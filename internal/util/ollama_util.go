package util

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Client Ollama 客户端结构
type Client struct {
	host       string
	model      string
	httpClient *http.Client
}

// EmbeddingRequest 请求结构
type EmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// EmbeddingResponse 响应结构
type EmbeddingResponse struct {
	Embedding []float32 `json:"embedding"`
}

// ChatRequest 聊天请求结构
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

// Message 消息结构
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponse 聊天响应结构
type ChatResponse struct {
	Message Message `json:"message"`
	Done    bool    `json:"done"`
}

// NewClient 创建一个新的 Ollama 客户端工具
func NewClient(host, model string, timeout time.Duration) *Client {
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &Client{
		host:  strings.TrimSuffix(host, "/"),
		model: model,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// GetEmbedding 核心工具方法：根据文本获取向量
func (c *Client) GetEmbedding(text string) ([]float32, error) {
	if text == "" {
		return nil, fmt.Errorf("prompt cannot be empty")
	}

	url := fmt.Sprintf("%s/api/embeddings", c.host)
	reqBody := EmbeddingRequest{
		Model:  c.model,
		Prompt: text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request failed: %w", err)
	}

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama api error: status %d", resp.StatusCode)
	}

	var res EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("decode response failed: %w", err)
	}

	if len(res.Embedding) == 0 {
		return nil, fmt.Errorf("received empty embedding from ollama")
	}

	return res.Embedding, nil
}

// GenerateText 生成文本（用于第三层推理）
func (c *Client) GenerateText(prompt string) (string, error) {
	if prompt == "" {
		return "", fmt.Errorf("prompt cannot be empty")
	}

	url := fmt.Sprintf("%s/api/chat", c.host)
	reqBody := ChatRequest{
		Model: c.model,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request failed: %w", err)
	}

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama api error: status %d", resp.StatusCode)
	}

	var res ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", fmt.Errorf("decode response failed: %w", err)
	}

	if res.Message.Content == "" {
		return "", fmt.Errorf("received empty response from ollama")
	}

	return res.Message.Content, nil
}

// GenerateTextWithContext 使用 context 生成文本（支持超时取消）
func (c *Client) GenerateTextWithContext(ctx context.Context, prompt string) (string, error) {
	if prompt == "" {
		return "", fmt.Errorf("prompt cannot be empty")
	}

	url := fmt.Sprintf("%s/api/chat", c.host)
	reqBody := ChatRequest{
		Model: c.model,
		Messages: []Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("create request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama api error: status %d", resp.StatusCode)
	}

	var res ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", fmt.Errorf("decode response failed: %w", err)
	}

	if res.Message.Content == "" {
		return "", fmt.Errorf("received empty response from ollama")
	}

	return res.Message.Content, nil
}
