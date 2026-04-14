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
	return &orchestrator{
		layer1Service:           layer1Service,
		layer2Service:           layer2Service,
		layer3Service:           layer3Service,
		wordRepo:                wordRepo,
		detectionHistoryService: detectionHistoryService,
		redisClient:             redisClient,
		logger:                  logger,
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

		// 记录检测历史
		if o.detectionHistoryService != nil {
			if err := o.detectionHistoryService.SaveDetectionHistory(tenantID, text, mode, result, duration); err != nil {
				if o.logger != nil {
					o.logger.Warn("Failed to save detection history",
						zap.String("tenantID", tenantID.String()),
						zap.Int("textLength", len(text)),
						zap.Error(err))
				}
			}
		}

		// 如果有新检测到的违禁词，添加到数据库和Redis
		if len(newlyDetectedWords) > 0 {
			o.addDetectedWordsToDatabase(tenantID, newlyDetectedWords)
			o.addDetectedWordsToRedis(tenantID, newlyDetectedWords)
		}

		// 缓存结果
		if o.redisClient != nil {
			cacheKey := generateCacheKey(tenantID, text)
			if cachedResult, err := json.Marshal(result); err == nil {
				// 分层缓存策略：根据检测结果风险等级设置不同的过期时间
				ttl := o.getCacheTTL(result)
				if err := o.redisClient.Set(context.Background(), cacheKey, cachedResult, ttl); err != nil {
					if o.logger != nil {
						o.logger.Warn("Failed to cache detection result",
							zap.String("tenantID", tenantID.String()),
							zap.String("cacheKey", cacheKey),
							zap.Error(err))
					}
				}
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

	// 并发执行Layer1和Layer2
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
			return result, nil
		}

		// 调用推理层
		layer3Start := time.Now()
		layer3Result, err := o.layer3Service.ReasonWithMatches(tenantID, text, matches, reasonContext)
		if o.logger != nil {
			o.logger.Debug("Layer3 execution completed",
				zap.String("tenantID", tenantID.String()),
				zap.Duration("duration", time.Since(layer3Start)))
		}
		if err != nil {
			return nil, fmt.Errorf("layer3 check failed: %w", err)
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
