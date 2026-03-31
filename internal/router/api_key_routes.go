package router

import (
	"jetwash/internal/config"
	"jetwash/internal/handler"
	"jetwash/internal/middleware"
	"jetwash/internal/repository"

	"github.com/gin-gonic/gin"
)

// SetupAPIKeyRoutes 设置API密钥管理路由
func SetupAPIKeyRoutes(router *gin.RouterGroup, apiKeyHandler *handler.APIKeyHandler, tenantRepo repository.TenantRepository, cfg *config.Config) {
	apiKeys := router.Group("/api-keys")
	apiKeys.Use(middleware.AuthMiddleware(tenantRepo, cfg))
	{
		apiKeys.POST("", apiKeyHandler.CreateAPIKey)
		apiKeys.GET("", apiKeyHandler.ListAPIKeys)
		apiKeys.GET("/:id", apiKeyHandler.GetAPIKey)
		apiKeys.PUT("/:id", apiKeyHandler.UpdateAPIKey)
		apiKeys.DELETE("/:id", apiKeyHandler.DeleteAPIKey)
		apiKeys.POST("/:id/activate", apiKeyHandler.ActivateAPIKey)
		apiKeys.POST("/:id/deactivate", apiKeyHandler.DeactivateAPIKey)
	}
}
