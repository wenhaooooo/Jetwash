package router

import (
	"time"

	"jetwash/internal/config"
	"jetwash/internal/handler"
	"jetwash/internal/logger"
	"jetwash/internal/middleware"
	"jetwash/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// SetupRouter 设置路由
func SetupRouter(cfg *config.Config, tenantRepo repository.TenantRepository, orchestratorHandler *handler.OrchestratorHandler, normalHandler *handler.NormalHandler, apiKeyHandler *handler.APIKeyHandler, tenantHandler *handler.TenantHandler) *gin.Engine {
	router := gin.Default()

	// 添加日志中间件（使用自定义日志）
	router.Use(func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()

		// 记录请求日志
		logger.Info("HTTP Request",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.Int("status", statusCode),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
		)
	})

	// 添加 CORS 中间件
	router.Use(middleware.CORS(cfg.CORS.AllowedOrigins))

	// 添加恢复中间件
	router.Use(gin.Recovery())

	// Prometheus metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// 健康检查接口（无需鉴权）
	router.GET("/health", orchestratorHandler.HealthCheck)

	// API 路由组
	v1 := router.Group("/api/v1")
	{
		// 注册路由模块
		SetupOrchestratorRoutes(v1, orchestratorHandler, tenantRepo, cfg)
		SetupNormalRoutes(v1, normalHandler, tenantRepo, cfg)
		SetupTenantRoutes(v1, tenantHandler)
		SetupAPIKeyRoutes(v1, apiKeyHandler, tenantRepo, cfg)
	}

	return router
}
