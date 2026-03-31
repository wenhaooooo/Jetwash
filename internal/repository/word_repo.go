package repository

import (
	"fmt"
	"jetwash/internal/models"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
)

// WordRepository 敏感词仓储接口
type WordRepository interface {
	// SearchByVector 根据向量搜索相似词（使用 TenantID 过滤）
	// threshold: 相似度阈值（0-1），只返回相似度大于等于阈值的词
	// limit: 返回结果数量限制
	SearchByVector(tenantID uuid.UUID, embedding pgvector.Vector, threshold float64, limit int) ([]models.SensitiveWord, error)

	// SearchByVectorWithDistance 根据向量搜索相似词，并返回距离分数
	SearchByVectorWithDistance(tenantID uuid.UUID, embedding pgvector.Vector, threshold float64, limit int) ([]WordWithDistance, error)

	// CreateSensitiveWord 创建敏感词
	CreateSensitiveWord(word *models.SensitiveWord) error

	// GetSensitiveWordByID 根据 ID 获取敏感词
	GetSensitiveWordByID(id uuid.UUID) (*models.SensitiveWord, error)

	// GetSensitiveWordsByTenant 获取租户的所有敏感词
	GetSensitiveWordsByTenant(tenantID uuid.UUID, offset, limit int) ([]models.SensitiveWord, int64, error)

	// GetSensitiveWordsByTenantWithFilters 获取租户的所有敏感词（支持过滤条件）
	GetSensitiveWordsByTenantWithFilters(tenantID uuid.UUID, filters map[string]interface{}, offset, limit int) ([]models.SensitiveWord, int64, error)

	// GetAllSensitiveWordsByTenant 获取租户的所有敏感词（不分页）
	GetAllSensitiveWordsByTenant(tenantID uuid.UUID) ([]models.SensitiveWord, error)
	// UpdateSensitiveWord 更新敏感词
	UpdateSensitiveWord(word *models.SensitiveWord) error

	// DeleteSensitiveWord 删除敏感词（软删除）
	DeleteSensitiveWord(id uint) error

	// UpdateStatus 更新敏感词状态
	UpdateStatus(id uint, status int) error

	// BatchCreateWords 批量创建敏感词
	BatchCreateWords(words []models.SensitiveWord) error

	// CheckWordExists 检查敏感词是否已存在
	CheckWordExists(tenantID uuid.UUID, wordText string) (bool, error)
}

// WordWithDistance 带距离分数的敏感词
type WordWithDistance struct {
	models.SensitiveWord
	Distance float64 `json:"distance"` // 相似度距离
}

// wordRepository 敏感词仓储实现
type wordRepository struct {
	db *gorm.DB
}

func (r *wordRepository) UpdateStatus(id uint, status int) error {
	updateStr := `UPDATE sensitive_words SET status = ? WHERE id = ?`
	if err := r.db.Exec(updateStr, status, id).Error; err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}
	return nil
}

// NewWordRepository 创建敏感词仓储实例
func NewWordRepository(db *gorm.DB) WordRepository {
	return &wordRepository{db: db}
}

// SearchByVector 根据向量搜索相似词（使用 TenantID 过滤）
// 使用欧几里得距离（L2 distance）<-> 操作符
func (r *wordRepository) SearchByVector(tenantID uuid.UUID, embedding pgvector.Vector, threshold float64, limit int) ([]models.SensitiveWord, error) {
	var words []models.SensitiveWord

	// 使用原生 SQL 进行向量相似度查询
	// <-> 表示欧几里得距离（L2 distance）
	// 距离越小表示越相似
	query := `
		SELECT *
		FROM sensitive_words
		WHERE tenant_id = ?
		  AND status = 1
		  AND deleted_at IS NULL
		  AND embedding <-> ? <= ?
		ORDER BY embedding <-> ?
		LIMIT ?
	`

	if err := r.db.Raw(query, tenantID, embedding, threshold, embedding, limit).Scan(&words).Error; err != nil {
		return nil, fmt.Errorf("failed to search by vector: %w", err)
	}

	return words, nil
}

// SearchByVectorWithDistance 根据向量搜索相似词，并返回距离分数
// 使用余弦距离（cosine distance）<=> 操作符
func (r *wordRepository) SearchByVectorWithDistance(tenantID uuid.UUID, embedding pgvector.Vector, threshold float64, limit int) ([]WordWithDistance, error) {
	var results []WordWithDistance

	// 使用原生 SQL 进行向量相似度查询
	// <=> 表示余弦距离（cosine distance）
	// 距离越小表示越相似（余弦距离范围：0-2，0表示完全相同）
	query := `
		SELECT *,
		       embedding <=> ? AS distance
		FROM sensitive_words
		WHERE tenant_id = ?
		  AND status = 1
		  AND deleted_at IS NULL
		  AND embedding <=> ? <= ?
		ORDER BY embedding <=> ?
		LIMIT ?
	`

	if err := r.db.Raw(query, embedding, tenantID, embedding, threshold, embedding, limit).Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to search by vector with distance: %w", err)
	}

	return results, nil
}

// CreateSensitiveWord 创建敏感词
func (r *wordRepository) CreateSensitiveWord(word *models.SensitiveWord) error {
	if err := r.db.Create(word).Error; err != nil {
		return fmt.Errorf("failed to create sensitive word: %w", err)
	}
	return nil
}

// GetSensitiveWordByID 根据 ID 获取敏感词
func (r *wordRepository) GetSensitiveWordByID(id uuid.UUID) (*models.SensitiveWord, error) {
	var word models.SensitiveWord
	if err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&word).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("sensitive word not found")
		}
		return nil, fmt.Errorf("failed to get sensitive word: %w", err)
	}
	return &word, nil
}

// GetSensitiveWordsByTenant 获取租户的所有敏感词
func (r *wordRepository) GetSensitiveWordsByTenant(tenantID uuid.UUID, offset, limit int) ([]models.SensitiveWord, int64, error) {
	var words []models.SensitiveWord
	var total int64

	// 获取总数
	if err := r.db.Model(&models.SensitiveWord{}).
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count sensitive words: %w", err)
	}

	// 获取分页数据
	if err := r.db.Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&words).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get sensitive words: %w", err)
	}

	return words, total, nil
}

// GetSensitiveWordsByTenantWithFilters 获取租户的所有敏感词（支持过滤条件）
func (r *wordRepository) GetSensitiveWordsByTenantWithFilters(tenantID uuid.UUID, filters map[string]interface{}, offset, limit int) ([]models.SensitiveWord, int64, error) {
	var words []models.SensitiveWord
	var total int64

	query := r.db.Model(&models.SensitiveWord{}).Where("tenant_id = ? AND deleted_at IS NULL", tenantID)

	if filters != nil {
		if category, ok := filters["category"]; ok && category != "" {
			query = query.Where("category = ?", category)
		}
		if status, ok := filters["status"]; ok && status != nil {
			query = query.Where("status = ?", status)
		}
		if riskLevel, ok := filters["risk_level"]; ok && riskLevel != nil {
			query = query.Where("risk_level = ?", riskLevel)
		}
		if minRiskLevel, ok := filters["min_risk_level"]; ok && minRiskLevel != nil {
			query = query.Where("risk_level >= ?", minRiskLevel)
		}
		if maxRiskLevel, ok := filters["max_risk_level"]; ok && maxRiskLevel != nil {
			query = query.Where("risk_level <= ?", maxRiskLevel)
		}
		if keyword, ok := filters["keyword"]; ok && keyword != "" {
			query = query.Where("word_text LIKE ?", "%"+keyword.(string)+"%")
		}
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count sensitive words: %w", err)
	}

	if err := query.Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&words).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get sensitive words: %w", err)
	}

	return words, total, nil
}

// GetAllSensitiveWordsByTenant 获取租户的所有敏感词（不分页）
func (r *wordRepository) GetAllSensitiveWordsByTenant(tenantID uuid.UUID) ([]models.SensitiveWord, error) {
	var words []models.SensitiveWord
	if err := r.db.Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		Order("created_at DESC").
		Find(&words).Error; err != nil {
		return nil, fmt.Errorf("failed to get all sensitive words: %w", err)
	}
	return words, nil
}

// UpdateSensitiveWord 更新敏感词
func (r *wordRepository) UpdateSensitiveWord(word *models.SensitiveWord) error {
	if err := r.db.Save(word).Error; err != nil {
		return fmt.Errorf("failed to update sensitive word: %w", err)
	}
	return nil
}

// DeleteSensitiveWord 删除敏感词（软删除）
func (r *wordRepository) DeleteSensitiveWord(id uint) error {
	if err := r.db.Delete(&models.SensitiveWord{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete sensitive word: %w", err)
	}
	return nil
}

// BatchCreateWords 批量创建敏感词
func (r *wordRepository) BatchCreateWords(words []models.SensitiveWord) error {
	if len(words) == 0 {
		return nil
	}

	// 使用事务确保原子性
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&words).Error; err != nil {
			return fmt.Errorf("failed to batch create sensitive words: %w", err)
		}
		return nil
	})
}

// CheckWordExists 检查敏感词是否已存在
func (r *wordRepository) CheckWordExists(tenantID uuid.UUID, wordText string) (bool, error) {
	var count int64
	if err := r.db.Model(&models.SensitiveWord{}).
		Where("tenant_id = ? AND word_text = ? AND deleted_at IS NULL", tenantID, wordText).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("failed to check word existence: %w", err)
	}
	return count > 0, nil
}
