package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// newR2Storage menginisialisasi storage berbasis Cloudflare R2 menggunakan klien S3-compatible (mendukung UploadStream/GetStream untuk objek besar).
// Ketentuan:
// - cfg.S3.Bucket harus diisi
// - cfg.S3.Endpoint harus diisi (contoh: https://<accountid>.r2.cloudflarestorage.com)
// Catatan:
// - Path-style addressing dipaksakan untuk R2.
// - Region dapat menggunakan "auto" untuk R2; proses signing menggunakan cfg.S3.Region.
func newR2Storage(ctx context.Context, cfg *Config) (Storage, error) {
	if cfg == nil {
		return nil, errors.New("r2 storage: missing config")
	}
	if cfg.S3.Bucket == "" {
		return nil, errors.New("r2 storage: bucket is required")
	}
	if cfg.S3.Endpoint == "" {
		return nil, errors.New("r2 storage: endpoint is required")
	}
	if cfg.S3.Region == "" {
		cfg.S3.Region = "auto"
	}

	creds := credentials.NewStaticCredentialsProvider(cfg.S3.AccessKeyID, cfg.S3.SecretAccessKey, "")

	awsCfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(cfg.S3.Region),
		config.WithCredentialsProvider(creds),
	)
	if err != nil {
		return nil, fmt.Errorf("r2 storage: load config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.BaseEndpoint = aws.String(cfg.S3.Endpoint)
	})

	return &s3Storage{
		client: client,
		bucket: cfg.S3.Bucket,
		public: cfg.Public,
	}, nil
}
