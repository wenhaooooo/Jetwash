package layer1_speed

import (
	"fmt"
	"sync"

	"jetwash/internal/models"

	"github.com/google/uuid"
)

// Layer1Service 第一层服务接口 - 快速匹配层
type Layer1Service interface {
	// CheckText 检查文本中的敏感词（快速匹配）
	CheckText(tenantID uuid.UUID, text string) (*Layer1Result, error)

	// Initialize 初始化AC自动机，加载指定租户的敏感词
	Initialize(tenantID uuid.UUID, words []models.SensitiveWord) error

	// BuildAutomaton 构建或重建 AC 自动机
	BuildAutomaton(words []string, payloads []*Payload) error

	// AddWord 添加敏感词到 AC 自动机
	AddWord(word string, payload *Payload) error

	// GetMatchedWords 获取匹配到的敏感词
	GetMatchedWords(tenantID uuid.UUID, text string) []*MatchResult

	// NormalizeText 规范化文本
	NormalizeText(text string) string

	// RegisterSegmenterWords 注册分词词典（用于歧义检测）
	RegisterSegmenterWords(words []string)

	// BuildEmojiBlacklist 从敏感词列表构建 emoji 黑名单
	BuildEmojiBlacklist(tenantID uuid.UUID, words []models.SensitiveWord)

	// CheckEmojiViolations 检查文本中的违规 emoji
	CheckEmojiViolations(tenantID uuid.UUID, text string) []EmojiViolation
}

// Layer1Result 第一层结果
type Layer1Result struct {
	HasMatch         bool              `json:"has_match"`         // 是否匹配到敏感词
	MatchedWords     []*MatchResult    `json:"matched_words"`     // 匹配到的敏感词列表
	Normalized       string            `json:"normalized"`        // 规范化后的文本
	RiskLevel        int               `json:"risk_level"`        // 最高风险等级
	Categories       []string          `json:"categories"`        // 涉及的分类
	AmbiguousMatches []*AmbiguousMatch `json:"ambiguous_matches"` // 歧义匹配（疑似误报）
	HasAmbiguity     bool              `json:"has_ambiguity"`     // 是否存在歧义

	// Emoji 违规检测
	EmojiViolations  []EmojiViolation `json:"emoji_violations,omitempty"` // 命中的违规 emoji 列表
	HasEmojiViolation bool            `json:"has_emoji_violation"`        // 是否存在 emoji 违规
}

// layer1Service 第一层服务实现
type layer1Service struct {
	automata       map[string]*ACAutomaton           // key: tenantID string
	emojiBlacklist map[string]map[rune]*EmojiViolation // key: tenantID string, value: emoji -> violation
	normalizer     *TextNormalizer
	segmenter      *WordSegmenter
	mu             sync.RWMutex
}

// NewLayer1Service 创建第一层服务实例
func NewLayer1Service() Layer1Service {
	return &layer1Service{
		automata:       make(map[string]*ACAutomaton),
		emojiBlacklist: make(map[string]map[rune]*EmojiViolation),
		normalizer:     NewTextNormalizer(),
		segmenter:      NewWordSegmenter(),
	}
}

// RegisterSegmenterWords 注册分词词典（用于歧义检测）
func (s *layer1Service) RegisterSegmenterWords(words []string) {
	s.segmenter.RegisterWords(words)
}

// CheckText 检查文本中的敏感词
func (s *layer1Service) CheckText(tenantID uuid.UUID, text string) (*Layer1Result, error) {
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	// 在规范化之前先检查 emoji 违规（规范化会删除 emoji）
	emojiViolations := s.CheckEmojiViolations(tenantID, text)

	// 获取租户的自动机
	automaton, err := s.getAutomaton(tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get automaton: %w", err)
	}

	// 规范化文本（此时 emoji 已被检查过，可以安全删除）
	normalized := s.NormalizeText(text)

	// 使用 AC 自动机匹配敏感词
	matchedWords := automaton.MatchWithTenantID(normalized, tenantID)

	// 构建结果（合并 emoji 违规）
	result := s.buildResult(matchedWords, normalized, emojiViolations)

	return result, nil
}

// BuildAutomaton 构建或重建 AC 自动机（兼容旧接口，使用第一个租户的ID）
func (s *layer1Service) BuildAutomaton(words []string, payloads []*Payload) error {
	if len(words) == 0 || len(payloads) == 0 {
		return fmt.Errorf("words and payloads cannot be empty")
	}

	// 使用第一个 payload 的租户ID
	tenantID := payloads[0].TenantID
	return s.BuildAutomatonForTenant(tenantID, words, payloads)
}

// AddWord 添加敏感词到 AC 自动机
func (s *layer1Service) AddWord(word string, payload *Payload) error {
	if word == "" {
		return fmt.Errorf("word cannot be empty")
	}

	if payload == nil {
		return fmt.Errorf("payload cannot be nil")
	}

	// 获取租户的自动机
	automaton, err := s.getAutomaton(payload.TenantID)
	if err != nil {
		return fmt.Errorf("failed to get automaton: %w", err)
	}

	// 规范化敏感词
	normalizedWord := s.NormalizeText(word)
	payload.WordText = normalizedWord

	if err := automaton.InsertAndBuildFail(normalizedWord, payload); err != nil {
		return fmt.Errorf("failed to insert word: %w", err)
	}

	return nil
}

// GetMatchedWords 获取匹配到的敏感词
func (s *layer1Service) GetMatchedWords(tenantID uuid.UUID, text string) []*MatchResult {
	automaton, err := s.getAutomaton(tenantID)
	if err != nil {
		return nil
	}
	normalized := s.NormalizeText(text)
	return automaton.MatchWithTenantID(normalized, tenantID)
}

// NormalizeText 规范化文本
func (s *layer1Service) NormalizeText(text string) string {
	return s.normalizer.NormalizeText(text)
}

// buildResult 构建结果
func (s *layer1Service) buildResult(matchedWords []*MatchResult, normalized string, emojiViolations []EmojiViolation) *Layer1Result {
	result := &Layer1Result{
		HasMatch:         len(matchedWords) > 0,
		MatchedWords:     matchedWords,
		Normalized:       normalized,
		RiskLevel:        0,
		Categories:        make([]string, 0),
		EmojiViolations:   emojiViolations,
		HasEmojiViolation: len(emojiViolations) > 0,
	}

	// 合并 emoji 违规的风险等级和分类
	categories := make(map[string]bool)
	for _, ev := range emojiViolations {
		if ev.RiskLevel > result.RiskLevel {
			result.RiskLevel = ev.RiskLevel
		}
		categories[ev.Category] = true
	}

	if !result.HasMatch && !result.HasEmojiViolation {
		return result
	}

	for _, match := range matchedWords {
		if match.Payload.RiskLevel > result.RiskLevel {
			result.RiskLevel = match.Payload.RiskLevel
		}
		categories[match.Payload.Category] = true
	}

	for category := range categories {
		result.Categories = append(result.Categories, category)
	}

	if s.segmenter != nil && len(matchedWords) > 0 {
		ambiguousMatches := s.segmenter.DetectAmbiguousMatches(normalized, matchedWords)
		result.AmbiguousMatches = ambiguousMatches
		for _, amb := range ambiguousMatches {
			if amb.IsAmbiguous {
				result.HasAmbiguity = true
				break
			}
		}
	}

	return result
}

// Initialize 初始化AC自动机，加载指定租户的敏感词
func (s *layer1Service) Initialize(tenantID uuid.UUID, words []models.SensitiveWord) error {
	if len(words) == 0 {
		return nil
	}

	// 准备数据
	wordsList := make([]string, len(words))
	payloads := make([]*Payload, len(words))

	for i, word := range words {
		wordsList[i] = word.WordText
		payloads[i] = &Payload{
			TenantID:  tenantID,
			WordText:  word.WordText,
			Category:  word.Category,
			RiskLevel: word.RiskLevel,
		}
	}

	// 构建AC自动机
	if err := s.BuildAutomatonForTenant(tenantID, wordsList, payloads); err != nil {
		return err
	}

	// 构建 emoji 黑名单
	s.BuildEmojiBlacklist(tenantID, words)

	return nil
}

// getAutomaton 获取租户的自动机
func (s *layer1Service) getAutomaton(tenantID uuid.UUID) (*ACAutomaton, error) {
	tenantIDStr := tenantID.String()

	s.mu.RLock()
	automaton, exists := s.automata[tenantIDStr]
	s.mu.RUnlock()

	if exists {
		return automaton, nil
	}

	// 如果自动机不存在，创建一个空的自动机
	s.mu.Lock()
	defer s.mu.Unlock()

	// 双重检查
	if automaton, exists = s.automata[tenantIDStr]; exists {
		return automaton, nil
	}

	automaton = NewACAutomaton()
	s.automata[tenantIDStr] = automaton

	return automaton, nil
}

// BuildAutomatonForTenant 为特定租户构建 AC 自动机
func (s *layer1Service) BuildAutomatonForTenant(tenantID uuid.UUID, words []string, payloads []*Payload) error {
	if len(words) != len(payloads) {
		return fmt.Errorf("words and payloads length mismatch")
	}

	tenantIDStr := tenantID.String()

	// 创建新的自动机
	automaton := NewACAutomaton()

	// 批量插入敏感词
	if err := automaton.BatchInsert(words, payloads); err != nil {
		return fmt.Errorf("failed to batch insert words: %w", err)
	}

	// 构建失败指针
	automaton.BuildFail()

	// 保存到映射
	s.mu.Lock()
	s.automata[tenantIDStr] = automaton
	s.mu.Unlock()

	return nil
}

// BuildEmojiBlacklist 从敏感词列表中提取 emoji 字符，构建 per-tenant 的 emoji 黑名单
// 敏感词 word_text 中如果包含 emoji 字符，该 emoji 会被视为违规内容
func (s *layer1Service) BuildEmojiBlacklist(tenantID uuid.UUID, words []models.SensitiveWord) {
	blacklist := make(map[rune]*EmojiViolation)
	for _, word := range words {
		emojis := s.normalizer.ExtractEmojis(word.WordText)
		for _, emoji := range emojis {
			if _, exists := blacklist[emoji]; !exists {
				blacklist[emoji] = &EmojiViolation{
					Emoji:     emoji,
					EmojiStr:  string(emoji),
					Category:  word.Category,
					RiskLevel: word.RiskLevel,
				}
			}
		}
	}

	tenantIDStr := tenantID.String()
	s.mu.Lock()
	s.emojiBlacklist[tenantIDStr] = blacklist
	s.mu.Unlock()
}

// CheckEmojiViolations 检查文本中是否包含违规 emoji
// 在文本规范化（RemoveEmojis）之前调用
func (s *layer1Service) CheckEmojiViolations(tenantID uuid.UUID, text string) []EmojiViolation {
	tenantIDStr := tenantID.String()

	s.mu.RLock()
	blacklist, exists := s.emojiBlacklist[tenantIDStr]
	s.mu.RUnlock()

	if !exists || len(blacklist) == 0 {
		return nil
	}

	emojis := s.normalizer.ExtractEmojis(text)
	if len(emojis) == 0 {
		return nil
	}

	// 去重：同一个 emoji 只报告一次
	seen := make(map[rune]bool)
	violations := make([]EmojiViolation, 0)
	for _, emoji := range emojis {
		if seen[emoji] {
			continue
		}
		seen[emoji] = true
		if violation, hit := blacklist[emoji]; hit {
			violations = append(violations, *violation)
		}
	}

	return violations
}
