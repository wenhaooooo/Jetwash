package handler

import (
	"strconv"
	"time"

	"jetwash/internal/middleware"
	"jetwash/internal/response"
	"jetwash/internal/service/api_key"
	"jetwash/pkg/ecode"

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
		response.Error(c, ecode.ErrUnauthorized)
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		response.ErrorWithMessage(c, ecode.ErrInvalidParams, "Invalid tenant_id: "+err.Error())
		return
	}

	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMessage(c, ecode.ErrInvalidParams, "Invalid request: "+err.Error())
		return
	}

	expiresAt, err := time.Parse(time.RFC3339, req.ExpiresAt)
	if err != nil {
		response.ErrorWithMessage(c, ecode.ErrInvalidParams, "Invalid expires_at format, use RFC3339")
		return
	}

	apiKey, err := h.apiKeyService.CreateAPIKey(tenantID, req.Name, expiresAt)
	if err != nil {
		response.ErrorWithMessage(c, ecode.ErrServer, "Failed to create API key: "+err.Error())
		return
	}

	response.OK(c, apiKey)
}

// GetAPIKey 获取API密钥
func (h *APIKeyHandler) GetAPIKey(c *gin.Context) {
	tenantIDStr, exists := middleware.GetTenantID(c)
	if !exists {
		response.Error(c, ecode.ErrUnauthorized)
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.ErrorWithMessage(c, ecode.ErrInvalidParams, "Invalid API key id")
		return
	}

	apiKey, err := h.apiKeyService.GetAPIKeyByID(id)
	if err != nil {
		response.Error(c, ecode.ErrNotFound)
		return
	}

	// 检查权限
	if apiKey.TenantID.String() != tenantIDStr {
		response.Error(c, ecode.ErrForbidden)
		return
	}

	response.OK(c, apiKey)
}

// ListAPIKeys 列API密钥
func (h *APIKeyHandler) ListAPIKeys(c *gin.Context) {
	tenantIDStr, exists := middleware.GetTenantID(c)
	if !exists {
		response.Error(c, ecode.ErrUnauthorized)
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		response.ErrorWithMessage(c, ecode.ErrInvalidParams, "Invalid tenant_id: "+err.Error())
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	apiKeys, total, err := h.apiKeyService.ListAPIKeys(tenantID, page, pageSize)
	if err != nil {
		response.ErrorWithMessage(c, ecode.ErrServer, "Failed to list API keys: "+err.Error())
		return
	}

	response.OK(c, gin.H{
		"api_keys":  apiKeys,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// UpdateAPIKey 更新API密钥
func (h *APIKeyHandler) UpdateAPIKey(c *gin.Context) {
	tenantIDStr, exists := middleware.GetTenantID(c)
	if !exists {
		response.Error(c, ecode.ErrUnauthorized)
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.ErrorWithMessage(c, ecode.ErrInvalidParams, "Invalid API key id")
		return
	}

	apiKey, err := h.apiKeyService.GetAPIKeyByID(id)
	if err != nil {
		response.Error(c, ecode.ErrNotFound)
		return
	}

	// 检查权限
	if apiKey.TenantID.String() != tenantIDStr {
		response.Error(c, ecode.ErrForbidden)
		return
	}

	var req UpdateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithMessage(c, ecode.ErrInvalidParams, "Invalid request: "+err.Error())
		return
	}

	apiKey.Name = req.Name
	if err := h.apiKeyService.UpdateAPIKey(apiKey); err != nil {
		response.ErrorWithMessage(c, ecode.ErrServer, "Failed to update API key: "+err.Error())
		return
	}

	response.OK(c, apiKey)
}

// DeleteAPIKey 删除API密钥
func (h *APIKeyHandler) DeleteAPIKey(c *gin.Context) {
	tenantIDStr, exists := middleware.GetTenantID(c)
	if !exists {
		response.Error(c, ecode.ErrUnauthorized)
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.ErrorWithMessage(c, ecode.ErrInvalidParams, "Invalid API key id")
		return
	}

	apiKey, err := h.apiKeyService.GetAPIKeyByID(id)
	if err != nil {
		response.Error(c, ecode.ErrNotFound)
		return
	}

	// 检查权限
	if apiKey.TenantID.String() != tenantIDStr {
		response.Error(c, ecode.ErrForbidden)
		return
	}

	if err := h.apiKeyService.DeleteAPIKey(id); err != nil {
		response.ErrorWithMessage(c, ecode.ErrServer, "Failed to delete API key: "+err.Error())
		return
	}

	response.OK(c, nil)
}

// ActivateAPIKey 激活API密钥
func (h *APIKeyHandler) ActivateAPIKey(c *gin.Context) {
	tenantIDStr, exists := middleware.GetTenantID(c)
	if !exists {
		response.Error(c, ecode.ErrUnauthorized)
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.ErrorWithMessage(c, ecode.ErrInvalidParams, "Invalid API key id")
		return
	}

	apiKey, err := h.apiKeyService.GetAPIKeyByID(id)
	if err != nil {
		response.Error(c, ecode.ErrNotFound)
		return
	}

	// 检查权限
	if apiKey.TenantID.String() != tenantIDStr {
		response.Error(c, ecode.ErrForbidden)
		return
	}

	if err := h.apiKeyService.ActivateAPIKey(id); err != nil {
		response.ErrorWithMessage(c, ecode.ErrServer, "Failed to activate API key: "+err.Error())
		return
	}

	response.OK(c, nil)
}

// DeactivateAPIKey 停用API密钥
func (h *APIKeyHandler) DeactivateAPIKey(c *gin.Context) {
	tenantIDStr, exists := middleware.GetTenantID(c)
	if !exists {
		response.Error(c, ecode.ErrUnauthorized)
		return
	}

	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.ErrorWithMessage(c, ecode.ErrInvalidParams, "Invalid API key id")
		return
	}

	apiKey, err := h.apiKeyService.GetAPIKeyByID(id)
	if err != nil {
		response.Error(c, ecode.ErrNotFound)
		return
	}

	// 检查权限
	if apiKey.TenantID.String() != tenantIDStr {
		response.Error(c, ecode.ErrForbidden)
		return
	}

	if err := h.apiKeyService.DeactivateAPIKey(id); err != nil {
		response.ErrorWithMessage(c, ecode.ErrServer, "Failed to deactivate API key: "+err.Error())
		return
	}

	response.OK(c, nil)
}
