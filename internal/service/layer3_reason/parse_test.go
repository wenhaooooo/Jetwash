package layer3_reason

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// mockLLMProvider 用于测试的 mock LLM 提供者
type mockLLMProvider struct {
	response string
	err      error
}

func (m *mockLLMProvider) GenerateText(ctx context.Context, prompt string) (string, error) {
	return m.response, m.err
}

func (m *mockLLMProvider) GenerateTextWithMessages(ctx context.Context, messages []Message) (string, error) {
	return m.response, m.err
}

func TestParseLLMResponseJSON_ValidJSON(t *testing.T) {
	svc := NewLayer3Service(&mockLLMProvider{})

	input := `{
		"risk_level": 3,
		"has_risk": true,
		"risk_reason": "检测到敏感内容",
		"detected_words": ["赌博", "色情"],
		"suggestions": ["建议删除相关内容"],
		"is_approved": false,
		"confidence": 0.92,
		"reasoning": "文本包含违禁词"
	}`

	result := svc.(*layer3Service).parseLLMResponseJSON(input)

	if result.RiskLevel != 3 {
		t.Errorf("RiskLevel = %d, want 3", result.RiskLevel)
	}
	if !result.HasRisk {
		t.Error("HasRisk = false, want true")
	}
	if result.RiskReason != "检测到敏感内容" {
		t.Errorf("RiskReason = %q, want %q", result.RiskReason, "检测到敏感内容")
	}
	if len(result.DetectedWords) != 2 {
		t.Errorf("DetectedWords len = %d, want 2", len(result.DetectedWords))
	}
	if result.DetectedWords[0] != "赌博" || result.DetectedWords[1] != "色情" {
		t.Errorf("DetectedWords = %v, want [赌博 色情]", result.DetectedWords)
	}
	if len(result.Suggestions) != 1 {
		t.Errorf("Suggestions len = %d, want 1", len(result.Suggestions))
	}
	if result.IsApproved {
		t.Error("IsApproved = true, want false")
	}
	if result.Confidence != 0.92 {
		t.Errorf("Confidence = %f, want 0.92", result.Confidence)
	}
	if result.Reasoning != "文本包含违禁词" {
		t.Errorf("Reasoning = %q, want %q", result.Reasoning, "文本包含违禁词")
	}
}

func TestParseLLMResponseJSON_MarkdownWrapped(t *testing.T) {
	svc := NewLayer3Service(&mockLLMProvider{})

	input := "这是分析结果：\n```json\n" + `{
		"risk_level": 0,
		"has_risk": false,
		"risk_reason": "无风险",
		"detected_words": [],
		"suggestions": [],
		"is_approved": true,
		"confidence": 0.98,
		"reasoning": "文本内容正常"
	}` + "\n```\n以上是分析结果。"

	result := svc.(*layer3Service).parseLLMResponseJSON(input)

	if result.RiskLevel != 0 {
		t.Errorf("RiskLevel = %d, want 0", result.RiskLevel)
	}
	if result.HasRisk {
		t.Error("HasRisk = true, want false")
	}
	if !result.IsApproved {
		t.Error("IsApproved = false, want true")
	}
	if result.Confidence != 0.98 {
		t.Errorf("Confidence = %f, want 0.98", result.Confidence)
	}
	if result.Reasoning != "文本内容正常" {
		t.Errorf("Reasoning = %q, want %q", result.Reasoning, "文本内容正常")
	}
	if len(result.DetectedWords) != 0 {
		t.Errorf("DetectedWords len = %d, want 0", len(result.DetectedWords))
	}
}

func TestParseLLMResponseJSON_InvalidJSON_FallbackToText(t *testing.T) {
	svc := NewLayer3Service(&mockLLMProvider{})

	input := `风险等级: 2
是否有风险: 是
风险理由: 存在违规内容
检测到的违禁词: [赌博]
建议: [删除相关内容]
是否批准: 否
置信度: 0.85
推理过程: 综合分析结果`

	result := svc.(*layer3Service).parseLLMResponseJSON(input)

	if result.RiskLevel != 2 {
		t.Errorf("RiskLevel = %d, want 2", result.RiskLevel)
	}
	if !result.HasRisk {
		t.Error("HasRisk = false, want true")
	}
	if result.RiskReason != "存在违规内容" {
		t.Errorf("RiskReason = %q, want %q", result.RiskReason, "存在违规内容")
	}
	if result.IsApproved {
		t.Error("IsApproved = true, want false")
	}
	if result.Confidence != 0.85 {
		t.Errorf("Confidence = %f, want 0.85", result.Confidence)
	}
}

func TestParseLLMResponseJSON_EmptyResponse(t *testing.T) {
	svc := NewLayer3Service(&mockLLMProvider{})

	result := svc.(*layer3Service).parseLLMResponseJSON("")

	if result.RiskLevel != 0 {
		t.Errorf("RiskLevel = %d, want 0", result.RiskLevel)
	}
	if result.HasRisk {
		t.Error("HasRisk = true, want false")
	}
	if !result.IsApproved {
		t.Error("IsApproved = false, want true")
	}
	if len(result.DetectedWords) != 0 {
		t.Errorf("DetectedWords len = %d, want 0", len(result.DetectedWords))
	}
	if len(result.Suggestions) != 0 {
		t.Errorf("Suggestions len = %d, want 0", len(result.Suggestions))
	}
}

func TestReasonText_UsesJSONParsing(t *testing.T) {
	jsonResponse := `{
		"risk_level": 1,
		"has_risk": false,
		"risk_reason": "轻微问题",
		"detected_words": [],
		"suggestions": ["注意措辞"],
		"is_approved": true,
		"confidence": 0.75,
		"reasoning": "整体风险较低"
	}`

	svc := NewLayer3Service(&mockLLMProvider{response: jsonResponse})
	result, err := svc.ReasonText(context.Background(), uuid.New(), "测试文本", nil)

	if err != nil {
		t.Fatalf("ReasonText returned error: %v", err)
	}
	if result.RiskLevel != 1 {
		t.Errorf("RiskLevel = %d, want 1", result.RiskLevel)
	}
	if !result.IsApproved {
		t.Error("IsApproved = false, want true")
	}
	if len(result.Suggestions) != 1 || result.Suggestions[0] != "注意措辞" {
		t.Errorf("Suggestions = %v, want [注意措辞]", result.Suggestions)
	}
}

func TestReasonWithMatches_UsesJSONParsing(t *testing.T) {
	jsonResponse := `{
		"risk_level": 4,
		"has_risk": true,
		"risk_reason": "严重违规",
		"detected_words": ["违禁词A"],
		"suggestions": ["立即删除"],
		"is_approved": false,
		"confidence": 0.95,
		"reasoning": "匹配到高风险违禁词"
	}`

	svc := NewLayer3Service(&mockLLMProvider{response: jsonResponse})
	matches := []MatchInfo{
		{WordText: "违禁词A", Category: "违法", RiskLevel: 4},
	}
	result, err := svc.ReasonWithMatches(context.Background(), uuid.New(), "测试文本", matches, nil)

	if err != nil {
		t.Fatalf("ReasonWithMatches returned error: %v", err)
	}
	if result.RiskLevel != 4 {
		t.Errorf("RiskLevel = %d, want 4", result.RiskLevel)
	}
	if result.IsApproved {
		t.Error("IsApproved = true, want false")
	}
	if len(result.DetectedWords) != 1 || result.DetectedWords[0] != "违禁词A" {
		t.Errorf("DetectedWords = %v, want [违禁词A]", result.DetectedWords)
	}
}

func TestGeneratePrompt_RequestsJSONFormat(t *testing.T) {
	svc := NewLayer3Service(&mockLLMProvider{})
	prompt := svc.GeneratePrompt(uuid.New(), "测试文本", nil, nil)

	if !contains(prompt, "JSON") {
		t.Error("Prompt should request JSON format")
	}
	if !contains(prompt, "risk_level") {
		t.Error("Prompt should contain risk_level field example")
	}
	if !contains(prompt, "```json") {
		t.Error("Prompt should contain json code block")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
