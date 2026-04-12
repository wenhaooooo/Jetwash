package layer3_reason

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"jetwash/internal/util"

	"github.com/google/uuid"
)

// LLMProvider LLM 服务提供者接口
type LLMProvider interface {
	// GenerateText 生成文本
	GenerateText(prompt string) (string, error)

	// GenerateTextWithMessages 使用消息列表生成文本
	GenerateTextWithMessages(messages []Message) (string, error)
}

// Message LLM 消息
type Message struct {
	Role    string `json:"role"` // system, user, assistant
	Content string `json:"content"`
}

// Layer3Result 第三层结果
type Layer3Result struct {
	HasRisk       bool     `json:"has_risk"`
	RiskLevel     int      `json:"risk_level"`
	RiskReason    string   `json:"risk_reason"`
	Suggestions   []string `json:"suggestions"`
	IsApproved    bool     `json:"is_approved"`
	Confidence    float64  `json:"confidence"`
	Reasoning     string   `json:"reasoning"`
	DetectedWords []string `json:"detected_words"` // LLM识别出的违禁词
}

// Layer3Service 第三层服务接口 - 推理层
type Layer3Service interface {
	// ReasonText 对文本进行推理分析
	ReasonText(tenantID uuid.UUID, text string, context *ReasonContext) (*Layer3Result, error)

	// ReasonWithMatches 基于匹配结果进行推理
	ReasonWithMatches(tenantID uuid.UUID, text string, matches []MatchInfo, context *ReasonContext) (*Layer3Result, error)

	// GeneratePrompt 生成 Prompt
	GeneratePrompt(tenantID uuid.UUID, text string, matches []MatchInfo, context *ReasonContext) string

	// GenerateSystemPrompt 生成系统 Prompt
	GenerateSystemPrompt(tenantID uuid.UUID) string
}

// layer3Service 第三层服务实现
type layer3Service struct {
	llmProvider LLMProvider
}

// NewLayer3Service 创建第三层服务实例
func NewLayer3Service(llmProvider LLMProvider) Layer3Service {
	return &layer3Service{
		llmProvider: llmProvider,
	}
}

// ReasonText 对文本进行推理分析
func (s *layer3Service) ReasonText(tenantID uuid.UUID, text string, context *ReasonContext) (*Layer3Result, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	if context == nil {
		context = NewReasonContext()
	}

	// 生成 Prompt
	prompt := s.GeneratePrompt(tenantID, text, nil, context)

	// 调用 LLM
	response, err := s.llmProvider.GenerateText(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate text: %w", err)
	}

	// 解析响应
	result := s.parseLLMResponse(response)

	return result, nil
}

// ReasonWithMatches 基于匹配结果进行推理
func (s *layer3Service) ReasonWithMatches(tenantID uuid.UUID, text string, matches []MatchInfo, context *ReasonContext) (*Layer3Result, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	if context == nil {
		context = NewReasonContext()
	}

	// 生成 Prompt
	prompt := s.GeneratePrompt(tenantID, text, matches, context)

	// 调用 LLM
	response, err := s.llmProvider.GenerateText(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate text: %w", err)
	}

	// 解析响应
	result := s.parseLLMResponse(response)

	return result, nil
}

// GeneratePrompt 生成 Prompt
func (s *layer3Service) GeneratePrompt(tenantID uuid.UUID, text string, matches []MatchInfo, context *ReasonContext) string {
	var builder strings.Builder

	// 系统提示
	builder.WriteString(s.GenerateSystemPrompt(tenantID))
	builder.WriteString("\n\n")

	// 用户提示
	builder.WriteString("## 待审查文本\n\n")
	builder.WriteString(text)
	builder.WriteString("\n\n")

	// 如果有匹配结果，添加匹配信息
	if len(matches) > 0 {
		builder.WriteString("## 匹配到的敏感词\n\n")
		for i, match := range matches {
			builder.WriteString(fmt.Sprintf("%d. %s (分类: %s, 风险等级: %d)\n", i+1, match.WordText, match.Category, match.RiskLevel))
		}
		builder.WriteString("\n")
	}

	// 添加上下文信息
	if context != nil {
		if context.Scenario != "" {
			builder.WriteString(fmt.Sprintf("## 应用场景\n\n%s\n\n", context.Scenario))
		}

		if context.UserContext != "" {
			builder.WriteString(fmt.Sprintf("## 用户上下文\n\n%s\n\n", context.UserContext))
		}

		if len(context.CustomRules) > 0 {
			builder.WriteString("## 自定义规则\n\n")
			for i, rule := range context.CustomRules {
				builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, rule))
			}
			builder.WriteString("\n")
		}

		if len(context.Examples) > 0 {
			builder.WriteString("## 参考示例\n\n")
			for i, example := range context.Examples {
				builder.WriteString(fmt.Sprintf("示例 %d:\n", i+1))
				builder.WriteString(fmt.Sprintf("文本: %s\n", example.Text))
				builder.WriteString(fmt.Sprintf("结果: %s\n", example.Result))
				builder.WriteString(fmt.Sprintf("理由: %s\n\n", example.Reason))
			}
		}
	}

	// 添加任务要求
	builder.WriteString("## 任务要求\n\n")
	builder.WriteString("请基于以上信息，对文本进行深入分析，并按照以下格式返回结果：\n\n")
	builder.WriteString("```\n")
	builder.WriteString("风险等级: 0-5\n")
	builder.WriteString("是否有风险: 是/否\n")
	builder.WriteString("风险理由: [详细说明]\n")
	builder.WriteString("检测到的违禁词: [违禁词1, 违禁词2, ...]\n")
	builder.WriteString("建议: [建议1, 建议2, ...]\n")
	builder.WriteString("是否批准: 是/否\n")
	builder.WriteString("置信度: 0.0-1.0\n")
	builder.WriteString("推理过程: [详细推理过程]\n")
	builder.WriteString("```\n")

	return builder.String()
}

// GenerateSystemPrompt 生成系统 Prompt
func (s *layer3Service) GenerateSystemPrompt(tenantID uuid.UUID) string {
	return `你是一个专业的文本内容审查助手，负责分析和评估文本内容的合规性和风险。

## 你的职责

1. 仔细分析待审查文本，识别其中的潜在风险
2. 基于匹配到的敏感词和上下文信息，进行综合判断
3. 提供详细的风险分析和合理的建议
4. 给出明确的审查结论和置信度

## 风险等级定义

- 0: 无风险 - 文本完全合规，没有任何问题
- 1: 低风险 - 文本存在轻微问题，但不影响整体合规性
- 2: 中低风险 - 文本存在一些问题，需要关注
- 3: 中等风险 - 文本存在明显问题，需要谨慎处理
- 4: 较高风险 - 文本存在严重问题，建议拒绝
- 5: 高风险 - 文本存在严重违规内容，必须拒绝

## 审查原则

1. 客观公正：基于事实和规则进行判断，不受主观偏见影响
2. 上下文理解：充分考虑文本的使用场景和上下文
3. 精准判断：准确识别风险，避免误判和漏判
4. 合理建议：提供有针对性的改进建议

## 输出要求

- 使用简洁明了的语言
- 提供充分的推理依据
- 给出明确的风险等级和审查结论
- 提供实用的改进建议
- 评估判断的置信度

请始终保持专业、客观、严谨的态度。`
}

// parseLLMResponse 解析 LLM 响应
func (s *layer3Service) parseLLMResponse(response string) *Layer3Result {
	result := &Layer3Result{
		HasRisk:       false,
		RiskLevel:     0,
		RiskReason:    "",
		Suggestions:   make([]string, 0),
		IsApproved:    true,
		Confidence:    0.0,
		Reasoning:     "",
		DetectedWords: make([]string, 0),
	}

	// 简单解析响应（实际应用中应该使用更复杂的解析逻辑）
	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "风险等级:") {
			fmt.Sscanf(line, "风险等级: %d", &result.RiskLevel)
		} else if strings.HasPrefix(line, "是否有风险:") {
			var hasRiskStr string
			fmt.Sscanf(line, "是否有风险: %s", &hasRiskStr)
			result.HasRisk = (hasRiskStr == "是")
		} else if strings.HasPrefix(line, "风险理由:") {
			result.RiskReason = strings.TrimPrefix(line, "风险理由:")
			result.RiskReason = strings.TrimSpace(result.RiskReason)
		} else if strings.HasPrefix(line, "检测到的违禁词:") {
			detectedWordsStr := strings.TrimPrefix(line, "检测到的违禁词:")
			detectedWordsStr = strings.TrimSpace(detectedWordsStr)
			if detectedWordsStr != "" && detectedWordsStr != "[]" {
				// 移除方括号并分割违禁词
				detectedWordsStr = strings.TrimPrefix(detectedWordsStr, "[")
				detectedWordsStr = strings.TrimSuffix(detectedWordsStr, "]")
				result.DetectedWords = strings.Split(detectedWordsStr, ",")
				for i := range result.DetectedWords {
					result.DetectedWords[i] = strings.TrimSpace(result.DetectedWords[i])
				}
			}
		} else if strings.HasPrefix(line, "建议:") {
			suggestionsStr := strings.TrimPrefix(line, "建议:")
			suggestionsStr = strings.TrimSpace(suggestionsStr)
			if suggestionsStr != "" {
				// 简单分割建议
				result.Suggestions = strings.Split(suggestionsStr, ",")
				for i := range result.Suggestions {
					result.Suggestions[i] = strings.TrimSpace(result.Suggestions[i])
				}
			}
		} else if strings.HasPrefix(line, "是否批准:") {
			var isApprovedStr string
			fmt.Sscanf(line, "是否批准: %s", &isApprovedStr)
			result.IsApproved = (isApprovedStr == "是")
		} else if strings.HasPrefix(line, "置信度:") {
			fmt.Sscanf(line, "置信度: %f", &result.Confidence)
		} else if strings.HasPrefix(line, "推理过程:") {
			result.Reasoning = strings.TrimPrefix(line, "推理过程:")
			result.Reasoning = strings.TrimSpace(result.Reasoning)
		}
	}

	return result
}

// MatchInfo 匹配信息
type MatchInfo struct {
	WordText  string  `json:"word_text"`
	Category  string  `json:"category"`
	RiskLevel int     `json:"risk_level"`
	Distance  float64 `json:"distance"`
	Position  int     `json:"position"`
	MatchType string  `json:"match_type"`
}

// ReasonContext 推理上下文
type ReasonContext struct {
	Scenario     string    `json:"scenario"`      // 应用场景
	UserContext  string    `json:"user_context"`  // 用户上下文
	CustomRules  []string  `json:"custom_rules"`  // 自定义规则
	Examples     []Example `json:"examples"`      // 参考示例
	Temperature  float64   `json:"temperature"`   // LLM 温度参数
	MaxTokens    int       `json:"max_tokens"`    // 最大 token 数
	EnableReason bool      `json:"enable_reason"` // 是否启用推理
}

// Example 参考示例
type Example struct {
	Text   string `json:"text"`
	Result string `json:"result"`
	Reason string `json:"reason"`
}

// NewReasonContext 创建推理上下文
func NewReasonContext() *ReasonContext {
	return &ReasonContext{
		Scenario:     "",
		UserContext:  "",
		CustomRules:  make([]string, 0),
		Examples:     make([]Example, 0),
		Temperature:  0.7,
		MaxTokens:    1000,
		EnableReason: true,
	}
}

// WithScenario 设置应用场景
func (rc *ReasonContext) WithScenario(scenario string) *ReasonContext {
	rc.Scenario = scenario
	return rc
}

// WithUserContext 设置用户上下文
func (rc *ReasonContext) WithUserContext(userContext string) *ReasonContext {
	rc.UserContext = userContext
	return rc
}

// WithCustomRule 添加自定义规则
func (rc *ReasonContext) WithCustomRule(rule string) *ReasonContext {
	rc.CustomRules = append(rc.CustomRules, rule)
	return rc
}

// WithCustomRules 设置自定义规则列表
func (rc *ReasonContext) WithCustomRules(rules []string) *ReasonContext {
	rc.CustomRules = rules
	return rc
}

// WithExample 添加参考示例
func (rc *ReasonContext) WithExample(example Example) *ReasonContext {
	rc.Examples = append(rc.Examples, example)
	return rc
}

// WithExamples 设置参考示例列表
func (rc *ReasonContext) WithExamples(examples []Example) *ReasonContext {
	rc.Examples = examples
	return rc
}

// WithTemperature 设置温度参数
func (rc *ReasonContext) WithTemperature(temperature float64) *ReasonContext {
	rc.Temperature = temperature
	return rc
}

// WithMaxTokens 设置最大 token 数
func (rc *ReasonContext) WithMaxTokens(maxTokens int) *ReasonContext {
	rc.MaxTokens = maxTokens
	return rc
}

// WithEnableReason 设置是否启用推理
func (rc *ReasonContext) WithEnableReason(enableReason bool) *ReasonContext {
	rc.EnableReason = enableReason
	return rc
}

// MockLLMProvider Mock LLM 提供者（用于测试）
type MockLLMProvider struct {
	Response string
}

// NewMockLLMProvider 创建 Mock LLM 提供者
func NewMockLLMProvider(response string) *MockLLMProvider {
	return &MockLLMProvider{
		Response: response,
	}
}

// GenerateText 生成文本
func (m *MockLLMProvider) GenerateText(prompt string) (string, error) {
	return m.Response, nil
}

// GenerateTextWithMessages 使用消息列表生成文本
func (m *MockLLMProvider) GenerateTextWithMessages(messages []Message) (string, error) {
	return m.Response, nil
}

// OpenAILLMProvider OpenAI LLM 提供者（示例实现）
type OpenAILLMProvider struct {
	APIKey      string
	Model       string
	BaseURL     string
	MaxTokens   int
	Temperature float64
}

// NewOpenAILLMProvider 创建 OpenAI LLM 提供者
func NewOpenAILLMProvider(apiKey, model string) *OpenAILLMProvider {
	return &OpenAILLMProvider{
		APIKey:      apiKey,
		Model:       model,
		BaseURL:     "https://api.openai.com/v1",
		MaxTokens:   1000,
		Temperature: 0.7,
	}
}

// GenerateText 生成文本
func (o *OpenAILLMProvider) GenerateText(prompt string) (string, error) {
	// TODO: 实现实际的 OpenAI API 调用
	// 这里需要使用 http.Client 调用 OpenAI API
	return "", fmt.Errorf("not implemented yet")
}

// GenerateTextWithMessages 使用消息列表生成文本
func (o *OpenAILLMProvider) GenerateTextWithMessages(messages []Message) (string, error) {
	// TODO: 实现实际的 OpenAI API 调用
	// 这里需要使用 http.Client 调用 OpenAI API
	return "", fmt.Errorf("not implemented yet")
}

// OnlineLLMProvider 在线 LLM 提供者
type OnlineLLMProvider struct {
	APIKey      string
	Model       string
	BaseURL     string
	MaxTokens   int
	Temperature float64
}

// NewOnlineLLMProvider 创建在线 LLM 提供者
func NewOnlineLLMProvider(apiKey, model, baseURL string) *OnlineLLMProvider {
	return &OnlineLLMProvider{
		APIKey:      apiKey,
		Model:       model,
		BaseURL:     baseURL,
		MaxTokens:   65536,
		Temperature: 1.0,
	}
}

// GenerateText 生成文本
func (o *OnlineLLMProvider) GenerateText(prompt string) (string, error) {
	messages := []Message{
		{
			Role:    "user",
			Content: prompt,
		},
	}
	return o.GenerateTextWithMessages(messages)
}

// GenerateTextWithMessages 使用消息列表生成文本
func (o *OnlineLLMProvider) GenerateTextWithMessages(messages []Message) (string, error) {
	// 构建请求体
	reqBody := struct {
		Model       string    `json:"model"`
		Messages    []Message `json:"messages"`
		Thinking    Thinking  `json:"thinking"`
		MaxTokens   int       `json:"max_tokens"`
		Temperature float64   `json:"temperature"`
	}{
		Model:       o.Model,
		Messages:    messages,
		Thinking:    Thinking{Type: "enabled"},
		MaxTokens:   o.MaxTokens,
		Temperature: o.Temperature,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建 HTTP 请求
	req, err := http.NewRequest("POST", o.BaseURL+"/chat/completions", bytes.NewBuffer(data))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.APIKey)

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var respBody struct {
		Choices []struct {
			Message Message `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(respBody.Choices) == 0 {
		return "", fmt.Errorf("no response from zhipu api")
	}

	return respBody.Choices[0].Message.Content, nil
}

// Thinking 智谱思考配置
type Thinking struct {
	Type string `json:"type"`
}

// OllamaLLMProvider Ollama LLM 提供者
type OllamaLLMProvider struct {
	client *util.Client
}

// NewOllamaLLMProvider 创建 Ollama LLM 提供者
func NewOllamaLLMProvider(host, model string) *OllamaLLMProvider {
	return &OllamaLLMProvider{
		client: util.NewClient(host, model, 60*time.Second),
	}
}

// GenerateText 生成文本
func (o *OllamaLLMProvider) GenerateText(prompt string) (string, error) {
	return o.client.GenerateText(prompt)
}

// GenerateTextWithMessages 使用消息列表生成文本
func (o *OllamaLLMProvider) GenerateTextWithMessages(messages []Message) (string, error) {
	// 将消息转换为单个prompt
	var promptBuilder strings.Builder
	for _, msg := range messages {
		promptBuilder.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
	}
	return o.client.GenerateText(promptBuilder.String())
}
