package cache

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestWithTTL(t *testing.T) {
	ttl := 5 * time.Second
	opt := WithTTL(ttl)
	if opt == nil {
		t.Error("WithTTL returned nil")
	}

	so := &SetOptions{}
	opt(so)

	if so.TTL != ttl {
		t.Errorf("TTL not set correctly: got %v, want %v", so.TTL, ttl)
	}
	if !so.ExpiresAt.IsZero() {
		t.Error("ExpiresAt should be zero when using WithTTL")
	}
}

func TestWithExpiresAt(t *testing.T) {
	expiresAt := time.Now().Add(10 * time.Second)
	opt := WithExpiresAt(expiresAt)
	if opt == nil {
		t.Error("WithExpiresAt returned nil")
	}

	so := &SetOptions{}
	opt(so)

	if so.ExpiresAt != expiresAt {
		t.Errorf("ExpiresAt not set correctly: got %v, want %v", so.ExpiresAt, expiresAt)
	}
	if so.TTL != 0 {
		t.Error("TTL should be zero when using WithExpiresAt")
	}
}

func TestInMemoryCacheGet(t *testing.T) {
	cache := NewInMemoryCache[string, string](100, 0)
	ctx := context.Background()

	tests := []struct {
		name    string
		setup   func()
		key     string
		wantVal string
		wantOK  bool
	}{
		{
			name: "missing key",
			setup: func() {
			},
			key:     "missing",
			wantVal: "",
			wantOK:  false,
		},
		{
			name: "existing key",
			setup: func() {
				cache.Set(ctx, "exists", "value")
			},
			key:     "exists",
			wantVal: "value",
			wantOK:  true,
		},
		{
			name: "expired key",
			setup: func() {
				cache.Set(ctx, "expired", "value", WithExpiresAt(time.Now().Add(-1*time.Second)))
			},
			key:     "expired",
			wantVal: "",
			wantOK:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewInMemoryCache[string, string](100, 0)
			tt.setup = func() {
				if tt.name == "existing key" {
					c.Set(ctx, "exists", "value")
				} else if tt.name == "expired key" {
					c.Set(ctx, "expired", "value", WithExpiresAt(time.Now().Add(-1*time.Second)))
				}
			}
			tt.setup()

			val, ok := c.Get(ctx, tt.key)
			if ok != tt.wantOK {
				t.Errorf("Get() ok = %v, wantOK %v", ok, tt.wantOK)
			}
			if ok && val != tt.wantVal {
				t.Errorf("Get() val = %q, wantVal %q", val, tt.wantVal)
			}
		})
	}
}

func TestInMemoryCacheSet(t *testing.T) {
	cache := NewInMemoryCache[string, int](100, 0)
	ctx := context.Background()

	tests := []struct {
		name      string
		key       string
		value     int
		opts      []SetOption
		wantError bool
	}{
		{
			name:  "simple set",
			key:   "key1",
			value: 42,
		},
		{
			name:      "set with TTL",
			key:       "key2",
			value:     100,
			opts:      []SetOption{WithTTL(5 * time.Second)},
			wantError: false,
		},
		{
			name:      "set with ExpiresAt",
			key:       "key3",
			value:     200,
			opts:      []SetOption{WithExpiresAt(time.Now().Add(10 * time.Second))},
			wantError: false,
		},
		{
			name:  "overwrite existing",
			key:   "key1",
			value: 99,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache.Set(ctx, tt.key, tt.value, tt.opts...)

			val, ok := cache.Get(ctx, tt.key)
			if !ok {
				t.Error("Set() key not found after Set")
				return
			}
			if val != tt.value {
				t.Errorf("Set() value mismatch: got %d, want %d", val, tt.value)
			}
		})
	}
}

func TestInMemoryCacheDelete(t *testing.T) {
	cache := NewInMemoryCache[string, string](100, 0)
	ctx := context.Background()

	cache.Set(ctx, "key1", "value1")

	tests := []struct {
		name   string
		key    string
		wantOK bool
	}{
		{
			name:   "delete existing",
			key:    "key1",
			wantOK: false,
		},
		{
			name:   "delete missing (idempotent)",
			key:    "missing",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache.Delete(ctx, tt.key)

			_, ok := cache.Get(ctx, tt.key)
			if ok != tt.wantOK {
				t.Errorf("Get() after Delete() ok = %v, wantOK %v", ok, tt.wantOK)
			}
		})
	}
}

func TestInMemoryCacheStats(t *testing.T) {
	cache := NewInMemoryCache[string, string](100, 0)
	ctx := context.Background()

	cache.Set(ctx, "key1", "value1")
	cache.Set(ctx, "key2", "value2")

	// Get hits
	cache.Get(ctx, "key1")
	cache.Get(ctx, "key1")

	// Get misses
	cache.Get(ctx, "missing1")
	cache.Get(ctx, "missing2")

	stats := cache.Stats()

	if stats.Items != 2 {
		t.Errorf("Stats() Items = %d, want 2", stats.Items)
	}
	if stats.Capacity != 100 {
		t.Errorf("Stats() Capacity = %d, want 100", stats.Capacity)
	}
	if stats.Hits != 2 {
		t.Errorf("Stats() Hits = %d, want 2", stats.Hits)
	}
	if stats.Misses != 2 {
		t.Errorf("Stats() Misses = %d, want 2", stats.Misses)
	}
}

func TestInMemoryCacheCapacity(t *testing.T) {
	cache := NewInMemoryCache[string, int](3, 0)
	ctx := context.Background()

	cache.Set(ctx, "key1", 1)
	cache.Set(ctx, "key2", 2)
	cache.Set(ctx, "key3", 3)

	stats := cache.Stats()
	if stats.Items != 3 {
		t.Errorf("expected 3 items, got %d", stats.Items)
	}

	// Adding one more should evict the oldest (LRU)
	cache.Set(ctx, "key4", 4)

	stats = cache.Stats()
	if stats.Items != 3 {
		t.Errorf("expected 3 items after LRU, got %d", stats.Items)
	}
	if stats.Evictions != 1 {
		t.Errorf("expected 1 eviction, got %d", stats.Evictions)
	}

	// key1 should be evicted
	_, ok := cache.Get(ctx, "key1")
	if ok {
		t.Error("key1 should be evicted")
	}

	// key4 should exist
	_, ok = cache.Get(ctx, "key4")
	if !ok {
		t.Error("key4 should exist")
	}
}

func TestInMemoryCacheTTL(t *testing.T) {
	cache := NewInMemoryCache[string, string](100, 0)
	ctx := context.Background()

	// Set with short TTL
	cache.Set(ctx, "short_ttl", "value", WithTTL(100*time.Millisecond))

	val, ok := cache.Get(ctx, "short_ttl")
	if !ok || val != "value" {
		t.Error("value should be retrievable immediately after Set")
	}

	time.Sleep(150 * time.Millisecond)

	_, ok = cache.Get(ctx, "short_ttl")
	if ok {
		t.Error("value should be expired after TTL")
	}

	stats := cache.Stats()
	if stats.Evictions != 1 {
		t.Errorf("expected 1 eviction due to TTL, got %d", stats.Evictions)
	}
}

func TestInMemoryCacheDefaultTTL(t *testing.T) {
	cache := NewInMemoryCache[string, string](100, 100*time.Millisecond)
	ctx := context.Background()

	cache.Set(ctx, "key1", "value1")

	_, ok := cache.Get(ctx, "key1")
	if !ok {
		t.Error("value should be retrievable immediately")
	}

	time.Sleep(150 * time.Millisecond)

	_, ok = cache.Get(ctx, "key1")
	if ok {
		t.Error("value should expire after defaultTTL")
	}
}

func TestInMemoryCacheExpiresAtPriority(t *testing.T) {
	cache := NewInMemoryCache[string, string](100, 10*time.Second)
	ctx := context.Background()

	// ExpiresAt should take priority over TTL and defaultTTL
	// Pass ExpiresAt after TTL so it takes priority (last writer wins in option order)
	expiresAt := time.Now().Add(-1 * time.Second)
	cache.Set(ctx, "key1", "value1", WithTTL(10*time.Second), WithExpiresAt(expiresAt))

	_, ok := cache.Get(ctx, "key1")
	if ok {
		t.Error("value should expire at ExpiresAt time")
	}
}

func TestInMemoryCacheLRUEviction(t *testing.T) {
	cache := NewInMemoryCache[string, int](3, 0)
	ctx := context.Background()

	// Fill cache
	cache.Set(ctx, "a", 1)
	cache.Set(ctx, "b", 2)
	cache.Set(ctx, "c", 3)

	// Access 'a' to move it to front
	cache.Get(ctx, "a")

	// Add new item, 'b' should be evicted (least recently used)
	cache.Set(ctx, "d", 4)

	_, ok := cache.Get(ctx, "b")
	if ok {
		t.Error("'b' should be evicted as LRU")
	}

	_, ok = cache.Get(ctx, "a")
	if !ok {
		t.Error("'a' should still exist (was recently accessed)")
	}
}

func TestInMemoryCacheClose(t *testing.T) {
	cache := NewInMemoryCache[string, string](100, 0)

	err := cache.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

func TestInMemoryCacheMinCapacity(t *testing.T) {
	// Capacity <= 0 should be set to 1
	cache := NewInMemoryCache[string, string](0, 0)
	stats := cache.Stats()

	if stats.Capacity != 1 {
		t.Errorf("capacity should be 1 for <= 0 input, got %d", stats.Capacity)
	}
}

func TestWithCloner(t *testing.T) {
	cloneCount := 0
	cloner := func(s string) string {
		cloneCount++
		return "cloned_" + s
	}

	cache := NewInMemoryCache[string, string](100, 0, WithCloner[string, string](cloner))
	ctx := context.Background()

	cache.Set(ctx, "key1", "original")

	val, ok := cache.Get(ctx, "key1")
	if !ok {
		t.Error("key should exist")
	}
	if !strings.HasPrefix(val, "cloned_") {
		t.Errorf("cloner should be applied on Get: got %q", val)
	}
	if cloneCount < 1 {
		t.Error("cloner should have been called")
	}
}

func TestInMemoryCacheContextCancellation(t *testing.T) {
	cache := NewInMemoryCache[string, string](100, 0)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Operations should still work even with cancelled context
	// (InMemoryCache doesn't respect context cancellation for this implementation)
	cache.Set(ctx, "key1", "value1")
	val, ok := cache.Get(ctx, "key1")

	if !ok {
		t.Error("Get should work with cancelled context")
	}
	if val != "value1" {
		t.Errorf("value mismatch: got %q, want value1", val)
	}
}

func TestInMemoryCacheMultipleTypes(t *testing.T) {
	t.Run("string cache", func(t *testing.T) {
		cache := NewInMemoryCache[string, string](10, 0)
		ctx := context.Background()
		cache.Set(ctx, "key", "value")
		val, ok := cache.Get(ctx, "key")
		if !ok || val != "value" {
			t.Error("string cache failed")
		}
	})

	t.Run("int cache", func(t *testing.T) {
		cache := NewInMemoryCache[int, int](10, 0)
		ctx := context.Background()
		cache.Set(ctx, 1, 100)
		val, ok := cache.Get(ctx, 1)
		if !ok || val != 100 {
			t.Error("int cache failed")
		}
	})

	t.Run("struct cache", func(t *testing.T) {
		type User struct {
			ID   int
			Name string
		}
		cache := NewInMemoryCache[string, User](10, 0)
		ctx := context.Background()
		user := User{ID: 1, Name: "Alice"}
		cache.Set(ctx, "alice", user)
		val, ok := cache.Get(ctx, "alice")
		if !ok || val.Name != "Alice" {
			t.Error("struct cache failed")
		}
	})
}

func TestInMemoryCacheStatsAccuracy(t *testing.T) {
	cache := NewInMemoryCache[string, string](5, 0)
	ctx := context.Background()

	// Initial stats
	stats := cache.Stats()
	if stats.Hits != 0 || stats.Misses != 0 || stats.Items != 0 {
		t.Error("initial stats should be zero")
	}

	// Add items
	for i := 1; i <= 5; i++ {
		cache.Set(ctx, string(rune('a'+i-1)), "value")
	}

	stats = cache.Stats()
	if stats.Items != 5 {
		t.Errorf("expected 5 items, got %d", stats.Items)
	}

	// Trigger hits and misses
	cache.Get(ctx, "a")
	cache.Get(ctx, "b")
	cache.Get(ctx, "missing1")
	cache.Get(ctx, "missing2")

	stats = cache.Stats()
	if stats.Hits != 2 {
		t.Errorf("expected 2 hits, got %d", stats.Hits)
	}
	if stats.Misses != 2 {
		t.Errorf("expected 2 misses, got %d", stats.Misses)
	}
}
