package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"jetwash/internal/cache"
	"jetwash/internal/service/layer1_speed"
	"jetwash/internal/service/layer3_reason"
)

// LLMReviewTask LLM 异步审核任务
type LLMReviewTask struct {
	ReviewID  string                    `json:"review_id"`
	TenantID  string                    `json:"tenant_id"`
	Text      string                    `json:"text"`
	Matches   []layer3_reason.MatchInfo `json:"matches"`
	MaxRisk   int                       `json:"max_risk"`
	CreatedAt int64                     `json:"created_at"`
}

// LLMReviewWorker LLM 异步审核 Worker
// 消费 Redis Stream 中的审核任务，调用 LLM 推理，发现新词后更新 AC 自动机和 DB
type LLMReviewWorker struct {
	layer3Service layer3_reason.Layer3Service
	layer1Service layer1_speed.Layer1Service
	redisClient   *cache.RedisClient
	wordRepo      WordRepository
	logger        *zap.Logger
	consumerName  string
}

// WordRepository 词库仓库接口（用于新词入库）
type WordRepository interface {
	CheckWordExists(tenantID uuid.UUID, word string) (bool, error)
}

// NewLLMReviewWorker 创建 LLM 审核 Worker
func NewLLMReviewWorker(
	layer3Service layer3_reason.Layer3Service,
	layer1Service layer1_speed.Layer1Service,
	redisClient *cache.RedisClient,
	wordRepo WordRepository,
	logger *zap.Logger,
) *LLMReviewWorker {
	return &LLMReviewWorker{
		layer3Service: layer3Service,
		layer1Service: layer1Service,
		redisClient:   redisClient,
		wordRepo:      wordRepo,
		logger:        logger,
		consumerName:  fmt.Sprintf("worker-%s", uuid.New().String()[:8]),
	}
}

// Start 启动 Worker 消费循环
func (w *LLMReviewWorker) Start(ctx context.Context) {
	// 初始化消费组
	if err := w.redisClient.XGroupCreate(ctx, cache.LLMReviewStream, cache.LLMReviewGroup, "0"); err != nil {
		if w.logger != nil {
			w.logger.Warn("Failed to create LLM review consumer group",
				zap.Error(err))
		}
	}

	if w.logger != nil {
		w.logger.Info("LLM Review Worker started",
			zap.String("consumer", w.consumerName),
			zap.String("stream", cache.LLMReviewStream))
	}

	for {
		select {
		case <-ctx.Done():
			if w.logger != nil {
				w.logger.Info("LLM Review Worker shutting down")
			}
			return
		default:
			w.processOne(ctx)
		}
	}
}

// processOne 处理一条审核任务
func (w *LLMReviewWorker) processOne(ctx context.Context) {
	streams, err := w.redisClient.XReadGroup(ctx, cache.LLMReviewStream, cache.LLMReviewGroup, w.consumerName, 1)
	if err != nil {
		if err != redis.Nil {
			if w.logger != nil {
				w.logger.Debug("No messages in LLM review stream")
			}
		} else {
			if w.logger != nil {
				w.logger.Warn("Failed to read from LLM review stream",
					zap.Error(err))
			}
		}
		return
	}

	for _, stream := range streams {
		for _, message := range stream.Messages {
			w.handleMessage(ctx, message.ID, message.Values)
			// 确认消息
			if err := w.redisClient.XAck(ctx, cache.LLMReviewStream, cache.LLMReviewGroup, message.ID); err != nil {
				if w.logger != nil {
					w.logger.Warn("Failed to ACK LLM review message",
						zap.String("msgID", message.ID),
						zap.Error(err))
				}
			}
		}
	}
}

// handleMessage 处理单条消息
func (w *LLMReviewWorker) handleMessage(ctx context.Context, msgID string, values map[string]interface{}) {
	_ = ctx
	start := time.Now()

	// 解析任务数据
	dataStr, ok := values["data"].(string)
	if !ok {
		if w.logger != nil {
			w.logger.Warn("Invalid LLM review task: missing data field", zap.String("msgID", msgID))
		}
		return
	}

	var task LLMReviewTask
	if err := json.Unmarshal([]byte(dataStr), &task); err != nil {
		if w.logger != nil {
			w.logger.Warn("Failed to unmarshal LLM review task",
				zap.String("msgID", msgID),
				zap.Error(err))
		}
		return
	}

	tenantID, err := uuid.Parse(task.TenantID)
	if err != nil {
		if w.logger != nil {
			w.logger.Warn("Invalid tenant ID in LLM review task",
				zap.String("msgID", msgID),
				zap.String("tenantID", task.TenantID),
				zap.Error(err))
		}
		return
	}

	// 调用 LLM 推理
	layer3Result, err := w.layer3Service.ReasonWithMatches(tenantID, task.Text, task.Matches, nil)
	if err != nil {
		if w.logger != nil {
			w.logger.Warn("LLM review failed",
				zap.String("reviewID", task.ReviewID),
				zap.String("tenantID", tenantID.String()),
				zap.Error(err))
		}
		return
	}

	// 如果 LLM 发现了新的违禁词且判断有风险，更新 AC 自动机和 DB
	if layer3Result.HasRisk && !layer3Result.IsApproved && len(layer3Result.DetectedWords) > 0 {
		for _, word := range layer3Result.DetectedWords {
			word = trimWord(word)
			if word == "" {
				continue
			}

			// 检查词是否已存在
			exists := false
			if w.wordRepo != nil {
				exists, _ = w.wordRepo.CheckWordExists(tenantID, word)
			}

			if !exists {
				// 增量更新 AC 自动机
				if w.layer1Service != nil {
					if addErr := w.layer1Service.AddWord(word, &layer1_speed.Payload{
						TenantID:  tenantID,
						WordText:  word,
						Category:  "LLM",
						RiskLevel: layer3Result.RiskLevel,
					}); addErr != nil {
						if w.logger != nil {
							w.logger.Warn("Failed to add word to AC automaton",
								zap.String("word", word),
								zap.Error(addErr))
						}
					}
				}

				// 更新 Redis 敏感词集合
				if w.redisClient != nil {
					redisKey := fmt.Sprintf("sensitive_words:%s", tenantID.String())
					w.redisClient.SAdd(context.Background(), redisKey, word)
				}

				if w.logger != nil {
					w.logger.Info("LLM detected new sensitive word, added to AC automaton",
						zap.String("word", word),
						zap.String("reviewID", task.ReviewID),
						zap.String("tenantID", tenantID.String()))
				}
			}
		}
	}

	if w.logger != nil {
		w.logger.Info("LLM review completed",
			zap.String("reviewID", task.ReviewID),
			zap.String("tenantID", tenantID.String()),
			zap.Bool("hasRisk", layer3Result.HasRisk),
			zap.Int("riskLevel", layer3Result.RiskLevel),
			zap.Int("detectedWords", len(layer3Result.DetectedWords)),
			zap.Duration("duration", time.Since(start)))
	}
}

// trimWord 清理词语
func trimWord(word string) string {
	// 简单清理：去除空格和方括号
	word = trimAll(word, " ", "[", "]", "\"", "'")
	return word
}

func trimAll(s string, cuts ...string) string {
	for _, c := range cuts {
		s = replaceAll(s, c, "")
	}
	return s
}

func replaceAll(s, old, new string) string {
	result := ""
	for i := 0; i < len(s); i++ {
		if i+len(old) <= len(s) && s[i:i+len(old)] == old {
			result += new
			i += len(old) - 1
		} else {
			result += string(s[i])
		}
	}
	return result
}
