package handler

import (
	"net/http"
	"strconv"

	"jetwash/internal/middleware"
	"jetwash/internal/models"
	"jetwash/internal/repository"
	"jetwash/internal/util"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TenantHandler 租户处理器
type TenantHandler struct {
	tenantRepo repository.TenantRepository
}

// NewTenantHandler 创建租户处理器实例
func NewTenantHandler(tenantRepo repository.TenantRepository) *TenantHandler {
	return &TenantHandler{
		tenantRepo: tenantRepo,
	}
}

// CreateTenantRequest 创建租户请求
type CreateTenantRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// UpdateTenantRequest 更新租户请求
type UpdateTenantRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password"`
}

// CreateTenant 创建租户
func (h *TenantHandler) CreateTenant(c *gin.Context) {
	var req CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid request: " + err.Error(),
		})
		return
	}

	// 获取当前登录租户ID
	var parentID *uuid.UUID
	tenantIDStr, exists := middleware.GetTenantID(c)
	if exists {
		currentTenantID, err := uuid.Parse(tenantIDStr)
		if err == nil {
			// 检查当前租户层级是否已达到5层
			currentLevel, err := h.tenantRepo.GetTenantLevel(currentTenantID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"code":    500,
					"message": "Failed to check tenant level: " + err.Error(),
				})
				return
			}

			// 如果当前租户层级已达到5层，不允许创建子租户
			if currentLevel >= 4 {
				c.JSON(http.StatusForbidden, gin.H{
					"code":    403,
					"message": "Cannot create tenant: maximum tenant level (5) reached",
				})
				return
			}

			parentID = &currentTenantID
		}
	}

	// 检查租户名称是否已存在
	existingTenant, err := h.tenantRepo.GetTenantByName(req.Name)
	if err == nil && existingTenant != nil {
		c.JSON(http.StatusConflict, gin.H{
			"code":    409,
			"message": "Tenant name already exists",
		})
		return
	}

	// 检查邮箱是否已存在
	existingTenant, err = h.tenantRepo.GetTenantByEmail(req.Email)
	if err == nil && existingTenant != nil {
		c.JSON(http.StatusConflict, gin.H{
			"code":    409,
			"message": "Email already exists",
		})
		return
	}

	// 生成密码哈希
	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to hash password: " + err.Error(),
		})
		return
	}

	// 生成API密钥
	apiKey := util.GenerateAPIKey()

	tenant := &models.Tenant{
		ParentID: parentID,
		APIKey:   apiKey,
		Name:     req.Name,
		Email:    req.Email,
		Password: hashedPassword,
		Status:   1, // 1: active
	}

	if err := h.tenantRepo.CreateTenant(tenant); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to create tenant: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data":    tenant,
	})
}

// GetTenant 获取租户
func (h *TenantHandler) GetTenant(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid tenant id",
		})
		return
	}

	tenant, err := h.tenantRepo.GetTenantByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "Tenant not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data":    tenant,
	})
}

// ListTenants 列租户
func (h *TenantHandler) ListTenants(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize
	tenants, total, err := h.tenantRepo.ListTenants(offset, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to list tenants: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data": gin.H{
			"tenants":   tenants,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// UpdateTenant 更新租户
func (h *TenantHandler) UpdateTenant(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid tenant id",
		})
		return
	}

	tenant, err := h.tenantRepo.GetTenantByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "Tenant not found",
		})
		return
	}

	var req UpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid request: " + err.Error(),
		})
		return
	}

	// 检查租户名称是否已存在（排除当前租户）
	if req.Name != tenant.Name {
		existingTenant, err := h.tenantRepo.GetTenantByName(req.Name)
		if err == nil && existingTenant != nil && existingTenant.ID != id {
			c.JSON(http.StatusConflict, gin.H{
				"code":    409,
				"message": "Tenant name already exists",
			})
			return
		}
	}

	// 检查邮箱是否已存在（排除当前租户）
	if req.Email != tenant.Email {
		existingTenant, err := h.tenantRepo.GetTenantByEmail(req.Email)
		if err == nil && existingTenant != nil && existingTenant.ID != id {
			c.JSON(http.StatusConflict, gin.H{
				"code":    409,
				"message": "Email already exists",
			})
			return
		}
	}

	// 更新租户信息
	tenant.Name = req.Name
	tenant.Email = req.Email

	// 如果提供了新密码，更新密码
	if req.Password != "" {
		hashedPassword, err := util.HashPassword(req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "Failed to hash password: " + err.Error(),
			})
			return
		}
		tenant.Password = hashedPassword
	}

	if err := h.tenantRepo.UpdateTenant(tenant); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to update tenant: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data":    tenant,
	})
}

// DeleteTenant 删除租户
func (h *TenantHandler) DeleteTenant(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "Invalid tenant id",
		})
		return
	}

	// 检查租户是否存在
	_, err = h.tenantRepo.GetTenantByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "Tenant not found",
		})
		return
	}

	if err := h.tenantRepo.DeleteTenant(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "Failed to delete tenant: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
	})
}
