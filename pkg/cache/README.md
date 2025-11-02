# pkg/cache

Cache in-memory generik, aman terhadap konkurensi, dengan TTL dan kebijakan pengusiran LRU untuk Ziswapp. Fokus satu tanggung jawab:
- Menyimpan nilai (apa pun) di memori proses untuk akses cepat.
- Mendukung TTL per entri (relatif/absolut).
- Mengusir entri berdasarkan LRU saat kapasitas tercapai.

Tidak ada dependensi eksternal; tidak ada I/O jaringan. Cocok untuk caching hot-path di satu proses.

---

## Fitur

- Antarmuka generik: `Cache[K comparable, V any]`
- Implementasi in-memory LRU dengan TTL per entri
- Default TTL global (opsional) saat membuat cache
- Opsi TTL relatif (`WithTTL`) dan absolut (`WithExpiresAt`)
- Statistik: hits, misses, evictions, items, capacity
- Cloner opsional untuk nilai agar aman dari mutasi eksternal
- Aman terhadap konkurensi (mutex + atomics)

---

## Instalasi dan dependensi

Paket berada di dalam repository (tidak memerlukan dependensi eksternal). Pastikan modul Anda tersinkron bila baru menambahkan paket ini:

```
go mod tidy
```

---

## Variabel environment

Tidak ada. Semua konfigurasi cache dilakukan secara programatik di kode.

---

## API

Antarmuka
```go
type Cache[K comparable, V any] interface {
    Get(ctx context.Context, key K) (V, bool)
    Set(ctx context.Context, key K, value V, opts ...SetOption)
    Delete(ctx context.Context, key K)
    Stats() Stats
    Close() error
}
```

Struktur opsi
```go
type SetOptions struct {
    TTL       time.Duration // TTL relatif
    ExpiresAt time.Time     // kedaluwarsa absolut (prioritas di atas TTL)
}

type SetOption func(*SetOptions)

func WithTTL(ttl time.Duration) SetOption
func WithExpiresAt(ts time.Time) SetOption
```

Statistik
```go
type Stats struct {
    Hits      uint64
    Misses    uint64
    Evictions uint64
    Items     int
    Capacity  int
}
```

Implementasi In-Memory
```go
// CloneFunc opsional untuk defensif menyalin nilai saat Set/Get
type CloneFunc[V any] func(V) V

// Opsi konstruktor
func WithCloner[K comparable, V any](fn CloneFunc[V]) InMemoryOption[K, V]

// Konstruktor
func NewInMemoryCache[K comparable, V any](capacity int, defaultTTL time.Duration, opts ...InMemoryOption[K, V]) *InMemoryCache[K, V]
```

Catatan
- Get mengembalikan (value, true) jika ditemukan dan belum kedaluwarsa; jika tidak (zero, false). Perhatikan zero value pada tipe nilai Anda; gunakan bool untuk membedakan miss.

---

## Quick start

Buat cache sederhana untuk string→string, TTL default 5 menit, kapasitas 10.000 item:
```go
c := cache.NewInMemoryCache[string, string](10_000, 5*time.Minute)

// Set dengan TTL eksplisit 60 detik (menimpa default)
c.Set(context.Background(), "greeting", "hello", cache.WithTTL(60*time.Second))

// Get
if v, ok := c.Get(context.Background(), "greeting"); ok {
    fmt.Println("value:", v) // "hello"
}

// Hapus
c.Delete(context.Background(), "greeting")

// Statistik
s := c.Stats()
fmt.Printf("hits=%d misses=%d items=%d cap=%d evictions=%d\n",
    s.Hits, s.Misses, s.Items, s.Capacity, s.Evictions)

// Tutup (no-op untuk in-memory)
_ = c.Close()
```

---

## Contoh penggunaan

### 1) Default TTL, TTL relatif, dan ExpiresAt absolut
```go
// default TTL 10 menit untuk entri tanpa TTL eksplisit
c := cache.NewInMemoryCache[string, []byte](5_000, 10*time.Minute)

// Tanpa opsi → gunakan default TTL (10 menit)
c.Set(ctx, "profile:123", []byte("..."))

// TTL relatif 30 detik
c.Set(ctx, "rate-limit:user:123", []byte("1"), cache.WithTTL(30*time.Second))

// ExpiresAt absolut (prioritas di atas TTL)
exp := time.Now().Add(24 * time.Hour)
c.Set(ctx, "report:daily:2025-11-01", []byte("ready"), cache.WithExpiresAt(exp))
```

### 2) Struktur kompleks dan cloner defensif
Gunakan `WithCloner` agar objek yang diambil tidak bisa dimutasi caller dan “bocor” ke cache.
```go
type Session struct {
    ID     string
    UserID string
    Data   map[string]any
}

// cloner deep-copy sederhana (contoh)
cloneSession := func(s Session) Session {
    out := Session{
        ID:     s.ID,
        UserID: s.UserID,
        Data:   make(map[string]any, len(s.Data)),
    }
    for k, v := range s.Data {
        out.Data[k] = v // sesuaikan jika butuh deep copy yang lebih dalam
    }
    return out
}

c := cache.NewInMemoryCache[string, Session](100_000, 0,
    cache.WithCloner[string, Session](cloneSession),
)

// Set/Get
sess := Session{ID: "abc", UserID: "u1", Data: map[string]any{"role": "admin"}}
c.Set(ctx, "session:abc", sess)

got, ok := c.Get(ctx, "session:abc")
if ok {
    got.Data["role"] = "user" // tidak mempengaruhi nilai di cache karena cloner
}
```

### 3) LRU eviction saat kapasitas terlampaui
```go
c := cache.NewInMemoryCache[int, string](3, 0)

c.Set(ctx, 1, "A")
c.Set(ctx, 2, "B")
c.Set(ctx, 3, "C")
// Akses key=1 agar menjadi paling baru
_, _ = c.Get(ctx, 1)

// Tambah item ke-4, item “terlama” (bukan yang baru diakses) akan dieliminasi
c.Set(ctx, 4, "D")
```

### 4) Pola cache-aside (misalnya cache halaman daftar)
```go
key := fmt.Sprintf("org:%s:projects:page=%d:limit=%d:q=%s", orgID, page, limit, q)

if b, ok := c.Get(ctx, key); ok {
    // parse dan kembalikan dari cache
    return decodeProjects(b), nil
}

// ambil dari DB
items, err := repo.ListProjects(ctx, orgID, page, limit, q)
if err != nil { return nil, err }

// simpan ke cache selama 2 menit
c.Set(ctx, key, encodeProjects(items), cache.WithTTL(2*time.Minute))
return items, nil
```

---

## Praktik terbaik

- Kapasitas
  - Pilih kapasitas berdasarkan peak working set Anda. Mulai dari konservatif (mis. 10–100 ribu item) lalu ukur.
- TTL
  - Gunakan TTL pendek untuk data yang berubah cepat (mis. 30–120 detik).
  - Gunakan TTL lebih panjang untuk data statis (mis. 10–60 menit).
  - Pertimbangkan `ExpiresAt` absolut untuk data yang memiliki siklus harian/bulanan.
- Concurrency
  - Implementasi sudah aman untuk banyak goroutine. Namun, hindari menyimpan nilai yang berisi mutasi bersama; gunakan cloner bila diperlukan.
- Nilai besar
  - Hindari menyimpan payload besar (MBs) dalam jumlah banyak; memori proses akan membengkak. Pertimbangkan fragmentasi atau simpan sebagian di storage.
- Zero value di Get
  - Selalu cek boolean dari `Get`. Nilai zero bukan berarti miss.
- Invalidasi
  - Gunakan `Delete` pada jalur mutasi data back-end (create/update/delete) untuk invalidasi dini entry relevan.
- Durability
  - Cache ini proses-lokal dan volatil. Jangan gunakan sebagai sumber data otoritatif. Restart proses akan mengosongkan cache.
- Distribusi
  - Cache ini tidak terdistribusi dan tidak sinkron antar node. Untuk skala multi-node, pertimbangkan lapisan cache terdistribusi (mis. Redis) di atas pola yang sama.
- Context
  - Parameter `context.Context` saat ini tidak digunakan untuk `Get/Set/Delete` (operasi in-memory), namun disediakan untuk konsistensi API dan masa depan.

---

## Catatan dan keterbatasan

- In-memory, proses-lokal: tidak ada replikasi, tidak ada persistensi.
- LRU + TTL: TTL ditegakkan saat `Get` (lazy expiration) dan saat `Set`/evict; item kedaluwarsa yang tidak diakses dapat bertahan hingga disentuh kembali.
- Cloner opsional: tanpa cloner, Anda bertanggung jawab mencegah mutasi nilai yang dibaca/ditulis ke cache mempengaruhi konsistensi.
- Tidak ada mekanisme “single-flight” bawaan; gunakan pola eksternal jika ingin mencegah stampede (mis. `singleflight`).