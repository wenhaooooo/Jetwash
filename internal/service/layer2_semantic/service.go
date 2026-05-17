package layer2_semantic

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"jetwash/internal/cache"
	"jetwash/internal/config"
	"jetwash/internal/util"
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"go.uber.org/zap"
)

// SemanticResult 语义检索结果
type SemanticResult struct {
	WordText  string    `json:"word_text"`
	Category  string    `json:"category"`
	RiskLevel int       `json:"risk_level"`
	Distance  float64   `json:"distance"`
	TenantID  uuid.UUID `json:"tenant_id"`
}

// Layer2Result 第二层结果
type Layer2Result struct {
	HasMatch     bool              `json:"has_match"`
	MatchedWords []*SemanticResult `json:"matched_words"`
	RiskLevel    int               `json:"risk_level"`
	Categories   []string          `json:"categories"`
	Threshold    float64           `json:"threshold"`
	TotalMatches int               `json:"total_matches"`
}

// SemanticRepository 语义检索仓库接口
type SemanticRepository interface {
	// SearchByVectorWithFilter 使用向量搜索，支持 Filter 条件
	SearchByVectorWithFilter(tenantID uuid.UUID, vector pgvector.Vector, threshold float64, limit int, filters map[string]interface{}) ([]*SemanticResult, error)
}

// Layer2Service 第二层服务接口 - 语义检索层
type Layer2Service interface {
	// SemanticSearch 语义检索
	SemanticSearch(tenantID uuid.UUID, text string, threshold float64, limit int, filters map[string]interface{}) (*Layer2Result, error)

	// SemanticSearchWithVector 使用向量进行语义检索
	SemanticSearchWithVector(tenantID uuid.UUID, vector pgvector.Vector, threshold float64, limit int, filters map[string]interface{}) (*Layer2Result, error)
}

// layer2Service 第二层服务实现
type layer2Service struct {
	repo              SemanticRepository
	embeddingProvider util.EmbeddingProvider
	cache             *cache.RedisClient
	logger            *zap.Logger
}

// NewLayer2Service 创建第二层服务实例
func NewLayer2Service(repo SemanticRepository, cfg *config.Config, redisCache *cache.RedisClient) Layer2Service {
	var embeddingProvider util.EmbeddingProvider

	// 根据配置选择embedding提供者
	if cfg.LLM.Provider == "ollama" {
		// 使用Ollama embedding提供者
		fullHost := fmt.Sprintf("%s:%d", cfg.LLM.Ollama.Host, cfg.LLM.Ollama.Port)
		embeddingProvider = util.NewOllamaEmbeddingProvider(
			fullHost,
			cfg.LLM.Ollama.EmbeddingModel,
			30*time.Second,
		)
	} else {
		// 使用在线embedding提供者
		embeddingProvider = util.NewOnlineEmbeddingProvider(
			cfg.LLM.Online.APIKey,
			cfg.LLM.Online.EmbeddingModel,
			cfg.LLM.Online.BaseURL,
			30*time.Second,
		)
	}

	return &layer2Service{
		repo:              repo,
		embeddingProvider: embeddingProvider,
		cache:             redisCache,
		logger:            zap.NewNop(),
	}
}

// SemanticSearch 语义检索
func (s *layer2Service) SemanticSearch(tenantID uuid.UUID, text string, threshold float64, limit int, filters map[string]interface{}) (*Layer2Result, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	if threshold < 0 || threshold > 1 {
		return nil, fmt.Errorf("threshold must be between 0 and 1")
	}

	if limit <= 0 || limit > 100 {
		return nil, fmt.Errorf("limit must be between 1 and 100")
	}

	// 生成文本的 SHA256 哈希作为缓存键
	hash := sha256.Sum256([]byte(text))
	textHash := hex.EncodeToString(hash[:])

	// 尝试从 Redis 缓存获取 embedding
	var fVec []float32
	ctx := context.Background()
	cachedEmbedding, err := s.cache.GetEmbeddingCache(ctx, textHash)
	if err == nil && len(cachedEmbedding) > 0 {
		// 缓存命中
		fVec = cachedEmbedding
	} else {
		// 缓存未命中，调用 embedding API
		fVec, err = s.embeddingProvider.GetEmbedding(text)
		if err != nil {
			return nil, fmt.Errorf("failed to get embedding: %w", err)
		}
		// 将结果存入缓存（异步，不阻塞主流程）
		go func() {
			cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if cacheErr := s.cache.SetEmbeddingCache(cacheCtx, textHash, fVec); cacheErr != nil {
				s.logger.Warn("Failed to cache embedding", zap.Error(cacheErr))
			}
		}()
	}

	vector := pgvector.NewVector(fVec)
	// 使用向量进行语义检索
	return s.SemanticSearchWithVector(tenantID, vector, threshold, limit, filters)
}

// SemanticSearchWithVector 使用向量进行语义检索
func (s *layer2Service) SemanticSearchWithVector(tenantID uuid.UUID, vector pgvector.Vector, threshold float64, limit int, filters map[string]interface{}) (*Layer2Result, error) {
	if threshold < 0 || threshold > 1 {
		return nil, fmt.Errorf("threshold must be between 0 and 1")
	}

	if limit <= 0 || limit > 100 {
		return nil, fmt.Errorf("limit must be between 1 and 100")
	}

	// 调用仓库层进行向量检索
	matchedWords, err := s.repo.SearchByVectorWithFilter(tenantID, vector, threshold, limit, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to search by vector: %w", err)
	}

	// 构建结果
	result := s.buildResult(matchedWords, threshold)

	return result, nil
}

// buildResult 构建结果
func (s *layer2Service) buildResult(matchedWords []*SemanticResult, threshold float64) *Layer2Result {
	result := &Layer2Result{
		HasMatch:     len(matchedWords) > 0,
		MatchedWords: matchedWords,
		RiskLevel:    0,
		Categories:   make([]string, 0),
		Threshold:    threshold,
		TotalMatches: len(matchedWords),
	}

	if !result.HasMatch {
		return result
	}

	// 计算最高风险等级
	categories := make(map[string]bool)
	for _, match := range matchedWords {
		if match.RiskLevel > result.RiskLevel {
			result.RiskLevel = match.RiskLevel
		}
		categories[match.Category] = true
	}

	// 提取分类列表
	for category := range categories {
		result.Categories = append(result.Categories, category)
	}

	return result
}

// FilterBuilder 过滤条件构建器
type FilterBuilder struct {
	filters map[string]interface{}
}

// NewFilterBuilder 创建过滤条件构建器
func NewFilterBuilder() *FilterBuilder {
	return &FilterBuilder{
		filters: make(map[string]interface{}),
	}
}

// WithCategory 添加分类过滤
func (fb *FilterBuilder) WithCategory(category string) *FilterBuilder {
	fb.filters["category"] = category
	return fb
}

// WithCategories 添加多个分类过滤（OR 关系）
func (fb *FilterBuilder) WithCategories(categories []string) *FilterBuilder {
	fb.filters["categories"] = categories
	return fb
}

// WithRiskLevel 添加风险等级过滤
func (fb *FilterBuilder) WithRiskLevel(riskLevel int) *FilterBuilder {
	fb.filters["risk_level"] = riskLevel
	return fb
}

// WithRiskLevelRange 添加风险等级范围过滤
func (fb *FilterBuilder) WithRiskLevelRange(min, max int) *FilterBuilder {
	fb.filters["risk_level_range"] = []int{min, max}
	return fb
}

// WithStatus 添加状态过滤
func (fb *FilterBuilder) WithStatus(status int) *FilterBuilder {
	fb.filters["status"] = status
	return fb

}

// Build 构建过滤条件
func (fb *FilterBuilder) Build() map[string]interface{} {
	return fb.filters
}

// Clear 清空过滤条件
func (fb *FilterBuilder) Clear() *FilterBuilder {
	fb.filters = make(map[string]interface{})
	return fb
}
