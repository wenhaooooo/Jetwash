package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"jetwash/internal/cache"
	"jetwash/internal/service/orchestrator"
	"jetwash/internal/types"
)

// DetectionTask 检测任务
type DetectionTask struct {
	ID       string                 `json:"id"`
	TenantID uuid.UUID              `json:"tenant_id"`
	Text     string                 `json:"text"`
	Config   *types.OrchestratorConfig `json:"config"`
	Callback string                 `json:"callback,omitempty"`
	CreatedAt time.Time             `json:"created_at"`
}

// DetectionResult 检测结果
type DetectionResult struct {
	TaskID    string                  `json:"task_id"`
	Result    *types.OrchestratorResult `json:"result"`
	Error     string                  `json:"error,omitempty"`
	CompletedAt time.Time             `json:"completed_at"`
}

// QueueService 队列服务接口
type QueueService interface {
	// Enqueue 入队
	Enqueue(ctx context.Context, task *DetectionTask) error
	
	// Dequeue 出队
	Dequeue(ctx context.Context) (*DetectionTask, error)
	
	// Process 处理任务
	Process(ctx context.Context, orchestrator orchestrator.Orchestrator) error
	
	// GetResult 获取任务结果
	GetResult(ctx context.Context, taskID string) (*DetectionResult, error)
	
	// SaveResult 保存任务结果
	SaveResult(ctx context.Context, result *DetectionResult) error
}

// queueService 队列服务实现
type queueService struct {
	redisClient *cache.RedisClient
	queueName   string
	resultPrefix string
}

// NewQueueService 创建队列服务实例
func NewQueueService(redisClient *cache.RedisClient) QueueService {
	return &queueService{
		redisClient: redisClient,
		queueName:   "detection_queue",
		resultPrefix: "detection_result:",
	}
}

// Enqueue 入队
func (q *queueService) Enqueue(ctx context.Context, task *DetectionTask) error {
	if task.ID == "" {
		task.ID = uuid.New().String()
	}
	task.CreatedAt = time.Now()
	
	taskJSON, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}
	
	return q.redisClient.Enqueue(ctx, q.queueName, string(taskJSON))
}

// Dequeue 出队
func (q *queueService) Dequeue(ctx context.Context) (*DetectionTask, error) {
	taskJSON, err := q.redisClient.Dequeue(ctx, q.queueName)
	if err != nil {
		return nil, err
	}
	
	var task DetectionTask
	if err := json.Unmarshal([]byte(taskJSON), &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}
	
	return &task, nil
}

// Process 处理任务
func (q *queueService) Process(ctx context.Context, orchestrator orchestrator.Orchestrator) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		task, err := q.Dequeue(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.Printf("Error dequeuing task: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		log.Printf("Processing task: %s", task.ID)

		// 执行检测
		result, err := orchestrator.CheckTextWithConfig(task.TenantID, task.Text, task.Config)

		// 保存结果
		detectionResult := &DetectionResult{
			TaskID:      task.ID,
			Result:      result,
			CompletedAt: time.Now(),
		}

		if err != nil {
			detectionResult.Error = err.Error()
			log.Printf("Error processing task %s: %v", task.ID, err)
		} else {
			log.Printf("Task %s completed successfully", task.ID)
		}

		if err := q.SaveResult(ctx, detectionResult); err != nil {
			log.Printf("Error saving result: %v", err)
		}
	}
}

// GetResult 获取任务结果
func (q *queueService) GetResult(ctx context.Context, taskID string) (*DetectionResult, error) {
	resultKey := q.resultPrefix + taskID
	resultJSON, err := q.redisClient.Get(ctx, resultKey)
	if err != nil {
		return nil, err
	}
	
	var result DetectionResult
	if err := json.Unmarshal([]byte(resultJSON), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}
	
	return &result, nil
}

// SaveResult 保存任务结果
func (q *queueService) SaveResult(ctx context.Context, result *DetectionResult) error {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}
	
	resultKey := q.resultPrefix + result.TaskID
	return q.redisClient.Set(ctx, resultKey, resultJSON, 24*time.Hour)
}
