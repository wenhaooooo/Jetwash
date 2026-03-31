package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// EmbeddingProvider embedding提供者接口
type EmbeddingProvider interface {
	// GetEmbedding 获取文本的向量表示
	GetEmbedding(text string) ([]float32, error)
}

// OllamaEmbeddingProvider Ollama embedding提供者
type OllamaEmbeddingProvider struct {
	client *Client
}

// NewOllamaEmbeddingProvider 创建Ollama embedding提供者
func NewOllamaEmbeddingProvider(host, model string, timeout time.Duration) *OllamaEmbeddingProvider {
	return &OllamaEmbeddingProvider{
		client: NewClient(host, model, timeout),
	}
}

// GetEmbedding 获取文本的向量表示
func (o *OllamaEmbeddingProvider) GetEmbedding(text string) ([]float32, error) {
	return o.client.GetEmbedding(text)
}

// OnlineEmbeddingProvider 在线embedding提供者
type OnlineEmbeddingProvider struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

// NewOnlineEmbeddingProvider 创建在线embedding提供者
func NewOnlineEmbeddingProvider(apiKey, model, baseURL string, timeout time.Duration) *OnlineEmbeddingProvider {
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	return &OnlineEmbeddingProvider{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// GetEmbedding 获取文本的向量表示
func (o *OnlineEmbeddingProvider) GetEmbedding(text string) ([]float32, error) {
	// 构建请求体
	reqBody := struct {
		Model string `json:"model"`
		Input string `json:"input"`
	}{
		Model: o.model,
		Input: text,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequest("POST", o.baseURL+"/embeddings", bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	// 发送请求
	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// 解析响应
	var respBody struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(respBody.Data) == 0 {
		return nil, fmt.Errorf("no embedding data in response")
	}

	return respBody.Data[0].Embedding, nil
}
