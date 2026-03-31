package layer2_semantic

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
)

// semanticRepository 语义检索仓库实现
type semanticRepository struct {
	db *gorm.DB
}

// NewSemanticRepository 创建语义检索仓库实例
func NewSemanticRepository(db *gorm.DB) SemanticRepository {
	return &semanticRepository{
		db: db,
	}
}

// SearchByVectorWithFilter 使用向量搜索，支持 Filter 条件
func (r *semanticRepository) SearchByVectorWithFilter(tenantID uuid.UUID, vector pgvector.Vector, threshold float64, limit int, filters map[string]interface{}) ([]*SemanticResult, error) {
	// 构建查询
	query := r.db.Table("sensitive_words").
		Select("word_text, category, risk_level, tenant_id, embedding <-> ? as distance", vector).
		Where("tenant_id = ?", tenantID).
		Where("status = ?", 1).
		Where("embedding <-> ? <= ?", vector, 1-threshold).
		Limit(limit)

	// 应用过滤条件
	query = r.applyFilters(query, filters)

	// 执行查询
	type Result struct {
		WordText  string    `gorm:"column:word_text"`
		Category  string    `gorm:"column:category"`
		RiskLevel int       `gorm:"column:risk_level"`
		TenantID  uuid.UUID `gorm:"column:tenant_id"`
		Distance  float64   `gorm:"column:distance"`
	}

	var results []Result
	if err := query.Find(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to search by vector: %w", err)
	}

	// 转换为 SemanticResult
	semanticResults := make([]*SemanticResult, len(results))
	for i, result := range results {
		semanticResults[i] = &SemanticResult{
			WordText:  result.WordText,
			Category:  result.Category,
			RiskLevel: result.RiskLevel,
			Distance:  result.Distance,
			TenantID:  result.TenantID,
		}
	}

	return semanticResults, nil
}

// applyFilters 应用过滤条件
func (r *semanticRepository) applyFilters(query *gorm.DB, filters map[string]interface{}) *gorm.DB {
	if filters == nil {
		return query
	}

	// 分类过滤
	if category, ok := filters["category"]; ok {
		query = query.Where("category = ?", category)
	}

	// 多分类过滤（OR 关系）
	if categories, ok := filters["categories"]; ok {
		if catSlice, ok := categories.([]string); ok && len(catSlice) > 0 {
			query = query.Where("category IN ?", catSlice)
		}
	}

	// 风险等级过滤
	if riskLevel, ok := filters["risk_level"]; ok {
		query = query.Where("risk_level = ?", riskLevel)
	}

	// 风险等级范围过滤
	if riskLevelRange, ok := filters["risk_level_range"]; ok {
		if rangeSlice, ok := riskLevelRange.([]int); ok && len(rangeSlice) == 2 {
			query = query.Where("risk_level BETWEEN ? AND ?", rangeSlice[0], rangeSlice[1])
		}
	}

	// 状态过滤
	if status, ok := filters["status"]; ok {
		query = query.Where("status = ?", status)
	}

	return query
}

// AdvancedSearch 高级检索，支持更多过滤条件
func (r *semanticRepository) AdvancedSearch(tenantID uuid.UUID, vector pgvector.Vector, options *SearchOptions) ([]*SemanticResult, error) {
	if options == nil {
		options = &SearchOptions{
			Threshold: 0.3,
			Limit:     10,
		}
	}

	// 构建查询
	query := r.db.Table("sensitive_words").
		Select("word_text, category, risk_level, tenant_id, embedding <-> ? as distance", vector).
		Where("tenant_id = ?", tenantID).
		Where("status = ?", 1).
		Where("embedding <-> ? <= ?", vector, 1-options.Threshold)

	// 应用过滤条件
	if len(options.Categories) > 0 {
		query = query.Where("category IN ?", options.Categories)
	}

	if options.MinRiskLevel > 0 {
		query = query.Where("risk_level >= ?", options.MinRiskLevel)
	}

	if options.MaxRiskLevel > 0 {
		query = query.Where("risk_level <= ?", options.MaxRiskLevel)
	}

	// 排序
	switch options.SortBy {
	case "risk_level":
		query = query.Order("risk_level DESC")
	case "distance":
		// 不使用 Order，因为距离已经在 SELECT 中计算了
	default:
		// 不使用 Order，因为距离已经在 SELECT 中计算了
	}

	// 限制结果数量
	if options.Limit > 0 {
		query = query.Limit(options.Limit)
	}

	// 执行查询
	type Result struct {
		WordText  string    `gorm:"column:word_text"`
		Category  string    `gorm:"column:category"`
		RiskLevel int       `gorm:"column:risk_level"`
		TenantID  uuid.UUID `gorm:"column:tenant_id"`
		Distance  float64   `gorm:"column:distance"`
	}

	var results []Result
	if err := query.Find(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to advanced search: %w", err)
	}

	// 转换为 SemanticResult
	semanticResults := make([]*SemanticResult, len(results))
	for i, result := range results {
		semanticResults[i] = &SemanticResult{
			WordText:  result.WordText,
			Category:  result.Category,
			RiskLevel: result.RiskLevel,
			Distance:  result.Distance,
			TenantID:  result.TenantID,
		}
	}

	return semanticResults, nil
}

// SearchOptions 检索选项
type SearchOptions struct {
	Threshold    float64  `json:"threshold"`      // 相似度阈值
	Limit        int      `json:"limit"`          // 结果数量限制
	Categories   []string `json:"categories"`     // 分类过滤
	MinRiskLevel int      `json:"min_risk_level"` // 最小风险等级
	MaxRiskLevel int      `json:"max_risk_level"` // 最大风险等级
	SortBy       string   `json:"sort_by"`        // 排序字段: distance, risk_level
}

// HybridSearch 混合检索：结合向量检索和关键词检索
func (r *semanticRepository) HybridSearch(tenantID uuid.UUID, text string, vector pgvector.Vector, options *SearchOptions) ([]*SemanticResult, error) {
	if options == nil {
		options = &SearchOptions{
			Threshold: 0.3,
			Limit:     10,
		}
	}

	// 向量检索
	vectorResults, err := r.AdvancedSearch(tenantID, vector, options)
	if err != nil {
		return nil, fmt.Errorf("failed to vector search: %w", err)
	}

	// 关键词检索（模糊匹配）
	var keywordResults []struct {
		WordText  string    `gorm:"column:word_text"`
		Category  string    `gorm:"column:category"`
		RiskLevel int       `gorm:"column:risk_level"`
		TenantID  uuid.UUID `gorm:"column:tenant_id"`
	}

	keywordQuery := r.db.Table("sensitive_words").
		Select("word_text, category, risk_level, tenant_id").
		Where("tenant_id = ?", tenantID).
		Where("status = ?", 1)

	// 应用分类过滤
	if len(options.Categories) > 0 {
		keywordQuery = keywordQuery.Where("category IN ?", options.Categories)
	}

	// 应用风险等级过滤
	if options.MinRiskLevel > 0 {
		keywordQuery = keywordQuery.Where("risk_level >= ?", options.MinRiskLevel)
	}
	if options.MaxRiskLevel > 0 {
		keywordQuery = keywordQuery.Where("risk_level <= ?", options.MaxRiskLevel)
	}

	// 关键词模糊匹配
	if text != "" {
		keywordQuery = keywordQuery.Where("word_text LIKE ?", "%"+text+"%")
	}

	if err := keywordQuery.Limit(options.Limit).Find(&keywordResults).Error; err != nil {
		return nil, fmt.Errorf("failed to keyword search: %w", err)
	}

	// 合并结果（去重）
	resultMap := make(map[string]*SemanticResult)

	// 添加向量检索结果
	for _, result := range vectorResults {
		key := fmt.Sprintf("%s_%s", result.WordText, result.Category)
		resultMap[key] = result
	}

	// 添加关键词检索结果
	for _, result := range keywordResults {
		key := fmt.Sprintf("%s_%s", result.WordText, result.Category)
		if _, exists := resultMap[key]; !exists {
			resultMap[key] = &SemanticResult{
				WordText:  result.WordText,
				Category:  result.Category,
				RiskLevel: result.RiskLevel,
				Distance:  0, // 关键词匹配的距离设为 0
				TenantID:  result.TenantID,
			}
		}
	}

	// 转换为切片
	results := make([]*SemanticResult, 0, len(resultMap))
	for _, result := range resultMap {
		results = append(results, result)
	}

	return results, nil
}

// GetSimilarWords 获取相似词
func (r *semanticRepository) GetSimilarWords(tenantID uuid.UUID, word string, vector pgvector.Vector, limit int) ([]*SemanticResult, error) {
	// 使用向量检索相似词
	return r.SearchByVectorWithFilter(tenantID, vector, 0.5, limit, nil)
}

// BatchGetSimilarWords 批量获取相似词
func (r *semanticRepository) BatchGetSimilarWords(tenantID uuid.UUID, words []string, v pgvector.Vector, limit int) ([][]*SemanticResult, error) {
	results := make([][]*SemanticResult, len(words))

	for i, word := range words {
		similarWords, err := r.GetSimilarWords(tenantID, word, v, limit)
		if err != nil {
			return nil, fmt.Errorf("failed to get similar words for '%s': %w", word, err)
		}
		results[i] = similarWords
	}

	return results, nil
}

// GetWordsByCategory 按分类获取敏感词
func (r *semanticRepository) GetWordsByCategory(tenantID uuid.UUID, category string, limit int) ([]*SemanticResult, error) {
	type Result struct {
		WordText  string    `gorm:"column:word_text"`
		Category  string    `gorm:"column:category"`
		RiskLevel int       `gorm:"column:risk_level"`
		TenantID  uuid.UUID `gorm:"column:tenant_id"`
	}

	var results []Result
	query := r.db.Table("sensitive_words").
		Select("word_text, category, risk_level, tenant_id").
		Where("tenant_id = ?", tenantID).
		Where("category = ?", category).
		Where("status = ?", 1)

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to get words by category: %w", err)
	}

	// 转换为 SemanticResult
	semanticResults := make([]*SemanticResult, len(results))
	for i, result := range results {
		semanticResults[i] = &SemanticResult{
			WordText:  result.WordText,
			Category:  result.Category,
			RiskLevel: result.RiskLevel,
			Distance:  0,
			TenantID:  result.TenantID,
		}
	}

	return semanticResults, nil
}

// GetWordStatistics 获取敏感词统计信息
func (r *semanticRepository) GetWordStatistics(tenantID uuid.UUID) (*WordStatistics, error) {
	type Stats struct {
		TotalWords    int64 `gorm:"column:total_words"`
		ActiveWords   int64 `gorm:"column:active_words"`
		InactiveWords int64 `gorm:"column:inactive_words"`
	}

	var stats Stats

	// 总词数
	if err := r.db.Table("sensitive_words").
		Where("tenant_id = ?", tenantID).
		Count(&stats.TotalWords).Error; err != nil {
		return nil, fmt.Errorf("failed to count total words: %w", err)
	}

	// 活跃词数
	if err := r.db.Table("sensitive_words").
		Where("tenant_id = ?", tenantID).
		Where("status = ?", 1).
		Count(&stats.ActiveWords).Error; err != nil {
		return nil, fmt.Errorf("failed to count active words: %w", err)
	}

	// 非活跃词数
	if err := r.db.Table("sensitive_words").
		Where("tenant_id = ?", tenantID).
		Where("status = ?", 0).
		Count(&stats.InactiveWords).Error; err != nil {
		return nil, fmt.Errorf("failed to count inactive words: %w", err)
	}

	// 按分类统计
	type CategoryStats struct {
		Category string `gorm:"column:category"`
		Count    int64  `gorm:"column:count"`
	}

	var categoryStats []CategoryStats
	if err := r.db.Table("sensitive_words").
		Select("category, COUNT(*) as count").
		Where("tenant_id = ?", tenantID).
		Where("status = ?", 1).
		Group("category").
		Find(&categoryStats).Error; err != nil {
		return nil, fmt.Errorf("failed to get category statistics: %w", err)
	}

	// 转换分类统计
	categoryCountMap := make(map[string]int64)
	for _, cs := range categoryStats {
		categoryCountMap[cs.Category] = cs.Count
	}

	// 按风险等级统计
	type RiskLevelStats struct {
		RiskLevel int   `gorm:"column:risk_level"`
		Count     int64 `gorm:"column:count"`
	}

	var riskLevelStats []RiskLevelStats
	if err := r.db.Table("sensitive_words").
		Select("risk_level, COUNT(*) as count").
		Where("tenant_id = ?", tenantID).
		Where("status = ?", 1).
		Group("risk_level").
		Find(&riskLevelStats).Error; err != nil {
		return nil, fmt.Errorf("failed to get risk level statistics: %w", err)
	}

	// 转换风险等级统计
	riskLevelCountMap := make(map[int]int64)
	for _, rls := range riskLevelStats {
		riskLevelCountMap[rls.RiskLevel] = rls.Count
	}

	return &WordStatistics{
		TotalWords:     stats.TotalWords,
		ActiveWords:    stats.ActiveWords,
		InactiveWords:  stats.InactiveWords,
		CategoryCount:  categoryCountMap,
		RiskLevelCount: riskLevelCountMap,
	}, nil
}

// WordStatistics 敏感词统计信息
type WordStatistics struct {
	TotalWords     int64            `json:"total_words"`
	ActiveWords    int64            `json:"active_words"`
	InactiveWords  int64            `json:"inactive_words"`
	CategoryCount  map[string]int64 `json:"category_count"`
	RiskLevelCount map[int]int64    `json:"risk_level_count"`
}
