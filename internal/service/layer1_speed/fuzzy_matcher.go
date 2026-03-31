package layer1_speed

import (
	"fmt"
	"strings"
	"sync"
	"unicode"

	"github.com/google/uuid"
)

// FuzzyMatcher 模糊匹配器
type FuzzyMatcher struct {
	words map[string]*FuzzyWord
	mu    sync.RWMutex
}

// FuzzyWord 模糊词
type FuzzyWord struct {
	WordText string
	Payloads []*Payload
}

// NewFuzzyMatcher 创建模糊匹配器
func NewFuzzyMatcher() *FuzzyMatcher {
	return &FuzzyMatcher{
		words: make(map[string]*FuzzyWord),
	}
}

// AddWord 添加模糊词
func (fm *FuzzyMatcher) AddWord(word string, payloads []*Payload) error {
	if word == "" {
		return fmt.Errorf("word cannot be empty")
	}

	fm.mu.Lock()
	defer fm.mu.Unlock()

	normalized := normalizeForFuzzy(word)
	fm.words[normalized] = &FuzzyWord{
		WordText: word,
		Payloads: payloads,
	}

	return nil
}

// Match 模糊匹配文本中的词
func (fm *FuzzyMatcher) Match(text string, threshold float64) []*FuzzyMatchResult {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	results := make([]*FuzzyMatchResult, 0)

	// 分词
	tokens := tokenizeForFuzzy(text)

	for _, token := range tokens {
		normalizedToken := normalizeForFuzzy(token)

		for key, fuzzyWord := range fm.words {
			// 计算相似度
			similarity := calculateSimilarity(normalizedToken, key)

			if similarity >= threshold {
				for _, payload := range fuzzyWord.Payloads {
					position := findPosition(text, token)
					results = append(results, &FuzzyMatchResult{
						Payload:    payload,
						Position:   position,
						Matched:    token,
						Target:     fuzzyWord.WordText,
						Similarity: similarity,
						MatchType:  "fuzzy",
					})
				}
			}
		}
	}

	return results
}

// MatchWithTenantID 仅匹配指定租户的模糊词
func (fm *FuzzyMatcher) MatchWithTenantID(text string, tenantID uuid.UUID, threshold float64) []*FuzzyMatchResult {
	allResults := fm.Match(text, threshold)
	filtered := make([]*FuzzyMatchResult, 0)

	for _, result := range allResults {
		if result.Payload.TenantID == tenantID {
			filtered = append(filtered, result)
		}
	}

	return filtered
}

// RemoveWord 移除模糊词
func (fm *FuzzyMatcher) RemoveWord(word string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	normalized := normalizeForFuzzy(word)
	delete(fm.words, normalized)
}

// Clear 清空所有词
func (fm *FuzzyMatcher) Clear() {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	fm.words = make(map[string]*FuzzyWord)
}

// FuzzyMatchResult 模糊匹配结果
type FuzzyMatchResult struct {
	Payload    *Payload `json:"payload"`
	Position   int      `json:"position"`
	Matched    string   `json:"matched"`
	Target     string   `json:"target"`
	Similarity float64  `json:"similarity"`
	MatchType  string   `json:"match_type"`
}

// normalizeForFuzzy 规范化文本用于模糊匹配
func normalizeForFuzzy(text string) string {
	// 转小写
	text = strings.ToLower(text)

	// 移除空格和标点
	var result []rune
	for _, r := range text {
		if !unicode.IsSpace(r) && !unicode.IsPunct(r) {
			result = append(result, r)
		}
	}

	return string(result)
}

// tokenizeForFuzzy 分词用于模糊匹配
func tokenizeForFuzzy(text string) []string {
	// 简单分词：按空格和标点符号分割
	var tokens []string
	var currentToken []rune

	for _, r := range text {
		if unicode.IsSpace(r) || unicode.IsPunct(r) {
			if len(currentToken) > 0 {
				tokens = append(tokens, string(currentToken))
				currentToken = currentToken[:0]
			}
		} else {
			currentToken = append(currentToken, r)
		}
	}

	if len(currentToken) > 0 {
		tokens = append(tokens, string(currentToken))
	}

	return tokens
}

// calculateSimilarity 计算两个字符串的相似度（使用编辑距离）
func calculateSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}

	if len(s1) == 0 || len(s2) == 0 {
		return 0.0
	}

	// 计算编辑距离
	distance := levenshteinDistance(s1, s2)

	// 计算相似度
	maxLen := max(len(s1), len(s2))
	similarity := 1.0 - float64(distance)/float64(maxLen)

	return similarity
}

// levenshteinDistance 计算编辑距离
func levenshteinDistance(s1, s2 string) int {
	m := len(s1)
	n := len(s2)

	// 创建 DP 表
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}

	// 初始化
	for i := 0; i <= m; i++ {
		dp[i][0] = i
	}
	for j := 0; j <= n; j++ {
		dp[0][j] = j
	}

	// 填充 DP 表
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if s1[i-1] == s2[j-1] {
				dp[i][j] = dp[i-1][j-1]
			} else {
				dp[i][j] = min(
					dp[i-1][j]+1,   // 删除
					dp[i][j-1]+1,   // 插入
					dp[i-1][j-1]+1, // 替换
				)
			}
		}
	}

	return dp[m][n]
}

// min 返回最小值
func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// findPosition 查找词在文本中的位置
func findPosition(text, word string) int {
	return strings.Index(text, word)
}
