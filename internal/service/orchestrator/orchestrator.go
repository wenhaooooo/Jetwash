package orchestrator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"jetwash/internal/cache"
	"jetwash/internal/models"
	"jetwash/internal/repository"
	"jetwash/internal/service/detection_history"
	"jetwash/internal/service/layer1_speed"
	"jetwash/internal/service/layer2_semantic"
	"jetwash/internal/service/layer3_reason"
	"jetwash/internal/types"
)

// generateCacheKey 生成缓存键（使用 SHA256 哈希）
func generateCacheKey(tenantID uuid.UUID, text string) string {
	hash := sha256.Sum256([]byte(text))
	return fmt.Sprintf("detection:%s:%s", tenantID, hex.EncodeToString(hash[:]))
}

// DefaultOrchestratorConfig 默认编排配置
func DefaultOrchestratorConfig() *types.OrchestratorConfig {
	return &types.OrchestratorConfig{
		EnableLayer1:               true,
		EnableLayer2:               true,
		EnableLayer3:               true,
		StopAtLayer1:               false,
		StopAtLayer2:               false,
		EnableAmbiguityPassThrough: true,
		Layer2Threshold:            0.3,
		Layer2Limit:                10,
		Layer3EnableReason:         true,
		EnableFastPass:             true, // 默认开启非敏感词快速放行
		Layer3TimeoutMs:            3000, // LLM 推理超时 3 秒，超时后降级为规则判断
		EnableAsyncLLM:             true, // 默认开启异步 LLM 审核（通过 Redis Stream）
	}
}

// Orchestrator 编排器接口
type Orchestrator interface {
	// CheckText 检查文本（漏斗模式）
	CheckText(tenantID uuid.UUID, text string) (*types.OrchestratorResult, error)

	// CheckTextWithConfig 使用配置检查文本
	CheckTextWithConfig(tenantID uuid.UUID, text string, config *types.OrchestratorConfig) (*types.OrchestratorResult, error)

	// CheckTextWithContext 使用上下文检查文本
	CheckTextWithContext(tenantID uuid.UUID, text string, context *layer3_reason.ReasonContext) (*types.OrchestratorResult, error)

	// CheckTextWithConfigAndContext 使用配置和上下文检查文本
	CheckTextWithConfigAndContext(tenantID uuid.UUID, text string, config *types.OrchestratorConfig, context *layer3_reason.ReasonContext) (*types.OrchestratorResult, error)

	// Warmup 预热所有服务
	Warmup(tenantIDs []uuid.UUID) error
}

// orchestrator 编排器实现
type orchestrator struct {
	layer1Service           layer1_speed.Layer1Service
	layer2Service           layer2_semantic.Layer2Service
	layer3Service           layer3_reason.Layer3Service
	wordRepo                repository.WordRepository
	detectionHistoryService detection_history.DetectionHistoryService
	redisClient             *cache.RedisClient
	logger                  *zap.Logger
	layer1Cache             sync.Map // key: tenantID string, value: layer1_cache_entry

	// 异步写入：将检测历史、缓存写入等非关键路径操作从请求链路中剥离
	// 通过带缓冲 channel + 后台 worker 实现，避免 DB 写入阻塞检测响应
	historyChan chan *asyncTask
}

// asyncTask 异步任务，包含检测历史写入和缓存写入
type asyncTask struct {
	tenantID           uuid.UUID
	text               string
	mode               string
	result             *types.OrchestratorResult
	duration           int64
	newlyDetectedWords []string
}

// layer1_cache_entry Layer1 缓存条目
type layer1_cache_entry struct {
	lastUpdated time.Time
}

// NewOrchestrator 创建编排器实例
func NewOrchestrator(
	layer1Service layer1_speed.Layer1Service,
	layer2Service layer2_semantic.Layer2Service,
	layer3Service layer3_reason.Layer3Service,
	wordRepo repository.WordRepository,
	detectionHistoryService detection_history.DetectionHistoryService,
	redisClient *cache.RedisClient,
	logger *zap.Logger,
) Orchestrator {
	o := &orchestrator{
		layer1Service:           layer1Service,
		layer2Service:           layer2Service,
		layer3Service:           layer3Service,
		wordRepo:                wordRepo,
		detectionHistoryService: detectionHistoryService,
		redisClient:             redisClient,
		logger:                  logger,
		historyChan:             make(chan *asyncTask, 1000), // 带缓冲 channel，避免阻塞发送方
	}
	// 启动后台 worker 消费异步任务（检测历史写入、缓存写入等）
	go o.asyncWorker()
	return o
}

// asyncWorker 后台 worker，异步消费检测历史写入和缓存写入任务
// 将 DB 写入从请求关键路径中剥离，避免阻塞检测响应
func (o *orchestrator) asyncWorker() {
	for task := range o.historyChan {
		// 1. 异步写入检测历史（DB 事务）
		if o.detectionHistoryService != nil {
			if err := o.detectionHistoryService.SaveDetectionHistory(
				task.tenantID, task.text, task.mode, task.result, task.duration,
			); err != nil {
				if o.logger != nil {
					o.logger.Warn("Failed to save detection history (async)",
						zap.String("tenantID", task.tenantID.String()),
						zap.Int("textLength", len(task.text)),
						zap.Error(err))
				}
			}
		}

		// 2. 异步写入新检测到的违禁词到 DB 和 Redis
		if len(task.newlyDetectedWords) > 0 {
			o.addDetectedWordsToDatabase(task.tenantID, task.newlyDetectedWords)
			o.addDetectedWordsToRedis(task.tenantID, task.newlyDetectedWords)
		}

		// 3. 异步写入缓存结果
		if o.redisClient != nil {
			cacheKey := generateCacheKey(task.tenantID, task.text)
			if cachedResult, err := json.Marshal(task.result); err == nil {
				ttl := o.getCacheTTL(task.result)
				if err := o.redisClient.Set(context.Background(), cacheKey, cachedResult, ttl); err != nil {
					if o.logger != nil {
						o.logger.Warn("Failed to cache detection result (async)",
							zap.String("tenantID", task.tenantID.String()),
							zap.String("cacheKey", cacheKey),
							zap.Error(err))
					}
				}
			}
		}
	}
}

// CheckText 检查文本（漏斗模式）
func (o *orchestrator) CheckText(tenantID uuid.UUID, text string) (*types.OrchestratorResult, error) {
	config := DefaultOrchestratorConfig()
	return o.CheckTextWithConfig(tenantID, text, config)
}

// CheckTextWithConfig 使用配置检查文本
func (o *orchestrator) CheckTextWithConfig(tenantID uuid.UUID, text string, config *types.OrchestratorConfig) (*types.OrchestratorResult, error) {
	return o.CheckTextWithConfigAndContext(tenantID, text, config, nil)
}

// CheckTextWithContext 使用上下文检查文本
func (o *orchestrator) CheckTextWithContext(tenantID uuid.UUID, text string, context *layer3_reason.ReasonContext) (*types.OrchestratorResult, error) {
	config := DefaultOrchestratorConfig()
	return o.CheckTextWithConfigAndContext(tenantID, text, config, context)
}

// CheckTextWithConfigAndContext 使用配置和上下文检查文本
func (o *orchestrator) CheckTextWithConfigAndContext(tenantID uuid.UUID, text string, config *types.OrchestratorConfig, reasonContext *layer3_reason.ReasonContext) (*types.OrchestratorResult, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	// 尝试从缓存获取结果
	if o.redisClient != nil {
		cacheKey := generateCacheKey(tenantID, text)
		cachedResult, err := o.redisClient.GetWithRefresh(context.Background(), cacheKey, 1*time.Hour)
		if err == nil {
			if o.logger != nil {
				o.logger.Debug("Cache hit for text detection",
					zap.String("tenantID", tenantID.String()),
					zap.Int("textLength", len(text)))
			}
			var result types.OrchestratorResult
			if err := json.Unmarshal([]byte(cachedResult), &result); err == nil {
				result.FromCache = true
				return &result, nil
			}
		} else {
			if o.logger != nil {
				o.logger.Debug("Cache miss for text detection",
					zap.String("tenantID", tenantID.String()),
					zap.Int("textLength", len(text)))
			}
		}
	}

	startTime := time.Now()
	mode := "standard"
	if config != nil {
		if config.EnableLayer1 && config.EnableLayer2 && config.EnableLayer3 {
			mode = "full"
		} else if config.EnableLayer1 && !config.EnableLayer2 && !config.EnableLayer3 {
			mode = "layer1_only"
		} else if config.EnableLayer1 && config.EnableLayer2 && !config.EnableLayer3 {
			mode = "layer1_layer2"
		}
	}

	result := &types.OrchestratorResult{
		Passed:       true,
		RiskLevel:    0,
		Message:      "文本审查通过",
		TotalMatches: 0,
	}

	// 存储新检测到的违禁词，用于后续添加到数据库和Redis
	var newlyDetectedWords []string

	defer func() {
		duration := time.Since(startTime).Milliseconds()

		// 异步写入：将检测历史、新词入库、缓存写入等非关键操作发送到后台 worker
		// 通过带缓冲 channel 实现，不阻塞检测响应返回
		// 设计依据：SaveDetectionHistory 包含 DB 事务（2+ 次 SQL 写入），
		// 同步执行会增加 50~100ms 延迟；异步化后请求链路仅增加 channel 发送开销（< 1μs）
		task := &asyncTask{
			tenantID:           tenantID,
			text:               text,
			mode:               mode,
			result:             result,
			duration:           duration,
			newlyDetectedWords: newlyDetectedWords,
		}
		select {
		case o.historyChan <- task:
			// 成功发送到异步队列
		default:
			// channel 已满（极端高并发），记录警告但不阻塞请求
			if o.logger != nil {
				o.logger.Warn("Async task channel full, dropping detection history",
					zap.String("tenantID", tenantID.String()),
					zap.Int("textLength", len(text)))
			}
		}
	}()

	// 获取默认配置
	defaultConfig := DefaultOrchestratorConfig()

	// 如果用户提供了配置，合并用户配置和默认配置
	if config != nil {
		mergedConfig := *defaultConfig
		mergedConfig.EnableLayer1 = config.EnableLayer1
		mergedConfig.EnableLayer2 = config.EnableLayer2
		mergedConfig.EnableLayer3 = config.EnableLayer3
		mergedConfig.StopAtLayer1 = config.StopAtLayer1
		mergedConfig.StopAtLayer2 = config.StopAtLayer2
		mergedConfig.EnableAmbiguityPassThrough = config.EnableAmbiguityPassThrough
		if config.Layer2Threshold > 0 {
			mergedConfig.Layer2Threshold = config.Layer2Threshold
		}
		if config.Layer2Limit > 0 {
			mergedConfig.Layer2Limit = config.Layer2Limit
		}
		mergedConfig.Layer3EnableReason = config.Layer3EnableReason
		mergedConfig.EnableFastPass = config.EnableFastPass
		mergedConfig.Layer3TimeoutMs = config.Layer3TimeoutMs
		mergedConfig.EnableAsyncLLM = config.EnableAsyncLLM
		config = &mergedConfig
	} else {
		config = defaultConfig
	}

	if reasonContext == nil {
		reasonContext = layer3_reason.NewReasonContext()
	}

	// 初始化Layer1 AC自动机，加载租户的敏感词（使用缓存）
	if config.EnableLayer1 {
		if err := o.ensureLayer1Initialized(tenantID); err != nil {
			return nil, fmt.Errorf("failed to ensure layer1 initialized: %w", err)
		}
	}

	// ========== 非敏感词快速放行（Fast Pass） ==========
	// 当 EnableFastPass 开启时，先单独执行 Layer1：
	//   - 如果 Layer1 无匹配 → 直接返回通过，跳过 Layer2/3（节省 Embedding API 开销）
	//   - 如果 Layer1 有匹配 → 继续并发执行 Layer2，进入完整漏斗流程
	// 设计依据：实际场景中 95%+ 的文本都是非敏感词，Layer1 O(n) 匹配仅需 ~5ms，
	// 而 Layer2 的 Embedding 调用需要 50~300ms，对非敏感词来说是纯浪费。
	if config.EnableFastPass && config.EnableLayer1 {
		layer1Start := time.Now()
		layer1Result, layer1Err := o.layer1Service.CheckText(tenantID, text)
		if o.logger != nil {
			o.logger.Debug("Layer1 fast-pass check completed",
				zap.String("tenantID", tenantID.String()),
				zap.Duration("duration", time.Since(layer1Start)))
		}
		if layer1Err != nil {
			return nil, fmt.Errorf("layer1 fast-pass check failed: %w", layer1Err)
		}

		result.Layer1Result = layer1Result
		result.TotalMatches += len(layer1Result.MatchedWords)

		// Layer1 无匹配 → 非敏感词快速放行
		if !layer1Result.HasMatch {
			result.Passed = true
			result.RiskLevel = 0
			result.Message = "文本审查通过（快速放行）"
			return result, nil
		}

		// Layer1 有匹配 → 检查是否可以直接在 Layer1 停止
		allAmbiguous := layer1Result.HasAmbiguity &&
			len(layer1Result.AmbiguousMatches) == len(layer1Result.MatchedWords)
		if config.StopAtLayer1 && !allAmbiguous {
			result.Passed = false
			result.RiskLevel = layer1Result.RiskLevel
			result.Message = fmt.Sprintf("文本包含敏感内容（Layer1 匹配）: %v", layer1Result.Categories)
			return result, nil
		}

		// Layer1 有匹配但需要继续 → 仅启动 Layer2（Layer1 已完成，无需重复执行）
		if config.EnableLayer2 {
			var layer2Err error
			var layer2Result *layer2_semantic.Layer2Result
			layer2Start := time.Now()
			layer2Result, layer2Err = o.layer2Service.SemanticSearch(
				tenantID,
				text,
				config.Layer2Threshold,
				config.Layer2Limit,
				nil, // filters
			)
			if o.logger != nil {
				o.logger.Debug("Layer2 execution completed (after fast-pass)",
					zap.String("tenantID", tenantID.String()),
					zap.Duration("duration", time.Since(layer2Start)))
			}
			if layer2Err != nil {
				return nil, fmt.Errorf("layer2 check failed: %w", layer2Err)
			}

			// 处理 Layer2 结果
			result.Layer2Result = layer2Result
			result.TotalMatches += layer2Result.TotalMatches
			if layer2Result.HasMatch && config.StopAtLayer2 {
				result.Passed = false
				result.RiskLevel = layer2Result.RiskLevel
				result.Message = fmt.Sprintf("文本包含敏感内容（Layer2 匹配）: %v", layer2Result.Categories)
				return result, nil
			}
		}
	} else {
		// ========== 原始流程：Layer1 和 Layer2 并发执行 ==========
		var wg sync.WaitGroup
		var layer1Err, layer2Err error
		var layer1Result *layer1_speed.Layer1Result
		var layer2Result *layer2_semantic.Layer2Result

		// Layer 1: 快速匹配层
		if config.EnableLayer1 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				layer1Start := time.Now()
				result, err := o.layer1Service.CheckText(tenantID, text)
				if o.logger != nil {
					o.logger.Debug("Layer1 execution completed",
						zap.String("tenantID", tenantID.String()),
						zap.Duration("duration", time.Since(layer1Start)))
				}
				if err != nil {
					layer1Err = fmt.Errorf("layer1 check failed: %w", err)
					return
				}
				layer1Result = result
			}()
		}

		// Layer 2: 语义检索层
		if config.EnableLayer2 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				layer2Start := time.Now()
				result, err := o.layer2Service.SemanticSearch(
					tenantID,
					text,
					config.Layer2Threshold,
					config.Layer2Limit,
					nil, // filters
				)
				if o.logger != nil {
					o.logger.Debug("Layer2 execution completed",
						zap.String("tenantID", tenantID.String()),
						zap.Duration("duration", time.Since(layer2Start)))
				}
				if err != nil {
					layer2Err = fmt.Errorf("layer2 check failed: %w", err)
					return
				}
				layer2Result = result
			}()
		}

		// 等待Layer1和Layer2完成
		wg.Wait()

		// 检查错误
		if layer1Err != nil {
			return nil, layer1Err
		}
		if layer2Err != nil {
			return nil, layer2Err
		}

		// 处理Layer1结果
		if config.EnableLayer1 && layer1Result != nil {
			result.Layer1Result = layer1Result
			result.TotalMatches += len(layer1Result.MatchedWords)

			if layer1Result.HasMatch {
				allAmbiguous := layer1Result.HasAmbiguity &&
					len(layer1Result.AmbiguousMatches) == len(layer1Result.MatchedWords)

				shouldStop := config.StopAtLayer1
				if allAmbiguous && config.EnableAmbiguityPassThrough {
					shouldStop = false
				}

				if shouldStop && !allAmbiguous {
					result.Passed = false
					result.RiskLevel = layer1Result.RiskLevel
					result.Message = fmt.Sprintf("文本包含敏感内容（Layer1 匹配）: %v", layer1Result.Categories)
					return result, nil
				}
			}
		}

		// 处理Layer2结果
		if config.EnableLayer2 && layer2Result != nil {
			result.Layer2Result = layer2Result
			result.TotalMatches += layer2Result.TotalMatches

			// 如果第二层匹配到敏感词，且配置为在第二层停止
			if layer2Result.HasMatch && config.StopAtLayer2 {
				result.Passed = false
				result.RiskLevel = layer2Result.RiskLevel
				result.Message = fmt.Sprintf("文本包含敏感内容（Layer2 匹配）: %v", layer2Result.Categories)
				return result, nil
			}
		}
	}

	// Layer 3: 推理层
	if config.EnableLayer3 {
		// 收集所有匹配信息
		matches := o.collectMatches(result)

		// 在调用LLM之前，先检查是否有高风险匹配
		maxRiskLevel := 0
		for _, match := range matches {
			if match.RiskLevel > maxRiskLevel {
				maxRiskLevel = match.RiskLevel
			}
		}

		// 如果有高风险匹配（风险等级>=4），直接拒绝，不依赖LLM判断
		if maxRiskLevel >= 4 {
			result.Passed = false
			result.RiskLevel = maxRiskLevel
			result.Message = fmt.Sprintf("文本包含高风险敏感内容（风险等级: %d）", maxRiskLevel)
			result.ReviewStatus = "completed"
			return result, nil
		}

		// 如果没有匹配到任何内容，无需 LLM 审查
		if len(matches) == 0 {
			result.Passed = true
			result.RiskLevel = 0
			result.Message = "文本审查通过"
			result.ReviewStatus = "completed"
			return result, nil
		}

		// ========== 异步 LLM 审核（EnableAsyncLLM） ==========
		// 将 LLM 推理任务发送到 Redis Stream，由后台 worker 异步执行
		// API 立即返回初步结果（基于 Layer1/2 匹配），LLM 完成后更新 AC 自动机和 DB
		if config.EnableAsyncLLM && o.redisClient != nil {
			reviewID := uuid.New().String()

			// 构建审核任务消息
			taskData, _ := json.Marshal(map[string]interface{}{
				"review_id":  reviewID,
				"tenant_id":  tenantID.String(),
				"text":       text,
				"matches":    matches,
				"max_risk":   maxRiskLevel,
				"created_at": time.Now().Unix(),
			})

			if _, err := o.redisClient.XAdd(context.Background(), cache.LLMReviewStream, map[string]interface{}{
				"data": string(taskData),
			}); err != nil {
				// Stream 发送失败，降级为同步超时模式
				if o.logger != nil {
					o.logger.Warn("Failed to send LLM review task to stream, falling back to sync mode",
						zap.String("tenantID", tenantID.String()),
						zap.Error(err))
				}
			} else {
				// 异步发送成功，返回初步结果
				if maxRiskLevel >= 3 {
					result.Passed = false
					result.RiskLevel = maxRiskLevel
					result.Message = fmt.Sprintf("文本包含敏感内容（初步结果，LLM异步审核中，风险等级: %d）", maxRiskLevel)
				} else {
					result.Passed = true
					result.RiskLevel = 0
					result.Message = "文本审查通过（初步结果，LLM异步审核中）"
				}
				result.ReviewID = reviewID
				result.ReviewStatus = "pending_llm_review"

				if o.logger != nil {
					o.logger.Debug("LLM review task sent to stream",
						zap.String("reviewID", reviewID),
						zap.String("tenantID", tenantID.String()))
				}
				return result, nil
			}
		}

		// ========== 同步 LLM 调用（超时控制） ==========
		layer3Start := time.Now()
		var layer3Result *layer3_reason.Layer3Result
		var layer3Err error

		if config.Layer3TimeoutMs > 0 {
			// 带超时的 LLM 调用：超时后降级为基于已有匹配结果的规则判断
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Layer3TimeoutMs)*time.Millisecond)
			defer cancel()

			layer3Done := make(chan *layer3_reason.Layer3Result, 1)
			layer3ErrChan := make(chan error, 1)

			go func() {
				r, err := o.layer3Service.ReasonWithMatches(ctx, tenantID, text, matches, reasonContext)
				if err != nil {
					layer3ErrChan <- err
				} else {
					layer3Done <- r
				}
			}()

			select {
			case r := <-layer3Done:
				layer3Result = r
			case err := <-layer3ErrChan:
				layer3Err = err
			case <-ctx.Done():
				// LLM 超时，降级处理：基于已有匹配结果做规则判断
				if o.logger != nil {
					o.logger.Warn("Layer3 LLM timeout, falling back to rule-based judgment",
						zap.String("tenantID", tenantID.String()),
						zap.Int("timeoutMs", config.Layer3TimeoutMs),
						zap.Duration("actualDuration", time.Since(layer3Start)))
				}
				// 降级策略：如果有中等风险匹配（>=3），拒绝；否则通过
				if maxRiskLevel >= 3 {
					result.Passed = false
					result.RiskLevel = maxRiskLevel
					result.Message = fmt.Sprintf("文本包含敏感内容（LLM超时降级，基于规则判断，风险等级: %d）", maxRiskLevel)
				} else if maxRiskLevel > 0 {
					// 有低风险匹配但未达阈值，保守放行
					result.Passed = true
					result.RiskLevel = 0
					result.Message = "文本审查通过（LLM超时降级，低风险匹配未达阈值）"
				}
				// LLM 超时不视为错误，直接返回降级结果
				return result, nil
			}
		} else {
			// 无超时限制，直接调用
			layer3Result, layer3Err = o.layer3Service.ReasonWithMatches(context.Background(), tenantID, text, matches, reasonContext)
		}

		if o.logger != nil {
			o.logger.Debug("Layer3 execution completed",
				zap.String("tenantID", tenantID.String()),
				zap.Duration("duration", time.Since(layer3Start)))
		}
		if layer3Err != nil {
			return nil, fmt.Errorf("layer3 check failed: %w", layer3Err)
		}

		result.Layer3Result = layer3Result

		// 根据推理结果更新最终结果
		if layer3Result.HasRisk {
			result.Passed = !layer3Result.IsApproved
			result.RiskLevel = layer3Result.RiskLevel
			result.Message = layer3Result.RiskReason

			// 如果LLM检测到新的违禁词且判断文本有风险，记录这些违禁词
			if len(layer3Result.DetectedWords) > 0 && !layer3Result.IsApproved {
				newlyDetectedWords = layer3Result.DetectedWords
			}
		} else if maxRiskLevel >= 3 {
			// 如果LLM判断无风险，但有中等风险匹配（风险等级>=3），仍然拒绝
			result.Passed = false
			result.RiskLevel = maxRiskLevel
			result.Message = fmt.Sprintf("文本包含中等风险敏感内容（风险等级: %d）", maxRiskLevel)
		}
	}

	// 如果没有匹配到任何敏感词，返回通过结果
	if result.TotalMatches == 0 {
		result.Passed = true
		result.RiskLevel = 0
		result.Message = "文本审查通过"
	}

	return result, nil
}

// collectMatches 收集所有层的匹配信息
func (o *orchestrator) collectMatches(result *types.OrchestratorResult) []layer3_reason.MatchInfo {
	matches := make([]layer3_reason.MatchInfo, 0)

	// 收集 Layer1 的匹配信息
	if result.Layer1Result != nil && result.Layer1Result.HasMatch {
		for _, match := range result.Layer1Result.MatchedWords {
			matches = append(matches, layer3_reason.MatchInfo{
				WordText:  match.Matched,
				Category:  match.Payload.Category,
				RiskLevel: match.Payload.RiskLevel,
				Distance:  0, // 精确匹配，距离为 0
				Position:  match.Position,
				MatchType: match.MatchType,
			})
		}
	}

	// 收集 Layer2 的匹配信息
	if result.Layer2Result != nil && result.Layer2Result.HasMatch {
		for _, match := range result.Layer2Result.MatchedWords {
			matches = append(matches, layer3_reason.MatchInfo{
				WordText:  match.WordText,
				Category:  match.Category,
				RiskLevel: match.RiskLevel,
				Distance:  match.Distance,
				Position:  -1, // 语义匹配没有位置信息
				MatchType: "semantic",
			})
		}
	}

	return matches
}

// AggregateRiskLevel 聚合风险等级
func (o *orchestrator) AggregateRiskLevel(result *types.OrchestratorResult) int {
	maxRiskLevel := 0

	if result.Layer1Result != nil && result.Layer1Result.RiskLevel > maxRiskLevel {
		maxRiskLevel = result.Layer1Result.RiskLevel
	}

	if result.Layer2Result != nil && result.Layer2Result.RiskLevel > maxRiskLevel {
		maxRiskLevel = result.Layer2Result.RiskLevel
	}

	if result.Layer3Result != nil && result.Layer3Result.RiskLevel > maxRiskLevel {
		maxRiskLevel = result.Layer3Result.RiskLevel
	}

	return maxRiskLevel
}

// AggregateCategories 聚合分类
func (o *orchestrator) AggregateCategories(result *types.OrchestratorResult) []string {
	categoryMap := make(map[string]bool)

	if result.Layer1Result != nil {
		for _, category := range result.Layer1Result.Categories {
			categoryMap[category] = true
		}
	}

	if result.Layer2Result != nil {
		for _, category := range result.Layer2Result.Categories {
			categoryMap[category] = true
		}
	}

	categories := make([]string, 0, len(categoryMap))
	for category := range categoryMap {
		categories = append(categories, category)
	}

	return categories
}

// BuildSummary 构建摘要
func (o *orchestrator) BuildSummary(result *types.OrchestratorResult) string {
	var summary string

	if result.Passed {
		summary = "文本审查通过，未发现敏感内容"
	} else {
		summary = fmt.Sprintf("文本审查未通过，风险等级: %d", result.RiskLevel)

		if result.Layer1Result != nil && result.Layer1Result.HasMatch {
			summary += fmt.Sprintf("，Layer1 匹配: %d 个", len(result.Layer1Result.MatchedWords))
		}

		if result.Layer2Result != nil && result.Layer2Result.HasMatch {
			summary += fmt.Sprintf("，Layer2 匹配: %d 个", result.Layer2Result.TotalMatches)
		}

		if result.Layer3Result != nil && result.Layer3Result.HasRisk {
			summary += fmt.Sprintf("，Layer3 风险: %s", result.Layer3Result.RiskReason)
		}
	}

	return summary
}

// addDetectedWordsToDatabase 将检测到的违禁词添加到数据库
func (o *orchestrator) addDetectedWordsToDatabase(tenantID uuid.UUID, words []string) {
	if o.wordRepo == nil {
		return
	}

	for _, wordText := range words {
		wordText = strings.TrimSpace(wordText)
		if wordText == "" {
			continue
		}

		// 检查违禁词是否已存在
		exists, err := o.wordRepo.CheckWordExists(tenantID, wordText)
		if err != nil {
			continue
		}
		if exists {
			continue
		}

		// 创建新的敏感词记录
		newWord := &models.SensitiveWord{
			TenantID:  tenantID,
			WordText:  wordText,
			Category:  "llm_detected", // 标记为LLM检测到的
			RiskLevel: 3,              // 默认中等风险
			Status:    1,              // 启用状态
		}

		if err := o.wordRepo.CreateSensitiveWord(newWord); err != nil {
			continue
		}

		// 增量更新 AC 自动机，使新词立即生效
		payload := &layer1_speed.Payload{
			TenantID:  tenantID,
			WordText:  wordText,
			Category:  "llm_detected",
			RiskLevel: 3,
		}

		if err := o.layer1Service.AddWord(wordText, payload); err != nil {
			if o.logger != nil {
				o.logger.Warn("Failed to add word to AC automaton",
					zap.String("tenantID", tenantID.String()),
					zap.String("word", wordText),
					zap.Error(err))
			}
		} else if o.logger != nil {
			o.logger.Debug("Successfully added new word to AC automaton",
				zap.String("tenantID", tenantID.String()),
				zap.String("word", wordText))
		}
	}
}

// addDetectedWordsToRedis 将检测到的违禁词添加到Redis缓存
func (o *orchestrator) addDetectedWordsToRedis(tenantID uuid.UUID, words []string) {
	if o.redisClient == nil {
		return
	}

	ctx := context.Background()

	// 更新敏感词列表缓存
	for _, wordText := range words {
		wordText = strings.TrimSpace(wordText)
		if wordText == "" {
			continue
		}

		// 添加到敏感词集合
		sensitiveWordsKey := fmt.Sprintf("sensitive_words:%s", tenantID)
		o.redisClient.SAdd(ctx, sensitiveWordsKey, wordText)

		// 设置过期时间为7天（与数据库同步频率一致）
		o.redisClient.Expire(ctx, sensitiveWordsKey, 7*24*time.Hour)
	}
}

// getCacheTTL 根据检测结果的风险等级返回合适的缓存过期时间
// 分层缓存策略：高风险结果保留更长时间用于审计，低风险结果保留较短时间
func (o *orchestrator) getCacheTTL(result *types.OrchestratorResult) time.Duration {
	if !result.Passed && result.RiskLevel >= 4 {
		// 高风险结果，保留7天（用于审计和合规）
		return 7 * 24 * time.Hour
	} else if !result.Passed {
		// 中等风险结果，保留1天
		return 24 * time.Hour
	} else {
		// 通过结果，保留1小时
		return 1 * time.Hour
	}
}

// ensureLayer1Initialized 确保 Layer1 自动机已初始化（使用缓存机制）
// 缓存有效期为10分钟，过期后自动重新加载
func (o *orchestrator) ensureLayer1Initialized(tenantID uuid.UUID) error {
	tenantIDStr := tenantID.String()
	cacheExpiry := 10 * time.Minute

	// 检查缓存
	if cachedEntry, ok := o.layer1Cache.Load(tenantIDStr); ok {
		entry := cachedEntry.(layer1_cache_entry)
		// 如果缓存未过期，直接返回
		if time.Since(entry.lastUpdated) < cacheExpiry {
			if o.logger != nil {
				o.logger.Debug("Using cached Layer1 automaton",
					zap.String("tenantID", tenantIDStr))
			}
			return nil
		}
		// 缓存过期，记录日志
		if o.logger != nil {
			o.logger.Debug("Layer1 automaton cache expired, reloading",
				zap.String("tenantID", tenantIDStr))
		}
	} else {
		// 缓存未命中，记录日志
		if o.logger != nil {
			o.logger.Debug("Layer1 automaton cache miss, loading",
				zap.String("tenantID", tenantIDStr))
		}
	}

	// 需要重新加载敏感词并初始化
	words, err := o.wordRepo.GetAllSensitiveWordsByTenant(tenantID)
	if err != nil {
		return fmt.Errorf("failed to load sensitive words for layer1: %w", err)
	}

	if err := o.layer1Service.Initialize(tenantID, words); err != nil {
		return fmt.Errorf("failed to initialize layer1 automaton: %w", err)
	}

	// 注册分词词典
	wordTexts := make([]string, len(words))
	for i, w := range words {
		wordTexts[i] = w.WordText
	}
	o.layer1Service.RegisterSegmenterWords(wordTexts)

	// 更新缓存
	o.layer1Cache.Store(tenantIDStr, layer1_cache_entry{
		lastUpdated: time.Now(),
	})

	return nil
}

// Warmup 预热所有服务
func (o *orchestrator) Warmup(tenantIDs []uuid.UUID) error {
	if o.logger != nil {
		o.logger.Info("Starting orchestrator warmup...")
	}

	startTime := time.Now()

	// 1. 预热 Layer1 AC 自动机（为每个租户加载敏感词）
	if o.logger != nil {
		o.logger.Info("Warming up Layer1 automaton...")
	}
	for _, tenantID := range tenantIDs {
		if err := o.ensureLayer1Initialized(tenantID); err != nil {
			if o.logger != nil {
				o.logger.Warn("Failed to warmup Layer1 for tenant",
					zap.String("tenantID", tenantID.String()),
					zap.Error(err))
			}
		} else if o.logger != nil {
			o.logger.Debug("Layer1 warmed up for tenant",
				zap.String("tenantID", tenantID.String()))
		}
	}

	// 2. 预热 LLM 模型（发送一个测试请求让模型加载到内存）
	if o.logger != nil {
		o.logger.Info("Warming up Layer3 LLM model...")
	}
	if o.layer3Service != nil {
		warmupStart := time.Now()
		// 发送一个简单的测试请求来预热模型
		_, err := o.layer3Service.ReasonText(
			context.Background(),
			uuid.New(),
			"预热测试",
			layer3_reason.NewReasonContext(),
		)
		if err != nil {
			if o.logger != nil {
				o.logger.Warn("Failed to warmup Layer3 LLM",
					zap.Error(err))
			}
		} else if o.logger != nil {
			o.logger.Info("Layer3 LLM warmed up",
				zap.Duration("duration", time.Since(warmupStart)))
		}
	}

	// 3. 预热 Layer2 语义检索（可选，会在首次请求时自动加载）
	if o.logger != nil {
		o.logger.Info("Layer2 semantic search will be warmed up on first request")
	}

	if o.logger != nil {
		o.logger.Info("Orchestrator warmup completed",
			zap.Duration("totalDuration", time.Since(startTime)))
	}

	return nil
}
