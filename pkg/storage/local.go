package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// localStorage mengimplementasikan Storage menggunakan filesystem lokal.
type localStorage struct {
	basePath string
	public   bool
}

// newLocalStorage membangun driver penyimpanan filesystem lokal.
// cfg.Local.BasePath digunakan sebagai root; default ke "storage" jika kosong.
func newLocalStorage(cfg *Config) (Storage, error) {
	if cfg == nil {
		return nil, errors.New("local storage: missing config")
	}
	base := cfg.Local.BasePath
	if base == "" {
		base = "storage"
	}
	// Pastikan direktori dasar (base) tersedia
	if err := os.MkdirAll(base, 0o755); err != nil {
		return nil, fmt.Errorf("local storage: mkdir base %s: %w", base, err)
	}
	return &localStorage{
		basePath: base,
		public:   cfg.Public,
	}, nil
}

// Upload menulis konten ke filesystem lokal pada path (key) yang diberikan.
// Mengembalikan path (key) kanonik dan error jika ada.
func (l *localStorage) Upload(ctx context.Context, path string, content []byte, opts ...Option) (string, error) {
	if path == "" {
		return "", errors.New("local storage: empty path")
	}

	// Normalisasi dan cegah path traversal keluar dari basePath.
	cleanRel := filepath.Clean("/" + path) // paksa absolut lalu bersihkan
	full := filepath.Join(l.basePath, cleanRel)

	dir := filepath.Dir(full)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("local storage: mkdir %s: %w", dir, err)
	}

	if err := os.WriteFile(full, content, 0o644); err != nil {
		return "", fmt.Errorf("local storage: write %s: %w", full, err)
	}

	return path, nil
}

// Get membaca dan mengembalikan konten file dari filesystem lokal untuk path (key) yang diberikan.
func (l *localStorage) Get(ctx context.Context, path string) ([]byte, error) {
	if path == "" {
		return nil, errors.New("local storage: empty path")
	}

	// Normalisasi dan cegah path traversal keluar dari basePath.
	cleanRel := filepath.Clean("/" + path)
	full := filepath.Join(l.basePath, cleanRel)

	b, err := os.ReadFile(full)
	if err != nil {
		return nil, fmt.Errorf("local storage: read %s: %w", full, err)
	}
	return b, nil
}

// UploadStream mengunggah konten berukuran besar dari io.Reader secara streaming.
// Mengembalikan path (key) kanonik dan error (jika ada).
func (l *localStorage) UploadStream(ctx context.Context, path string, r io.Reader, opts ...Option) (string, error) {
	if path == "" {
		return "", errors.New("local storage: empty path")
	}
	if r == nil {
		return "", errors.New("local storage: reader is nil")
	}

	// Normalisasi dan cegah path traversal keluar dari basePath.
	cleanRel := filepath.Clean("/" + path)
	full := filepath.Join(l.basePath, cleanRel)

	dir := filepath.Dir(full)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("local storage: mkdir %s: %w", dir, err)
	}

	f, err := os.Create(full)
	if err != nil {
		return "", fmt.Errorf("local storage: create %s: %w", full, err)
	}
	defer func() { _ = f.Close() }()

	if _, err := io.Copy(f, r); err != nil {
		return "", fmt.Errorf("local storage: write stream %s: %w", full, err)
	}

	return path, nil
}

// GetStream membaca konten berukuran besar sebagai io.ReadCloser.
// Caller bertanggung jawab menutup ReadCloser yang dikembalikan.
func (l *localStorage) GetStream(ctx context.Context, path string) (io.ReadCloser, error) {
	if path == "" {
		return nil, errors.New("local storage: empty path")
	}

	// Normalisasi dan cegah path traversal keluar dari basePath.
	cleanRel := filepath.Clean("/" + path)
	full := filepath.Join(l.basePath, cleanRel)

	f, err := os.Open(full)
	if err != nil {
		return nil, fmt.Errorf("local storage: open %s: %w", full, err)
	}

	return f, nil
}

// Delete menghapus objek pada path (key) tertentu dari filesystem lokal.
func (l *localStorage) Delete(ctx context.Context, path string) error {
	if path == "" {
		return errors.New("local storage: empty path")
	}

	// Normalisasi dan cegah path traversal keluar dari basePath.
	cleanRel := filepath.Clean("/" + path)
	full := filepath.Join(l.basePath, cleanRel)

	if err := os.Remove(full); err != nil {
		if os.IsNotExist(err) {
			// Idempotent: jika tidak ada, anggap sukses
			return nil
		}
		return fmt.Errorf("local storage: delete %s: %w", full, err)
	}
	return nil
}

// Has memeriksa apakah objek dengan path (key) ada di filesystem lokal.
func (l *localStorage) Has(ctx context.Context, path string) (bool, error) {
	if path == "" {
		return false, errors.New("local storage: empty path")
	}

	// Normalisasi dan cegah path traversal keluar dari basePath.
	cleanRel := filepath.Clean("/" + path)
	full := filepath.Join(l.basePath, cleanRel)

	if _, err := os.Stat(full); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("local storage: stat %s: %w", full, err)
	}
	return true, nil
}
