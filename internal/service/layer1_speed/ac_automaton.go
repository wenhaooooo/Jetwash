package layer1_speed

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
)

// Payload 敏感词的负载信息
type Payload struct {
	TenantID  uuid.UUID `json:"tenant_id"`
	WordText  string    `json:"word_text"`
	Category  string    `json:"category"`
	RiskLevel int       `json:"risk_level"`
}

// ACNode AC 自动机节点
type ACNode struct {
	children map[rune]*ACNode // 子节点
	fail     *ACNode          // 失败指针
	payloads []*Payload       // 该节点关联的敏感词负载
	isEnd    bool             // 是否为某个敏感词的结尾
}

// ACAutomaton AC 自动机
type ACAutomaton struct {
	root *ACNode
	mu   sync.RWMutex
}

// NewACAutomaton 创建新的 AC 自动机
func NewACAutomaton() *ACAutomaton {
	return &ACAutomaton{
		root: &ACNode{
			children: make(map[rune]*ACNode),
			fail:     nil,
			payloads: make([]*Payload, 0),
			isEnd:    false,
		},
	}
}

// Insert 插入敏感词
func (ac *ACAutomaton) Insert(word string, payload *Payload) error {
	if word == "" {
		return fmt.Errorf("word cannot be empty")
	}

	ac.mu.Lock()
	defer ac.mu.Unlock()

	node := ac.root
	for _, r := range word {
		if _, exists := node.children[r]; !exists {
			node.children[r] = &ACNode{
				children: make(map[rune]*ACNode),
				fail:     nil,
				payloads: make([]*Payload, 0),
				isEnd:    false,
			}
		}
		node = node.children[r]
	}

	node.isEnd = true
	node.payloads = append(node.payloads, payload)

	return nil
}

// BuildFail 构建失败指针
func (ac *ACAutomaton) BuildFail() {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	queue := make([]*ACNode, 0)

	// 第一层节点的失败指针指向根节点
	for _, child := range ac.root.children {
		child.fail = ac.root
		queue = append(queue, child)
	}

	// BFS 构建其他节点的失败指针
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for r, child := range current.children {
			queue = append(queue, child)

			// 找到父节点的失败指针
			failNode := current.fail
			for failNode != nil {
				if nextNode, exists := failNode.children[r]; exists {
					child.fail = nextNode
					break
				}
				failNode = failNode.fail
			}

			// 如果没有找到，失败指针指向根节点
			if child.fail == nil {
				child.fail = ac.root
			}

			// 合并失败指针的 payloads
			if child.fail.isEnd {
				child.payloads = append(child.payloads, child.fail.payloads...)
			}
		}
	}
}

// Match 匹配文本中的敏感词
func (ac *ACAutomaton) Match(text string) []*MatchResult {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	results := make([]*MatchResult, 0)
	node := ac.root

	for i, r := range text {
		// 沿着失败指针查找
		for node != ac.root && node.children[r] == nil {
			node = node.fail
		}

		if nextNode, exists := node.children[r]; exists {
			node = nextNode
		} else {
			node = ac.root
		}

		// 如果当前节点是敏感词结尾，记录匹配结果
		if node.isEnd {
			for _, payload := range node.payloads {
				// 使用字符长度而不是字节长度
				wordRunes := []rune(payload.WordText)
				startPos := i - len(wordRunes) + 1
				if startPos >= 0 {
					results = append(results, &MatchResult{
						Payload:   payload,
						Position:  startPos,
						Matched:   payload.WordText,
						MatchType: "exact",
					})
				}
			}
		}
	}

	return results
}

// MatchWithTenantID 仅匹配指定租户的敏感词
func (ac *ACAutomaton) MatchWithTenantID(text string, tenantID uuid.UUID) []*MatchResult {
	allResults := ac.Match(text)
	filtered := make([]*MatchResult, 0)

	for _, result := range allResults {
		if result.Payload.TenantID == tenantID {
			filtered = append(filtered, result)
		}
	}

	return filtered
}

// MatchResult 匹配结果
type MatchResult struct {
	Payload   *Payload `json:"payload"`
	Position  int      `json:"position"`
	Matched   string   `json:"matched"`
	MatchType string   `json:"match_type"` // exact, fuzzy
}

// BatchInsert 批量插入敏感词
func (ac *ACAutomaton) BatchInsert(words []string, payloads []*Payload) error {
	if len(words) != len(payloads) {
		return fmt.Errorf("words and payloads length mismatch")
	}

	for i, word := range words {
		if err := ac.Insert(word, payloads[i]); err != nil {
			return fmt.Errorf("failed to insert word at index %d: %w", i, err)
		}
	}

	return nil
}

// Clear 清空 AC 自动机
func (ac *ACAutomaton) Clear() {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	ac.root = &ACNode{
		children: make(map[rune]*ACNode),
		fail:     nil,
		payloads: make([]*Payload, 0),
		isEnd:    false,
	}
}
