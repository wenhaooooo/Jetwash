package router

import (
	"github.com/gin-gonic/gin"
	"jetwash/internal/config"
	"jetwash/internal/handler"
	"jetwash/internal/middleware"
	"jetwash/internal/repository"
)

// SetupOrchestratorRoutes 设置编排器路由
func SetupOrchestratorRoutes(router *gin.RouterGroup, handler *handler.OrchestratorHandler, tenantRepo repository.TenantRepository, cfg *config.Config) {
	orchestratorGroup := router.Group("/orchestrator")
	orchestratorGroup.Use(middleware.AuthMiddleware(tenantRepo, cfg))
	{
		orchestratorGroup.POST("/check", handler.CheckText)
		orchestratorGroup.POST("/check/config", handler.CheckTextWithConfig)
		orchestratorGroup.POST("/check/context", handler.CheckTextWithContext)
		orchestratorGroup.POST("/check/full", handler.CheckTextWithConfigAndContext)
		orchestratorGroup.GET("/histories", handler.GetDetectionHistories)
		orchestratorGroup.GET("/histories/:id", handler.GetDetectionHistoryByID)
	}
}