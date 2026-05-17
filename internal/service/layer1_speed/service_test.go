package layer1_speed

import (
	"sync"
	"testing"

	"jetwash/internal/models"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLayer1Service(t *testing.T) {
	svc := NewLayer1Service()
	assert.NotNil(t, svc)
}

func TestCheckText_EmptyText(t *testing.T) {
	svc := NewLayer1Service()
	_, err := svc.CheckText(uuid.New(), "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "text cannot be empty")
}

func TestCheckText_NoWords(t *testing.T) {
	svc := NewLayer1Service()
	tenantID := uuid.New()
	result, err := svc.CheckText(tenantID, "hello world")
	require.NoError(t, err)
	assert.False(t, result.HasMatch)
	assert.Equal(t, 0, result.RiskLevel)
}

func TestCheckText_MatchSensitiveWord(t *testing.T) {
	svc := NewLayer1Service()
	tenantID := uuid.New()

	words := []models.SensitiveWord{
		{WordText: "badword", Category: "profanity", RiskLevel: 3},
		{WordText: "anotherbad", Category: "violence", RiskLevel: 4},
	}

	err := svc.Initialize(tenantID, words)
	require.NoError(t, err)

	result, err := svc.CheckText(tenantID, "this contains badword in it")
	require.NoError(t, err)
	assert.True(t, result.HasMatch)
	assert.Equal(t, 3, result.RiskLevel)
	assert.Len(t, result.MatchedWords, 1)
	assert.Equal(t, "badword", result.MatchedWords[0].Matched)
}

func TestCheckText_NoMatch(t *testing.T) {
	svc := NewLayer1Service()
	tenantID := uuid.New()

	words := []models.SensitiveWord{
		{WordText: "badword", Category: "profanity", RiskLevel: 3},
	}

	err := svc.Initialize(tenantID, words)
	require.NoError(t, err)

	result, err := svc.CheckText(tenantID, "this is a clean text")
	require.NoError(t, err)
	assert.False(t, result.HasMatch)
}

func TestCheckText_MultipleMatches(t *testing.T) {
	svc := NewLayer1Service()
	tenantID := uuid.New()

	words := []models.SensitiveWord{
		{WordText: "word1", Category: "cat1", RiskLevel: 2},
		{WordText: "word2", Category: "cat2", RiskLevel: 4},
	}

	err := svc.Initialize(tenantID, words)
	require.NoError(t, err)

	result, err := svc.CheckText(tenantID, "text with word1 and word2")
	require.NoError(t, err)
	assert.True(t, result.HasMatch)
	assert.Len(t, result.MatchedWords, 2)
	assert.Equal(t, 4, result.RiskLevel)
	assert.ElementsMatch(t, []string{"cat1", "cat2"}, result.Categories)
}

func TestCheckText_TenantIsolation(t *testing.T) {
	svc := NewLayer1Service()
	tenant1 := uuid.New()
	tenant2 := uuid.New()

	words1 := []models.SensitiveWord{
		{WordText: "alphaone", Category: "cat", RiskLevel: 1},
	}
	words2 := []models.SensitiveWord{
		{WordText: "alphatwo", Category: "cat", RiskLevel: 2},
	}

	require.NoError(t, svc.Initialize(tenant1, words1))
	require.NoError(t, svc.Initialize(tenant2, words2))

	result1, err := svc.CheckText(tenant1, "contains alphaone here")
	require.NoError(t, err)
	assert.True(t, result1.HasMatch)

	result1b, err := svc.CheckText(tenant1, "contains alphatwo here")
	require.NoError(t, err)
	assert.False(t, result1b.HasMatch)

	result2, err := svc.CheckText(tenant2, "contains alphatwo here")
	require.NoError(t, err)
	assert.True(t, result2.HasMatch)
}

func TestNormalizeText(t *testing.T) {
	svc := NewLayer1Service()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"lowercase", "Hello", "hello"},
		{"collapse and trim spaces", "  hello   world  ", "hello world"},
		{"fullwidth to halfwidth", "Ｈｅｌｌｏ", "hello"},
		{"remove duplicate chars 3+", "aaa", "a"},
		{"preserve double chars", "aa", "aa"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := svc.NormalizeText(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAddWord(t *testing.T) {
	svc := NewLayer1Service()
	tenantID := uuid.New()

	err := svc.Initialize(tenantID, []models.SensitiveWord{
		{WordText: "existing", Category: "cat", RiskLevel: 1},
	})
	require.NoError(t, err)

	payload := &Payload{
		TenantID:  tenantID,
		WordText:  "newword",
		Category:  "cat",
		RiskLevel: 2,
	}
	err = svc.AddWord("newword", payload)
	require.NoError(t, err)

	result, err := svc.CheckText(tenantID, "this has newword in it")
	require.NoError(t, err)
	assert.True(t, result.HasMatch)
}

func TestBuildAutomaton_EmptyWords(t *testing.T) {
	svc := NewLayer1Service()
	err := svc.BuildAutomaton(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "words and payloads cannot be empty")
}

func TestInitialize_EmptyWords(t *testing.T) {
	svc := NewLayer1Service()
	err := svc.Initialize(uuid.New(), nil)
	assert.NoError(t, err) // empty words is a no-op, not an error
}

func TestCheckText_MatchesAreCaseInsensitive(t *testing.T) {
	svc := NewLayer1Service()
	tenantID := uuid.New()

	words := []models.SensitiveWord{
		{WordText: "badword", Category: "cat", RiskLevel: 3},
	}
	require.NoError(t, svc.Initialize(tenantID, words))

	result, err := svc.CheckText(tenantID, "this contains BADWORD here")
	require.NoError(t, err)
	assert.True(t, result.HasMatch)
	assert.Equal(t, "badword", result.MatchedWords[0].Matched)
}

func TestAddWord_EmptyWord(t *testing.T) {
	svc := NewLayer1Service()
	tenantID := uuid.New()

	require.NoError(t, svc.Initialize(tenantID, []models.SensitiveWord{
		{WordText: "test", Category: "cat", RiskLevel: 1},
	}))

	payload := &Payload{TenantID: tenantID, WordText: "", Category: "cat", RiskLevel: 1}
	err := svc.AddWord("", payload)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "word cannot be empty")
}

func TestAddWord_NilPayload(t *testing.T) {
	svc := NewLayer1Service()
	err := svc.AddWord("word", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "payload cannot be nil")
}

func TestGetMatchedWords(t *testing.T) {
	svc := NewLayer1Service()
	tenantID := uuid.New()

	words := []models.SensitiveWord{
		{WordText: "target", Category: "cat", RiskLevel: 5},
	}
	require.NoError(t, svc.Initialize(tenantID, words))

	matches := svc.GetMatchedWords(tenantID, "find the target here")
	assert.Len(t, matches, 1)
	assert.Equal(t, "target", matches[0].Matched)
	assert.Equal(t, 5, matches[0].Payload.RiskLevel)
}

func TestGetMatchedWords_NoAutomaton(t *testing.T) {
	svc := NewLayer1Service()
	matches := svc.GetMatchedWords(uuid.New(), "some text")
	assert.Empty(t, matches)
}

func TestCheckText_ConcurrentAccess(t *testing.T) {
	svc := NewLayer1Service()
	tenantID := uuid.New()

	words := []models.SensitiveWord{
		{WordText: "badword", Category: "profanity", RiskLevel: 3},
	}
	require.NoError(t, svc.Initialize(tenantID, words))

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := svc.CheckText(tenantID, "this has badword in it")
			assert.NoError(t, err)
			assert.True(t, result.HasMatch)
		}()
	}
	wg.Wait()
}
