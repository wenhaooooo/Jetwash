package router

import (
	"jetwash/internal/handler"

	"github.com/gin-gonic/gin"
)

// SetupTenantRoutes 设置租户管理路由
func SetupTenantRoutes(router *gin.RouterGroup, tenantHandler *handler.TenantHandler) {
	tenants := router.Group("/tenants")
	{
		tenants.POST("", tenantHandler.CreateTenant)
		tenants.GET("", tenantHandler.ListTenants)
		tenants.GET("/:id", tenantHandler.GetTenant)
		tenants.PUT("/:id", tenantHandler.UpdateTenant)
		tenants.DELETE("/:id", tenantHandler.DeleteTenant)
	}
}
