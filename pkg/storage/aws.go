package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// s3Storage adalah implementasi Storage untuk AWS S3/S3-compatible.
type s3Storage struct {
	client *s3.Client
	bucket string
	public bool
}

// newS3Storage menginisialisasi driver S3 berdasarkan Config (mendukung endpoint S3-compatible dan path-style).
func newS3Storage(ctx context.Context, cfg *Config) (Storage, error) {
	if cfg == nil {
		return nil, errors.New("s3 storage: missing config")
	}
	if cfg.S3.Bucket == "" {
		return nil, errors.New("s3 storage: bucket is required")
	}
	if cfg.S3.Region == "" {
		cfg.S3.Region = "auto"
	}

	creds := credentials.NewStaticCredentialsProvider(cfg.S3.AccessKeyID, cfg.S3.SecretAccessKey, "")

	loadOpts := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.S3.Region),
		config.WithCredentialsProvider(creds),
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return nil, fmt.Errorf("s3 storage: load config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.S3.ForcePathStyle
		if cfg.S3.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.S3.Endpoint)
		}
	})

	return &s3Storage{
		client: client,
		bucket: cfg.S3.Bucket,
		public: cfg.Public,
	}, nil
}

// Upload menyimpan konten biner ke bucket/key dan mengembalikan path (key) kanonik.
func (s *s3Storage) Upload(ctx context.Context, path string, content []byte, opts ...Option) (string, error) {
	if path == "" {
		return "", errors.New("s3 storage: empty path")
	}
	o := applyOptions(s.public, opts)

	input := &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
		Body:   bytes.NewReader(content),
	}
	if o.contentType != "" {
		input.ContentType = aws.String(o.contentType)
	}
	if o.public != nil && *o.public {
		input.ACL = s3types.ObjectCannedACLPublicRead
	}

	if _, err := s.client.PutObject(ctx, input); err != nil {
		return "", fmt.Errorf("s3 storage: put object: %w", err)
	}
	return path, nil
}

// Get mengambil konten biner berdasarkan path (key) dari bucket.
func (s *s3Storage) Get(ctx context.Context, path string) ([]byte, error) {
	if path == "" {
		return nil, errors.New("s3 storage: empty path")
	}

	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return nil, fmt.Errorf("s3 storage: get object: %w", err)
	}
	defer func() { _ = out.Body.Close() }()

	b, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, fmt.Errorf("s3 storage: read body: %w", err)
	}
	return b, nil
}

// UploadStream mengunggah konten berukuran besar dari io.Reader secara streaming.
// Mengembalikan path (key) kanonik dan error (jika ada).
func (s *s3Storage) UploadStream(ctx context.Context, path string, r io.Reader, opts ...Option) (string, error) {
	if path == "" {
		return "", errors.New("s3 storage: empty path")
	}
	if r == nil {
		return "", errors.New("s3 storage: reader is nil")
	}

	o := applyOptions(s.public, opts)

	input := &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
		Body:   r,
	}
	if o.contentType != "" {
		input.ContentType = aws.String(o.contentType)
	}
	if o.public != nil && *o.public {
		input.ACL = s3types.ObjectCannedACLPublicRead
	}

	if _, err := s.client.PutObject(ctx, input); err != nil {
		return "", fmt.Errorf("s3 storage: put object (stream): %w", err)
	}
	return path, nil
}

// GetStream mengambil konten berukuran besar sebagai io.ReadCloser.
// Caller bertanggung jawab menutup ReadCloser yang dikembalikan.
func (s *s3Storage) GetStream(ctx context.Context, path string) (io.ReadCloser, error) {
	if path == "" {
		return nil, errors.New("s3 storage: empty path")
	}

	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return nil, fmt.Errorf("s3 storage: get object (stream): %w", err)
	}

	// Jangan tutup di sini; pembaca (caller) yang akan menutup.
	return out.Body, nil
}

// Delete menghapus objek berdasarkan path (key) dari bucket.
func (s *s3Storage) Delete(ctx context.Context, path string) error {
	if path == "" {
		return errors.New("s3 storage: empty path")
	}

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		return fmt.Errorf("s3 storage: delete object: %w", err)
	}
	// Catatan: DeleteObject di S3 bersifat idempotent; objek yang tidak ada tidak dianggap error.
	return nil
}

// Has memeriksa apakah objek dengan path (key) ada di bucket.
// Mengembalikan true jika ada, false jika tidak ada.
func (s *s3Storage) Has(ctx context.Context, path string) (bool, error) {
	if path == "" {
		return false, errors.New("s3 storage: empty path")
	}

	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	if err == nil {
		return true, nil
	}

	// Tangani kasus not found tanpa mengandalkan tipe error spesifik SDK.
	msg := err.Error()
	if strings.Contains(msg, "NotFound") || strings.Contains(msg, "404") || strings.Contains(msg, "NoSuchKey") {
		return false, nil
	}

	return false, fmt.Errorf("s3 storage: head object: %w", err)
}
