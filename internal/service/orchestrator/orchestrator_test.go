package orchestrator

import (
	"testing"
	"time"

	"jetwash/internal/service/layer1_speed"
	"jetwash/internal/service/layer2_semantic"
	"jetwash/internal/service/layer3_reason"
	"jetwash/internal/types"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestDefaultOrchestratorConfig(t *testing.T) {
	cfg := DefaultOrchestratorConfig()
	assert.True(t, cfg.EnableLayer1)
	assert.True(t, cfg.EnableLayer2)
	assert.True(t, cfg.EnableLayer3)
	assert.True(t, cfg.EnableFastPass)
	assert.Equal(t, 0.3, cfg.Layer2Threshold)
	assert.Equal(t, 10, cfg.Layer2Limit)
	assert.Equal(t, 30000, cfg.Layer3TimeoutMs)
}

func TestAggregateRiskLevel(t *testing.T) {
	o := &orchestrator{}
	result := &types.OrchestratorResult{
		Layer1Result: &layer1_speed.Layer1Result{RiskLevel: 2},
		Layer2Result: &layer2_semantic.Layer2Result{RiskLevel: 4},
		Layer3Result: &layer3_reason.Layer3Result{RiskLevel: 3},
	}
	assert.Equal(t, 4, o.AggregateRiskLevel(result))
}

func TestAggregateRiskLevel_NilLayers(t *testing.T) {
	o := &orchestrator{}
	result := &types.OrchestratorResult{}
	assert.Equal(t, 0, o.AggregateRiskLevel(result))
}

func TestAggregateCategories(t *testing.T) {
	o := &orchestrator{}
	result := &types.OrchestratorResult{
		Layer1Result: &layer1_speed.Layer1Result{Categories: []string{"profanity", "violence"}},
		Layer2Result: &layer2_semantic.Layer2Result{Categories: []string{"violence", "spam"}},
	}
	categories := o.AggregateCategories(result)
	assert.Len(t, categories, 3)
}

func TestBuildSummary_Passed(t *testing.T) {
	o := &orchestrator{}
	result := &types.OrchestratorResult{Passed: true}
	summary := o.BuildSummary(result)
	assert.Contains(t, summary, "审查通过")
}

func TestBuildSummary_Failed(t *testing.T) {
	o := &orchestrator{}
	result := &types.OrchestratorResult{
		Passed:    false,
		RiskLevel: 4,
		Layer1Result: &layer1_speed.Layer1Result{
			HasMatch:     true,
			MatchedWords: []*layer1_speed.MatchResult{{Matched: "bad"}},
		},
	}
	summary := o.BuildSummary(result)
	assert.Contains(t, summary, "未通过")
	assert.Contains(t, summary, "4")
}

func TestGenerateCacheKey_DifferentForDifferentText(t *testing.T) {
	tenantID := uuid.New()
	key1 := generateCacheKey(tenantID, "full", "text one")
	key2 := generateCacheKey(tenantID, "full", "text two")
	assert.NotEqual(t, key1, key2)
}

func TestGenerateCacheKey_DifferentForDifferentMode(t *testing.T) {
	tenantID := uuid.New()
	key1 := generateCacheKey(tenantID, "full", "same text")
	key2 := generateCacheKey(tenantID, "basic", "same text")
	assert.NotEqual(t, key1, key2)
}

func TestGetCacheTTL(t *testing.T) {
	o := &orchestrator{}

	// High risk (passed=false, risk>=4) -> 7 days
	result := &types.OrchestratorResult{Passed: false, RiskLevel: 4}
	assert.Equal(t, 7*24*time.Hour, o.getCacheTTL(result))

	// Medium risk (passed=false, risk<4) -> 1 day
	result = &types.OrchestratorResult{Passed: false, RiskLevel: 2}
	assert.Equal(t, 24*time.Hour, o.getCacheTTL(result))

	// Passed -> 1 hour
	result = &types.OrchestratorResult{Passed: true, RiskLevel: 0}
	assert.Equal(t, 1*time.Hour, o.getCacheTTL(result))
}
