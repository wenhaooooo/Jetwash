package layer3_reason

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLayer3Service(t *testing.T) {
	mock := NewMockLLMProvider("test response")
	svc := NewLayer3Service(mock)
	assert.NotNil(t, svc)
}

func TestReasonText_EmptyText(t *testing.T) {
	mock := NewMockLLMProvider("")
	svc := NewLayer3Service(mock)
	_, err := svc.ReasonText(context.Background(), uuid.New(), "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "text cannot be empty")
}

func TestReasonText_ValidText(t *testing.T) {
	mock := NewMockLLMProvider(`风险等级: 0
是否有风险: 否
风险理由: 无风险
检测到的违禁词: []
建议: 无需修改
是否批准: 是
置信度: 0.95
推理过程: 文本内容合规`)
	svc := NewLayer3Service(mock)
	result, err := svc.ReasonText(context.Background(), uuid.New(), "hello world", nil)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.HasRisk)
	assert.Equal(t, 0, result.RiskLevel)
	assert.True(t, result.IsApproved)
	assert.InDelta(t, 0.95, result.Confidence, 0.01)
}

func TestReasonText_NilReasonContext(t *testing.T) {
	mock := NewMockLLMProvider(`风险等级: 0
是否有风险: 否
风险理由: 无
是否批准: 是
置信度: 0.9
推理过程: 正常`)
	svc := NewLayer3Service(mock)
	result, err := svc.ReasonText(context.Background(), uuid.New(), "safe text", nil)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

func TestReasonWithMatches_EmptyText(t *testing.T) {
	mock := NewMockLLMProvider("")
	svc := NewLayer3Service(mock)
	_, err := svc.ReasonWithMatches(context.Background(), uuid.New(), "", nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "text cannot be empty")
}

func TestReasonWithMatches_WithMatches(t *testing.T) {
	mock := NewMockLLMProvider(`风险等级: 3
是否有风险: 是
风险理由: 包含不当内容
检测到的违禁词: [bad]
建议: [删除不当内容]
是否批准: 否
置信度: 0.85
推理过程: 文本包含违规词汇`)
	svc := NewLayer3Service(mock)
	matches := []MatchInfo{
		{WordText: "bad", Category: "profanity", RiskLevel: 3},
	}
	result, err := svc.ReasonWithMatches(context.Background(), uuid.New(), "some text", matches, nil)
	require.NoError(t, err)
	assert.True(t, result.HasRisk)
	assert.Equal(t, 3, result.RiskLevel)
	assert.False(t, result.IsApproved)
}

func TestGeneratePrompt_ContainsText(t *testing.T) {
	mock := NewMockLLMProvider("")
	svc := NewLayer3Service(mock)
	prompt := svc.GeneratePrompt(uuid.New(), "test text", nil, nil)
	assert.Contains(t, prompt, "test text")
	assert.Contains(t, prompt, "风险等级")
}

func TestGeneratePrompt_WithMatches(t *testing.T) {
	mock := NewMockLLMProvider("")
	svc := NewLayer3Service(mock)
	matches := []MatchInfo{
		{WordText: "bad", Category: "profanity", RiskLevel: 3},
	}
	prompt := svc.GeneratePrompt(uuid.New(), "test text", matches, nil)
	assert.Contains(t, prompt, "bad")
	assert.Contains(t, prompt, "profanity")
	assert.Contains(t, prompt, "匹配到的敏感词")
}

func TestGeneratePrompt_WithContext(t *testing.T) {
	mock := NewMockLLMProvider("")
	svc := NewLayer3Service(mock)
	ctx := NewReasonContext().
		WithScenario("social media").
		WithCustomRule("no profanity")
	prompt := svc.GeneratePrompt(uuid.New(), "test text", nil, ctx)
	assert.Contains(t, prompt, "social media")
	assert.Contains(t, prompt, "no profanity")
	assert.Contains(t, prompt, "应用场景")
	assert.Contains(t, prompt, "自定义规则")
}

func TestGeneratePrompt_WithUserContext(t *testing.T) {
	mock := NewMockLLMProvider("")
	svc := NewLayer3Service(mock)
	ctx := NewReasonContext().
		WithUserContext("new user")
	prompt := svc.GeneratePrompt(uuid.New(), "test text", nil, ctx)
	assert.Contains(t, prompt, "new user")
	assert.Contains(t, prompt, "用户上下文")
}

func TestGeneratePrompt_WithExamples(t *testing.T) {
	mock := NewMockLLMProvider("")
	svc := NewLayer3Service(mock)
	ctx := NewReasonContext().
		WithExample(Example{Text: "example text", Result: "approved", Reason: "safe"})
	prompt := svc.GeneratePrompt(uuid.New(), "test text", nil, ctx)
	assert.Contains(t, prompt, "example text")
	assert.Contains(t, prompt, "approved")
	assert.Contains(t, prompt, "参考示例")
}

func TestGenerateSystemPrompt_ContainsKeywords(t *testing.T) {
	mock := NewMockLLMProvider("")
	svc := NewLayer3Service(mock)
	prompt := svc.GenerateSystemPrompt(uuid.New())
	assert.Contains(t, prompt, "文本内容审查助手")
	assert.Contains(t, prompt, "风险等级")
	assert.Contains(t, prompt, "审查原则")
}

func TestParseLLMResponse_ValidResponse(t *testing.T) {
	svc := &layer3Service{}
	response := `风险等级: 3
是否有风险: 是
风险理由: 包含不当内容
检测到的违禁词: [bad, worse]
建议: [删除不当内容, 修改措辞]
是否批准: 否
置信度: 0.85
推理过程: 文本包含明显违规词汇`
	result := svc.parseLLMResponse(response)
	assert.Equal(t, 3, result.RiskLevel)
	assert.True(t, result.HasRisk)
	assert.Contains(t, result.RiskReason, "不当内容")
	assert.False(t, result.IsApproved)
	assert.InDelta(t, 0.85, result.Confidence, 0.01)
	assert.Len(t, result.DetectedWords, 2)
	assert.Contains(t, result.DetectedWords, "bad")
	assert.Len(t, result.Suggestions, 2)
}

func TestParseLLMResponse_EmptyResponse(t *testing.T) {
	svc := &layer3Service{}
	result := svc.parseLLMResponse("")
	assert.False(t, result.HasRisk)
	assert.Equal(t, 0, result.RiskLevel)
	assert.True(t, result.IsApproved)
	assert.InDelta(t, 0.0, result.Confidence, 0.001)
	assert.Empty(t, result.DetectedWords)
	assert.Empty(t, result.Suggestions)
}

func TestParseLLMResponse_PartialResponse(t *testing.T) {
	svc := &layer3Service{}
	response := `风险等级: 2
是否有风险: 是`
	result := svc.parseLLMResponse(response)
	assert.Equal(t, 2, result.RiskLevel)
	assert.True(t, result.HasRisk)
	// Remaining fields should have defaults
	assert.True(t, result.IsApproved)
	assert.Empty(t, result.RiskReason)
}

func TestParseLLMResponse_NoRisk(t *testing.T) {
	svc := &layer3Service{}
	response := `风险等级: 0
是否有风险: 否
风险理由: 无
是否批准: 是
置信度: 0.9
推理过程: 内容合规`
	result := svc.parseLLMResponse(response)
	assert.Equal(t, 0, result.RiskLevel)
	assert.False(t, result.HasRisk)
	assert.True(t, result.IsApproved)
	assert.InDelta(t, 0.9, result.Confidence, 0.01)
}

func TestReasonContext_Builder(t *testing.T) {
	ctx := NewReasonContext().
		WithScenario("test").
		WithUserContext("user info").
		WithCustomRule("rule1").
		WithCustomRules([]string{"rule2", "rule3"}).
		WithExample(Example{Text: "ex", Result: "ok", Reason: "because"}).
		WithTemperature(0.5).
		WithMaxTokens(500).
		WithEnableReason(false)
	assert.Equal(t, "test", ctx.Scenario)
	assert.Equal(t, "user info", ctx.UserContext)
	// WithCustomRules replaces the slice, so only "rule2" and "rule3" remain
	assert.Len(t, ctx.CustomRules, 2)
	assert.Contains(t, ctx.CustomRules, "rule2")
	assert.Contains(t, ctx.CustomRules, "rule3")
	assert.Len(t, ctx.Examples, 1)
	assert.InDelta(t, 0.5, ctx.Temperature, 0.001)
	assert.Equal(t, 500, ctx.MaxTokens)
	assert.False(t, ctx.EnableReason)
}

func TestReasonContext_DefaultValues(t *testing.T) {
	ctx := NewReasonContext()
	assert.Equal(t, "", ctx.Scenario)
	assert.Equal(t, "", ctx.UserContext)
	assert.Empty(t, ctx.CustomRules)
	assert.Empty(t, ctx.Examples)
	assert.InDelta(t, 0.7, ctx.Temperature, 0.001)
	assert.Equal(t, 1000, ctx.MaxTokens)
	assert.True(t, ctx.EnableReason)
}

func TestReasonContext_ChainingOrder(t *testing.T) {
	// Test that WithCustomRule appends while WithCustomRules replaces
	ctx := NewReasonContext().
		WithCustomRule("rule1").
		WithCustomRule("rule2").
		WithCustomRules([]string{"rule3", "rule4", "rule5"})
	assert.Len(t, ctx.CustomRules, 3)
	assert.Equal(t, "rule3", ctx.CustomRules[0])
	assert.Equal(t, "rule4", ctx.CustomRules[1])
	assert.Equal(t, "rule5", ctx.CustomRules[2])
}

func TestReasonContext_WithExamples(t *testing.T) {
	ctx := NewReasonContext().
		WithExample(Example{Text: "ex1", Result: "ok", Reason: "r1"}).
		WithExamples([]Example{
			{Text: "ex2", Result: "no", Reason: "r2"},
			{Text: "ex3", Result: "ok", Reason: "r3"},
		})
	// WithExamples replaces the slice
	assert.Len(t, ctx.Examples, 2)
	assert.Equal(t, "ex2", ctx.Examples[0].Text)
	assert.Equal(t, "ex3", ctx.Examples[1].Text)
}

func TestMockLLMProvider(t *testing.T) {
	mock := NewMockLLMProvider("test response")
	text, err := mock.GenerateText(context.Background(), "prompt")
	require.NoError(t, err)
	assert.Equal(t, "test response", text)

	text, err = mock.GenerateTextWithMessages(context.Background(), []Message{{Role: "user", Content: "hi"}})
	require.NoError(t, err)
	assert.Equal(t, "test response", text)
}

func TestReasonText_LLMError(t *testing.T) {
	mock := NewMockLLMProviderError()
	svc := NewLayer3Service(mock)
	_, err := svc.ReasonText(context.Background(), uuid.New(), "some text", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to generate text")
}

func TestReasonWithMatches_LLMError(t *testing.T) {
	mock := NewMockLLMProviderError()
	svc := NewLayer3Service(mock)
	_, err := svc.ReasonWithMatches(context.Background(), uuid.New(), "some text", nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to generate text")
}

func TestMatchInfo_Fields(t *testing.T) {
	match := MatchInfo{
		WordText:  "test",
		Category:  "profanity",
		RiskLevel: 3,
		Distance:  0.95,
		Position:  10,
		MatchType: "exact",
	}
	assert.Equal(t, "test", match.WordText)
	assert.Equal(t, "profanity", match.Category)
	assert.Equal(t, 3, match.RiskLevel)
	assert.InDelta(t, 0.95, match.Distance, 0.001)
	assert.Equal(t, 10, match.Position)
	assert.Equal(t, "exact", match.MatchType)
}
