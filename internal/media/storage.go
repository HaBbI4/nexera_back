package media

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
)

// Storage абстрагирует хранение файлов медиа от конкретного бэкенда (локальный диск, S3 и т.д.)
type Storage interface {
	// Save сохраняет содержимое r под ключом key. contentType и size передаются
	// бэкендам, которым они нужны (например, S3).
	Save(ctx context.Context, key string, r io.Reader, size int64, contentType string) error

	// Delete удаляет файл по ключу
	Delete(ctx context.Context, key string) error

	// URL возвращает постоянную публичную ссылку на файл. Ссылка остаётся
	// действительной, пока файл не будет удалён — без срока истечения.
	URL(key string) string
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
