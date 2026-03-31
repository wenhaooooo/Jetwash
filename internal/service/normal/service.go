package normal

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"jetwash/internal/config"
	"jetwash/internal/models"
	"jetwash/internal/repository"
	"jetwash/internal/util"
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

type NormalService interface {
	// UpdateStatus 启用/禁用违禁词
	UpdateStatus(wordId uint, status int) error
	// BatchImportWords 批量导入敏感词
	BatchImportWords(ctx context.Context, tenantID uuid.UUID, words []models.SensitiveWord) error
	// BatchImportWordsAsync 异步批量导入敏感词
	BatchImportWordsAsync(ctx context.Context, tenantID uuid.UUID, filename string, words []models.SensitiveWord) (uuid.UUID, error)
	// GetImportTaskStatus 获取导入任务状态
	GetImportTaskStatus(taskID uuid.UUID) (*models.ImportTask, error)
	// GetImportTasks 获取租户的所有导入任务
	GetImportTasks(tenantID uuid.UUID, offset, limit int) ([]models.ImportTask, int64, error)
	// ImportWord 单个导入敏感词
	ImportWord(ctx context.Context, tenantID uuid.UUID, word *models.SensitiveWord) error
	// GetWords 获取敏感词列表
	GetWords(tenantID uuid.UUID, offset, limit int) ([]models.SensitiveWord, int64, error)
	// GetWordsWithFilters 获取敏感词列表（支持过滤条件）
	GetWordsWithFilters(tenantID uuid.UUID, filters map[string]interface{}, offset, limit int) ([]models.SensitiveWord, int64, error)
	// RegisterTenant 注册新租户
	RegisterTenant(name, email, password string) (*models.Tenant, error)
}

type normalService struct {
	repo              repository.WordRepository
	tenantRepo        repository.TenantRepository
	embeddingProvider util.EmbeddingProvider
	taskRepo          repository.ImportTaskRepository
}

func (n normalService) UpdateStatus(wordId uint, status int) error {
	return n.repo.UpdateStatus(wordId, status)
}

func (n normalService) BatchImportWords(ctx context.Context, tenantID uuid.UUID, words []models.SensitiveWord) error {
	// 先生成所有embedding向量
	for i := range words {
		words[i].TenantID = tenantID
		if words[i].Status == 0 {
			words[i].Status = 1 // 默认启用
		}

		// 生成embedding向量
		embedding, err := n.embeddingProvider.GetEmbedding(words[i].WordText)
		if err != nil {
			return fmt.Errorf("failed to generate embedding for word '%s': %w", words[i].WordText, err)
		}
		words[i].Embedding = pgvector.NewVector(embedding)
	}

	// 所有embedding向量生成成功后，再批量插入
	return n.repo.BatchCreateWords(words)
}

func (n normalService) BatchImportWordsAsync(ctx context.Context, tenantID uuid.UUID, filename string, words []models.SensitiveWord) (uuid.UUID, error) {
	// 创建导入任务
	task := &models.ImportTask{
		TenantID: tenantID,
		FileName: filename,
		Status:   "pending",
		Total:    len(words),
	}

	if err := n.taskRepo.Create(task); err != nil {
		return uuid.Nil, fmt.Errorf("failed to create import task: %w", err)
	}

	// 启动异步处理
	go n.processImportTask(task, words)

	return task.ID, nil
}

func (n normalService) processImportTask(task *models.ImportTask, words []models.SensitiveWord) {
	// 更新任务状态为处理中
	now := time.Now()
	task.Status = "processing"
	task.StartedAt = &now
	if err := n.taskRepo.Update(task); err != nil {
		task.Status = "failed"
		task.ErrorMsg = "Failed to update task status: " + err.Error()
		task.CompletedAt = &now
		n.taskRepo.Update(task)
		return
	}

	// 批量导入敏感词
	imported := 0
	failed := 0
	skipped := 0

	for i := range words {
		words[i].TenantID = task.TenantID
		if words[i].Status == 0 {
			words[i].Status = 1
		}

		// 检查敏感词是否已存在
		exists, err := n.repo.CheckWordExists(task.TenantID, words[i].WordText)
		if err != nil {
			failed++
			continue
		}
		if exists {
			skipped++
			continue
		}

		// 生成embedding向量
		embedding, err := n.embeddingProvider.GetEmbedding(words[i].WordText)
		if err != nil {
			failed++
			continue
		}
		words[i].Embedding = pgvector.NewVector(embedding)

		// 插入数据库
		if err := n.repo.CreateSensitiveWord(&words[i]); err != nil {
			failed++
			continue
		}
		imported++

		// 每处理100个词更新一次进度
		if (i+1)%100 == 0 {
			task.Imported = imported
			task.Failed = failed
			n.taskRepo.Update(task)
		}
	}

	// 更新任务状态
	completedAt := time.Now()
	task.Imported = imported
	task.Failed = failed
	task.CompletedAt = &completedAt

	if failed > 0 {
		task.Status = "completed"
		task.ErrorMsg = fmt.Sprintf("Imported %d words, skipped %d existing words, failed %d words", imported, skipped, failed)
	} else if skipped > 0 {
		task.Status = "completed"
		task.ErrorMsg = fmt.Sprintf("Imported %d words, skipped %d existing words", imported, skipped)
	} else {
		task.Status = "completed"
	}

	if err := n.taskRepo.Update(task); err != nil {
		task.Status = "failed"
		task.ErrorMsg = "Failed to update task status: " + err.Error()
		n.taskRepo.Update(task)
	}
}

func (n normalService) GetImportTaskStatus(taskID uuid.UUID) (*models.ImportTask, error) {
	return n.taskRepo.GetByID(taskID)
}

func (n normalService) GetImportTasks(tenantID uuid.UUID, offset, limit int) ([]models.ImportTask, int64, error) {
	return n.taskRepo.GetByTenantID(tenantID, offset, limit)
}

func (n normalService) ImportWord(ctx context.Context, tenantID uuid.UUID, word *models.SensitiveWord) error {
	// 设置租户ID和默认状态
	word.TenantID = tenantID
	if word.Status == 0 {
		word.Status = 1 // 默认启用
	}

	// 检查敏感词是否已存在
	exists, err := n.repo.CheckWordExists(tenantID, word.WordText)
	if err != nil {
		return fmt.Errorf("failed to check word existence: %w", err)
	}
	if exists {
		return fmt.Errorf("sensitive word '%s' already exists", word.WordText)
	}

	// 生成embedding向量
	embedding, err := n.embeddingProvider.GetEmbedding(word.WordText)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}
	word.Embedding = pgvector.NewVector(embedding)

	return n.repo.CreateSensitiveWord(word)
}

func (n normalService) GetWords(tenantID uuid.UUID, offset, limit int) ([]models.SensitiveWord, int64, error) {
	return n.repo.GetSensitiveWordsByTenant(tenantID, offset, limit)
}

func (n normalService) GetWordsWithFilters(tenantID uuid.UUID, filters map[string]interface{}, offset, limit int) ([]models.SensitiveWord, int64, error) {
	return n.repo.GetSensitiveWordsByTenantWithFilters(tenantID, filters, offset, limit)
}

func (n normalService) RegisterTenant(name, email, password string) (*models.Tenant, error) {
	// 检查用户名是否已存在
	_, err := n.tenantRepo.GetTenantByName(name)
	if err == nil {
		return nil, fmt.Errorf("tenant name already exists")
	}

	// 检查邮箱是否已存在
	_, err = n.tenantRepo.GetTenantByEmail(email)
	if err == nil {
		return nil, fmt.Errorf("tenant email already exists")
	}

	// 哈希密码
	hashedPassword, err := util.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// 生成API Key
	apiKey, err := generateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	// 创建租户
	tenant := &models.Tenant{
		Name:     name,
		Email:    email,
		Password: hashedPassword,
		APIKey:   apiKey,
		Status:   1, // 默认激活
	}

	err = n.tenantRepo.CreateTenant(tenant)
	if err != nil {
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}

	return tenant, nil
}

// generateAPIKey 生成32字节的随机API Key
func generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func NewNormalService(repo repository.WordRepository, tenantRepo repository.TenantRepository, taskRepo repository.ImportTaskRepository, cfg *config.Config) NormalService {
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

	return &normalService{
		repo:              repo,
		tenantRepo:        tenantRepo,
		embeddingProvider: embeddingProvider,
		taskRepo:          taskRepo,
	}
}
