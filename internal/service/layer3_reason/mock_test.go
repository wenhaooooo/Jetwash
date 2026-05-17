package layer3_reason

import (
	"context"
	"fmt"
)

// MockLLMProvider Mock LLM provider (test only).
type MockLLMProvider struct {
	Response string
}

// NewMockLLMProvider creates a MockLLMProvider that returns the given response.
func NewMockLLMProvider(response string) *MockLLMProvider {
	return &MockLLMProvider{Response: response}
}

func (m *MockLLMProvider) GenerateText(_ context.Context, _ string) (string, error) {
	return m.Response, nil
}

func (m *MockLLMProvider) GenerateTextWithMessages(_ context.Context, _ []Message) (string, error) {
	return m.Response, nil
}

// MockLLMProviderError is an LLM provider that always returns an error.
type MockLLMProviderError struct{}

// NewMockLLMProviderError creates a provider whose methods always fail.
func NewMockLLMProviderError() *MockLLMProviderError {
	return &MockLLMProviderError{}
}

func (m *MockLLMProviderError) GenerateText(_ context.Context, _ string) (string, error) {
	return "", fmt.Errorf("simulated LLM failure")
}

func (m *MockLLMProviderError) GenerateTextWithMessages(_ context.Context, _ []Message) (string, error) {
	return "", fmt.Errorf("simulated LLM failure")
}
