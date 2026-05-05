package handler

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"jetwash/internal/config"
	"jetwash/internal/middleware"
	"jetwash/internal/models"
	"jetwash/internal/repository"
	"jetwash/internal/service/detection_history"
	"jetwash/internal/service/normal"
	"jetwash/internal/util"
	"jetwash/pkg/ecode"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// 通用处理器

type NormalHandler struct {
	normalHandler           normal.NormalService
	tenantRepo              repository.TenantRepository
	config                  *config.Config
	detectionHistoryService detection_history.DetectionHistoryService
}

func NewNormalHandler(normalHandler normal.NormalService, tenantRepo repository.TenantRepository, config *config.Config, detectionHistoryService detection_history.DetectionHistoryService) *NormalHandler {
	return &NormalHandler{
		normalHandler:           normalHandler,
		tenantRepo:              tenantRepo,
		config:                  config,
		detectionHistoryService: detectionHistoryService,
	}
}

func (h *NormalHandler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    ecode.ErrInvalidParams,
			"message": "Invalid request body, email and password are required",
		})
		return
	}

	tenant, err := h.tenantRepo.GetTenantByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    ecode.InvalidAPIKey,
			"message": "Invalid email or password",
		})
		return
	}

	if !util.CheckPassword(req.Password, tenant.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    ecode.InvalidAPIKey,
			"message": "Invalid email or password",
		})
		return
	}

	if tenant.Status != 1 {
		c.JSON(http.StatusForbidden, gin.H{
			"code":    ecode.Forbidden,
			"message": "Tenant is not active",
		})
		return
	}

	token, err := util.GenerateToken(tenant.ID.String(), tenant.Name, h.config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    ecode.ErrServer,
			"message": "Failed to generate token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    ecode.Success,
		"message": "Login success",
		"data": gin.H{
			"id":      tenant.ID,
			"api_key": tenant.APIKey,
			"name":    tenant.Name,
			"email":   tenant.Email,
			"status":  tenant.Status,
			"token":   token,
		},
	})
}

func (h *NormalHandler) Register(c *gin.Context) {
	var req struct {
		Name     string `json:"username" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    ecode.ErrInvalidParams,
			"message": "Invalid request body, username, email and password (min 6 chars) are required",
		})
		return
	}

	tenant, err := h.normalHandler.RegisterTenant(req.Name, req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    ecode.ErrInvalidParams,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    ecode.Success,
		"message": "Registration successful",
		"data": gin.H{
			"tenant_id": tenant.ID,
			"api_key":   tenant.APIKey,
		},
	})
}

func (h *NormalHandler) UpdateStatus(c *gin.Context) {
	wordIdStr := c.Param("id")
	statusStr := c.Param("status")

	wordId, err := strconv.ParseUint(wordIdStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    ecode.ErrInvalidParams,
			"message": "Invalid word ID",
		})
		return
	}

	status, err := strconv.Atoi(statusStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    ecode.ErrInvalidParams,
			"message": "Invalid status",
		})
		return
	}

	if status < 1 || status > 2 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    ecode.ErrInvalidParams,
			"message": "Status must be 1 (enabled) or 2 (disabled)",
		})
		return
	}

	err = h.normalHandler.UpdateStatus(uint(wordId), status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    ecode.ErrServer,
			"message": "Failed to update status",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    ecode.Success,
		"message": "Status updated successfully",
	})
}

func (h *NormalHandler) BatchImportWords(c *gin.Context) {
	tenantIDStr, exists := middleware.GetTenantID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    ecode.Unauthorized,
			"message": "Tenant ID not found",
		})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    ecode.ErrInvalidParams,
			"message": "Invalid tenant ID",
		})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    ecode.ErrInvalidParams,
			"message": "No file uploaded",
		})
		return
	}

	if file.Size > 10*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    ecode.ErrInvalidParams,
			"message": "File size exceeds 10MB limit",
		})
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    ecode.ErrServer,
			"message": "Failed to open file",
		})
		return
	}
	defer src.Close()

	words, err := parseCSVFile(src)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    ecode.ErrInvalidParams,
			"message": "Failed to parse CSV file: " + err.Error(),
		})
		return
	}

	if len(words) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    ecode.ErrInvalidParams,
			"message": "No valid words found in CSV file",
		})
		return
	}

	err = h.normalHandler.BatchImportWords(c.Request.Context(), tenantID, words)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    ecode.ErrServer,
			"message": "Failed to import words: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    ecode.Success,
		"message": "Words imported successfully",
		"data": gin.H{
			"total": len(words),
		},
	})
}

func (h *NormalHandler) BatchImportWordsAsync(c *gin.Context) {
	tenantIDStr, exists := middleware.GetTenantID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    ecode.Unauthorized,
			"message": "Tenant ID not found",
		})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    ecode.ErrInvalidParams,
			"message": "Invalid tenant ID",
		})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    ecode.ErrInvalidParams,
			"message": "No file uploaded",
		})
		return
	}

	if file.Size > 10*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    ecode.ErrInvalidParams,
			"message": "File size exceeds 10MB limit",
		})
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    ecode.ErrServer,
			"message": "Failed to open file",
		})
		return
	}
	defer src.Close()

	words, err := parseCSVFile(src)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    ecode.ErrInvalidParams,
			"message": "Failed to parse CSV file: " + err.Error(),
		})
		return
	}

	if len(words) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    ecode.ErrInvalidParams,
			"message": "No valid words found in CSV file",
		})
		return
	}

	taskID, err := h.normalHandler.BatchImportWordsAsync(c.Request.Context(), tenantID, file.Filename, words)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    ecode.ErrServer,
			"message": "Failed to create import task: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    ecode.Success,
		"message": "Import task created successfully",
		"data": gin.H{
			"task_id": taskID,
			"total":   len(words),
		},
	})
}

func (h *NormalHandler) GetImportTaskStatus(c *gin.Context) {
	taskIDStr := c.Param("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    ecode.ErrInvalidParams,
			"message": "Invalid task ID",
		})
		return
	}

	task, err := h.normalHandler.GetImportTaskStatus(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    ecode.ErrNotFound,
			"message": "Import task not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    ecode.Success,
		"message": "Success",
		"data":    task,
	})
}

func (h *NormalHandler) GetImportTasks(c *gin.Context) {
	tenantIDStr, exists := middleware.GetTenantID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    ecode.Unauthorized,
			"message": "Tenant ID not found",
		})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    ecode.ErrInvalidParams,
			"message": "Invalid tenant ID",
		})
		return
	}

	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	tasks, total, err := h.normalHandler.GetImportTasks(tenantID, offset, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    ecode.ErrServer,
			"message": "Failed to get import tasks",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    ecode.Success,
		"message": "Success",
		"data": gin.H{
			"tasks":       tasks,
			"total":       total,
			"page":        page,
			"page_size":   pageSize,
			"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

func (h *NormalHandler) ImportWord(c *gin.Context) {
	tenantIDStr, exists := middleware.GetTenantID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    ecode.Unauthorized,
			"message": "Tenant ID not found",
		})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    ecode.ErrInvalidParams,
			"message": "Invalid tenant ID",
		})
		return
	}

	var req struct {
		WordText  string `json:"word_text" binding:"required"`
		Category  string `json:"category" binding:"required"`
		RiskLevel int    `json:"risk_level"`
		Status    int    `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    ecode.ErrInvalidParams,
			"message": "Invalid request body",
		})
		return
	}

	word := models.SensitiveWord{
		WordText:  req.WordText,
		Category:  req.Category,
		RiskLevel: req.RiskLevel,
		Status:    req.Status,
	}
	if word.RiskLevel == 0 {
		word.RiskLevel = 1
	}
	if word.Status == 0 {
		word.Status = 1
	}

	err = h.normalHandler.ImportWord(c.Request.Context(), tenantID, &word)
	if err != nil {
		// 检查是否是敏感词已存在的错误
		if strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, gin.H{
				"code":    ecode.ErrInvalidParams,
				"message": err.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    ecode.ErrServer,
			"message": "Failed to import word",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    ecode.Success,
		"message": "Word imported successfully",
		"data": gin.H{
			"id":         word.ID,
			"word_text":  word.WordText,
			"category":   word.Category,
			"risk_level": word.RiskLevel,
			"status":     word.Status,
		},
	})
}

func parseCSVFile(reader io.Reader) ([]models.SensitiveWord, error) {
	csvReader := csv.NewReader(reader)
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("CSV file is empty")
	}

	var words []models.SensitiveWord

	for i, record := range records {
		if i == 0 {
			continue
		}

		if len(record) < 2 {
			return nil, fmt.Errorf("record on line %d: missing required fields (word_text and category)", i+1)
		}

		wordText := record[0]
		category := record[1]
		riskLevel := 1
		status := 1

		if len(record) >= 3 && record[2] != "" {
			rl, err := strconv.Atoi(record[2])
			if err != nil {
				return nil, fmt.Errorf("record on line %d: invalid risk_level value", i+1)
			}
			if rl < 1 || rl > 5 {
				return nil, fmt.Errorf("record on line %d: risk_level must be between 1 and 5", i+1)
			}
			riskLevel = rl
		}

		if len(record) >= 4 && record[3] != "" {
			s, err := strconv.Atoi(record[3])
			if err != nil {
				return nil, fmt.Errorf("record on line %d: invalid status value", i+1)
			}
			if s != 1 && s != 2 {
				return nil, fmt.Errorf("record on line %d: status must be 1 (enabled) or 2 (disabled)", i+1)
			}
			status = s
		}

		if wordText == "" || category == "" {
			return nil, fmt.Errorf("record on line %d: word_text or category cannot be empty", i+1)
		}

		words = append(words, models.SensitiveWord{
			WordText:  wordText,
			Category:  category,
			RiskLevel: riskLevel,
			Status:    status,
		})
	}

	if len(words) == 0 {
		return nil, fmt.Errorf("no valid words found in CSV file")
	}

	return words, nil
}

func (h *NormalHandler) GetWords(c *gin.Context) {
	tenantIDStr, exists := middleware.GetTenantID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    ecode.Unauthorized,
			"message": "Tenant ID not found",
		})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    ecode.ErrInvalidParams,
			"message": "Invalid tenant ID",
		})
		return
	}

	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	filters := make(map[string]interface{})

	if category := c.Query("category"); category != "" {
		filters["category"] = category
	}

	if statusStr := c.Query("status"); statusStr != "" {
		if status, err := strconv.Atoi(statusStr); err == nil {
			filters["status"] = status
		}
	}

	if riskLevelStr := c.Query("risk_level"); riskLevelStr != "" {
		if riskLevel, err := strconv.Atoi(riskLevelStr); err == nil {
			filters["risk_level"] = riskLevel
		}
	}

	if minRiskLevelStr := c.Query("min_risk_level"); minRiskLevelStr != "" {
		if minRiskLevel, err := strconv.Atoi(minRiskLevelStr); err == nil {
			filters["min_risk_level"] = minRiskLevel
		}
	}

	if maxRiskLevelStr := c.Query("max_risk_level"); maxRiskLevelStr != "" {
		if maxRiskLevel, err := strconv.Atoi(maxRiskLevelStr); err == nil {
			filters["max_risk_level"] = maxRiskLevel
		}
	}

	if keyword := c.Query("keyword"); keyword != "" {
		filters["keyword"] = keyword
	}

	var words []models.SensitiveWord
	var total int64

	if len(filters) > 0 {
		words, total, err = h.normalHandler.GetWordsWithFilters(tenantID, filters, offset, pageSize)
	} else {
		words, total, err = h.normalHandler.GetWords(tenantID, offset, pageSize)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    ecode.ErrServer,
			"message": "Failed to get words",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    ecode.Success,
		"message": "Success",
		"data": gin.H{
			"words":       words,
			"total":       total,
			"page":        page,
			"page_size":   pageSize,
			"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

func (h *NormalHandler) GetDetectionHistories(c *gin.Context) {
	tenantIDStr, exists := middleware.GetTenantID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    ecode.Unauthorized,
			"message": "Tenant ID not found",
		})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    ecode.ErrInvalidParams,
			"message": "Invalid tenant ID",
		})
		return
	}

	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	filters := make(map[string]interface{})

	if mode := c.Query("mode"); mode != "" {
		filters["mode"] = mode
	}

	if startTime := c.Query("start_time"); startTime != "" {
		filters["start_time"] = startTime
	}

	if endTime := c.Query("end_time"); endTime != "" {
		filters["end_time"] = endTime
	}

	histories, total, err := h.detectionHistoryService.GetByTenantID(tenantID, filters, offset, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    ecode.ErrServer,
			"message": "Failed to get detection histories",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    ecode.Success,
		"message": "Success",
		"data": gin.H{
			"histories":   histories,
			"total":       total,
			"page":        page,
			"page_size":   pageSize,
			"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

func (h *NormalHandler) GetDetectionHistoryByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    ecode.ErrInvalidParams,
			"message": "Invalid history ID",
		})
		return
	}

	history, err := h.detectionHistoryService.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    ecode.ErrNotFound,
			"message": "Detection history not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    ecode.Success,
		"message": "Success",
		"data":    history,
	})
}

func (h *NormalHandler) DeleteDetectionHistory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    ecode.ErrInvalidParams,
			"message": "Invalid history ID",
		})
		return
	}

	err = h.detectionHistoryService.Delete(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    ecode.ErrServer,
			"message": "Failed to delete detection history",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    ecode.Success,
		"message": "Deleted successfully",
	})
}

func (h *NormalHandler) ClearDetectionHistories(c *gin.Context) {
	tenantIDStr, exists := middleware.GetTenantID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    ecode.Unauthorized,
			"message": "Tenant ID not found",
		})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    ecode.ErrInvalidParams,
			"message": "Invalid tenant ID",
		})
		return
	}

	err = h.detectionHistoryService.ClearByTenantID(tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    ecode.ErrServer,
			"message": "Failed to clear detection histories",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    ecode.Success,
		"message": "Cleared successfully",
	})
}
