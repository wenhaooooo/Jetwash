package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"jetwash/internal/middleware"
	"jetwash/internal/service/detection_history"
	"jetwash/internal/service/layer3_reason"
	"jetwash/internal/types"
)

// OrchestratorHandler 编排器处理器
type OrchestratorHandler struct {
	orchestrator            interface{}
	detectionHistoryService detection_history.DetectionHistoryService
}

// NewOrchestratorHandler 创建编排器处理器
func NewOrchestratorHandler(orchestrator interface{}, detectionHistoryService detection_history.DetectionHistoryService) *OrchestratorHandler {
	return &OrchestratorHandler{
		orchestrator:            orchestrator,
		detectionHistoryService: detectionHistoryService,
	}
}

// OrchestratorCheckTextRequest 编排器检查文本请求
type OrchestratorCheckTextRequest struct {
	Text string `json:"text" binding:"required"`
}

// OrchestratorCheckTextResponse 编排器检查文本响应
type OrchestratorCheckTextResponse struct {
	Code    int                       `json:"code"`
	Message string                    `json:"message"`
	Data    *types.OrchestratorResult `json:"data"`
}

// CheckText 检查文本
func (h *OrchestratorHandler) CheckText(c *gin.Context) {
	var req OrchestratorCheckTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, OrchestratorCheckTextResponse{
			Code:    400,
			Message: "Invalid request: " + err.Error(),
			Data:    nil,
		})
		return
	}

	tenantIDStr, exists := middleware.GetTenantID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, OrchestratorCheckTextResponse{
			Code:    401,
			Message: "Unauthorized: tenant_id not found in context",
			Data:    nil,
		})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, OrchestratorCheckTextResponse{
			Code:    400,
			Message: "Invalid tenant_id: " + err.Error(),
			Data:    nil,
		})
		return
	}

	result, err := h.orchestrator.(interface {
		CheckText(tenantID uuid.UUID, text string) (*types.OrchestratorResult, error)
	}).CheckText(tenantID, req.Text)
	if err != nil {
		c.JSON(http.StatusInternalServerError, OrchestratorCheckTextResponse{
			Code:    500,
			Message: "Failed to check text: " + err.Error(),
			Data:    nil,
		})
		return
	}

	c.JSON(http.StatusOK, OrchestratorCheckTextResponse{
		Code:    0,
		Message: "Success",
		Data:    result,
	})
}

// CheckTextWithConfigRequest 使用配置检查文本请求
type CheckTextWithConfigRequest struct {
	Text   string                    `json:"text" binding:"required"`
	Config *types.OrchestratorConfig `json:"config"`
}

// CheckTextWithConfig 使用配置检查文本
func (h *OrchestratorHandler) CheckTextWithConfig(c *gin.Context) {
	var req CheckTextWithConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, OrchestratorCheckTextResponse{
			Code:    400,
			Message: "Invalid request: " + err.Error(),
			Data:    nil,
		})
		return
	}

	tenantIDStr, exists := middleware.GetTenantID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, OrchestratorCheckTextResponse{
			Code:    401,
			Message: "Unauthorized: tenant_id not found in context",
			Data:    nil,
		})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, OrchestratorCheckTextResponse{
			Code:    400,
			Message: "Invalid tenant_id: " + err.Error(),
			Data:    nil,
		})
		return
	}

	result, err := h.orchestrator.(interface {
		CheckTextWithConfig(tenantID uuid.UUID, text string, config *types.OrchestratorConfig) (*types.OrchestratorResult, error)
	}).CheckTextWithConfig(tenantID, req.Text, req.Config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, OrchestratorCheckTextResponse{
			Code:    500,
			Message: "Failed to check text: " + err.Error(),
			Data:    nil,
		})
		return
	}

	c.JSON(http.StatusOK, OrchestratorCheckTextResponse{
		Code:    0,
		Message: "Success",
		Data:    result,
	})
}

// CheckTextWithContextRequest 使用上下文检查文本请求
type CheckTextWithContextRequest struct {
	Text    string                       `json:"text" binding:"required"`
	Context *layer3_reason.ReasonContext `json:"context"`
}

// CheckTextWithContext 使用上下文检查文本
func (h *OrchestratorHandler) CheckTextWithContext(c *gin.Context) {
	var req CheckTextWithContextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, OrchestratorCheckTextResponse{
			Code:    400,
			Message: "Invalid request: " + err.Error(),
			Data:    nil,
		})
		return
	}

	tenantIDStr, exists := middleware.GetTenantID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, OrchestratorCheckTextResponse{
			Code:    401,
			Message: "Unauthorized: tenant_id not found in context",
			Data:    nil,
		})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, OrchestratorCheckTextResponse{
			Code:    400,
			Message: "Invalid tenant_id: " + err.Error(),
			Data:    nil,
		})
		return
	}

	result, err := h.orchestrator.(interface {
		CheckTextWithContext(tenantID uuid.UUID, text string, context *layer3_reason.ReasonContext) (*types.OrchestratorResult, error)
	}).CheckTextWithContext(tenantID, req.Text, req.Context)
	if err != nil {
		c.JSON(http.StatusInternalServerError, OrchestratorCheckTextResponse{
			Code:    500,
			Message: "Failed to check text: " + err.Error(),
			Data:    nil,
		})
		return
	}

	c.JSON(http.StatusOK, OrchestratorCheckTextResponse{
		Code:    0,
		Message: "Success",
		Data:    result,
	})
}

// CheckTextWithConfigAndContextRequest 使用配置和上下文检查文本请求
type CheckTextWithConfigAndContextRequest struct {
	Text    string                       `json:"text" binding:"required"`
	Config  *types.OrchestratorConfig    `json:"config"`
	Context *layer3_reason.ReasonContext `json:"context"`
}

// CheckTextWithConfigAndContext 使用配置和上下文检查文本
func (h *OrchestratorHandler) CheckTextWithConfigAndContext(c *gin.Context) {
	var req CheckTextWithConfigAndContextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, OrchestratorCheckTextResponse{
			Code:    400,
			Message: "Invalid request: " + err.Error(),
			Data:    nil,
		})
		return
	}

	tenantIDStr, exists := middleware.GetTenantID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, OrchestratorCheckTextResponse{
			Code:    401,
			Message: "Unauthorized: tenant_id not found in context",
			Data:    nil,
		})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, OrchestratorCheckTextResponse{
			Code:    400,
			Message: "Invalid tenant_id: " + err.Error(),
			Data:    nil,
		})
		return
	}

	result, err := h.orchestrator.(interface {
		CheckTextWithConfigAndContext(tenantID uuid.UUID, text string, config *types.OrchestratorConfig, context *layer3_reason.ReasonContext) (*types.OrchestratorResult, error)
	}).CheckTextWithConfigAndContext(tenantID, req.Text, req.Config, req.Context)
	if err != nil {
		c.JSON(http.StatusInternalServerError, OrchestratorCheckTextResponse{
			Code:    500,
			Message: "Failed to check text: " + err.Error(),
			Data:    nil,
		})
		return
	}

	c.JSON(http.StatusOK, OrchestratorCheckTextResponse{
		Code:    0,
		Message: "Success",
		Data:    result,
	})
}

// HealthCheck 健康检查
func (h *OrchestratorHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "orchestrator",
	})
}

// GetDetectionHistories 获取检测历史列表
func (h *OrchestratorHandler) GetDetectionHistories(c *gin.Context) {
	tenantIDStr, exists := middleware.GetTenantID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "Unauthorized: tenant_id not found in context",
		})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid tenant_id: " + err.Error(),
		})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	histories, total, err := h.detectionHistoryService.GetDetectionHistories(tenantID, offset, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to get detection histories: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data": gin.H{
			"histories": histories,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// GetDetectionHistoryByID 获取单个检测历史详情
func (h *OrchestratorHandler) GetDetectionHistoryByID(c *gin.Context) {
	tenantIDStr, exists := middleware.GetTenantID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "Unauthorized: tenant_id not found in context",
		})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid tenant_id: " + err.Error(),
		})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid history id: " + err.Error(),
		})
		return
	}

	history, err := h.detectionHistoryService.GetDetectionHistoryByID(id, tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "Detection history not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data":    history,
	})
}
