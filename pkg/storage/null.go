package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
)

type nullStorage struct {
	data sync.Map // kunci: path(string) -> []byte
}

func newNullStorage() *nullStorage {
	return &nullStorage{}
}

func (n *nullStorage) Upload(ctx context.Context, path string, content []byte, opts ...Option) (string, error) {
	if path == "" {
		return "", errors.New("null storage: empty path")
	}
	cp := make([]byte, len(content))
	copy(cp, content)
	n.data.Store(path, cp)
	return path, nil
}

func (n *nullStorage) Get(ctx context.Context, path string) ([]byte, error) {
	if path == "" {
		return nil, errors.New("null storage: empty path")
	}
	v, ok := n.data.Load(path)
	if !ok {
		return nil, fmt.Errorf("null storage: object %q not found", path)
	}
	src := v.([]byte)
	cp := make([]byte, len(src))
	copy(cp, src)
	return cp, nil
}

// UploadStream mengunggah konten berukuran besar dari io.Reader secara streaming.
// Mengembalikan path (key) kanonik dan error (jika ada).
func (n *nullStorage) UploadStream(ctx context.Context, path string, r io.Reader, opts ...Option) (string, error) {
	if path == "" {
		return "", errors.New("null storage: empty path")
	}
	if r == nil {
		return "", errors.New("null storage: reader is nil")
	}

	b, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("null storage: read stream: %w", err)
	}

	cp := make([]byte, len(b))
	copy(cp, b)
	n.data.Store(path, cp)
	return path, nil
}

// GetStream mengambil konten berukuran besar sebagai io.ReadCloser.
// Caller bertanggung jawab menutup ReadCloser yang dikembalikan.
func (n *nullStorage) GetStream(ctx context.Context, path string) (io.ReadCloser, error) {
	if path == "" {
		return nil, errors.New("null storage: empty path")
	}

	v, ok := n.data.Load(path)
	if !ok {
		return nil, fmt.Errorf("null storage: object %q not found", path)
	}

	src := v.([]byte)
	cp := make([]byte, len(src))
	copy(cp, src)

	return io.NopCloser(bytes.NewReader(cp)), nil
}

// Delete menghapus objek berdasarkan path (key) dari penyimpanan in-memory (idempotent).
func (n *nullStorage) Delete(ctx context.Context, path string) error {
	if path == "" {
		return errors.New("null storage: empty path")
	}
	n.data.Delete(path)
	return nil
}

// Has memeriksa apakah objek dengan path (key) ada di penyimpanan in-memory.
func (n *nullStorage) Has(ctx context.Context, path string) (bool, error) {
	if path == "" {
		return false, errors.New("null storage: empty path")
	}
	_, ok := n.data.Load(path)
	return ok, nil
}
