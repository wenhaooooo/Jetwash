package benchmark

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"

	"jetwash/internal/service/orchestrator"
	"jetwash/internal/service/queue"
	"jetwash/internal/types"
)

// 敏感词测试样本（用于触发layer1匹配）
var sensitiveWordSamples = []string{
	"优惠券", "抢购", "特价促销", "限时优惠", "微信", "红包",
	"投资", "理财", "高收益", "赌博", "色情", "暴力",
	"免费领取", "点击领取", "立即领取", "买一送一", "假一赔十",
	"反党", "反革命", "分裂国家", "台独", "港独", "藏独",
	"杀人", "自杀", "爆炸", "绑架", "抢劫", "强奸",
	"裸体", "性爱", "卖淫", "嫖娼", "性器官",
}

// 普通文本样本（不会触发敏感词）
var normalTextSamples = []string{
	"今天天气真好", "我喜欢阅读", "这是一个测试", "项目进展顺利",
	"用户体验很好", "产品质量优秀", "服务态度好", "工作效率高",
	"学习编程很有趣", "周末去公园散步", "晚上吃什么好呢",
	"音乐很好听", "电影很精彩", "书籍很有价值",
	"运动有益健康", "旅行开阔眼界", "朋友聚会很开心",
}

// 生成随机测试文本
func generateRandomText() string {
	// 使用时间戳+纳秒作为种子，确保每次调用都产生不同结果
	seed := time.Now().UnixNano() + int64(rand.Intn(1000000))
	r := rand.New(rand.NewSource(seed))

	// 随机选择文本类型：50%普通文本，50%包含敏感词的文本
	if r.Float64() < 0.5 {
		// 生成包含敏感词的文本
		sensitiveWord := sensitiveWordSamples[r.Intn(len(sensitiveWordSamples))]
		normalText := normalTextSamples[r.Intn(len(normalTextSamples))]

		// 添加随机后缀，确保每次生成的文本都不同
		randomSuffix := fmt.Sprintf("_%d", r.Intn(1000000))

		if r.Float64() < 0.5 {
			return fmt.Sprintf("%s，%s%s", sensitiveWord, normalText, randomSuffix)
		}
		return fmt.Sprintf("%s，%s%s", normalText, sensitiveWord, randomSuffix)
	}

	// 生成普通文本，添加随机后缀
	randomSuffix := fmt.Sprintf("_%d", r.Intn(1000000))
	return fmt.Sprintf("%s%s", normalTextSamples[r.Intn(len(normalTextSamples))], randomSuffix)
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
}

// BenchmarkConfig 性能测试配置
type BenchmarkConfig struct {
	TotalRequests      int                       `json:"total_requests"`
	ConcurrentRequests int                       `json:"concurrent_requests"`
	TestText           string                    `json:"test_text"`
	TenantID           uuid.UUID                 `json:"tenant_id"`
	Config             *types.OrchestratorConfig `json:"config"`
	UseRandomText      bool                      `json:"use_random_text"` // 是否使用随机文本
}

// RunBenchmark 运行性能测试
func RunBenchmark(
	orchestrator orchestrator.Orchestrator,
	queueService queue.QueueService,
	config BenchmarkConfig,
) (*BenchmarkResult, error) {
	result := &BenchmarkResult{
		TotalRequests:      config.TotalRequests,
		ConcurrentRequests: config.ConcurrentRequests,
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var cacheHits, cacheMisses, errors int
	startTime := time.Now()

	semaphore := make(chan struct{}, config.ConcurrentRequests)

	for i := 0; i < config.TotalRequests; i++ {
		semaphore <- struct{}{}
		wg.Add(1)

		go func() {
			defer func() {
				wg.Done()
				<-semaphore
			}()

			// 根据配置选择测试文本
			text := config.TestText
			if config.UseRandomText {
				text = generateRandomText()
			}

			result, err := orchestrator.CheckTextWithConfig(
				config.TenantID,
				text,
				config.Config,
			)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				errors++
			} else {
				// 使用结果中的 FromCache 字段判断是否缓存命中
				if result.FromCache {
					cacheHits++
				} else {
					cacheMisses++
				}
			}
		}()
	}

	wg.Wait()
	result.TotalTime = time.Since(startTime)
	result.AverageTime = result.TotalTime / time.Duration(config.TotalRequests)
	result.RequestsPerSecond = float64(config.TotalRequests) / result.TotalTime.Seconds()
	result.CacheHits = cacheHits
	result.CacheMisses = cacheMisses
	result.Errors = errors

	return result, nil
}

// RunQueueBenchmark 运行队列性能测试
func RunQueueBenchmark(
	queueService queue.QueueService,
	totalTasks int,
	testText string,
	tenantID uuid.UUID,
	useRandomText bool,
) (*BenchmarkResult, error) {
	result := &BenchmarkResult{
		TotalRequests: totalTasks,
	}

	startTime := time.Now()

	// 入队所有任务
	taskIDs := make([]string, totalTasks)
	for i := 0; i < totalTasks; i++ {
		// 根据配置选择测试文本
		text := testText
		if useRandomText {
			text = generateRandomText()
		}

		task := &queue.DetectionTask{
			TenantID: tenantID,
			Text:     text,
			Config:   orchestrator.DefaultOrchestratorConfig(),
		}

		if err := queueService.Enqueue(context.Background(), task); err != nil {
			return nil, fmt.Errorf("failed to enqueue task: %w", err)
		}
		taskIDs[i] = task.ID
	}

	// 等待队列处理完成（简单实现，实际应该根据任务状态判断）
	time.Sleep(time.Duration(totalTasks) * 100 * time.Millisecond)

	// 检查任务结果
	errors := 0
	for _, taskID := range taskIDs {
		_, err := queueService.GetResult(context.Background(), taskID)
		if err != nil {
			errors++
		}
	}

	result.TotalTime = time.Since(startTime)
	result.AverageTime = result.TotalTime / time.Duration(totalTasks)
	result.RequestsPerSecond = float64(totalTasks) / result.TotalTime.Seconds()
	result.Errors = errors

	return result, nil
}

// PrintResult 打印测试结果
func PrintResult(result *BenchmarkResult) {
	fmt.Printf("\n===== 性能测试结果 =====\n")
	fmt.Printf("总请求数: %d\n", result.TotalRequests)
	fmt.Printf("并发请求数: %d\n", result.ConcurrentRequests)
	fmt.Printf("总耗时: %v\n", result.TotalTime)
	fmt.Printf("平均响应时间: %v\n", result.AverageTime)
	fmt.Printf("每秒请求数: %.2f\n", result.RequestsPerSecond)
	fmt.Printf("缓存命中: %d\n", result.CacheHits)
	fmt.Printf("缓存未命中: %d\n", result.CacheMisses)
	fmt.Printf("错误数: %d\n", result.Errors)
	fmt.Printf("=====================\n")
}
