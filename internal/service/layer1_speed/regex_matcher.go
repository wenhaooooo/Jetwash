package layer1_speed

import (
	"fmt"
	"regexp"
	"sync"

	"github.com/google/uuid"
)

// RegexPattern 正则表达式模式
type RegexPattern struct {
	Pattern  string         `json:"pattern"`
	Payloads []*Payload     `json:"payloads"`
	Compiled *regexp.Regexp `json:"-"`
}

// RegexMatcher 正则表达式匹配器
type RegexMatcher struct {
	patterns map[string]*RegexPattern
	mu       sync.RWMutex
}

// NewRegexMatcher 创建正则表达式匹配器
func NewRegexMatcher() *RegexMatcher {
	return &RegexMatcher{
		patterns: make(map[string]*RegexPattern),
	}
}

// AddPattern 添加正则表达式模式
func (rm *RegexMatcher) AddPattern(pattern string, payloads []*Payload) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	// 编译正则表达式
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("failed to compile regex pattern: %w", err)
	}

	rm.patterns[pattern] = &RegexPattern{
		Pattern:  pattern,
		Payloads: payloads,
		Compiled: compiled,
	}

	return nil
}

// Match 匹配文本中的正则表达式
func (rm *RegexMatcher) Match(text string) []*RegexMatchResult {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	results := make([]*RegexMatchResult, 0)

	for _, pattern := range rm.patterns {
		matches := pattern.Compiled.FindAllStringSubmatchIndex(text, -1)
		for _, match := range matches {
			start, end := match[0], match[1]
			matchedText := text[start:end]

			for _, payload := range pattern.Payloads {
				results = append(results, &RegexMatchResult{
					Pattern:   pattern.Pattern,
					Payload:   payload,
					Position:  start,
					Matched:   matchedText,
					MatchType: "regex",
				})
			}
		}
	}

	return results
}

// MatchWithTenantID 仅匹配指定租户的正则表达式
func (rm *RegexMatcher) MatchWithTenantID(text string, tenantID uuid.UUID) []*RegexMatchResult {
	allResults := rm.Match(text)
	filtered := make([]*RegexMatchResult, 0)

	for _, result := range allResults {
		if result.Payload.TenantID == tenantID {
			filtered = append(filtered, result)
		}
	}

	return filtered
}

// RemovePattern 移除正则表达式模式
func (rm *RegexMatcher) RemovePattern(pattern string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	delete(rm.patterns, pattern)
}

// Clear 清空所有模式
func (rm *RegexMatcher) Clear() {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.patterns = make(map[string]*RegexPattern)
}

// RegexMatchResult 正则表达式匹配结果
type RegexMatchResult struct {
	Pattern   string   `json:"pattern"`
	Payload   *Payload `json:"payload"`
	Position  int      `json:"position"`
	Matched   string   `json:"matched"`
	MatchType string   `json:"match_type"`
}
