package router

import (
	"jetwash/internal/cache"
	"jetwash/internal/config"
	"jetwash/internal/handler"
	"jetwash/internal/middleware"
	"jetwash/internal/repository"

	"github.com/gin-gonic/gin"
)

// SetupAPIKeyRoutes 设置API密钥管理路由
func SetupAPIKeyRoutes(router *gin.RouterGroup, apiKeyHandler *handler.APIKeyHandler, tenantRepo repository.TenantRepository, cfg *config.Config, redisClient *cache.RedisClient) {
	apiKeys := router.Group("/api-keys")
	apiKeys.Use(middleware.AuthMiddleware(tenantRepo, cfg))
	if cfg.RateLimit.Enabled {
		apiKeys.Use(middleware.RateLimit(redisClient, cfg.RateLimit.RequestsPerMinute))
	}
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
