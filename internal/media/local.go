package media

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LocalStorage хранит файлы на локальном диске
type LocalStorage struct {
	basePath      string
	publicBaseURL string
}

// NewLocalStorage создаёт локальное файловое хранилище с базовой директорией basePath.
// publicBaseURL — префикс, по которому файлы отдаются наружу (см. handlers.NewLocalMediaHandler,
// смонтированный на GET /media/{path...}); если не задан, используется "/media".
func NewLocalStorage(basePath, publicBaseURL string) (*LocalStorage, error) {
	if basePath == "" {
		return nil, fmt.Errorf("local media path is not configured")
	}
	if err := os.MkdirAll(basePath, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create media directory %s: %w", basePath, err)
	}
	if publicBaseURL == "" {
		publicBaseURL = "/media"
	}
	return &LocalStorage{basePath: basePath, publicBaseURL: strings.TrimRight(publicBaseURL, "/")}, nil
}

func (s *LocalStorage) Save(_ context.Context, key string, r io.Reader, _ int64, _ string) error {
	path := filepath.Join(s.basePath, filepath.FromSlash(key))

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", key, err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", key, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, r); err != nil {
		return fmt.Errorf("failed to write file %s: %w", key, err)
	}

	return nil
}

func (s *LocalStorage) Delete(_ context.Context, key string) error {
	path := filepath.Join(s.basePath, filepath.FromSlash(key))
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file %s: %w", key, err)
	}
	return nil
}

func (s *LocalStorage) URL(key string) string {
	return s.publicBaseURL + "/" + key
}
