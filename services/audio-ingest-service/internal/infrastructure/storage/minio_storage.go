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

type MinIOConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

type minioClient interface {
	PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error)
	GetObject(ctx context.Context, bucketName, objectName string, opts minio.GetObjectOptions) (*minio.Object, error)
	RemoveObject(ctx context.Context, bucketName, objectName string, opts minio.RemoveObjectOptions) error
	BucketExists(ctx context.Context, bucketName string) (bool, error)
	MakeBucket(ctx context.Context, bucketName string, opts minio.MakeBucketOptions) error
}

type MinIOStorage struct {
	client minioClient
	bucket string
}

func NewMinIOStorage(ctx context.Context, cfg MinIOConfig) (*MinIOStorage, error) {
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

	return newMinIOStorageWithClient(ctx, client, cfg.Bucket)
}

func newMinIOStorageWithClient(ctx context.Context, client minioClient, bucket string) (*MinIOStorage, error) {
	if client == nil {
		return nil, errors.New("minio client is nil")
	}
	if strings.TrimSpace(bucket) == "" {
		return nil, errors.New("minio bucket is empty")
	}

	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("check minio bucket: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("create minio bucket: %w", err)
		}
	}

	return &MinIOStorage{
		client: client,
		bucket: bucket,
	}, nil
}

func (s *MinIOStorage) Put(ctx context.Context, objectKey string, body io.Reader) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(objectKey) == "" {
		return errors.New("object key is empty")
	}

	_, err := s.client.PutObject(ctx, s.bucket, objectKey, body, -1, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		return fmt.Errorf("put object to minio: %w", err)
	}
	return nil
}

func (s *MinIOStorage) Get(ctx context.Context, objectKey string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(objectKey) == "" {
		return nil, errors.New("object key is empty")
	}

	object, err := s.client.GetObject(ctx, s.bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object from minio: %w", err)
	}

	return object, nil
}

func (s *MinIOStorage) Delete(ctx context.Context, objectKey string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(objectKey) == "" {
		return errors.New("object key is empty")
	}

	if err := s.client.RemoveObject(ctx, s.bucket, objectKey, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("remove object from minio: %w", err)
	}
	return nil
}
