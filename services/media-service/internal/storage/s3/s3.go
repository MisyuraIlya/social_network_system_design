package s3

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	UseSSL    bool
	Bucket    string
}

type Storage struct {
	cfg    Config
	client *minio.Client
}

func New(cfg Config) (*Storage, error) {
	cl, err := minio.New(strings.TrimPrefix(cfg.Endpoint, "http://"), &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, err
	}
	return &Storage{cfg: cfg, client: cl}, nil
}

func (s *Storage) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.cfg.Bucket)
	if err != nil {
		return err
	}
	if !exists {
		return s.client.MakeBucket(ctx, s.cfg.Bucket, minio.MakeBucketOptions{})
	}
	return nil
}

func (s *Storage) Put(ctx context.Context, key string, contentType string, data []byte) error {
	_, err := s.client.PutObject(ctx, s.cfg.Bucket, key,
		strings.NewReader(string(data)), int64(len(data)),
		minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (s *Storage) FPut(ctx context.Context, key, path, contentType string) error {
	_, err := s.client.FPutObject(ctx, s.cfg.Bucket, key, path, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

func (s *Storage) Remove(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.cfg.Bucket, key, minio.RemoveObjectOptions{})
}

func (s *Storage) PresignGet(ctx context.Context, key string, ttl time.Duration) (*url.URL, error) {
	return s.client.PresignedGetObject(ctx, s.cfg.Bucket, key, ttl, nil)
}

func (s *Storage) PresignPut(ctx context.Context, key string, ttl time.Duration, contentType string) (*url.URL, error) {
	reqParams := make(url.Values)
	if contentType != "" {
		reqParams.Set("content-type", contentType)
	}
	return s.client.PresignedPutObject(ctx, s.cfg.Bucket, key, ttl)
}
