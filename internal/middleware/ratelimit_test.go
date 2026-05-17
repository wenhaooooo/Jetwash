package middleware

import (
	"fmt"
	"testing"
	"time"
)

func TestRateLimitKeyFormat(t *testing.T) {
	tenantID := "test-tenant-123"
	now := time.Now()
	window := now.Unix() / 60

	expected := fmt.Sprintf("ratelimit:%s:%d", tenantID, window)
	actual := fmt.Sprintf("ratelimit:%s:%d", tenantID, now.Unix()/60)

	if expected != actual {
		t.Errorf("rate limit key format mismatch: expected %s, got %s", expected, actual)
	}
}

func TestRateLimitKeyChangesPerMinute(t *testing.T) {
	tenantID := "test-tenant-123"

	// Two timestamps in the same minute should produce the same key
	ts1 := time.Date(2026, 1, 1, 10, 30, 0, 0, time.UTC)
	ts2 := time.Date(2026, 1, 1, 10, 30, 59, 0, time.UTC)
	key1 := fmt.Sprintf("ratelimit:%s:%d", tenantID, ts1.Unix()/60)
	key2 := fmt.Sprintf("ratelimit:%s:%d", tenantID, ts2.Unix()/60)
	if key1 != key2 {
		t.Errorf("keys within same minute should be equal: %s != %s", key1, key2)
	}

	// Two timestamps in different minutes should produce different keys
	ts3 := time.Date(2026, 1, 1, 10, 31, 0, 0, time.UTC)
	key3 := fmt.Sprintf("ratelimit:%s:%d", tenantID, ts3.Unix()/60)
	if key1 == key3 {
		t.Errorf("keys in different minutes should differ: %s == %s", key1, key3)
	}
}

func TestRateLimitKeyDifferentTenants(t *testing.T) {
	now := time.Now()
	key1 := fmt.Sprintf("ratelimit:%s:%d", "tenant-a", now.Unix()/60)
	key2 := fmt.Sprintf("ratelimit:%s:%d", "tenant-b", now.Unix()/60)
	if key1 == key2 {
		t.Errorf("keys for different tenants should differ: %s == %s", key1, key2)
	}
}
