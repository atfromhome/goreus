package cache

import (
	"context"
	"testing"
	"time"
)

// Test dasar Set/Get dan metrik statistik (Hits/Misses/Items/Capacity)
func TestInMemoryCache_SetGet_Basic(t *testing.T) {
	c := NewInMemoryCache[string, int](10, 0)
	defer c.Close()

	ctx := context.Background()

	// Awalnya miss
	if _, ok := c.Get(ctx, "x"); ok {
		t.Fatalf("seharusnya miss pada key yang belum ada")
	}
	if s := c.Stats(); s.Misses != 1 {
		t.Fatalf("misses harus 1, got %d", s.Misses)
	}

	// Set lalu Get
	c.Set(ctx, "x", 42)
	v, ok := c.Get(ctx, "x")
	if !ok {
		t.Fatalf("seharusnya hit setelah Set")
	}
	if v != 42 {
		t.Fatalf("nilai tidak sesuai, got %d want 42", v)
	}

	// Periksa statistik
	s := c.Stats()
	if s.Hits != 1 {
		t.Fatalf("hits harus 1, got %d", s.Hits)
	}
	if s.Items != 1 || s.Capacity != 10 {
		t.Fatalf("stats tidak sesuai, Items=%d Capacity=%d", s.Items, s.Capacity)
	}
}

// Test TTL kedaluwarsa dengan opsi WithTTL
func TestInMemoryCache_TTL_Expiry(t *testing.T) {
	c := NewInMemoryCache[string, int](10, 0)
	defer c.Close()

	ctx := context.Background()
	c.Set(ctx, "ttl", 1, WithTTL(50*time.Millisecond))

	time.Sleep(70 * time.Millisecond) // beri waktu agar TTL lewat

	if _, ok := c.Get(ctx, "ttl"); ok {
		t.Fatalf("seharusnya miss setelah TTL kedaluwarsa")
	}
	s := c.Stats()
	if s.Evictions != 1 {
		t.Fatalf("evictions harus 1 karena entri kedaluwarsa, got %d", s.Evictions)
	}
}

// Test prioritas ExpiresAt dibanding TTL ketika keduanya diberikan.
// Catatan: karena opsi diterapkan secara berurutan, gunakan urutan (WithTTL, WithExpiresAt)
// agar ExpiresAt yang terakhir menetapkan nilai final.
func TestInMemoryCache_ExpiresAt_TakesPrecedence(t *testing.T) {
	c := NewInMemoryCache[string, int](10, 0)
	defer c.Close()

	ctx := context.Background()
	exp := time.Now().Add(50 * time.Millisecond)

	// Berikan TTL panjang, namun ExpiresAt lebih cepat → ExpiresAt harus menang.
	c.Set(ctx, "key", 7, WithTTL(1*time.Hour), WithExpiresAt(exp))

	time.Sleep(70 * time.Millisecond)

	if _, ok := c.Get(ctx, "key"); ok {
		t.Fatalf("seharusnya miss karena ExpiresAt telah lewat meski TTL panjang")
	}
}

// Test defaultTTL pada konstruktor digunakan ketika Set tanpa opsi TTL/ExpiresAt
func TestInMemoryCache_DefaultTTL(t *testing.T) {
	c := NewInMemoryCache[string, int](10, 50*time.Millisecond)
	defer c.Close()

	ctx := context.Background()
	c.Set(ctx, "k", 99) // tanpa opsi → pakai defaultTTL

	time.Sleep(70 * time.Millisecond)

	if _, ok := c.Get(ctx, "k"); ok {
		t.Fatalf("seharusnya miss karena defaultTTL kedaluwarsa")
	}
	if c.Stats().Evictions != 1 {
		t.Fatalf("evictions harus 1 karena defaultTTL kedaluwarsa")
	}
}

// Test kebijakan LRU (capacity eviction) dan urutan akses memengaruhi entri yang disingkirkan
func TestInMemoryCache_CapacityEviction_LRU(t *testing.T) {
	c := NewInMemoryCache[string, int](2, 0)
	defer c.Close()

	ctx := context.Background()

	c.Set(ctx, "a", 1)
	c.Set(ctx, "b", 2)

	// Akses "a" agar menjadi paling baru
	if _, ok := c.Get(ctx, "a"); !ok {
		t.Fatalf("harusnya ada 'a'")
	}

	// Tambah "c" → kapasitas terlampaui, LRU ("b") harus dikeluarkan
	c.Set(ctx, "c", 3)

	if _, ok := c.Get(ctx, "b"); ok {
		t.Fatalf("'b' harusnya ter-evict")
	}
	if _, ok := c.Get(ctx, "a"); !ok {
		t.Fatalf("'a' harusnya masih ada (baru diakses)")
	}
	if _, ok := c.Get(ctx, "c"); !ok {
		t.Fatalf("'c' harusnya ada")
	}

	if c.Stats().Evictions != 1 {
		t.Fatalf("evictions harus 1 setelah melampaui kapasitas")
	}
}

// Test WithCloner: memastikan salinan defensif pada Set dan Get
func TestInMemoryCache_WithCloner_DefensiveCopy(t *testing.T) {
	type blob struct {
		arr []int
	}
	cloner := func(b blob) blob {
		if b.arr == nil {
			return b
		}
		cp := make([]int, len(b.arr))
		copy(cp, b.arr)
		return blob{arr: cp}
	}

	c := NewInMemoryCache[string, blob](10, 0, WithCloner[string, blob](cloner))
	defer c.Close()

	ctx := context.Background()

	orig := blob{arr: []int{1, 2, 3}}
	c.Set(ctx, "k", orig)

	// Mutasi sumber setelah Set tidak memengaruhi cache
	orig.arr[0] = 99

	got1, ok := c.Get(ctx, "k")
	if !ok {
		t.Fatalf("harusnya hit pada 'k'")
	}
	if got1.arr[0] != 1 {
		t.Fatalf("nilai di cache berubah akibat mutasi luar, got %v", got1.arr)
	}

	// Mutasi nilai hasil Get tidak memengaruhi penyimpanan internal
	got1.arr[0] = 77
	got2, _ := c.Get(ctx, "k")
	if got2.arr[0] != 1 {
		t.Fatalf("nilai internal seharusnya tetap tidak berubah, got %v", got2.arr)
	}
}

// Test Delete menghapus entri
func TestInMemoryCache_Delete_Removes(t *testing.T) {
	c := NewInMemoryCache[string, string](10, 0)
	defer c.Close()

	ctx := context.Background()
	c.Set(ctx, "rm", "x")

	if _, ok := c.Get(ctx, "rm"); !ok {
		t.Fatalf("harusnya ada sebelum delete")
	}

	c.Delete(ctx, "rm")

	if _, ok := c.Get(ctx, "rm"); ok {
		t.Fatalf("seharusnya miss setelah delete")
	}
	if c.Stats().Items != 0 {
		t.Fatalf("Items harus 0 setelah delete, got %d", c.Stats().Items)
	}
}

// Test kapasitas 0 diklem menjadi 1 (sesuai implementasi)
func TestInMemoryCache_ZeroCapacity_ClampedToOne(t *testing.T) {
	c := NewInMemoryCache[string, int](0, 0)
	defer c.Close()

	ctx := context.Background()

	c.Set(ctx, "k1", 1)
	c.Set(ctx, "k2", 2) // memicu eviction karena kapasitas efektif = 1

	if _, ok := c.Get(ctx, "k1"); ok {
		t.Fatalf("'k1' harusnya ter-evict karena kapasitas 1")
	}
	v, ok := c.Get(ctx, "k2")
	if !ok || v != 2 {
		t.Fatalf("'k2' harusnya ada dengan nilai 2, got %d ok=%v", v, ok)
	}

	s := c.Stats()
	if s.Items != 1 || s.Capacity != 1 {
		t.Fatalf("Items/Capacity tidak sesuai, %+v", s)
	}
	if s.Evictions < 1 {
		t.Fatalf("harus ada eviction setidaknya 1, got %d", s.Evictions)
	}
}
