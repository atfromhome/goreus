package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// Storage adalah antarmuka minimal untuk menyimpan dan mengambil objek biner.
// Mendukung dua mode:
// - Byte-slice: Upload/Get untuk ukuran kecil-menengah.
// - Streaming: UploadStream/GetStream untuk objek besar (io.Reader/io.ReadCloser).
// Catatan:
// - Upload/UploadStream mengembalikan path (key) kanonik dan error (jika ada).
// - Get mengembalikan seluruh konten sebagai []byte; GetStream mengembalikan io.ReadCloser.
// - Delete menghapus objek pada path (key) tertentu.
// - Has memeriksa keberadaan objek untuk path (key) tertentu.
type Storage interface {
	Upload(ctx context.Context, path string, content []byte, opts ...Option) (string, error)
	Get(ctx context.Context, path string) ([]byte, error)
	UploadStream(ctx context.Context, path string, r io.Reader, opts ...Option) (string, error)
	GetStream(ctx context.Context, path string) (io.ReadCloser, error)
	Delete(ctx context.Context, path string) error
	Has(ctx context.Context, path string) (bool, error)
}

// Config menyimpan konfigurasi storage. Gunakan FromEnv() untuk memuat dari variabel environment.
//
// Driver:
// - "s3"     : AWS S3 atau S3-compatible
// - "r2"     : Cloudflare R2 (S3-compatible; umumnya perlu path-style + endpoint kustom)
// - "local"  : Penyimpanan filesystem lokal
// - "null"   : Penyimpanan in-memory (untuk pengujian); alias: "mock"
type Config struct {
	Driver string
	// Public menentukan visibilitas/ACL bawaan saat upload (tergantung driver).
	// Implementasi sebaiknya menjaga objek tetap private secara default kecuali disetel true atau dioverride per-upload.
	Public bool

	S3 struct {
		Bucket          string
		Region          string
		AccessKeyID     string
		SecretAccessKey string
		Endpoint        string
		ForcePathStyle  bool
	}

	Local struct {
		// BasePath adalah direktori root untuk penyimpanan lokal.
		// Jika kosong, gunakan default "storage".
		BasePath string
	}
}

// FromEnv memuat nilai Config storage dari variabel environment.
//
// Variabel:
// - STORAGE_DRIVER: s3 | r2 | local | null (default: null)
// - STORAGE_PUBLIC: true | false (default: false)
// - S3_BUCKET, AWS_REGION, AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY
// - S3_ENDPOINT (opsional; wajib untuk sebagian penyedia S3-compatible seperti R2)
// - S3_FORCE_PATH_STYLE: true | false (disarankan true untuk R2)
// - LOCAL_BASE_PATH: direktori dasar untuk driver "local" (default: storage)
func FromEnv() *Config {
	cfg := &Config{
		Driver: strings.ToLower(strings.TrimSpace(os.Getenv("STORAGE_DRIVER"))),
	}

	if v := strings.TrimSpace(os.Getenv("STORAGE_PUBLIC")); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.Public = b
		}
	}

	// S3/R2
	cfg.S3.Bucket = strings.TrimSpace(os.Getenv("S3_BUCKET"))
	cfg.S3.Region = strings.TrimSpace(os.Getenv("AWS_REGION"))
	if cfg.S3.Region == "" {
		cfg.S3.Region = strings.TrimSpace(os.Getenv("S3_REGION"))
	}
	cfg.S3.AccessKeyID = strings.TrimSpace(os.Getenv("AWS_ACCESS_KEY_ID"))
	if cfg.S3.AccessKeyID == "" {
		cfg.S3.AccessKeyID = strings.TrimSpace(os.Getenv("S3_ACCESS_KEY_ID"))
	}
	cfg.S3.SecretAccessKey = strings.TrimSpace(os.Getenv("AWS_SECRET_ACCESS_KEY"))
	if cfg.S3.SecretAccessKey == "" {
		cfg.S3.SecretAccessKey = strings.TrimSpace(os.Getenv("S3_SECRET_ACCESS_KEY"))
	}
	cfg.S3.Endpoint = strings.TrimSpace(os.Getenv("S3_ENDPOINT"))
	if v := strings.TrimSpace(os.Getenv("S3_FORCE_PATH_STYLE")); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			cfg.S3.ForcePathStyle = b
		}
	}

	// Local
	cfg.Local.BasePath = strings.TrimSpace(os.Getenv("LOCAL_BASE_PATH"))
	if cfg.Local.BasePath == "" {
		cfg.Local.BasePath = "storage"
	}

	if cfg.Driver == "" {
		cfg.Driver = "null"
	}

	return cfg
}

// NewFromEnv membuat Storage dari variabel environment.
func NewFromEnv(ctx context.Context) (Storage, error) {
	return New(ctx, FromEnv())
}

// New membuat Storage berdasarkan Config yang diberikan.
// Factory ini mendelegasikan ke konstruktor driver di file lain:
//   - aws.go / cloudflare.go: newS3Storage, newR2Storage
//   - local.go:               newLocalStorage
//   - null.go:                newNullStorage
func New(ctx context.Context, cfg *Config) (Storage, error) {
	if cfg == nil {
		return nil, errors.New("storage: missing config")
	}

	switch strings.ToLower(strings.TrimSpace(cfg.Driver)) {
	case "s3":
		return newS3Storage(ctx, cfg)
	case "r2":
		return newR2Storage(ctx, cfg)
	case "local":
		return newLocalStorage(cfg)
	case "null", "mock", "":
		return newNullStorage(), nil
	default:
		return nil, fmt.Errorf("storage: unsupported driver %q", cfg.Driver)
	}
}

// Option mengatur perilaku saat upload.
type Option func(*uploadOptions)

type uploadOptions struct {
	public      *bool
	contentType string
}

// WithPublic menimpa visibilitas default untuk satu kali pemanggilan Upload.
func WithPublic(public bool) Option {
	return func(o *uploadOptions) { o.public = &public }
}

// WithContentType mengatur metadata content-type (jika didukung driver).
func WithContentType(ct string) Option {
	return func(o *uploadOptions) { o.contentType = ct }
}

// applyOptions menggabungkan opsi yang diberikan dengan nilai default (level driver) untuk public.
// Implementasi driver sebaiknya memanggil ini untuk menentukan opsi efektif.
func applyOptions(defaultPublic bool, opts []Option) uploadOptions {
	o := uploadOptions{}
	for _, fn := range opts {
		if fn != nil {
			fn(&o)
		}
	}
	if o.public == nil {
		o.public = &defaultPublic
	}
	return o
}
