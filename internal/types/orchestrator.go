package types

import (
	"jetwash/internal/service/layer1_speed"
	"jetwash/internal/service/layer2_semantic"
	"jetwash/internal/service/layer3_reason"
)

// OrchestratorResult 编排结果
type OrchestratorResult struct {
	// 最终结果
	Passed    bool   `json:"passed"`
	RiskLevel int    `json:"risk_level"`
	Message   string `json:"message"`

	// 各层结果
	Layer1Result *layer1_speed.Layer1Result    `json:"layer1_result"`
	Layer2Result *layer2_semantic.Layer2Result `json:"layer2_result"`
	Layer3Result *layer3_reason.Layer3Result   `json:"layer3_result"`

	// 统计信息
	TotalMatches  int   `json:"total_matches"`
	ExecutionTime int64 `json:"execution_time"` // 毫秒

	// 是否来自缓存
	FromCache bool `json:"from_cache"`

	// 异步 LLM 审核相关字段
	// 当 EnableAsyncLLM 开启时，Layer3 推理异步执行，API 立即返回初步结果
	ReviewID     string `json:"review_id,omitempty"`     // 审核任务 ID，可用于查询最终结果
	ReviewStatus string `json:"review_status,omitempty"` // 审核状态: completed / pending_llm_review
}

// OrchestratorConfig 编排配置
type OrchestratorConfig struct {
	// 是否启用各层
	EnableLayer1 bool `json:"enable_layer1"`
	EnableLayer2 bool `json:"enable_layer2"`
	EnableLayer3 bool `json:"enable_layer3"`

	// 是否在第一层就停止（如果匹配到敏感词）
	StopAtLayer1 bool `json:"stop_at_layer1"`

	// 是否在第二层就停止（如果匹配到敏感词）
	StopAtLayer2 bool `json:"stop_at_layer2"`

	// 歧义放行：Layer1 歧义匹配是否放行到 Layer2
	EnableAmbiguityPassThrough bool `json:"enable_ambiguity_pass_through"`

	// Layer2 配置
	Layer2Threshold float64 `json:"layer2_threshold"`
	Layer2Limit     int     `json:"layer2_limit"`

	// Layer3 配置
	Layer3EnableReason bool `json:"layer3_enable_reason"`

	// 非敏感词快速放行：Layer1 无匹配时跳过 Layer2/3，直接返回通过
	// 适用于非敏感词占绝大多数（95%+）的高并发场景，可显著降低延迟和 Embedding API 开销
	EnableFastPass bool `json:"enable_fast_pass"`

	// Layer3 超时控制（毫秒）：LLM 推理的最大允许耗时
	// 超时后降级为基于已有匹配结果的规则判断，不阻塞请求
	// 设为 0 表示不限制超时
	Layer3TimeoutMs int `json:"layer3_timeout_ms"`

	// 异步 LLM 审核：Layer1/2 完成后立即返回初步结果，Layer3 通过 Redis Stream 异步执行
	// 适用于"先发布后审核"场景，用最终一致性换取零 LLM 延迟
	// 开启后 Layer3TimeoutMs 不再生效（LLM 不在请求链路上）
	EnableAsyncLLM bool `json:"enable_async_llm"`
}
