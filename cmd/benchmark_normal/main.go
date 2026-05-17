package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

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
)

// 非敏感词测试样本（不会触发任何敏感词检测）
var normalTextSamples = []string{
	"今天天气真好",
	"我喜欢阅读",
	"项目进展顺利",
	"用户体验很好",
	"产品质量优秀",
	"服务态度好",
	"工作效率高",
	"学习编程很有趣",
	"周末去公园散步",
	"晚上吃什么好呢",
	"音乐很好听",
	"电影很精彩",
	"书籍很有价值",
	"运动有益健康",
	"旅行开阔眼界",
	"朋友聚会很开心",
	"团队合作愉快",
	"代码质量很高",
	"文档写得很详细",
	"测试覆盖率达标",
	"代码review完成",
	"需求分析完毕",
	"设计文档已更新",
	"会议准时开始",
	"任务已完成",
}

// 生成随机非敏感文本
func generateNormalText() string {
	seed := time.Now().UnixNano() + int64(rand.Intn(1000000))
	r := rand.New(rand.NewSource(seed))

	text := normalTextSamples[r.Intn(len(normalTextSamples))]
	randomSuffix := fmt.Sprintf("_%d", r.Intn(1000000))

	return fmt.Sprintf("%s%s", text, randomSuffix)
}

// BenchmarkResult 性能测试结果
type BenchmarkResult struct {
	TotalRequests      int           `json:"total_requests"`
	ConcurrentRequests int           `json:"concurrent_requests"`
	TotalTime          time.Duration `json:"total_time"`
	AverageTime        time.Duration `json:"average_time"`
	RequestsPerSecond  float64       `json:"requests_per_second"`
	CacheHits          int           `json:"cache_hits"`
	CacheMisses        int           `json:"cache_misses"`
	Errors             int           `json:"errors"`
	FastPassCount      int           `json:"fast_pass_count"` // 快速放行数量
	Layer2Count        int           `json:"layer2_count"`    // 进入Layer2的数量
	Layer3Count        int           `json:"layer3_count"`    // 进入Layer3的数量
}

func main() {
	// 解析命令行参数
	totalRequests := flag.Int("total", 1000, "Total number of requests")
	concurrentRequests := flag.Int("concurrent", 100, "Number of concurrent requests")
	configPath := flag.String("config", "config.yaml", "Path to config file")
	tenantID := flag.String("tenant", "", "Tenant ID to use for testing")
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
	layer2Service := layer2_semantic.NewLayer2Service(semanticRepo, cfg, redisClient)

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
		context.Background(),
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

	// 解析或生成测试租户ID
	var testTenantID uuid.UUID
	if *tenantID != "" {
		var err error
		testTenantID, err = uuid.Parse(*tenantID)
		if err != nil {
			log.Fatalf("Invalid tenant ID: %v", err)
		}
		fmt.Printf("Using provided tenant ID: %s\n", testTenantID)
	} else {
		testTenantID = uuid.New()
		fmt.Printf("Generated new tenant ID: %s\n", testTenantID)
	}

	// 预热服务
	fmt.Println("Warming up services...")
	warmupStart := time.Now()
	if err := orchestratorService.Warmup([]uuid.UUID{testTenantID}); err != nil {
		fmt.Printf("Warmup completed with warnings: %v\n", err)
	} else {
		fmt.Printf("Warmup completed in %v\n", time.Since(warmupStart))
	}

	// 运行性能测试（仅使用非敏感词）
	fmt.Println("Running benchmark with normal (non-sensitive) text...")
	fmt.Println("All requests should go through fast-pass and NOT hit LLM")

	result := &BenchmarkResult{
		TotalRequests:      *totalRequests,
		ConcurrentRequests: *concurrentRequests,
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var cacheHits, cacheMisses, errors, fastPassCount, layer2Count, layer3Count int
	startTime := time.Now()

	semaphore := make(chan struct{}, *concurrentRequests)

	for i := 0; i < *totalRequests; i++ {
		semaphore <- struct{}{}
		wg.Add(1)

		go func() {
			defer func() {
				wg.Done()
				<-semaphore
			}()

			// 生成非敏感词文本
			text := generateNormalText()

			result, err := orchestratorService.CheckText(testTenantID, text)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				errors++
			} else {
				// 统计缓存命中
				if result.FromCache {
					cacheHits++
				} else {
					cacheMisses++
				}

				// 统计各层执行情况
				if result.Layer1Result != nil && !result.Layer1Result.HasMatch {
					fastPassCount++ // 快速放行
				}
				if result.Layer2Result != nil {
					layer2Count++ // 进入Layer2
				}
				if result.Layer3Result != nil {
					layer3Count++ // 进入Layer3
				}
			}
		}()
	}

	wg.Wait()
	result.TotalTime = time.Since(startTime)
	result.AverageTime = result.TotalTime / time.Duration(*totalRequests)
	result.RequestsPerSecond = float64(*totalRequests) / result.TotalTime.Seconds()
	result.CacheHits = cacheHits
	result.CacheMisses = cacheMisses
	result.Errors = errors
	result.FastPassCount = fastPassCount
	result.Layer2Count = layer2Count
	result.Layer3Count = layer3Count

	// 打印结果
	printResult(result)
}

func printResult(result *BenchmarkResult) {
	fmt.Printf("\n===== 非敏感词性能测试结果 =====\n")
	fmt.Printf("总请求数: %d\n", result.TotalRequests)
	fmt.Printf("并发请求数: %d\n", result.ConcurrentRequests)
	fmt.Printf("总耗时: %v\n", result.TotalTime)
	fmt.Printf("平均响应时间: %v\n", result.AverageTime)
	fmt.Printf("每秒请求数: %.2f\n", result.RequestsPerSecond)
	fmt.Printf("缓存命中: %d\n", result.CacheHits)
	fmt.Printf("缓存未命中: %d\n", result.CacheMisses)
	fmt.Printf("错误数: %d\n", result.Errors)
	fmt.Printf("\n各层执行统计:\n")
	fmt.Printf("  快速放行: %d (%.1f%%)\n", result.FastPassCount,
		float64(result.FastPassCount)/float64(result.TotalRequests)*100)
	fmt.Printf("  进入Layer2: %d (%.1f%%)\n", result.Layer2Count,
		float64(result.Layer2Count)/float64(result.TotalRequests)*100)
	fmt.Printf("  进入Layer3(LLM): %d (%.1f%%)\n", result.Layer3Count,
		float64(result.Layer3Count)/float64(result.TotalRequests)*100)
	fmt.Printf("=============================\n")
}
