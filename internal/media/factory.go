package media

import (
	"context"
	"fmt"

	"nexera_back/internal/config"
)

const (
	DriverLocal = "local"
	DriverS3    = "s3"
)

// NewStorage создаёт реализацию Storage согласно драйверу, выбранному в cfg.Driver
func NewStorage(ctx context.Context, cfg *config.MediaConfig) (Storage, error) {
	switch cfg.Driver {
	case DriverLocal:
		return NewLocalStorage(cfg.Local.Path, cfg.Local.PublicBaseURL)
	case DriverS3:
		return NewS3Storage(ctx, S3Config{
			Bucket:        cfg.S3.Bucket,
			Region:        cfg.S3.Region,
			Endpoint:      cfg.S3.Endpoint,
			AccessKey:     cfg.S3.AccessKey,
			SecretKey:     cfg.S3.SecretKey,
			PublicBaseURL: cfg.S3.PublicBaseURL,
		})
	default:
		return nil, fmt.Errorf("unsupported media driver %q (expected %q or %q)", cfg.Driver, DriverLocal, DriverS3)
	}
}
