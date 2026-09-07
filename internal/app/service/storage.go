package service

import (
	"context"
	"fmt"
	"time"

	"github.com/minio/minio-go/v7"
)

type StorageService struct {
	client     *minio.Client
	bucketName string
	baseURL    string
}

func NewStorageService(client *minio.Client, bucketName, baseURL string) *StorageService {
	return &StorageService{
		client:     client,
		bucketName: bucketName,
		baseURL:    baseURL,
	}
}

func (s *StorageService) GetURL(ctx context.Context, objectKey string) (string, error) {
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
