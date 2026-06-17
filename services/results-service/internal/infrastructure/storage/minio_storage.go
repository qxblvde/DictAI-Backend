package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

type MinIOStorage struct {
	client *minio.Client
	bucket string
}

func New(ctx context.Context, cfg Config) (*MinIOStorage, error) {
	if strings.TrimSpace(cfg.Endpoint) == "" {
		return nil, errors.New("minio endpoint is empty")
	}
	if strings.TrimSpace(cfg.AccessKey) == "" || strings.TrimSpace(cfg.SecretKey) == "" {
		return nil, errors.New("minio credentials are empty")
	}
	if strings.TrimSpace(cfg.Bucket) == "" {
		return nil, errors.New("minio bucket is empty")
	}

	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("check minio bucket: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("create minio bucket: %w", err)
		}
	}

	return &MinIOStorage{client: client, bucket: cfg.Bucket}, nil
}

func (s *MinIOStorage) Put(ctx context.Context, key string, body io.Reader, size int64, contentType string) error {
	if strings.TrimSpace(key) == "" {
		return errors.New("object key is empty")
	}
	_, err := s.client.PutObject(ctx, s.bucket, key, body, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("put object to minio: %w", err)
	}
	return nil
}

func (s *MinIOStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	if strings.TrimSpace(key) == "" {
		return nil, errors.New("object key is empty")
	}
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object from minio: %w", err)
	}
	// MinIO SDK returns object lazily — stat to confirm it exists before returning
	if _, err := obj.Stat(); err != nil {
		_ = obj.Close()
		return nil, fmt.Errorf("stat object: %w", err)
	}
	return obj, nil
}

func (s *MinIOStorage) Delete(ctx context.Context, key string) error {
	if strings.TrimSpace(key) == "" {
		return errors.New("object key is empty")
	}
	if err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("remove object from minio: %w", err)
	}
	return nil
}
