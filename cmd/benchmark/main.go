package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/google/uuid"

	"jetwash/internal/cache"
	"jetwash/internal/config"
	"jetwash/internal/logger"
	"jetwash/internal/repository"
	"jetwash/internal/service/detection_history"
	"jetwash/internal/service/layer1_speed"
	"jetwash/internal/service/layer2_semantic"
	"jetwash/internal/service/layer3_reason"
	"jetwash/internal/service/orchestrator"
	"jetwash/internal/service/queue"
	"jetwash/pkg/benchmark"
)

func main() {
	// 解析命令行参数
	totalRequests := flag.Int("total", 1000, "Total number of requests")
	concurrentRequests := flag.Int("concurrent", 100, "Number of concurrent requests")
	testText := flag.String("text", "This is a test text for performance benchmarking", "Test text to check")
	queueTest := flag.Bool("queue", false, "Run queue benchmark instead of direct benchmark")
	configPath := flag.String("config", "config.yaml", "Path to config file")
	mode := flag.String("mode", "full", "Detection mode: full, basic (layer1 only), semantic (layer1+layer2)")
	flag.Parse()

	// 加载配置
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化日志
	if err := logger.InitLogger(cfg.Server.Mode); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// 初始化数据库
	db, err := repository.InitDB(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer repository.CloseDB()

	// 初始化Redis
	redisClient, err := cache.NewRedisClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB, logger.GetLogger())
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// 初始化Repository
	wordRepo := repository.NewWordRepository(db)
	detectionHistoryService := detection_history.NewDetectionHistoryService(db)

	// 初始化服务
	layer1Service := layer1_speed.NewLayer1Service()
	semanticRepo := layer2_semantic.NewSemanticRepository(db)
	layer2Service := layer2_semantic.NewLayer2Service(semanticRepo, cfg)

	// 初始化LLM服务
	var layer3Service layer3_reason.Layer3Service
	if cfg.LLM.Provider == "ollama" {
		ollamaLLMProvider := layer3_reason.NewOllamaLLMProvider(
			fmt.Sprintf("%s:%d", cfg.LLM.Ollama.Host, cfg.LLM.Ollama.Port),
			cfg.LLM.Ollama.ReasoningModel,
		)
		layer3Service = layer3_reason.NewLayer3Service(ollamaLLMProvider)
	} else {
		onlineLLMProvider := layer3_reason.NewOnlineLLMProvider(
			cfg.LLM.Online.APIKey,
			cfg.LLM.Online.Model,
			cfg.LLM.Online.BaseURL,
		)
		layer3Service = layer3_reason.NewLayer3Service(onlineLLMProvider)
	}

	// 初始化Orchestrator
	orchestratorService := orchestrator.NewOrchestrator(
		layer1Service,
		layer2Service,
		layer3Service,
		wordRepo,
		detectionHistoryService,
		redisClient,
		logger.GetLogger(),
	)

	// 初始化队列服务
	queueService := queue.NewQueueService(redisClient)

	// 启动队列处理
	go func() {
		if err := queueService.Process(context.Background(), orchestratorService); err != nil {
			log.Printf("Queue processing error: %v", err)
		}
	}()

	// 生成测试租户ID
	tenantID := uuid.New()

	if *queueTest {
		// 运行队列性能测试
		fmt.Println("Running queue benchmark...")
		result, err := benchmark.RunQueueBenchmark(
			queueService,
			*totalRequests,
			*testText,
			tenantID,
		)
		if err != nil {
			log.Fatalf("Failed to run queue benchmark: %v", err)
		}
		benchmark.PrintResult(result)
	} else {
		// 根据模式创建配置
		orchConfig := orchestrator.DefaultOrchestratorConfig()
		switch *mode {
		case "basic":
			orchConfig.EnableLayer2 = false
			orchConfig.EnableLayer3 = false
		case "semantic":
			orchConfig.EnableLayer3 = false
		case "full":
			// 默认配置，所有层都启用
		default:
			fmt.Printf("Unknown mode: %s, using full mode\n", *mode)
		}

		// 运行直接性能测试
		fmt.Printf("Running direct benchmark in %s mode...\n", *mode)
		benchmarkConfig := benchmark.BenchmarkConfig{
			TotalRequests:      *totalRequests,
			ConcurrentRequests: *concurrentRequests,
			TestText:           *testText,
			TenantID:           tenantID,
			Config:             orchConfig,
		}

		result, err := benchmark.RunBenchmark(
			orchestratorService,
			queueService,
			benchmarkConfig,
		)
		if err != nil {
			log.Fatalf("Failed to run benchmark: %v", err)
		}
		benchmark.PrintResult(result)
	}
}
