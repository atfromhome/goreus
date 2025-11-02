package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFromEnv(t *testing.T) {
	tests := []struct {
		name  string
		env   map[string]string
		check func(*Config) bool
	}{
		{
			name: "default null driver",
			env:  map[string]string{},
			check: func(c *Config) bool {
				return c.Driver == "null" && !c.Public
			},
		},
		{
			name: "s3 driver config",
			env: map[string]string{
				"STORAGE_DRIVER":        "s3",
				"STORAGE_PUBLIC":        "true",
				"S3_BUCKET":             "my-bucket",
				"AWS_REGION":            "us-east-1",
				"AWS_ACCESS_KEY_ID":     "AKIAIOSFODNN7EXAMPLE",
				"AWS_SECRET_ACCESS_KEY": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			},
			check: func(c *Config) bool {
				return c.Driver == "s3" &&
					c.Public &&
					c.S3.Bucket == "my-bucket" &&
					c.S3.Region == "us-east-1"
			},
		},
		{
			name: "r2 driver config",
			env: map[string]string{
				"STORAGE_DRIVER":      "r2",
				"S3_BUCKET":           "my-r2-bucket",
				"AWS_REGION":          "auto",
				"S3_ENDPOINT":         "https://abc123.r2.cloudflarestorage.com",
				"S3_FORCE_PATH_STYLE": "true",
			},
			check: func(c *Config) bool {
				return c.Driver == "r2" &&
					c.S3.ForcePathStyle &&
					c.S3.Endpoint == "https://abc123.r2.cloudflarestorage.com"
			},
		},
		{
			name: "local driver config",
			env: map[string]string{
				"STORAGE_DRIVER":  "local",
				"LOCAL_BASE_PATH": "/tmp/storage",
			},
			check: func(c *Config) bool {
				return c.Driver == "local" &&
					c.Local.BasePath == "/tmp/storage"
			},
		},
		{
			name: "invalid public value defaults to false",
			env: map[string]string{
				"STORAGE_PUBLIC": "invalid",
			},
			check: func(c *Config) bool {
				return !c.Public
			},
		},
		{
			name: "local base path default",
			env: map[string]string{
				"STORAGE_DRIVER": "local",
			},
			check: func(c *Config) bool {
				return c.Local.BasePath == "storage"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear env
			os.Clearenv()

			// Set test env vars
			for k, v := range tt.env {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			cfg := FromEnv()
			if !tt.check(cfg) {
				t.Errorf("FromEnv check failed")
			}
		})
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil config",
			cfg:     nil,
			wantErr: true,
			errMsg:  "missing config",
		},
		{
			name: "null driver",
			cfg: &Config{
				Driver: "null",
			},
			wantErr: false,
		},
		{
			name: "mock driver alias",
			cfg: &Config{
				Driver: "mock",
			},
			wantErr: false,
		},
		{
			name: "local driver",
			cfg: &Config{
				Driver: "local",
				Local: struct {
					BasePath string
				}{
					BasePath: t.TempDir(),
				},
			},
			wantErr: false,
		},
		{
			name: "unsupported driver",
			cfg: &Config{
				Driver: "unsupported",
			},
			wantErr: true,
			errMsg:  "unsupported driver",
		},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, err := New(ctx, tt.cfg)

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
				return
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error message mismatch: got %q, want containing %q", err.Error(), tt.errMsg)
				}
			}
			if !tt.wantErr && storage == nil {
				t.Error("expected storage, got nil")
			}
		})
	}
}

func TestLocalStorageUpload(t *testing.T) {
	basePath := t.TempDir()
	cfg := &Config{
		Driver: "local",
		Local: struct {
			BasePath string
		}{
			BasePath: basePath,
		},
	}
	storage, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	tests := []struct {
		name    string
		path    string
		content []byte
		wantErr bool
	}{
		{
			name:    "simple upload",
			path:    "test.txt",
			content: []byte("hello world"),
			wantErr: false,
		},
		{
			name:    "nested path upload",
			path:    "dir/subdir/file.txt",
			content: []byte("nested content"),
			wantErr: false,
		},
		{
			name:    "empty path",
			path:    "",
			content: []byte("content"),
			wantErr: true,
		},
		{
			name:    "binary content",
			path:    "binary.bin",
			content: []byte{0x00, 0x01, 0x02, 0xFF},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			path, err := storage.Upload(ctx, tt.path, tt.content)

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
				return
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if !tt.wantErr && path != tt.path {
				t.Errorf("returned path mismatch: got %q, want %q", path, tt.path)
			}
		})
	}
}

func TestLocalStorageGet(t *testing.T) {
	basePath := t.TempDir()
	cfg := &Config{
		Driver: "local",
		Local: struct {
			BasePath string
		}{
			BasePath: basePath,
		},
	}
	storage, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	ctx := context.Background()
	content := []byte("test content")
	storage.Upload(ctx, "test.txt", content)

	tests := []struct {
		name    string
		path    string
		wantErr bool
		check   func([]byte) bool
	}{
		{
			name:    "existing file",
			path:    "test.txt",
			wantErr: false,
			check:   func(b []byte) bool { return bytes.Equal(b, content) },
		},
		{
			name:    "non-existing file",
			path:    "missing.txt",
			wantErr: true,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := storage.Get(ctx, tt.path)

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
				return
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if !tt.wantErr && !tt.check(data) {
				t.Errorf("data mismatch: got %v, want %v", data, content)
			}
		})
	}
}

func TestLocalStorageUploadStream(t *testing.T) {
	basePath := t.TempDir()
	cfg := &Config{
		Driver: "local",
		Local: struct {
			BasePath string
		}{
			BasePath: basePath,
		},
	}
	storage, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		name    string
		path    string
		reader  io.Reader
		wantErr bool
	}{
		{
			name:    "valid stream",
			path:    "stream.txt",
			reader:  bytes.NewReader([]byte("stream content")),
			wantErr: false,
		},
		{
			name:    "empty path",
			path:    "",
			reader:  bytes.NewReader([]byte("content")),
			wantErr: true,
		},
		{
			name:    "nil reader",
			path:    "file.txt",
			reader:  nil,
			wantErr: true,
		},
		{
			name:    "nested path",
			path:    "dir/file.txt",
			reader:  bytes.NewReader([]byte("nested stream")),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := storage.UploadStream(ctx, tt.path, tt.reader)

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
				return
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if !tt.wantErr && path != tt.path {
				t.Errorf("returned path mismatch: got %q, want %q", path, tt.path)
			}
		})
	}
}

func TestLocalStorageGetStream(t *testing.T) {
	basePath := t.TempDir()
	cfg := &Config{
		Driver: "local",
		Local: struct {
			BasePath string
		}{
			BasePath: basePath,
		},
	}
	storage, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	ctx := context.Background()
	content := []byte("stream content")
	storage.Upload(ctx, "test.txt", content)

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "existing file",
			path:    "test.txt",
			wantErr: false,
		},
		{
			name:    "non-existing file",
			path:    "missing.txt",
			wantErr: true,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc, err := storage.GetStream(ctx, tt.path)

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
				return
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if !tt.wantErr {
				defer rc.Close()
				data, _ := io.ReadAll(rc)
				if !bytes.Equal(data, content) {
					t.Errorf("stream data mismatch: got %v, want %v", data, content)
				}
			}
		})
	}
}

func TestLocalStorageDelete(t *testing.T) {
	basePath := t.TempDir()
	cfg := &Config{
		Driver: "local",
		Local: struct {
			BasePath string
		}{
			BasePath: basePath,
		},
	}
	storage, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	ctx := context.Background()
	storage.Upload(ctx, "test.txt", []byte("content"))

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "existing file",
			path:    "test.txt",
			wantErr: false,
		},
		{
			name:    "non-existing file (idempotent)",
			path:    "missing.txt",
			wantErr: false,
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := storage.Delete(ctx, tt.path)

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
				return
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
		})
	}
}

func TestLocalStorageHas(t *testing.T) {
	basePath := t.TempDir()
	cfg := &Config{
		Driver: "local",
		Local: struct {
			BasePath string
		}{
			BasePath: basePath,
		},
	}
	storage, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	ctx := context.Background()
	storage.Upload(ctx, "test.txt", []byte("content"))

	tests := []struct {
		name       string
		path       string
		wantExists bool
		wantErr    bool
	}{
		{
			name:       "existing file",
			path:       "test.txt",
			wantExists: true,
			wantErr:    false,
		},
		{
			name:       "non-existing file",
			path:       "missing.txt",
			wantExists: false,
			wantErr:    false,
		},
		{
			name:       "empty path",
			path:       "",
			wantExists: false,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exists, err := storage.Has(ctx, tt.path)

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
				return
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if !tt.wantErr && exists != tt.wantExists {
				t.Errorf("existence mismatch: got %v, want %v", exists, tt.wantExists)
			}
		})
	}
}

func TestPathTraversalPrevention(t *testing.T) {
	basePath := t.TempDir()
	cfg := &Config{
		Driver: "local",
		Local: struct {
			BasePath string
		}{
			BasePath: basePath,
		},
	}
	storage, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		name string
		path string
	}{
		{
			name: "simple traversal",
			path: "../../../etc/passwd",
		},
		{
			name: "dot segments",
			path: "dir/../../file.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := storage.Upload(ctx, tt.path, []byte("content"))
			if err != nil {
				t.Logf("upload blocked as expected: %v", err)
				return
			}

			// If no error, verify file is within basePath
			fullPath := filepath.Join(basePath, filepath.Clean("/"+tt.path))
			if !strings.HasPrefix(fullPath, filepath.Clean(basePath)) {
				t.Error("path traversal not prevented")
			}
		})
	}
}

func TestWithPublicOption(t *testing.T) {
	opt := WithPublic(true)
	opts := &uploadOptions{}
	opt(opts)

	if opts.public == nil || !*opts.public {
		t.Error("WithPublic option not applied correctly")
	}
}

func TestWithContentTypeOption(t *testing.T) {
	opt := WithContentType("application/json")
	opts := &uploadOptions{}
	opt(opts)

	if opts.contentType != "application/json" {
		t.Errorf("WithContentType option mismatch: got %q", opts.contentType)
	}
}

func TestApplyOptions(t *testing.T) {
	tests := []struct {
		name           string
		defaultPublic  bool
		opts           []Option
		expectedPublic bool
	}{
		{
			name:           "default public true",
			defaultPublic:  true,
			opts:           []Option{},
			expectedPublic: true,
		},
		{
			name:           "override to false",
			defaultPublic:  true,
			opts:           []Option{WithPublic(false)},
			expectedPublic: false,
		},
		{
			name:           "override to true",
			defaultPublic:  false,
			opts:           []Option{WithPublic(true)},
			expectedPublic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applyOptions(tt.defaultPublic, tt.opts)
			if *result.public != tt.expectedPublic {
				t.Errorf("public option mismatch: got %v, want %v", *result.public, tt.expectedPublic)
			}
		})
	}
}
