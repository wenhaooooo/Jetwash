package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"jetwash/internal/cache"
	"jetwash/internal/config"
	"jetwash/internal/handler"
	"jetwash/internal/logger"
	"jetwash/internal/repository"
	"jetwash/internal/router"
	"jetwash/internal/service/api_key"
	"jetwash/internal/service/detection_history"
	"jetwash/internal/service/layer1_speed"
	"jetwash/internal/service/layer2_semantic"
	"jetwash/internal/service/layer3_reason"
	"jetwash/internal/service/normal"
	"jetwash/internal/service/orchestrator"
	"jetwash/internal/service/queue"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var (
	// 配置文件路径
	configPath = flag.String("config", "config.yaml", "Path to config file")
)

func main() {
	// 解析命令行参数
	flag.Parse()

	// 加载配置
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化日志系统
	if err := logger.InitLogger(cfg.Server.Mode); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Starting Jetwash Platform",
		zap.String("mode", cfg.Server.Mode),
		zap.String("port", fmt.Sprintf("%d", cfg.Server.Port)),
	)

	// 设置 Gin 运行模式
	gin.SetMode(cfg.Server.Mode)

	// 初始化数据库
	db, err := repository.InitDB(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// 初始化Redis客户端
	redisClient := cache.NewRedisClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB, logger.GetLogger())
	defer redisClient.Close()

	// 初始化 Repository 层
	tenantRepo := repository.NewTenantRepository(db)
	wordRepo := repository.NewWordRepository(db)
	importTaskRepo := repository.NewImportTaskRepository(db)
	apiKeyRepo := repository.NewAPIKeyRepository(db)

	// 初始化通用层
	normalService := normal.NewNormalService(wordRepo, tenantRepo, importTaskRepo, cfg)
	detectionHistoryService := detection_history.NewDetectionHistoryService(db)

	// 初始化 Layer1: 快速匹配层
	layer1Service := layer1_speed.NewLayer1Service()

	// 初始化 Layer2: 语义检索层
	semanticRepo := layer2_semantic.NewSemanticRepository(db)
	layer2Service := layer2_semantic.NewLayer2Service(semanticRepo, cfg)

	// 初始化 Layer3: 推理层
	var layer3Service layer3_reason.Layer3Service

	// 根据配置选择LLM提供者
	if cfg.LLM.Provider == "ollama" {
		// 使用Ollama LLM提供者
		ollamaLLMProvider := layer3_reason.NewOllamaLLMProvider(
			fmt.Sprintf("%s:%d", cfg.LLM.Ollama.Host, cfg.LLM.Ollama.Port),
			cfg.LLM.Ollama.ReasoningModel,
		)
		layer3Service = layer3_reason.NewLayer3Service(ollamaLLMProvider)
	} else {
		// 默认使用在线 LLM 提供者
		onlineLLMProvider := layer3_reason.NewOnlineLLMProvider(
			cfg.LLM.Online.APIKey,
			cfg.LLM.Online.Model,
			cfg.LLM.Online.BaseURL,
		)
		layer3Service = layer3_reason.NewLayer3Service(onlineLLMProvider)
	}

	// 初始化 Orchestrator: 编排层
	orchestratorService := orchestrator.NewOrchestrator(layer1Service, layer2Service, layer3Service, wordRepo, detectionHistoryService, redisClient)

	// 初始化队列服务
	queueService := queue.NewQueueService(redisClient)

	// 启动队列处理
	go func() {
		if err := queueService.Process(context.Background(), orchestratorService); err != nil {
			logger.Error("Queue processing error", zap.Error(err))
		}
	}()

	// 初始化服务层
	apiKeyService := api_key.NewAPIKeyService(apiKeyRepo)

	// 初始化 Handler 层
	orchestratorHandler := handler.NewOrchestratorHandler(orchestratorService, detectionHistoryService)
	normalHandler := handler.NewNormalHandler(normalService, tenantRepo, cfg, detectionHistoryService)
	apiKeyHandler := handler.NewAPIKeyHandler(apiKeyService)
	tenantHandler := handler.NewTenantHandler(tenantRepo)

	// 创建 Gin 路由
	ginRouter := router.SetupRouter(cfg, tenantRepo, orchestratorHandler, normalHandler, apiKeyHandler, tenantHandler)

	// 创建 HTTP 服务器
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      ginRouter,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	// 启动服务器（在 goroutine 中）
	go func() {
		logger.Info("Server started",
			zap.String("port", fmt.Sprintf("%d", cfg.Server.Port)),
			zap.String("mode", cfg.Server.Mode),
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// 在启动服务器前添加 LLM 服务连接性检查
	go func() {
		if cfg.LLM.Provider == "ollama" {
			// 检查Ollama服务
			ollamaUrl := fmt.Sprintf("%s:%d", cfg.LLM.Ollama.Host, cfg.LLM.Ollama.Port)
			resp, err := http.Get(fmt.Sprintf("%s/api/tags", ollamaUrl))
			if err != nil || resp.StatusCode != http.StatusOK {
				logger.Warn("Ollama service might be unreachable",
					zap.String("host", ollamaUrl),
					zap.Error(err),
				)
			} else {
				logger.Info("Connected to Ollama service successfully",
					zap.String("model", cfg.LLM.Ollama.EmbeddingModel))
			}
		} else {
			// 检查在线服务
			logger.Info("Using online LLM provider",
				zap.String("base_url", cfg.LLM.Online.BaseURL),
				zap.String("model", cfg.LLM.Online.Model))
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// 创建关闭上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 关闭 HTTP 服务器
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	}

	// 关闭数据库连接
	if err := repository.CloseDB(); err != nil {
		logger.Error("Failed to close database", zap.Error(err))
	}

	logger.Info("Server exited")
}
