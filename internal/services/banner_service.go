package services

import (
	"errors"
	"fmt"
	"time"

	"nexera_back/internal/logger"
	"nexera_back/internal/models"
	"nexera_back/internal/repository"
)

// ErrInvalidBanner оборачивает ошибки валидации данных баннера (см. validateBanner) —
// обработчик по нему отличает "клиент прислал некорректные данные" (400) от прочих ошибок.
var ErrInvalidBanner = errors.New("invalid banner")

// BannerService представляет сервис для работы с баннерами сайта.
type BannerService struct {
	repo repository.BannerRepository
	log  *logger.Logger
}

// NewBannerService создает новый сервис для работы с баннерами
func NewBannerService(repo repository.BannerRepository, log *logger.Logger) *BannerService {
	return &BannerService{
		repo: repo,
		log:  log,
	}
}

// ListBannersFilter описывает публичный запрос списка баннеров: что клиент сайта
// может отфильтровать и как отсортировать.
type ListBannersFilter struct {
	PageID *int
	Sort   repository.BannerSortField
	Order  repository.SortOrder
	Limit  int
	Offset int
}

// ListPublic возвращает баннеры, которые можно показать на сайте прямо сейчас.
// Отдает только активные (IsActive) баннеры, попадающие в окно показа (StartsAt/EndsAt) —
// это бизнес-правило видимости, и клиент не может переопределить его через фильтр.
func (svc *BannerService) ListPublic(filter ListBannersFilter) ([]*models.Banner, error) {
	return svc.repo.List(repository.BannerFilter{
		PageID:      filter.PageID,
		OnlyVisible: true,
		ActiveAt:    time.Now(),
		Sort:        filter.Sort,
		Order:       filter.Order,
		Limit:       filter.Limit,
		Offset:      filter.Offset,
	})
}

// GetPublicByID возвращает баннер по ID, если он видим прямо сейчас (см. ListPublic).
// Отключенный или находящийся вне окна показа баннер считается несуществующим —
// публичный API не должен подтверждать, что скрытый контент вообще есть.
func (svc *BannerService) GetPublicByID(id int) (*models.Banner, error) {
	banner, err := svc.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if !isBannerVisible(banner, time.Now()) {
		return nil, repository.ErrBannerNotFound
	}

	return banner, nil
}

func isBannerVisible(b *models.Banner, now time.Time) bool {
	if !b.IsActive {
		return false
	}
	if b.StartsAt != nil && b.StartsAt.After(now) {
		return false
	}
	if b.EndsAt != nil && b.EndsAt.Before(now) {
		return false
	}
	return true
}

// Create создает новый баннер. Вызывается только из админ-панели (см. handlers/admin) —
// публичные эндпоинты только читают.
func (svc *BannerService) Create(req models.CreateBannerRequest) (*models.Banner, error) {
	banner := &models.Banner{
		PageID:      req.PageID,
		Title:       req.Title,
		SubTitle:    req.SubTitle,
		Description: req.Description,
		LinkTitle:   req.LinkTitle,
		LinkURL:     req.LinkURL,
		ImageURL:    req.ImageURL,
		Position:    req.Position,
		StartsAt:    req.StartsAt,
		EndsAt:      req.EndsAt,
		IsActive:    req.IsActive,
	}

	if err := validateBanner(banner); err != nil {
		return nil, err
	}

	if err := svc.repo.Create(banner); err != nil {
		return nil, err
	}

	svc.log.WithFields(map[string]interface{}{
		"banner_id": banner.ID,
		"page_id":   banner.PageID,
	}).Info("created new banner")

	return banner, nil
}

// Update частично обновляет баннер: подгружает текущее состояние, накладывает на него
// только присланные в запросе поля (nil в запросе — поле остается прежним) и сохраняет
// итог целиком. Благодаря этому validateBanner всегда проверяет полную, согласованную
// сущность, а не "кусок" изменений — например, ends_at сверяется со starts_at, даже если
// в этом конкретном PATCH-запросе менялось только одно из двух полей.
func (svc *BannerService) Update(id int, req models.UpdateBannerRequest) (*models.Banner, error) {
	banner, err := svc.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if req.PageID != nil {
		banner.PageID = *req.PageID
	}
	if req.Title != nil {
		banner.Title = *req.Title
	}
	if req.SubTitle != nil {
		banner.SubTitle = req.SubTitle
	}
	if req.Description != nil {
		banner.Description = req.Description
	}
	if req.LinkTitle != nil {
		banner.LinkTitle = req.LinkTitle
	}
	if req.LinkURL != nil {
		banner.LinkURL = req.LinkURL
	}
	if req.ImageURL != nil {
		banner.ImageURL = *req.ImageURL
	}
	if req.Position != nil {
		banner.Position = *req.Position
	}
	if req.StartsAt != nil {
		banner.StartsAt = req.StartsAt
	}
	if req.EndsAt != nil {
		banner.EndsAt = req.EndsAt
	}
	if req.IsActive != nil {
		banner.IsActive = *req.IsActive
	}

	if err := validateBanner(banner); err != nil {
		return nil, err
	}

	if err := svc.repo.Update(banner); err != nil {
		return nil, err
	}

	svc.log.WithField("banner_id", banner.ID).Info("updated banner")

	return banner, nil
}

// Delete удаляет баннер по ID.
func (svc *BannerService) Delete(id int) error {
	if err := svc.repo.Delete(id); err != nil {
		return err
	}

	svc.log.WithField("banner_id", id).Info("deleted banner")
	return nil
}

// validateBanner проверяет бизнес-правила, обязательные для баннера независимо от того,
// как он получил текущий вид — целиком при Create или слиянием текущей записи с PATCH-телом
// при Update. Это тот же разделение ответственности, что и isBannerVisible: правило нельзя
// обойти, выбрав другой способ попасть в сервис.
func validateBanner(b *models.Banner) error {
	switch {
	case b.Title == "":
		return fmt.Errorf("%w: title is required", ErrInvalidBanner)
	case b.ImageURL == "":
		return fmt.Errorf("%w: image_url is required", ErrInvalidBanner)
	case b.PageID <= 0:
		return fmt.Errorf("%w: page_id is required", ErrInvalidBanner)
	case b.Position < 0:
		return fmt.Errorf("%w: position must be non-negative", ErrInvalidBanner)
	case b.StartsAt != nil && b.EndsAt != nil && b.EndsAt.Before(*b.StartsAt):
		return fmt.Errorf("%w: ends_at must be after starts_at", ErrInvalidBanner)
	default:
		return nil
	}
}
