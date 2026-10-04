package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"nexera_back/internal/database"
	"nexera_back/internal/logger"
	"nexera_back/internal/media"
	"nexera_back/internal/models"
)

// ErrOrderNotFound возвращается, когда заказ с указанным ID не существует
var ErrOrderNotFound = errors.New("order not found")

// OrderMediaService отвечает за привязку фото/видео к заказу
type OrderMediaService struct {
	db      *database.DB
	storage media.Storage
	log     *logger.Logger
}

// NewOrderMediaService создает новый сервис для работы с медиа заказов
func NewOrderMediaService(db *database.DB, storage media.Storage, log *logger.Logger) *OrderMediaService {
	return &OrderMediaService{
		db:      db,
		storage: storage,
		log:     log,
	}
}

// AddMedia сохраняет файл в хранилище и создает запись в order_media.
// Формат файла определяется по содержимому (magic bytes), а не по расширению
// или Content-Type, присланным клиентом.
func (svc *OrderMediaService) AddMedia(
	ctx context.Context,
	orderID int,
	stage string,
	fileSize int64,
	content io.Reader,
) (*models.OrderMedia, error) {
	var exists bool
	if err := svc.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM orders WHERE id = $1)", orderID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("failed to check order existence: %w", err)
	}
	if !exists {
		return nil, ErrOrderNotFound
	}

	header := make([]byte, media.SniffHeaderSize)
	n, err := io.ReadFull(content, header)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	header = header[:n]

	format, err := media.Sniff(header)
	if err != nil {
		return nil, err
	}

	key, err := media.GenerateKey(orderID, stage, format)
	if err != nil {
		return nil, err
	}

	// Часть файла уже прочитана для определения формата — дочитываем остаток
	fullContent := io.MultiReader(bytes.NewReader(header), content)

	if err := svc.storage.Save(ctx, key, fullContent, fileSize, format.MIME); err != nil {
		return nil, fmt.Errorf("failed to save media file: %w", err)
	}

	orderMedia := &models.OrderMedia{
		OrderID:    orderID,
		Type:       string(format.Kind),
		Stage:      stage,
		StorageKey: key,
		MimeType:   format.MIME,
		FileSize:   &fileSize,
		CreatedAt:  time.Now(),
	}

	query := `
		INSERT INTO order_media (order_id, type, stage, storage_key, mime_type, file_size, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	if err := svc.db.QueryRowContext(
		ctx,
		query,
		orderMedia.OrderID,
		orderMedia.Type,
		orderMedia.Stage,
		orderMedia.StorageKey,
		orderMedia.MimeType,
		orderMedia.FileSize,
		orderMedia.CreatedAt,
	).Scan(&orderMedia.ID); err != nil {
		// Файл уже сохранён в хранилище — подчищаем, чтобы не оставлять мусор при ошибке записи в БД
		if delErr := svc.storage.Delete(ctx, key); delErr != nil {
			svc.log.WithError(delErr).WithField("key", key).Error("failed to clean up orphaned media file")
		}
		return nil, fmt.Errorf("failed to save media record: %w", err)
	}

	orderMedia.URL = svc.storage.URL(key)

	svc.log.WithFields(map[string]interface{}{
		"order_id": orderID,
		"stage":    stage,
		"key":      key,
	}).Info("order media added")

	return orderMedia, nil
}

// GetMediaByOrderID возвращает фото/видео заказа с публичными ссылками на файлы
func (svc *OrderMediaService) GetMediaByOrderID(ctx context.Context, orderID int) ([]models.OrderMedia, error) {
	query := `
		SELECT id, order_id, type, stage, storage_key, mime_type, file_size, comment, created_at
		FROM order_media
		WHERE order_id = $1
		ORDER BY created_at
	`

	rows, err := svc.db.QueryContext(ctx, query, orderID)
	if err != nil {
		return nil, fmt.Errorf("error getting order media: %w", err)
	}
	defer rows.Close()

	var mediaList []models.OrderMedia
	for rows.Next() {
		var m models.OrderMedia
		if err := rows.Scan(
			&m.ID, &m.OrderID, &m.Type, &m.Stage, &m.StorageKey,
			&m.MimeType, &m.FileSize, &m.Comment, &m.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan order media: %w", err)
		}
		m.URL = svc.storage.URL(m.StorageKey)
		mediaList = append(mediaList, m)
	}

	return mediaList, nil
}
