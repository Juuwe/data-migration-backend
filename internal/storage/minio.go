package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/Juuwe/data-migration-backend/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOStore struct {
	client        *minio.Client
	bucket        string
	publicBaseURL string
}

func NewMinIO(cfg config.ObjectStorage) (*MinIOStore, error) {
	publicURL, err := url.Parse(cfg.PublicBaseURL)
	if err != nil || publicURL.Host == "" || (publicURL.Scheme != "http" && publicURL.Scheme != "https") {
		return nil, fmt.Errorf("invalid MinIO public URL %q", cfg.PublicBaseURL)
	}
	if cfg.Bucket == "" || cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("MinIO bucket and credentials are required")
	}

	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure:       cfg.Secure,
		BucketLookup: minio.BucketLookupPath,
	})
	if err != nil {
		return nil, fmt.Errorf("create MinIO client: %w", err)
	}
	return &MinIOStore{client: client, bucket: cfg.Bucket, publicBaseURL: strings.TrimRight(cfg.PublicBaseURL, "/")}, nil
}

func (s *MinIOStore) Put(ctx context.Context, key string, body io.ReadSeeker, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, body, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("upload MinIO object %q: %w", key, err)
	}
	return nil
}

func (s *MinIOStore) Delete(ctx context.Context, key string) error {
	if err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("delete MinIO object %q: %w", key, err)
	}
	return nil
}

func (s *MinIOStore) PublicURL(key string) string {
	return s.publicBaseURL + "/" + strings.TrimLeft(key, "/")
}
