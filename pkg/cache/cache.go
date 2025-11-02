package cache

import (
	"context"
	"time"
)

// Cache adalah antarmuka cache generik yang aman terhadap konkurensi.
// Implementasi bebas memilih kebijakan pengusiran (mis. LRU) dan dapat mendukung TTL melalui SetOptions.
//
// K harus comparable agar implementasi dapat menggunakannya sebagai key map.
// V adalah tipe nilai apa pun yang disimpan di cache.
type Cache[K comparable, V any] interface {
	// Get mengambil nilai untuk key tertentu.
	// Mengembalikan (nilai, true) jika ditemukan dan belum kedaluwarsa; jika tidak, (zero, false).
	Get(ctx context.Context, key K) (V, bool)

	// Set menyimpan nilai untuk key tertentu, menggantikan nilai yang sudah ada.
	// Implementasi sebaiknya menerapkan SetOptions (mis. TTL/ExpiresAt) jika diberikan.
	Set(ctx context.Context, key K, value V, opts ...SetOption)

	// Delete menghapus nilai untuk key tertentu jika ada.
	Delete(ctx context.Context, key K)

	// Stats mengembalikan ringkasan statistik cache saat ini.
	Stats() Stats

	// Close melepaskan resource apa pun yang digunakan cache.
	// Implementasi yang tidak memegang resource eksternal dapat mengembalikan nil.
	Close() error
}

// SetOptions membawa perilaku opsional untuk Set (mis. TTL/ExpiresAt).
// Implementasi sebaiknya menerapkan ExpiresAt jika tidak nol; jika tidak, bila TTL > 0,
// ExpiresAt dihitung sebagai now()+TTL saat Set dipanggil.
//
// Jika ExpiresAt dan TTL keduanya nol, entri tidak kedaluwarsa kecuali
// implementasi menerapkan kebijakan TTL bawaan.
type SetOptions struct {
	// TTL menentukan time-to-live relatif untuk entri.
	// Implementasi sebaiknya menghitung ExpiresAt = now()+TTL saat menerapkan opsi ini.
	TTL time.Duration

	// ExpiresAt menentukan waktu kedaluwarsa absolut untuk entri.
	// Jika tidak nol, ini memiliki prioritas dibanding TTL.
	ExpiresAt time.Time
}

// SetOption memodifikasi SetOptions; digunakan dengan pola functional options.
type SetOption func(*SetOptions)

// WithTTL menetapkan TTL untuk entri cache. Implementasi sebaiknya menghitung
// ExpiresAt sebagai now()+ttl saat menerapkan opsi ini pada Set.
func WithTTL(ttl time.Duration) SetOption {
	return func(so *SetOptions) {
		so.TTL = ttl
		so.ExpiresAt = time.Time{} // prefer TTL over stale absolute value
	}
}

// WithExpiresAt menetapkan waktu kedaluwarsa absolut untuk entri cache.
// Ini memiliki prioritas dibanding TTL jika keduanya diberikan.
func WithExpiresAt(ts time.Time) SetOption {
	return func(so *SetOptions) {
		so.ExpiresAt = ts
		so.TTL = 0 // prefer absolute over relative
	}
}

// Stats menyediakan metrik dasar untuk implementasi cache dalam proses.
type Stats struct {
	// Hits adalah jumlah hit cache.
	Hits uint64
	// Misses adalah jumlah miss cache.
	Misses uint64
	// Evictions adalah jumlah entri yang dieliminasi karena kapasitas/TTL.
	Evictions uint64
	// Items adalah jumlah item saat ini yang disimpan di cache.
	Items int
	// Capacity adalah jumlah maksimum item yang akan dipertahankan cache.
	Capacity int
}
