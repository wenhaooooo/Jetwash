package benchmark

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"jetwash/internal/service/orchestrator"
	"jetwash/internal/service/queue"
	"jetwash/internal/types"
)

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
	TotalRequests      int           `json:"total_requests"`
	ConcurrentRequests int           `json:"concurrent_requests"`
	TestText           string        `json:"test_text"`
	TenantID           uuid.UUID     `json:"tenant_id"`
	Config             *types.OrchestratorConfig `json:"config"`
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

			reqStart := time.Now()
			_, err := orchestrator.CheckTextWithConfig(
				config.TenantID,
				config.TestText,
				config.Config,
			)
			reqDuration := time.Since(reqStart)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				errors++
			} else {
				// 简单判断：如果请求时间非常短，视为缓存命中
				if reqDuration < 10*time.Millisecond {
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
) (*BenchmarkResult, error) {
	result := &BenchmarkResult{
		TotalRequests: totalTasks,
	}

	startTime := time.Now()

	// 入队所有任务
	taskIDs := make([]string, totalTasks)
	for i := 0; i < totalTasks; i++ {
		task := &queue.DetectionTask{
			TenantID: tenantID,
			Text:     testText,
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
