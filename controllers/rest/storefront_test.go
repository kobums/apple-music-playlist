package rest

import (
	"strings"
	"testing"
	"time"
)

func resetStorefrontCache(t *testing.T) {
	t.Helper()
	storefrontMu.Lock()
	storefrontCache = map[string]storefrontEntry{}
	storefrontMu.Unlock()
}

// The cache is keyed by a hash so a long-lived map never holds raw user tokens.
func TestStorefrontCacheKeyHidesTheToken(t *testing.T) {
	token := "AiaBcDeF-a-real-looking-music-user-token"
	key := storefrontCacheKey(token)

	if strings.Contains(key, token) {
		t.Error("cache key contains the raw token")
	}
	if len(key) != 64 {
		t.Errorf("key length = %d, want 64 hex chars", len(key))
	}
	if key != storefrontCacheKey(token) {
		t.Error("key is not stable for the same token")
	}
	if key == storefrontCacheKey(token+"x") {
		t.Error("different tokens produced the same key")
	}
}

func TestStorefrontCacheRoundTrip(t *testing.T) {
	resetStorefrontCache(t)
	key := storefrontCacheKey("token-a")

	if _, ok := cachedStorefront(key); ok {
		t.Fatal("empty cache returned a hit")
	}

	rememberStorefront(key, "kr")

	got, ok := cachedStorefront(key)
	if !ok || got != "kr" {
		t.Errorf("got (%q, %v), want (\"kr\", true)", got, ok)
	}

	// One user's storefront must not answer for another.
	if _, ok := cachedStorefront(storefrontCacheKey("token-b")); ok {
		t.Error("a different token hit the cache")
	}
}

func TestStorefrontCacheExpires(t *testing.T) {
	resetStorefrontCache(t)
	key := storefrontCacheKey("token-expired")

	storefrontMu.Lock()
	storefrontCache[key] = storefrontEntry{id: "us", expiresAt: time.Now().Add(-time.Minute)}
	storefrontMu.Unlock()

	if _, ok := cachedStorefront(key); ok {
		t.Error("an expired entry was served")
	}
}

// Writes prune expired entries so the map cannot grow without bound as tokens
// rotate.
func TestStorefrontCacheEvictsExpiredOnWrite(t *testing.T) {
	resetStorefrontCache(t)

	storefrontMu.Lock()
	storefrontCache["stale-1"] = storefrontEntry{id: "us", expiresAt: time.Now().Add(-time.Hour)}
	storefrontCache["stale-2"] = storefrontEntry{id: "jp", expiresAt: time.Now().Add(-time.Hour)}
	storefrontMu.Unlock()

	rememberStorefront(storefrontCacheKey("fresh"), "kr")

	storefrontMu.Lock()
	size := len(storefrontCache)
	storefrontMu.Unlock()

	if size != 1 {
		t.Errorf("cache holds %d entries after a write, want only the fresh one", size)
	}
}
