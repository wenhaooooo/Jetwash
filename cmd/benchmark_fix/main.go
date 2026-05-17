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

// 非敏感词测试样本
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
}

// 敏感词测试样本（从数据库中常见的敏感词类别）
var sensitiveWordSamples = []string{
	// 广告类
	"优惠券",
	"立即领取",
	"限时优惠",
	"假一赔十",
	"买一送一",
	// 暴力类
	"自杀",
	"杀人",
	"暴力",
	"殴打",
	"凶器",
	// 色情类
	"色情",
	"裸体",
	"性服务",
	"卖淫",
	"嫖娼",
	// 政治类
	"反动",
	"颠覆",
	"分裂",
	"谣言",
	"煽动",
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
	FastPassCount      int           `json:"fast_pass_count"`  // 快速放行数量
	Layer2Count        int           `json:"layer2_count"`     // 进入Layer2的数量
	Layer3Count        int           `json:"layer3_count"`     // 进入Layer3的数量
	SensitiveCount     int           `json:"sensitive_count"`  // 敏感词请求数量
	NormalCount        int           `json:"normal_count"`     // 非敏感词请求数量
}

// 生成随机文本（混合敏感词和非敏感词）
func generateMixedText(sensitiveRatio float64, addSuffix bool) (string, bool) {
	seed := time.Now().UnixNano() + int64(rand.Intn(1000000))
	r := rand.New(rand.NewSource(seed))

	isSensitive := r.Float64() < sensitiveRatio

	if isSensitive {
		// 生成包含敏感词的文本
		sensitiveWord := sensitiveWordSamples[r.Intn(len(sensitiveWordSamples))]
		normalText := normalTextSamples[r.Intn(len(normalTextSamples))]

		var text string
		if r.Float64() < 0.5 {
			text = fmt.Sprintf("%s，%s", sensitiveWord, normalText)
		} else {
			text = fmt.Sprintf("%s，%s", normalText, sensitiveWord)
		}

		if addSuffix {
			randomSuffix := fmt.Sprintf("_%d", r.Intn(1000000))
			text += randomSuffix
		}

		return text, true
	}

	// 生成非敏感词文本
	text := normalTextSamples[r.Intn(len(normalTextSamples))]
	if addSuffix {
		randomSuffix := fmt.Sprintf("_%d", r.Intn(1000000))
		text += randomSuffix
	}

	return text, false
}

func main() {
	// 解析命令行参数
	totalRequests := flag.Int("total", 1000, "Total number of requests")
	concurrentRequests := flag.Int("concurrent", 100, "Number of concurrent requests")
	configPath := flag.String("config", "config.yaml", "Path to config file")
	tenantID := flag.String("tenant", "", "Tenant ID to use for testing")
	sensitiveRatio := flag.Float64("ratio", 0.1, "Ratio of sensitive requests (0.0-1.0)")
	addSuffix := flag.Bool("unique", true, "Add unique suffix to each request")
	flag.Parse()

	// 参数校验
	if *sensitiveRatio < 0 || *sensitiveRatio > 1 {
		log.Fatalf("Invalid sensitive ratio: must be between 0 and 1")
	}

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

	// 运行性能测试（混合敏感词和非敏感词）
	fmt.Printf("\nRunning benchmark with mixed text (%.0f%% sensitive, %.0f%% normal)...\n",
		*sensitiveRatio*100, (1-*sensitiveRatio)*100)

	result := &BenchmarkResult{
		TotalRequests:      *totalRequests,
		ConcurrentRequests: *concurrentRequests,
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var cacheHits, cacheMisses, errors, fastPassCount, layer2Count, layer3Count, sensitiveCount, normalCount int
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

			// 生成混合文本
			text, isSensitive := generateMixedText(*sensitiveRatio, *addSuffix)

			orResult, err := orchestratorService.CheckText(testTenantID, text)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				errors++
			} else {
				// 统计敏感词/非敏感词
				if isSensitive {
					sensitiveCount++
				} else {
					normalCount++
				}

				// 统计缓存命中
				if orResult.FromCache {
					cacheHits++
				} else {
					cacheMisses++
				}

				// 统计各层执行情况
				if orResult.Layer1Result != nil && !orResult.Layer1Result.HasMatch {
					fastPassCount++ // 快速放行
				}
				if orResult.Layer2Result != nil {
					layer2Count++ // 进入Layer2
				}
				if orResult.Layer3Result != nil {
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
	result.SensitiveCount = sensitiveCount
	result.NormalCount = normalCount

	// 打印结果
	printResult(result, *sensitiveRatio)
}

func printResult(result *BenchmarkResult, sensitiveRatio float64) {
	fmt.Printf("\n===== 混合文本性能测试结果 =====\n")
	fmt.Printf("配置参数:\n")
	fmt.Printf("  总请求数: %d\n", result.TotalRequests)
	fmt.Printf("  并发请求数: %d\n", result.ConcurrentRequests)
	fmt.Printf("  敏感词比例: %.0f%%\n", sensitiveRatio*100)
	fmt.Printf("\n性能指标:\n")
	fmt.Printf("  总耗时: %v\n", result.TotalTime)
	fmt.Printf("  平均响应时间: %v\n", result.AverageTime)
	fmt.Printf("  每秒请求数: %.2f\n", result.RequestsPerSecond)
	fmt.Printf("  缓存命中: %d\n", result.CacheHits)
	fmt.Printf("  缓存未命中: %d\n", result.CacheMisses)
	fmt.Printf("  错误数: %d\n", result.Errors)
	fmt.Printf("\n请求类型分布:\n")
	fmt.Printf("  敏感词请求: %d (%.1f%%)\n", result.SensitiveCount,
		float64(result.SensitiveCount)/float64(result.TotalRequests)*100)
	fmt.Printf("  非敏感词请求: %d (%.1f%%)\n", result.NormalCount,
		float64(result.NormalCount)/float64(result.TotalRequests)*100)
	fmt.Printf("\n各层执行统计:\n")
	fmt.Printf("  快速放行: %d (%.1f%%)\n", result.FastPassCount,
		float64(result.FastPassCount)/float64(result.TotalRequests)*100)
	fmt.Printf("  进入Layer2: %d (%.1f%%)\n", result.Layer2Count,
		float64(result.Layer2Count)/float64(result.TotalRequests)*100)
	fmt.Printf("  进入Layer3(LLM): %d (%.1f%%)\n", result.Layer3Count,
		float64(result.Layer3Count)/float64(result.TotalRequests)*100)
	fmt.Printf("=============================\n")

	// 性能分析建议
	fmt.Println("\n📊 性能分析:")
	if result.Layer3Count > 0 {
		llmRatio := float64(result.Layer3Count) / float64(result.TotalRequests)
		fmt.Printf("  - LLM调用占比: %.1f%%\n", llmRatio*100)
		fmt.Printf("  - LLM是主要性能瓶颈，考虑优化或关闭\n")
	}
	if result.RequestsPerSecond < 100 {
		fmt.Println("  - QPS较低，建议降低并发数或使用更快的LLM模型")
	} else if result.RequestsPerSecond > 1000 {
		fmt.Println("  - QPS表现优秀，系统性能良好")
	}
}
