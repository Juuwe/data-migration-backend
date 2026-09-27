package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	MaxImageBytes  int64 = 10 << 20
	MaxVideoBytes  int64 = 100 << 20
	MaxUploadBytes int64 = MaxImageBytes + MaxVideoBytes + 1<<20
)

var ErrInvalidMedia = errors.New("некорректный медиафайл")

type ObjectStore interface {
	Put(ctx context.Context, key string, body io.ReadSeeker, size int64, contentType string) error
	Delete(ctx context.Context, key string) error
	PublicURL(key string) string
}

type MediaInput struct {
	Reader io.ReadSeeker
	Size   int64
}

type preparedMedia struct {
	key         string
	contentType string
	input       MediaInput
}

type MediaService struct {
	store ObjectStore
}

func NewMediaService(store ObjectStore) *MediaService {
	return &MediaService{store: store}
}

func (s *MediaService) prepareImage(input MediaInput) (preparedMedia, error) {
	return s.prepare(input, MaxImageBytes, "images", map[string]string{
		"image/jpeg": ".jpg",
		"image/png":  ".png",
		"image/webp": ".webp",
	})
}

func (s *MediaService) prepareVideo(input MediaInput) (preparedMedia, error) {
	return s.prepare(input, MaxVideoBytes, "videos", map[string]string{
		"video/mp4":  ".mp4",
		"video/webm": ".webm",
	})
}

func (s *MediaService) prepare(input MediaInput, maxBytes int64, directory string, extensions map[string]string) (preparedMedia, error) {
	if input.Reader == nil || input.Size <= 0 || input.Size > maxBytes {
		return preparedMedia{}, fmt.Errorf("%w: %s отсутствует или превышает допустимый размер", ErrInvalidMedia, directory)
	}

	header := make([]byte, 512)
	n, err := io.ReadFull(input.Reader, header)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return preparedMedia{}, fmt.Errorf("read %s: %w", directory, err)
	}
	if _, err := input.Reader.Seek(0, io.SeekStart); err != nil {
		return preparedMedia{}, fmt.Errorf("rewind %s: %w", directory, err)
	}
	contentType := http.DetectContentType(header[:n])
	extension, ok := extensions[contentType]
	if !ok {
		return preparedMedia{}, fmt.Errorf("%w: неподдерживаемый формат %s", ErrInvalidMedia, directory)
	}

	randomName := make([]byte, 16)
	if _, err := rand.Read(randomName); err != nil {
		return preparedMedia{}, fmt.Errorf("generate %s name: %w", directory, err)
	}
	return preparedMedia{
		key:         directory + "/" + hex.EncodeToString(randomName) + extension,
		contentType: contentType,
		input:       input,
	}, nil
}

func (s *MediaService) upload(ctx context.Context, media preparedMedia) error {
	return s.store.Put(ctx, media.key, media.input.Reader, media.input.Size, media.contentType)
}

func (s *MediaService) publicURL(key, fallbackURL string) string {
	if key == "" {
		return fallbackURL
	}
	return s.store.PublicURL(key)
}

func (s *MediaService) cleanup(ctx context.Context, keys ...string) error {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	var errs []error
	for _, key := range keys {
		if key == "" {
			continue
		}
		if err := s.store.Delete(cleanupCtx, key); err != nil {
			errs = append(errs, fmt.Errorf("cleanup %s: %w", key, err))
		}
	}
	return errors.Join(errs...)
}
