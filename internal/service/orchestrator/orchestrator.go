package orchestrator

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"jetwash/internal/repository"
	"jetwash/internal/service/detection_history"
	"jetwash/internal/service/layer1_speed"
	"jetwash/internal/service/layer2_semantic"
	"jetwash/internal/service/layer3_reason"
	"jetwash/internal/types"
)

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
}

// NewOrchestrator 创建编排器实例
func NewOrchestrator(
	layer1Service layer1_speed.Layer1Service,
	layer2Service layer2_semantic.Layer2Service,
	layer3Service layer3_reason.Layer3Service,
	wordRepo repository.WordRepository,
	detectionHistoryService detection_history.DetectionHistoryService,
) Orchestrator {
	return &orchestrator{
		layer1Service:           layer1Service,
		layer2Service:           layer2Service,
		layer3Service:           layer3Service,
		wordRepo:                wordRepo,
		detectionHistoryService: detectionHistoryService,
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
func (o *orchestrator) CheckTextWithConfigAndContext(tenantID uuid.UUID, text string, config *types.OrchestratorConfig, context *layer3_reason.ReasonContext) (*types.OrchestratorResult, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
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

	defer func() {
		duration := time.Since(startTime).Milliseconds()
		if o.detectionHistoryService != nil {
			_ = o.detectionHistoryService.SaveDetectionHistory(tenantID, text, mode, result, duration)
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

	if context == nil {
		context = layer3_reason.NewReasonContext()
	}

	// 初始化Layer1 AC自动机，加载租户的敏感词
	if config.EnableLayer1 {
		words, err := o.wordRepo.GetAllSensitiveWordsByTenant(tenantID)
		if err != nil {
			return nil, fmt.Errorf("failed to load sensitive words for layer1: %w", err)
		}
		if err := o.layer1Service.Initialize(tenantID, words); err != nil {
			return nil, fmt.Errorf("failed to initialize layer1 automaton: %w", err)
		}

		wordTexts := make([]string, len(words))
		for i, w := range words {
			wordTexts[i] = w.WordText
		}
		o.layer1Service.RegisterSegmenterWords(wordTexts)
	}

	// Layer 1: 快速匹配层
	if config.EnableLayer1 {
		layer1Result, err := o.layer1Service.CheckText(tenantID, text)
		if err != nil {
			return nil, fmt.Errorf("layer1 check failed: %w", err)
		}

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

	// Layer 2: 语义检索层
	if config.EnableLayer2 {
		layer2Result, err := o.layer2Service.SemanticSearch(
			tenantID,
			text,
			config.Layer2Threshold,
			config.Layer2Limit,
			nil, // filters
		)
		if err != nil {
			return nil, fmt.Errorf("layer2 check failed: %w", err)
		}

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
		layer3Result, err := o.layer3Service.ReasonWithMatches(tenantID, text, matches, context)
		if err != nil {
			return nil, fmt.Errorf("layer3 check failed: %w", err)
		}

		result.Layer3Result = layer3Result

		// 根据推理结果更新最终结果
		if layer3Result.HasRisk {
			result.Passed = !layer3Result.IsApproved
			result.RiskLevel = layer3Result.RiskLevel
			result.Message = layer3Result.RiskReason
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
