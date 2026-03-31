package router

import (
	"github.com/gin-gonic/gin"
	"jetwash/internal/config"
	"jetwash/internal/handler"
	"jetwash/internal/middleware"
	"jetwash/internal/repository"
)

// SetupNormalRoutes 设置普通用户路由
func SetupNormalRoutes(router *gin.RouterGroup, handler *handler.NormalHandler, tenantRepo repository.TenantRepository, cfg *config.Config) {
	// 登录（无需鉴权）
	normal := router.Group("/normal")
	{
		normal.POST("/login", handler.Login)
		normal.POST("/register", handler.Register)
	}

	// 停用/启用
	status := router.Group("/normal")
	status.Use(middleware.AuthMiddleware(tenantRepo, cfg))
	{
		status.GET("/words", handler.GetWords)                                  // 获取所有敏感词
		status.PUT("/words/:id/status/:status", handler.UpdateStatus)           // 更新敏感词状态
		status.POST("/words/batch-import", handler.BatchImportWordsAsync)         // 异步批量导入敏感词
		status.POST("/words/import", handler.ImportWord)                        // 导入敏感词
		status.GET("/import-tasks", handler.GetImportTasks)                      // 获取导入任务列表
		status.GET("/import-tasks/:id", handler.GetImportTaskStatus)            // 获取导入任务状态
		status.GET("/detection-history", handler.GetDetectionHistories)         // 获取检测历史列表
		status.GET("/detection-history/:id", handler.GetDetectionHistoryByID)   // 获取单个检测历史详情
		status.DELETE("/detection-history/:id", handler.DeleteDetectionHistory) // 删除检测历史
		status.DELETE("/detection-history", handler.ClearDetectionHistories)    // 清空检测历史
	}
}