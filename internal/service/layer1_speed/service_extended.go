package layer1_speed

import (
	"github.com/google/uuid"
)

// ExtendedLayer1Service 扩展的第一层服务接口
type ExtendedLayer1Service interface {
	Layer1Service

	// 正则表达式匹配
	AddRegexPattern(pattern string, payloads []*Payload) error
	MatchRegex(text string) []*RegexMatchResult
	MatchRegexWithTenantID(text string, tenantID uuid.UUID) []*RegexMatchResult

	// 模糊匹配
	AddFuzzyWord(word string, payloads []*Payload) error
	MatchFuzzy(text string, threshold float64) []*FuzzyMatchResult
	MatchFuzzyWithTenantID(text string, tenantID uuid.UUID, threshold float64) []*FuzzyMatchResult

	// 多语言匹配
	AddMultiLangWord(word string, language Language, payloads []*Payload) error
	MatchMultiLang(text string, language Language) []*MultiLangMatchResult
	MatchMultiLangWithTenantID(text string, tenantID uuid.UUID, language Language) []*MultiLangMatchResult

	// 综合匹配（所有匹配方式）
	MatchAll(text string, tenantID uuid.UUID, fuzzyThreshold float64, language Language) *Layer1ExtendedResult
}

// Layer1ExtendedResult 扩展的第一层结果
type Layer1ExtendedResult struct {
	Layer1Result

	// 正则匹配结果
	RegexMatches []*RegexMatchResult `json:"regex_matches"`

	// 模糊匹配结果
	FuzzyMatches []*FuzzyMatchResult `json:"fuzzy_matches"`

	// 多语言匹配结果
	MultiLangMatches []*MultiLangMatchResult `json:"multilang_matches"`
}

// extendedLayer1Service 扩展的第一层服务实现
type extendedLayer1Service struct {
	layer1Service
	regexMatcher     *RegexMatcher
	fuzzyMatcher     *FuzzyMatcher
	multilangMatcher *MultiLangMatcher
}

// NewExtendedLayer1Service 创建扩展的第一层服务实例
func NewExtendedLayer1Service() ExtendedLayer1Service {
	return &extendedLayer1Service{
		layer1Service: layer1Service{
			automata:       make(map[string]*ACAutomaton),
			emojiBlacklist: make(map[string]map[rune]*EmojiViolation),
			normalizer:     NewTextNormalizer(),
		},
		regexMatcher:     NewRegexMatcher(),
		fuzzyMatcher:     NewFuzzyMatcher(),
		multilangMatcher: NewMultiLangMatcher(),
	}
}

// AddRegexPattern 添加正则表达式模式
func (s *extendedLayer1Service) AddRegexPattern(pattern string, payloads []*Payload) error {
	return s.regexMatcher.AddPattern(pattern, payloads)
}

// MatchRegex 正则表达式匹配
func (s *extendedLayer1Service) MatchRegex(text string) []*RegexMatchResult {
	return s.regexMatcher.Match(text)
}

// MatchRegexWithTenantID 仅匹配指定租户的正则表达式
func (s *extendedLayer1Service) MatchRegexWithTenantID(text string, tenantID uuid.UUID) []*RegexMatchResult {
	return s.regexMatcher.MatchWithTenantID(text, tenantID)
}

// AddFuzzyWord 添加模糊词
func (s *extendedLayer1Service) AddFuzzyWord(word string, payloads []*Payload) error {
	return s.fuzzyMatcher.AddWord(word, payloads)
}

// MatchFuzzy 模糊匹配
func (s *extendedLayer1Service) MatchFuzzy(text string, threshold float64) []*FuzzyMatchResult {
	return s.fuzzyMatcher.Match(text, threshold)
}

// MatchFuzzyWithTenantID 仅匹配指定租户的模糊词
func (s *extendedLayer1Service) MatchFuzzyWithTenantID(text string, tenantID uuid.UUID, threshold float64) []*FuzzyMatchResult {
	return s.fuzzyMatcher.MatchWithTenantID(text, tenantID, threshold)
}

// AddMultiLangWord 添加多语言词
func (s *extendedLayer1Service) AddMultiLangWord(word string, language Language, payloads []*Payload) error {
	return s.multilangMatcher.AddWord(word, language, payloads)
}

// MatchMultiLang 多语言匹配
func (s *extendedLayer1Service) MatchMultiLang(text string, language Language) []*MultiLangMatchResult {
	return s.multilangMatcher.Match(text, language)
}

// MatchMultiLangWithTenantID 仅匹配指定租户的多语言词
func (s *extendedLayer1Service) MatchMultiLangWithTenantID(text string, tenantID uuid.UUID, language Language) []*MultiLangMatchResult {
	return s.multilangMatcher.MatchWithTenantID(text, tenantID, language)
}

// MatchAll 综合匹配（所有匹配方式）
func (s *extendedLayer1Service) MatchAll(text string, tenantID uuid.UUID, fuzzyThreshold float64, language Language) *Layer1ExtendedResult {
	// 规范化文本
	normalized := s.NormalizeText(text)

	// 获取租户的自动机
	automaton, err := s.getAutomaton(tenantID)
	if err != nil {
		automaton = NewACAutomaton()
	}

	// AC 自动机匹配
	acMatches := automaton.MatchWithTenantID(normalized, tenantID)

	// 正则表达式匹配
	regexMatches := s.regexMatcher.MatchWithTenantID(normalized, tenantID)

	// 模糊匹配
	fuzzyMatches := s.fuzzyMatcher.MatchWithTenantID(normalized, tenantID, fuzzyThreshold)

	// 多语言匹配
	multilangMatches := s.multilangMatcher.MatchWithTenantID(normalized, tenantID, language)

	// 转换 AC 匹配结果为通用格式
	layer1Result := s.buildExtendedResult(acMatches, regexMatches, fuzzyMatches, multilangMatches, normalized)

	return layer1Result
}

// buildExtendedResult 构建扩展结果
func (s *extendedLayer1Service) buildExtendedResult(
	acMatches []*MatchResult,
	regexMatches []*RegexMatchResult,
	fuzzyMatches []*FuzzyMatchResult,
	multilangMatches []*MultiLangMatchResult,
	normalized string,
) *Layer1ExtendedResult {
	result := &Layer1ExtendedResult{
		Layer1Result: Layer1Result{
			HasMatch:     false,
			MatchedWords: make([]*MatchResult, 0),
			Normalized:   normalized,
			RiskLevel:    0,
			Categories:   make([]string, 0),
		},
		RegexMatches:     regexMatches,
		FuzzyMatches:     fuzzyMatches,
		MultiLangMatches: multilangMatches,
	}

	// 检查是否有匹配
	if len(acMatches) > 0 || len(regexMatches) > 0 || len(fuzzyMatches) > 0 || len(multilangMatches) > 0 {
		result.HasMatch = true
	}

	// 处理 AC 匹配结果
	result.MatchedWords = acMatches

	// 计算最高风险等级
	categories := make(map[string]bool)

	// AC 匹配的风险等级
	for _, match := range acMatches {
		if match.Payload.RiskLevel > result.RiskLevel {
			result.RiskLevel = match.Payload.RiskLevel
		}
		categories[match.Payload.Category] = true
	}

	// 正则匹配的风险等级
	for _, match := range regexMatches {
		if match.Payload.RiskLevel > result.RiskLevel {
			result.RiskLevel = match.Payload.RiskLevel
		}
		categories[match.Payload.Category] = true
	}

	// 模糊匹配的风险等级
	for _, match := range fuzzyMatches {
		if match.Payload.RiskLevel > result.RiskLevel {
			result.RiskLevel = match.Payload.RiskLevel
		}
		categories[match.Payload.Category] = true
	}

	// 多语言匹配的风险等级
	for _, match := range multilangMatches {
		if match.Payload.RiskLevel > result.RiskLevel {
			result.RiskLevel = match.Payload.RiskLevel
		}
		categories[match.Payload.Category] = true
	}

	// 提取分类列表
	for category := range categories {
		result.Categories = append(result.Categories, category)
	}

	return result
}

// BuildAutomatonWithExtensions 构建或重建 AC 自动机（保持兼容性）
func (s *extendedLayer1Service) BuildAutomaton(words []string, payloads []*Payload) error {
	return s.layer1Service.BuildAutomaton(words, payloads)
}

// AddWord 添加敏感词到 AC 自动机（保持兼容性）
func (s *extendedLayer1Service) AddWord(word string, payload *Payload) error {
	return s.layer1Service.AddWord(word, payload)
}

// GetMatchedWords 获取匹配到的敏感词（保持兼容性）
func (s *extendedLayer1Service) GetMatchedWords(tenantID uuid.UUID, text string) []*MatchResult {
	return s.layer1Service.GetMatchedWords(tenantID, text)
}

// NormalizeText 规范化文本（保持兼容性）
func (s *extendedLayer1Service) NormalizeText(text string) string {
	return s.layer1Service.NormalizeText(text)
}

// CheckText 检查文本中的敏感词（保持兼容性）
func (s *extendedLayer1Service) CheckText(tenantID uuid.UUID, text string) (*Layer1Result, error) {
	return s.layer1Service.CheckText(tenantID, text)
}

// MatchAllWithDefault 使用默认参数进行综合匹配
func (s *extendedLayer1Service) MatchAllWithDefault(text string, tenantID uuid.UUID) *Layer1ExtendedResult {
	return s.MatchAll(text, tenantID, 0.7, LanguageAuto)
}

// GetTotalMatches 获取总匹配数
func (s *extendedLayer1Service) GetTotalMatches(result *Layer1ExtendedResult) int {
	return len(result.MatchedWords) + len(result.RegexMatches) + len(result.FuzzyMatches) + len(result.MultiLangMatches)
}

// GetAllCategories 获取所有涉及的分类
func (s *extendedLayer1Service) GetAllCategories(result *Layer1ExtendedResult) []string {
	categoryMap := make(map[string]bool)

	// AC 匹配的分类
	for _, match := range result.MatchedWords {
		categoryMap[match.Payload.Category] = true
	}

	// 正则匹配的分类
	for _, match := range result.RegexMatches {
		categoryMap[match.Payload.Category] = true
	}

	// 模糊匹配的分类
	for _, match := range result.FuzzyMatches {
		categoryMap[match.Payload.Category] = true
	}

	// 多语言匹配的分类
	for _, match := range result.MultiLangMatches {
		categoryMap[match.Payload.Category] = true
	}

	// 转换为列表
	categories := make([]string, 0, len(categoryMap))
	for category := range categoryMap {
		categories = append(categories, category)
	}

	return categories
}

// ClearAll 清空所有匹配器
func (s *extendedLayer1Service) ClearAll() {
	s.mu.Lock()
	s.automata = make(map[string]*ACAutomaton)
	s.mu.Unlock()
	s.regexMatcher.Clear()
	s.fuzzyMatcher.Clear()
	s.multilangMatcher.Clear()
}
