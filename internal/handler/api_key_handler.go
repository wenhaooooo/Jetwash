package handler

import (
	"net/http"
	"strconv"
	"time"

	"jetwash/internal/middleware"
	"jetwash/internal/service/api_key"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// APIKeyHandler API密钥处理器
type APIKeyHandler struct {
	apiKeyService api_key.APIKeyService
}

// NewAPIKeyHandler 创建API密钥处理器实例
func NewAPIKeyHandler(apiKeyService api_key.APIKeyService) *APIKeyHandler {
	return &APIKeyHandler{
		apiKeyService: apiKeyService,
	}
}

// CreateAPIKeyRequest 创建API密钥请求
type CreateAPIKeyRequest struct {
	Name      string `json:"name" binding:"required"`
	ExpiresAt string `json:"expires_at" binding:"required"`
}

// UpdateAPIKeyRequest 更新API密钥请求
type UpdateAPIKeyRequest struct {
	Name string `json:"name" binding:"required"`
}

// CreateAPIKey 创建API密钥
func (h *APIKeyHandler) CreateAPIKey(c *gin.Context) {
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

	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid request: " + err.Error(),
		})
		return
	}

	expiresAt, err := time.Parse(time.RFC3339, req.ExpiresAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid expires_at format, use RFC3339",
		})
		return
	}

	apiKey, err := h.apiKeyService.CreateAPIKey(tenantID, req.Name, expiresAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to create API key: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data":    apiKey,
	})
}

// GetAPIKey 获取API密钥
func (h *APIKeyHandler) GetAPIKey(c *gin.Context) {
	tenantIDStr, exists := middleware.GetTenantID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "Unauthorized: tenant_id not found in context",
		})
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid API key id",
		})
		return
	}

	apiKey, err := h.apiKeyService.GetAPIKeyByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "API key not found",
		})
		return
	}

	// 检查权限
	if apiKey.TenantID.String() != tenantIDStr {
		c.JSON(http.StatusForbidden, gin.H{
			"code":    403,
			"message": "Forbidden: API key does not belong to this tenant",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data":    apiKey,
	})
}

// ListAPIKeys 列API密钥
func (h *APIKeyHandler) ListAPIKeys(c *gin.Context) {
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

	apiKeys, total, err := h.apiKeyService.ListAPIKeys(tenantID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to list API keys: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data": gin.H{
			"api_keys": apiKeys,
			"total":    total,
			"page":     page,
			"page_size": pageSize,
		},
	})
}

// UpdateAPIKey 更新API密钥
func (h *APIKeyHandler) UpdateAPIKey(c *gin.Context) {
	tenantIDStr, exists := middleware.GetTenantID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "Unauthorized: tenant_id not found in context",
		})
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid API key id",
		})
		return
	}

	apiKey, err := h.apiKeyService.GetAPIKeyByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "API key not found",
		})
		return
	}

	// 检查权限
	if apiKey.TenantID.String() != tenantIDStr {
		c.JSON(http.StatusForbidden, gin.H{
			"code":    403,
			"message": "Forbidden: API key does not belong to this tenant",
		})
		return
	}

	var req UpdateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid request: " + err.Error(),
		})
		return
	}

	apiKey.Name = req.Name
	if err := h.apiKeyService.UpdateAPIKey(apiKey); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to update API key: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data":    apiKey,
	})
}

// DeleteAPIKey 删除API密钥
func (h *APIKeyHandler) DeleteAPIKey(c *gin.Context) {
	tenantIDStr, exists := middleware.GetTenantID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "Unauthorized: tenant_id not found in context",
		})
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid API key id",
		})
		return
	}

	apiKey, err := h.apiKeyService.GetAPIKeyByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "API key not found",
		})
		return
	}

	// 检查权限
	if apiKey.TenantID.String() != tenantIDStr {
		c.JSON(http.StatusForbidden, gin.H{
			"code":    403,
			"message": "Forbidden: API key does not belong to this tenant",
		})
		return
	}

	if err := h.apiKeyService.DeleteAPIKey(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to delete API key: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
	})
}

// ActivateAPIKey 激活API密钥
func (h *APIKeyHandler) ActivateAPIKey(c *gin.Context) {
	tenantIDStr, exists := middleware.GetTenantID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "Unauthorized: tenant_id not found in context",
		})
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid API key id",
		})
		return
	}

	apiKey, err := h.apiKeyService.GetAPIKeyByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "API key not found",
		})
		return
	}

	// 检查权限
	if apiKey.TenantID.String() != tenantIDStr {
		c.JSON(http.StatusForbidden, gin.H{
			"code":    403,
			"message": "Forbidden: API key does not belong to this tenant",
		})
		return
	}

	if err := h.apiKeyService.ActivateAPIKey(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to activate API key: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
	})
}

// DeactivateAPIKey 停用API密钥
func (h *APIKeyHandler) DeactivateAPIKey(c *gin.Context) {
	tenantIDStr, exists := middleware.GetTenantID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "Unauthorized: tenant_id not found in context",
		})
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid API key id",
		})
		return
	}

	apiKey, err := h.apiKeyService.GetAPIKeyByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "API key not found",
		})
		return
	}

	// 检查权限
	if apiKey.TenantID.String() != tenantIDStr {
		c.JSON(http.StatusForbidden, gin.H{
			"code":    403,
			"message": "Forbidden: API key does not belong to this tenant",
		})
		return
	}

	if err := h.apiKeyService.DeactivateAPIKey(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to deactivate API key: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
	})
}
