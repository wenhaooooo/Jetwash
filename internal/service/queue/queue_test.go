package queue

import (
	"testing"
	"time"

	"jetwash/internal/types"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestDetectionTask_ID(t *testing.T) {
	task := &DetectionTask{
		TenantID: uuid.New(),
		Text:     "test",
	}
	assert.Empty(t, task.ID)

	// Simulate what Enqueue does: assign ID if empty
	if task.ID == "" {
		task.ID = uuid.New().String()
	}
	assert.NotEmpty(t, task.ID)
}

func TestDetectionTask_Fields(t *testing.T) {
	tenantID := uuid.New()
	now := time.Now()
	config := &types.OrchestratorConfig{
		EnableLayer1: true,
		EnableLayer2: true,
		EnableLayer3: false,
	}

	task := &DetectionTask{
		ID:        "test-task-id",
		TenantID:  tenantID,
		Text:      "hello world",
		Config:    config,
		Callback:  "http://example.com/callback",
		CreatedAt: now,
	}

	assert.Equal(t, "test-task-id", task.ID)
	assert.Equal(t, tenantID, task.TenantID)
	assert.Equal(t, "hello world", task.Text)
	assert.NotNil(t, task.Config)
	assert.True(t, task.Config.EnableLayer1)
	assert.Equal(t, "http://example.com/callback", task.Callback)
	assert.Equal(t, now, task.CreatedAt)
}

func TestDetectionResult_Fields(t *testing.T) {
	result := &DetectionResult{
		TaskID: "test-id",
		Result: &types.OrchestratorResult{
			Passed:    true,
			RiskLevel: 0,
		},
		CompletedAt: time.Now(),
	}

	assert.Equal(t, "test-id", result.TaskID)
	assert.NotNil(t, result.Result)
	assert.True(t, result.Result.Passed)
	assert.Equal(t, 0, result.Result.RiskLevel)
	assert.Empty(t, result.Error)
}

func TestDetectionResult_WithError(t *testing.T) {
	result := &DetectionResult{
		TaskID:      "task-err",
		Result:      nil,
		Error:       "something went wrong",
		CompletedAt: time.Now(),
	}

	assert.Equal(t, "task-err", result.TaskID)
	assert.Nil(t, result.Result)
	assert.Equal(t, "something went wrong", result.Error)
}

func TestDetectionTask_EmptyCallback(t *testing.T) {
	task := &DetectionTask{
		TenantID: uuid.New(),
		Text:     "test",
	}

	assert.Empty(t, task.Callback)
}

func TestQueueServiceInterface(t *testing.T) {
	// Verify that queueService satisfies QueueService at compile time.
	// This is a compile-time check; if the type doesn't implement the
	// interface, this test file won't compile.
	var _ QueueService = (*queueService)(nil)
}
