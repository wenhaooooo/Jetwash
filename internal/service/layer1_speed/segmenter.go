package layer1_speed

import (
	"strings"
	"unicode"
)

type AmbiguousMatch struct {
	MatchedWord  string `json:"matched_word"`
	StartPos     int    `json:"start_pos"`
	EndPos       int    `json:"end_pos"`
	LeftContext  string `json:"left_context"`
	RightContext string `json:"right_context"`
	IsAmbiguous  bool   `json:"is_ambiguous"`
	Reason       string `json:"reason"`
}

type SegmentationResult struct {
	Words     []string
	Positions []int
}

type WordSegmenter struct {
	commonWords map[string]bool
}

func NewWordSegmenter() *WordSegmenter {
	return &WordSegmenter{
		commonWords: make(map[string]bool),
	}
}

func (ws *WordSegmenter) RegisterWords(words []string) {
	for _, word := range words {
		if len(word) >= 2 {
			ws.commonWords[word] = true
		}
	}
}

func (ws *WordSegmenter) Segment(text string) *SegmentationResult {
	result := &SegmentationResult{
		Words:     make([]string, 0),
		Positions: make([]int, 0),
	}

	if text == "" {
		return result
	}

	runes := []rune(text)
	n := len(runes)
	i := 0

	for i < n {
		bestLen := 1
		bestWord := string(runes[i])

		if ws.isChineseChar(runes[i]) {
			maxLen := 5
			if i+maxLen > n {
				maxLen = n - i
			}

			for length := maxLen; length >= 2; length-- {
				candidate := string(runes[i : i+length])
				if _, exists := ws.commonWords[candidate]; exists {
					bestLen = length
					bestWord = candidate
					break
				}
			}

			if bestLen == 1 && i+1 < n {
				for length := 2; length <= maxLen && length <= 4; length++ {
					candidate := string(runes[i : i+length])
					if _, exists := ws.commonWords[candidate]; exists {
						bestLen = length
						bestWord = candidate
					}
				}
			}
		}

		result.Words = append(result.Words, bestWord)
		result.Positions = append(result.Positions, i)
		i += bestLen
	}

	return result
}

func (ws *WordSegmenter) isChineseChar(r rune) bool {
	return unicode.Is(unicode.Han, r)
}

func (ws *WordSegmenter) DetectAmbiguousMatches(text string, matchedWords []*MatchResult) []*AmbiguousMatch {
	if len(matchedWords) == 0 {
		return nil
	}

	segmentation := ws.Segment(text)
	ambiguous := make([]*AmbiguousMatch, 0)

	for _, match := range matchedWords {
		startPos := match.Position
		endPos := startPos + len([]rune(match.Matched)) - 1

		leftWord := ws.getWordAtPosition(segmentation, startPos)
		rightWord := ws.getWordAtPosition(segmentation, endPos)

		isAmbiguous := ws.isCrossingBoundary(match.Matched, leftWord, rightWord, startPos, endPos)

		reason := ""
		if isAmbiguous {
			if leftWord != "" && rightWord != "" {
				reason = "matched word crosses word boundary: [" + leftWord + "] + [" + rightWord + "]"
			} else if leftWord != "" {
				reason = "matched word extends left boundary: [" + leftWord + "]"
			} else if rightWord != "" {
				reason = "matched word extends right boundary: [" + rightWord + "]"
			} else {
				reason = "matched word is not aligned with expected word boundaries"
			}
		}

		ambMatch := &AmbiguousMatch{
			MatchedWord:  match.Matched,
			StartPos:     startPos,
			EndPos:       endPos,
			LeftContext:  leftWord,
			RightContext: rightWord,
			IsAmbiguous:  isAmbiguous,
			Reason:       reason,
		}

		ambiguous = append(ambiguous, ambMatch)
	}

	return ambiguous
}

func (ws *WordSegmenter) getWordAtPosition(seg *SegmentationResult, pos int) string {
	for i, p := range seg.Positions {
		if p == pos {
			return seg.Words[i]
		}
		if i+1 < len(seg.Positions) && p < pos && seg.Positions[i+1] > pos {
			return seg.Words[i]
		}
	}
	return ""
}

func (ws *WordSegmenter) isCrossingBoundary(matchedWord, leftWord, rightWord string, startPos, endPos int) bool {
	if matchedWord == "" {
		return false
	}

	matchedRunes := []rune(matchedWord)
	if len(matchedRunes) < 3 {
		return false
	}

	if leftWord != "" && !strings.Contains(leftWord, matchedWord) && !strings.HasSuffix(leftWord, string(matchedRunes[0])) {
		return true
	}

	if rightWord != "" && !strings.Contains(rightWord, matchedWord) && !strings.HasPrefix(rightWord, string(matchedRunes[len(matchedRunes)-1])) {
		return true
	}

	return false
}

func (ws *WordSegmenter) IsAmbiguousMatch(text string, match *MatchResult) bool {
	segmentation := ws.Segment(text)
	startPos := match.Position
	endPos := startPos + len([]rune(match.Matched)) - 1

	leftWord := ws.getWordAtPosition(segmentation, startPos)
	rightWord := ws.getWordAtPosition(segmentation, endPos)

	return ws.isCrossingBoundary(match.Matched, leftWord, rightWord, startPos, endPos)
}