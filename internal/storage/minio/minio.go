package minio

import (
	"context"
	"fmt"
	"time"

	"github.com/minio/minio-go/v7"
)

type Storage struct {
	client     *minio.Client
	bucketName string
	baseURL    string
}

func New(client *minio.Client, bucketName, baseURL string) *Storage {
	return &Storage{
		client:     client,
		bucketName: bucketName,
		baseURL:    baseURL,
	}
}

func (s *Storage) GetURL(ctx context.Context, objectKey string) (string, error) {
	if objectKey == "" {
		return "", nil
	}

	if s.client == nil {
		return fmt.Sprintf("%s/%s/%s", s.baseURL, s.bucketName, objectKey), nil
	}

	presignedURL, err := s.client.PresignedGetObject(ctx, s.bucketName, objectKey, time.Hour*2, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned url for key '%s': %w", objectKey, err)
	}

	return presignedURL.String(), nil
}
