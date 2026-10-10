package media

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
)

// ErrNotAnImage возвращается, когда присланный файл распознан по содержимому (magic bytes),
// но это видео, а не изображение — для полей вроде image_url допустимо только фото.
var ErrNotAnImage = errors.New("uploaded file is not an image")

// UploadImage сохраняет один файл как изображение в поддиректорию category (например,
// "banners", "services") и возвращает его публичный URL.
//
// Это универсальная точка входа для "принять одну картинку в рамках более крупного запроса"
func UploadImage(ctx context.Context, storage Storage, category string, size int64, content io.Reader) (string, error) {
	header := make([]byte, SniffHeaderSize)
	n, err := io.ReadFull(content, header)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	header = header[:n]

	format, err := Sniff(header)
	if err != nil {
		return "", err
	}
	if format.Kind != KindPhoto {
		return "", ErrNotAnImage
	}

	key, err := generateCategoryKey(category, format)
	if err != nil {
		return "", err
	}

	// Часть файла уже прочитана для определения формата — дочитываем остаток.
	fullContent := io.MultiReader(bytes.NewReader(header), content)

	if err := storage.Save(ctx, key, fullContent, size, format.MIME); err != nil {
		return "", fmt.Errorf("failed to save image: %w", err)
	}

	return storage.URL(key), nil
}

// generateCategoryKey — более общий аналог GenerateKey (которая жестко заточена под заказы:
// "orders/{id}/{stage}/..."). Эта версия ничего не знает про домен — только про category.
func generateCategoryKey(category string, format Format) (string, error) {
	suffix, err := randomHex(16)
	if err != nil {
		return "", fmt.Errorf("failed to generate media key: %w", err)
	}
	return fmt.Sprintf("%s/%s%s", category, suffix, format.Extension), nil
}
