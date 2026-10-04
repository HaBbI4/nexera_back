package media

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
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

// GenerateKey генерирует уникальный ключ файла в рамках заказа и стадии съёмки
func GenerateKey(orderID int, stage string, format Format) (string, error) {
	suffix, err := randomHex(16)
	if err != nil {
		return "", fmt.Errorf("failed to generate media key: %w", err)
	}
	return fmt.Sprintf("orders/%d/%s/%s%s", orderID, stage, suffix, format.Extension), nil
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
