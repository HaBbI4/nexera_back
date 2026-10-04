package media

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Config настройки подключения к S3-совместимому хранилищу
type S3Config struct {
	Bucket    string
	Region    string
	Endpoint  string // задаётся для MinIO/R2 и других S3-совместимых хранилищ (со схемой http:// или https://); для AWS S3 оставить пустым
	AccessKey string
	SecretKey string

	// PublicBaseURL — публичный домен, под которым отдаются объекты бакета
	// (например, CDN перед S3). Если не задан, строится стандартный URL
	// вида https://{bucket}.s3.{region}.amazonaws.com или {endpoint}/{bucket}
	// для S3-совместимых хранилищ.
	PublicBaseURL string
}

// S3Storage хранит файлы в S3-совместимом объектном хранилище.
// Бакет должен быть настроен на публичное чтение (bucket policy) —
// сервис отдаёт постоянные ссылки, а не presigned URL с истечением срока.
type S3Storage struct {
	client        *s3.Client
	bucket        string
	publicBaseURL string
}

// NewS3Storage создаёт клиент S3-хранилища на основе конфигурации
func NewS3Storage(ctx context.Context, cfg S3Config) (*S3Storage, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("s3 bucket is not configured")
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			// S3-совместимые хранилища (MinIO и т.п.) обычно требуют path-style доступа
			o.UsePathStyle = true
		}
	})

	publicBaseURL := strings.TrimRight(cfg.PublicBaseURL, "/")
	if publicBaseURL == "" {
		switch {
		case cfg.Endpoint != "":
			publicBaseURL = strings.TrimRight(cfg.Endpoint, "/") + "/" + cfg.Bucket
		default:
			publicBaseURL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com", cfg.Bucket, cfg.Region)
		}
	}

	return &S3Storage{client: client, bucket: cfg.Bucket, publicBaseURL: publicBaseURL}, nil
}

func (s *S3Storage) Save(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        r,
		ContentType: aws.String(contentType),
	}
	if size > 0 {
		input.ContentLength = aws.Int64(size)
	}

	if _, err := s.client.PutObject(ctx, input); err != nil {
		return fmt.Errorf("failed to upload %s to s3: %w", key, err)
	}
	return nil
}

func (s *S3Storage) Delete(ctx context.Context, key string) error {
	if _, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}); err != nil {
		return fmt.Errorf("failed to delete %s from s3: %w", key, err)
	}
	return nil
}

func (s *S3Storage) URL(key string) string {
	return s.publicBaseURL + "/" + key
}
