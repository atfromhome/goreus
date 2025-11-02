# pkg/storage

Mesin penyimpanan sederhana dan minimal untuk Ziswapp dengan satu tanggung jawab:
- Mengunggah konten biner dan mendapatkan path (key) kanonik.
- Mengambil konten biner berdasarkan path (key) untuk diteruskan ke pengguna melalui API.

Prioritas driver:
- AWS S3 (atau penyedia S3-compatible)
- Cloudflare R2
- Filesystem lokal
- Null/Mock (untuk pengujian)

Secara default, semua file bersifat private. Visibilitas dapat dioverride per unggah (upload).

---

## Fitur

- API sangat kecil: `Upload(ctx, path, []byte)` / `Get(ctx, path)` dan dukungan streaming `UploadStream(ctx, path, io.Reader)` / `GetStream(ctx, path) io.ReadCloser` untuk objek besar
- Factory: pilih driver via environment atau secara programatik
- Implementasi S3/R2 maupun Local, plus driver Null/Mock in-memory untuk pengujian
- Tidak mengekspos temporary URL atau tautan publik
- Private secara default (bisa dioverride via opsi upload)
- Tidak ada business logic; murni lapisan integrasi

---

## Instalasi dan dependensi

Paket ini berada di dalam repository. Untuk driver S3/R2, pastikan modul berikut tersedia di `go.mod`:

- github.com/aws/aws-sdk-go-v2
- github.com/aws/aws-sdk-go-v2/config
- github.com/aws/aws-sdk-go-v2/credentials
- github.com/aws/aws-sdk-go-v2/service/s3

Jika Anda baru menambahkan paket ini atau driver S3/R2, pastikan dependensi tersinkron (mis. jalankan `go mod tidy` di proyek Anda).

---

## Variabel environment

Pemilihan driver
- `STORAGE_DRIVER`: `s3` | `r2` | `local` | `null` (default: `null`)
- `STORAGE_PUBLIC`: `true` | `false` (default: `false`). Visibilitas default saat upload (spesifik driver). Private secara default.

S3 / R2 (S3-compatible)
- `S3_BUCKET`: nama bucket (wajib untuk s3/r2)
- `AWS_REGION`: region (untuk R2 dapat menggunakan "auto") (`S3_REGION` didukung sebagai fallback)
- `AWS_ACCESS_KEY_ID`: access key ID (`S3_ACCESS_KEY_ID` didukung sebagai fallback)
- `AWS_SECRET_ACCESS_KEY`: secret access key (`S3_SECRET_ACCESS_KEY` didukung sebagai fallback)
- `S3_ENDPOINT`: URL endpoint (wajib untuk Cloudflare R2, opsional untuk S3 atau S3-compatible lain)
- `S3_FORCE_PATH_STYLE`: `true` | `false` (disarankan true untuk R2)

Local
- `LOCAL_BASE_PATH`: direktori dasar untuk menyimpan file (default: `storage`)

Contoh .env
```
# Driver storage
STORAGE_DRIVER=s3
STORAGE_PUBLIC=false

# Konfigurasi bersama untuk S3/R2
S3_BUCKET=ziswapp-uploads
AWS_REGION=ap-southeast-1
AWS_ACCESS_KEY_ID=AKIA...
AWS_SECRET_ACCESS_KEY=...

# Untuk R2 (endpoint S3-compatible + path style)
# STORAGE_DRIVER=r2
# S3_ENDPOINT=https://<accountid>.r2.cloudflarestorage.com
# S3_FORCE_PATH_STYLE=true

# Local fallback
# STORAGE_DRIVER=local
# LOCAL_BASE_PATH=./storage
```

---

## API

Konstruktor
- NewFromEnv(ctx): Storage
- New(ctx, cfg): Storage

Antarmuka
- Upload(ctx, path, content, ...Option) (string, error)
- Get(ctx, path) ([]byte, error)
- UploadStream(ctx, path, r io.Reader, ...Option) (string, error)
- GetStream(ctx, path) (io.ReadCloser, error)
- Delete(ctx, path) error
- Has(ctx, path) (bool, error)

Opsi
- WithPublic(bool): override visibilitas default per unggah
- WithContentType(string): set metadata content type (jika didukung driver)

---

## Quick start

Inisialisasi dari environment
```go
ctx := context.Background()

st, err := storage.NewFromEnv(ctx)
if err != nil {
    // tangani error (mis-konfigurasi, kredensial hilang, dsb.)
}
```

Konfigurasi programatik
```go
cfg := &storage.Config{
    Driver: "s3",   // atau "r2", "local", "null"
    Public: false,  // visibilitas default untuk Upload
}
cfg.S3.Bucket          = "ziswapp-uploads"
cfg.S3.Region          = "ap-southeast-1" // "auto" oke untuk R2
cfg.S3.AccessKeyID     = os.Getenv("AWS_ACCESS_KEY_ID")
cfg.S3.SecretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
cfg.S3.Endpoint        = ""               // isi untuk R2 atau S3-compatible
cfg.S3.ForcePathStyle  = false            // disarankan true untuk R2

st, err := storage.New(context.Background(), cfg)
if err != nil {
    // tangani error
}
```

Unggah file (PDF) - mode byte-slice
```go
pdf := []byte("%PDF-1.4...") // contoh bytes
key := fmt.Sprintf("org/%d/invoices/%s.pdf", orgID, invoiceNumber)

storedPath, err := st.Upload(ctx, key, pdf,
    storage.WithContentType("application/pdf"),
    // storage.WithPublic(true), // override per unggah (opsional)
)
if err != nil {
    // tangani error
}

// storedPath adalah key kanonik untuk disimpan di DB
fmt.Println("uploaded:", storedPath)
```

Unggah file besar (streaming) - UploadStream
```go
key := fmt.Sprintf("org/%d/invoices/%s.pdf", orgID, invoiceNumber)
// Batasi ukuran agar aman (contoh 50MB)
limited := http.MaxBytesReader(w, r.Body, 50<<20)
defer r.Body.Close()

storedPath, err := st.UploadStream(ctx, key, limited,
    storage.WithContentType("application/pdf"),
)
if err != nil {
    // tangani error
}

fmt.Println("uploaded stream:", storedPath)
```

Ambil file dan teruskan ke respons HTTP
Ambil file dan teruskan ke respons HTTP (streaming untuk objek besar) - GetStream
```go
rc, err := st.GetStream(ctx, storedPath)
if err != nil {
    // terjemahkan ke HTTP status yang sesuai (mis. 404 jika tidak ditemukan)
    w.WriteHeader(http.StatusNotFound)
    return
}
defer rc.Close()

// Jika Anda tahu Content-Type, set eksplisit; jika tidak, bisa dideteksi sebagian di sisi klien/CDN.
w.WriteHeader(http.StatusOK)
_, _ = io.Copy(w, rc)
```

Ambil file ukuran kecil-menengah (byte-slice) - Get
```go
data, err := st.Get(ctx, storedPath)
if err != nil {
    w.WriteHeader(http.StatusNotFound)
    return
}

// Tentukan Content-Type (jika diketahui) atau deteksi cepat
ct := http.DetectContentType(data)
w.Header().Set("Content-Type", ct)
w.WriteHeader(http.StatusOK)
_, _ = w.Write(data)
```

Contoh cepat Has dan Delete
```go
// Cek keberadaan file
exists, err := st.Has(ctx, storedPath)
if err != nil {
    // tangani error
}
fmt.Println("exists:", exists)

// Hapus file
if err := st.Delete(ctx, storedPath); err != nil {
    // tangani error
}
fmt.Println("deleted")
```

Praktik terbaik
- Simpan `storedPath` (key) yang dikembalikan ke DB; jangan simpan bucket atau endpoint per objek.
- Bucket/endpoint dikelola di environment/konfigurasi sehingga migrasi/perubahan tidak menyentuh DB.
- Pertahankan file tetap private kecuali ada kebutuhan publik yang jelas. Gunakan `WithPublic(true)` hanya bila perlu (dengan kebijakan bucket atau CDN).

---

## Rincian driver dan catatan

S3 (AWS S3)
- Konfigurasi AWS standar; endpoint opsional. Jika memakai endpoint kustom (S3-compatible), implementasi saat ini menggunakan `BaseEndpoint` pada opsi S3 (beserta pengaturan path-style bila diperlukan).
- `WithPublic(true)` akan set ACL `public-read` (tergantung kebijakan bucket). Lebih disarankan kebijakan bucket.

Cloudflare R2
- Wajib set S3_ENDPOINT (contoh: `https://<accountid>.r2.cloudflarestorage.com`).
- Path-style addressing dipaksakan; set `S3_FORCE_PATH_STYLE=true`.
- Region dapat `"auto"`.

Local
- Menulis di bawah `LOCAL_BASE_PATH` (default `./storage`).
- Ditujukan untuk development atau lingkungan single-node.

Null/Mock
- In-memory, tidak persisten.
- Untuk pengujian dan lokal saat tidak ingin menyentuh storage nyata.

---

## Contoh handler HTTP

Handler minimal yang mengunggah data dari request dan mengembalikan path penyimpanan (byte-slice):

```go
func UploadInvoiceHandler(st storage.Storage) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        invoiceNo := r.URL.Query().Get("no")
        if invoiceNo == "" {
            http.Error(w, "missing invoice number", http.StatusBadRequest)
            return
        }
        orgID := r.PathValue("org")

        b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 10<<20)) // batasi 10MB
        if err != nil {
            http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
            return
        }
        defer r.Body.Close()

        key := fmt.Sprintf("org/%s/invoices/%s.pdf", orgID, invoiceNo)
        storedPath, err := st.Upload(ctx, key, b, storage.WithContentType("application/pdf"))
        if err != nil {
            http.Error(w, "failed to store file", http.StatusInternalServerError)
            return
        }

        SendRespondWithJson(w, r, http.StatusCreated, map[string]any{
            "path": storedPath,
        })
    }
}
```

Handler minimal yang menyajikan konten tersimpan secara streaming (GetStream):

```go
func GetInvoiceStreamHandler(st storage.Storage) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        path := r.URL.Query().Get("path")
        if strings.TrimSpace(path) == "" {
            http.Error(w, "missing path", http.StatusBadRequest)
            return
        }

        rc, err := st.GetStream(ctx, path)
        if err != nil {
            http.Error(w, "not found", http.StatusNotFound)
            return
        }
        defer rc.Close()

        // Jika Anda tahu content-type, set eksplisit; jika tidak, biarkan klien/CDN menangani.
        w.WriteHeader(http.StatusOK)
        _, _ = io.Copy(w, rc)
    }
}
```

Handler minimal yang menyajikan konten kecil-menengah (Get):

```go
func GetInvoiceHandler(st storage.Storage) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        path := r.URL.Query().Get("path")
        if strings.TrimSpace(path) == "" {
            http.Error(w, "missing path", http.StatusBadRequest)
            return
        }

        data, err := st.Get(ctx, path)
        if err != nil {
            http.Error(w, "not found", http.StatusNotFound)
            return
        }

        // Tentukan Content-Type (jika diketahui) atau deteksi cepat
        ct := http.DetectContentType(data)
        w.Header().Set("Content-Type", ct)
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write(data)
    }
}
```

Handler minimal untuk unggah streaming (UploadStream):

```go
func UploadInvoiceStreamHandler(st storage.Storage) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        invoiceNo := r.URL.Query().Get("no")
        if invoiceNo == "" {
            http.Error(w, "missing invoice number", http.StatusBadRequest)
            return
        }
        orgID := r.PathValue("org")

        // Batasi ukuran payload (contoh 50MB)
        limited := http.MaxBytesReader(w, r.Body, 50<<20)
        defer r.Body.Close()

        key := fmt.Sprintf("org/%s/invoices/%s.pdf", orgID, invoiceNo)
        storedPath, err := st.UploadStream(ctx, key, limited, storage.WithContentType("application/pdf"))
        if err != nil {
            http.Error(w, "failed to store file", http.StatusInternalServerError)
            return
        }

        SendRespondWithJson(w, r, http.StatusCreated, map[string]any{
            "path": storedPath,
        })
    }
}
```

Handler minimal yang menyajikan kembali konten tersimpan:
```go
func GetInvoiceHandler(st storage.Storage) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        path := r.URL.Query().Get("path")
        if strings.TrimSpace(path) == "" {
            http.Error(w, "missing path", http.StatusBadRequest)
            return
        }

        data, err := st.Get(ctx, path)
        if err != nil {
            http.Error(w, "not found", http.StatusNotFound)
            return
        }

        // Jika Anda tahu content-type, set eksplisit; jika tidak, deteksi
        ct := http.DetectContentType(data)
        w.Header().Set("Content-Type", ct)
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write(data)
    }
}
```

---

Contoh handler HTTP tambahan (Has/Delete)

Handler minimal untuk pengecekan keberadaan file (Has):
```go
func HasInvoiceHandler(st storage.Storage) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        path := r.URL.Query().Get("path")
        if strings.TrimSpace(path) == "" {
            http.Error(w, "missing path", http.StatusBadRequest)
            return
        }

        exists, err := st.Has(ctx, path)
        if err != nil {
            http.Error(w, "failed to check", http.StatusInternalServerError)
            return
        }

        SendRespondWithJson(w, r, http.StatusOK, map[string]any{
            "path":   path,
            "exists": exists,
        })
    }
}
```

Handler minimal untuk menghapus file (Delete):
```go
func DeleteInvoiceHandler(st storage.Storage) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        path := r.URL.Query().Get("path")
        if strings.TrimSpace(path) == "" {
            http.Error(w, "missing path", http.StatusBadRequest)
            return
        }

        if err := st.Delete(ctx, path); err != nil {
            http.Error(w, "failed to delete", http.StatusInternalServerError)
            return
        }

        w.WriteHeader(http.StatusNoContent)
    }
}
```

## Pengujian dengan driver null

```go
st := storage.NewFromEnv // atau buat langsung
st = &struct{ storage.Storage }{} // Anda bisa membungkus atau gunakan New dengan konfigurasi null

// Rekomendasi: paksa driver null saat pengujian
os.Setenv("STORAGE_DRIVER", "null")

stg, err := storage.NewFromEnv(context.Background())
if err != nil {
    t.Fatalf("storage init: %v", err)
}

p, err := stg.Upload(ctx, "test/hello.txt", []byte("hello"))
if err != nil {
    t.Fatalf("upload: %v", err)
}
b, err := stg.Get(ctx, p)
if err != nil {
    t.Fatalf("get: %v", err)
}
if string(b) != "hello" {
    t.Fatalf("unexpected content: %q", string(b))
}
```

---

## Catatan dan keterbatasan

- Paket ini kini mendukung dua mode akses: byte-slice (Upload/Get) dan streaming (UploadStream/GetStream).
  - Gunakan mode streaming untuk objek besar agar hemat memori dan lebih stabil.
- Content-Type tidak dipaksakan oleh API storage. Sertakan via `WithContentType` bila diperlukan.
- Akses publik adalah concern spesifik driver. Default yang direkomendasikan adalah private. Jika harus dipublikasikan, atur kebijakan bucket atau aturan CDN secara eksplisit dan gunakan `WithPublic(true)` hanya saat diperlukan.
