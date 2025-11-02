package cache

import (
	"container/list"
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// CloneFunc secara opsional mengkloning nilai pada Set/Get untuk mencegah mutasi eksternal mempengaruhi keadaan cache.
type CloneFunc[V any] func(V) V

// InMemoryCache adalah cache LRU generik yang aman terhadap konkurensi dengan dukungan TTL.
// - K harus comparable.
// - V adalah tipe nilai apa pun.
// - TTL diterapkan per entri (absolut atau relatif melalui SetOptions).
// - Kebijakan pengusiran adalah LRU saat mencapai batas kapasitas; kedaluwarsa TTL ditegakkan secara malas saat Get.
//
// Model konkurensi:
// - Keadaan internal (map + daftar LRU) dilindungi oleh mutex.
// - Penghitung statistik menggunakan atomic untuk mengurangi kontensi.
type InMemoryCache[K comparable, V any] struct {
	mu         sync.Mutex
	ll         *list.List
	cache      map[K]*list.Element
	capacity   int
	defaultTTL time.Duration

	clone CloneFunc[V]

	hits      uint64
	misses    uint64
	evictions uint64
}

type lruNode[K comparable, V any] struct {
	k         K
	v         V
	expiresAt time.Time
}

// InMemoryOption mengonfigurasi InMemoryCache.
type InMemoryOption[K comparable, V any] func(*InMemoryCache[K, V])

// WithCloner menetapkan cloner defensif untuk nilai V guna mencegah mutasi eksternal
// terhadap keadaan cache. Jika disediakan, cloner digunakan pada Set dan Get.
func WithCloner[K comparable, V any](fn CloneFunc[V]) InMemoryOption[K, V] {
	return func(c *InMemoryCache[K, V]) {
		c.clone = fn
	}
}

// NewInMemoryCache membuat cache LRU dalam memori yang bersifat generik.
// - capacity: jumlah maksimum entri yang dipertahankan. Jika <= 0, dibatasi menjadi 1.
// - defaultTTL: jika > 0, entri yang diset tanpa TTL/ExpiresAt eksplisit akan default ke now+defaultTTL.
// - opts: opsi tambahan seperti cloner untuk nilai.
func NewInMemoryCache[K comparable, V any](capacity int, defaultTTL time.Duration, opts ...InMemoryOption[K, V]) *InMemoryCache[K, V] {
	if capacity <= 0 {
		capacity = 1
	}
	c := &InMemoryCache[K, V]{
		ll:         list.New(),
		cache:      make(map[K]*list.Element, capacity),
		capacity:   capacity,
		defaultTTL: defaultTTL,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Get mengambil nilai untuk key tertentu.
// Mengembalikan (nilai, true) jika ditemukan dan belum kedaluwarsa; jika tidak, (zero, false).
func (c *InMemoryCache[K, V]) Get(_ context.Context, key K) (V, bool) {
	var zero V
	now := time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	if ele, ok := c.cache[key]; ok {
		n := ele.Value.(*lruNode[K, V])

		// Pemeriksaan TTL
		if !n.expiresAt.IsZero() && now.After(n.expiresAt) {
			// Hapus entri kedaluwarsa
			c.removeElement(ele)
			atomic.AddUint64(&c.evictions, 1)
			atomic.AddUint64(&c.misses, 1)
			return zero, false
		}

		// Pembaruan posisi LRU (recency)
		c.ll.MoveToFront(ele)
		atomic.AddUint64(&c.hits, 1)

		if c.clone != nil {
			return c.clone(n.v), true
		}
		return n.v, true
	}

	atomic.AddUint64(&c.misses, 1)
	return zero, false
}

// Set menyimpan nilai untuk key tertentu, menggantikan nilai yang sudah ada.
// Menerapkan prioritas SetOptions: ExpiresAt > TTL > defaultTTL.
func (c *InMemoryCache[K, V]) Set(_ context.Context, key K, value V, opts ...SetOption) {
	now := time.Now()

	// Kumpulkan SetOptions
	var so SetOptions
	for _, opt := range opts {
		opt(&so)
	}

	// Tentukan waktu kedaluwarsa
	var exp time.Time
	switch {
	case !so.ExpiresAt.IsZero():
		exp = so.ExpiresAt
	case so.TTL > 0:
		exp = now.Add(so.TTL)
	case c.defaultTTL > 0:
		exp = now.Add(c.defaultTTL)
	default:
		// waktu nol => tidak kedaluwarsa
	}

	// Salinan defensif V jika cloner disediakan.
	if c.clone != nil {
		value = c.clone(value)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if ele, ok := c.cache[key]; ok {
		n := ele.Value.(*lruNode[K, V])
		n.v = value
		n.expiresAt = exp
		c.ll.MoveToFront(ele)
	} else {
		ele := c.ll.PushFront(&lruNode[K, V]{k: key, v: value, expiresAt: exp})
		c.cache[key] = ele
	}

	// Singkirkan jika melebihi kapasitas
	for len(c.cache) > c.capacity {
		c.evictOldest()
	}
}

// Delete menghapus nilai untuk key tertentu jika ada.
func (c *InMemoryCache[K, V]) Delete(_ context.Context, key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ele, ok := c.cache[key]; ok {
		c.removeElement(ele)
	}
}

// Stats mengembalikan snapshot statistik cache saat ini.
func (c *InMemoryCache[K, V]) Stats() Stats {
	c.mu.Lock()
	items := len(c.cache)
	capacity := c.capacity
	c.mu.Unlock()

	return Stats{
		Hits:      atomic.LoadUint64(&c.hits),
		Misses:    atomic.LoadUint64(&c.misses),
		Evictions: atomic.LoadUint64(&c.evictions),
		Items:     items,
		Capacity:  capacity,
	}
}

// Close melepaskan resource apa pun yang digunakan oleh cache.
// Untuk implementasi dalam memori tidak ada resource eksternal.
func (c *InMemoryCache[K, V]) Close() error {
	return nil
}

// evictOldest menghapus elemen yang paling jarang digunakan dari cache.
// Pemanggil harus memegang c.mu.
func (c *InMemoryCache[K, V]) evictOldest() {
	ele := c.ll.Back()
	if ele == nil {
		return
	}
	c.removeElement(ele)
	atomic.AddUint64(&c.evictions, 1)
}

// removeElement menghapus elemen tertentu dari list dan map.
// Pemanggil harus memegang c.mu.
func (c *InMemoryCache[K, V]) removeElement(e *list.Element) {
	n := e.Value.(*lruNode[K, V])
	delete(c.cache, n.k)
	c.ll.Remove(e)
}
