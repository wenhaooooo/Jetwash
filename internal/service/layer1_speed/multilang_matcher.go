package layer1_speed

import (
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"
)

// Language 语言类型
type Language string

const (
	LanguageZH   Language = "zh"   // 中文
	LanguageEN   Language = "en"   // 英文
	LanguageJA   Language = "ja"   // 日文
	LanguageKO   Language = "ko"   // 韩文
	LanguageAuto Language = "auto" // 自动检测
)

// MultiLangWord 多语言词
type MultiLangWord struct {
	WordText string
	Language Language
	Payloads []*Payload
}

// MultiLangMatcher 多语言匹配器
type MultiLangMatcher struct {
	wordsByLang map[Language]map[string]*MultiLangWord
	mu          sync.RWMutex
}

// NewMultiLangMatcher 创建多语言匹配器
func NewMultiLangMatcher() *MultiLangMatcher {
	return &MultiLangMatcher{
		wordsByLang: make(map[Language]map[string]*MultiLangWord),
	}
}

// AddWord 添加多语言词
func (mm *MultiLangMatcher) AddWord(word string, language Language, payloads []*Payload) error {
	if word == "" {
		return fmt.Errorf("word cannot be empty")
	}

	mm.mu.Lock()
	defer mm.mu.Unlock()

	// 初始化语言字典
	if _, exists := mm.wordsByLang[language]; !exists {
		mm.wordsByLang[language] = make(map[string]*MultiLangWord)
	}

	// 添加词
	mm.wordsByLang[language][word] = &MultiLangWord{
		WordText: word,
		Language: language,
		Payloads: payloads,
	}

	return nil
}

// Match 匹配文本中的多语言词
func (mm *MultiLangMatcher) Match(text string, language Language) []*MultiLangMatchResult {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	results := make([]*MultiLangMatchResult, 0)

	// 如果是自动检测，则检测语言
	if language == LanguageAuto {
		language = detectLanguage(text)
	}

	// 获取对应语言的词库
	words, exists := mm.wordsByLang[language]
	if !exists {
		return results
	}

	// 根据语言类型进行匹配
	switch language {
	case LanguageZH:
		results = mm.matchChinese(text, words)
	case LanguageEN:
		results = mm.matchEnglish(text, words)
	case LanguageJA:
		results = mm.matchJapanese(text, words)
	case LanguageKO:
		results = mm.matchKorean(text, words)
	default:
		// 默认使用中文匹配
		results = mm.matchChinese(text, words)
	}

	return results
}

// MatchWithTenantID 仅匹配指定租户的多语言词
func (mm *MultiLangMatcher) MatchWithTenantID(text string, tenantID uuid.UUID, language Language) []*MultiLangMatchResult {
	allResults := mm.Match(text, language)
	filtered := make([]*MultiLangMatchResult, 0)

	for _, result := range allResults {
		if result.Payload.TenantID == tenantID {
			filtered = append(filtered, result)
		}
	}

	return filtered
}

// matchChinese 匹配中文
func (mm *MultiLangMatcher) matchChinese(text string, words map[string]*MultiLangWord) []*MultiLangMatchResult {
	results := make([]*MultiLangMatchResult, 0)

	for word, multiLangWord := range words {
		position := strings.Index(text, word)
		if position != -1 {
			for _, payload := range multiLangWord.Payloads {
				results = append(results, &MultiLangMatchResult{
					Payload:   payload,
					Position:  position,
					Matched:   word,
					Language:  multiLangWord.Language,
					MatchType: "multilang",
				})
			}
		}
	}

	return results
}

// matchEnglish 匹配英文
func (mm *MultiLangMatcher) matchEnglish(text string, words map[string]*MultiLangWord) []*MultiLangMatchResult {
	results := make([]*MultiLangMatchResult, 0)

	// 转为小写进行匹配
	lowerText := strings.ToLower(text)

	for word, multiLangWord := range words {
		lowerWord := strings.ToLower(word)
		position := strings.Index(lowerText, lowerWord)
		if position != -1 {
			for _, payload := range multiLangWord.Payloads {
				results = append(results, &MultiLangMatchResult{
					Payload:   payload,
					Position:  position,
					Matched:   word,
					Language:  multiLangWord.Language,
					MatchType: "multilang",
				})
			}
		}
	}

	return results
}

// matchJapanese 匹配日文
func (mm *MultiLangMatcher) matchJapanese(text string, words map[string]*MultiLangWord) []*MultiLangMatchResult {
	results := make([]*MultiLangMatchResult, 0)

	for word, multiLangWord := range words {
		position := strings.Index(text, word)
		if position != -1 {
			for _, payload := range multiLangWord.Payloads {
				results = append(results, &MultiLangMatchResult{
					Payload:   payload,
					Position:  position,
					Matched:   word,
					Language:  multiLangWord.Language,
					MatchType: "multilang",
				})
			}
		}
	}

	return results
}

// matchKorean 匹配韩文
func (mm *MultiLangMatcher) matchKorean(text string, words map[string]*MultiLangWord) []*MultiLangMatchResult {
	results := make([]*MultiLangMatchResult, 0)

	for word, multiLangWord := range words {
		position := strings.Index(text, word)
		if position != -1 {
			for _, payload := range multiLangWord.Payloads {
				results = append(results, &MultiLangMatchResult{
					Payload:   payload,
					Position:  position,
					Matched:   word,
					Language:  multiLangWord.Language,
					MatchType: "multilang",
				})
			}
		}
	}

	return results
}

// RemoveWord 移除多语言词
func (mm *MultiLangMatcher) RemoveWord(word string, language Language) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	if words, exists := mm.wordsByLang[language]; exists {
		delete(words, word)
	}
}

// Clear 清空所有词
func (mm *MultiLangMatcher) Clear() {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	mm.wordsByLang = make(map[Language]map[string]*MultiLangWord)
}

// MultiLangMatchResult 多语言匹配结果
type MultiLangMatchResult struct {
	Payload   *Payload `json:"payload"`
	Position  int      `json:"position"`
	Matched   string   `json:"matched"`
	Language  Language `json:"language"`
	MatchType string   `json:"match_type"`
}

// detectLanguage 检测文本语言
func detectLanguage(text string) Language {
	if text == "" {
		return LanguageZH
	}

	// 简单的语言检测
	zhCount := 0
	enCount := 0
	jaCount := 0
	koCount := 0

	for _, r := range text {
		if isChinese(r) {
			zhCount++
		} else if isEnglish(r) {
			enCount++
		} else if isJapanese(r) {
			jaCount++
		} else if isKorean(r) {
			koCount++
		}
	}

	// 返回字符数最多的语言
	maxCount := zhCount
	language := LanguageZH

	if enCount > maxCount {
		maxCount = enCount
		language = LanguageEN
	}
	if jaCount > maxCount {
		maxCount = jaCount
		language = LanguageJA
	}
	if koCount > maxCount {
		language = LanguageKO
	}

	return language
}

// isChinese 判断是否为中文字符
func isChinese(r rune) bool {
	return (r >= 0x4E00 && r <= 0x9FFF) || // 基本汉字
		(r >= 0x3400 && r <= 0x4DBF) || // 扩展A
		(r >= 0x20000 && r <= 0x2A6DF) || // 扩展B
		(r >= 0x2A700 && r <= 0x2B73F) || // 扩展C
		(r >= 0x2B740 && r <= 0x2B81F) || // 扩展D
		(r >= 0x2B820 && r <= 0x2CEAF) || // 扩展E
		(r >= 0xF900 && r <= 0xFAFF) || // 兼容汉字
		(r >= 0x2F800 && r <= 0x2FA1F) // 兼容补充
}

// isEnglish 判断是否为英文字符
func isEnglish(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// isJapanese 判断是否为日文字符
func isJapanese(r rune) bool {
	return (r >= 0x3040 && r <= 0x309F) || // 平假名
		(r >= 0x30A0 && r <= 0x30FF) || // 片假名
		(r >= 0x4E00 && r <= 0x9FFF) // 汉字（日文也使用汉字）
}

// isKorean 判断是否为韩文字符
func isKorean(r rune) bool {
	return (r >= 0xAC00 && r <= 0xD7AF) || // 韩文音节
		(r >= 0x1100 && r <= 0x11FF) || // 韩文字母
		(r >= 0x3130 && r <= 0x318F) // 韩文兼容字母
}
