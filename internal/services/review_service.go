package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"nexera_back/internal/logger"
	"nexera_back/internal/models"
	"nexera_back/internal/repository"
)

// ErrInvalidReview оборачивает ошибки валидации данных отзыва (см. validateReview) —
// обработчик по нему отличает "клиент прислал некорректные данные" (400) от прочих ошибок.
var ErrInvalidReview = errors.New("invalid review")

const (
	// Ограничения соответствуют колонкам и CHECK таблицы reviews. text в БД — TEXT без
	// ограничения, поэтому лимит задает сервис (как у faq.answer), чтобы публичный
	// эндпоинт нельзя было использовать для записи мегабайтов текста.
	maxReviewNameLength  = 255
	maxReviewEmailLength = 100
	maxReviewTextLength  = 5000
	minReviewRating      = 1
	maxReviewRating      = 5
)

// ReviewService представляет сервис для работы с отзывами клиентов.
type ReviewService struct {
	repo repository.ReviewRepository
	// duplicateWindow — в течение какого времени повторный отзыв с тем же телефоном
	// отклоняется (ErrDuplicateSubmission); 0 — проверка отключена.
	duplicateWindow time.Duration
	log             *logger.Logger
}

// NewReviewService создает новый сервис для работы с отзывами.
func NewReviewService(repo repository.ReviewRepository, duplicateWindow time.Duration, log *logger.Logger) *ReviewService {
	return &ReviewService{
		repo:            repo,
		duplicateWindow: duplicateWindow,
		log:             log,
	}
}

// ListPublicReviewsFilter описывает публичный запрос списка отзывов: что клиент сайта
// может отфильтровать и как отсортировать.
type ListPublicReviewsFilter struct {
	RatingMin *int
	Sort      repository.ReviewSortField
	Order     repository.SortOrder
	Limit     int
	Offset    int
}

// ListPublic возвращает отзывы, которые можно показать на сайте: только одобренные
// (IsApproved) — это бизнес-правило видимости, и клиент не может переопределить его через
// фильтр. Отзывы отдаются как PublicReview — без телефона и email клиента.
func (svc *ReviewService) ListPublic(filter ListPublicReviewsFilter) ([]*models.PublicReview, error) {
	approved := true
	reviews, err := svc.repo.List(repository.ReviewFilter{
		IsApproved: &approved,
		RatingMin:  filter.RatingMin,
		Sort:       filter.Sort,
		Order:      filter.Order,
		Limit:      filter.Limit,
		Offset:     filter.Offset,
	})
	if err != nil {
		return nil, err
	}

	result := make([]*models.PublicReview, 0, len(reviews))
	for _, review := range reviews {
		result = append(result, review.ToPublic())
	}
	return result, nil
}

// GetPublicSummary возвращает количество и средний рейтинг одобренных отзывов — для блока
// "4.8 из 5, 37 отзывов" на сайте. Неодобренные в сводку не входят по тому же правилу
// видимости, что в ListPublic.
func (svc *ReviewService) GetPublicSummary() (*models.ReviewSummary, error) {
	approved := true
	return svc.repo.GetSummary(&approved)
}

// ListAdminReviewsFilter описывает запрос списка отзывов из админ-панели: в отличие от
// ListPublicReviewsFilter, одобрение не форсируется — админ видит и неодобренные, а по
// флагу is_approved может отфильтровать сам (например, очередь модерации).
type ListAdminReviewsFilter struct {
	IsApproved  *bool
	RatingMin   *int
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	Sort        repository.ReviewSortField
	Order       repository.SortOrder
	Limit       int
	Offset      int
}

// List возвращает отзывы для админ-панели вне зависимости от одобрения, вместе с
// контактами клиента.
func (svc *ReviewService) List(filter ListAdminReviewsFilter) ([]*models.Review, error) {
	return svc.repo.List(repository.ReviewFilter{
		IsApproved:  filter.IsApproved,
		RatingMin:   filter.RatingMin,
		CreatedFrom: filter.CreatedFrom,
		CreatedTo:   filter.CreatedTo,
		Sort:        filter.Sort,
		Order:       filter.Order,
		Limit:       filter.Limit,
		Offset:      filter.Offset,
	})
}

// GetByID возвращает отзыв по ID вне зависимости от одобрения — для админ-панели.
func (svc *ReviewService) GetByID(id int) (*models.Review, error) {
	return svc.repo.GetByID(id)
}

// Create создает отзыв клиента. Вызывается из публичного API (см. handlers/public) —
// клиенту сайта авторизация не нужна. Новый отзыв всегда неодобрен: в запросе клиента
// поля is_approved нет, а репозиторий берет DEFAULT FALSE из БД.
//
// Данные нормализуются так же, как у заявок (LeadService.Create): пробелы по краям
// обрезаются, из телефона убираются пробелы/скобки/дефисы, пустой email — не заполнен.
//
// Защита от спама — как у заявок: заполненное honeypot-поле — ErrSpamDetected, отзыв
// с тем же телефоном в пределах duplicateWindow — ErrDuplicateSubmission.
func (svc *ReviewService) Create(req models.CreateReviewRequest) (*models.Review, error) {
	if req.Website != "" {
		svc.log.Warn("review rejected: honeypot field filled")
		return nil, ErrSpamDetected
	}

	review := &models.Review{
		Name:   strings.TrimSpace(req.Name),
		Phone:  phoneFormatting.Replace(strings.TrimSpace(req.Phone)),
		Email:  trimmedOrNil(req.Email),
		Rating: req.Rating,
		Text:   strings.TrimSpace(req.Text),
	}

	if err := validateReview(review); err != nil {
		return nil, err
	}

	if svc.duplicateWindow > 0 {
		exists, err := svc.repo.ExistsByPhoneSince(review.Phone, time.Now().Add(-svc.duplicateWindow))
		if err != nil {
			return nil, err
		}
		if exists {
			svc.log.Warn("review rejected: duplicate phone within window")
			return nil, ErrDuplicateSubmission
		}
	}

	if err := svc.repo.Create(review); err != nil {
		return nil, err
	}

	svc.log.WithFields(map[string]interface{}{
		"review_id": review.ID,
		"rating":    review.Rating,
	}).Info("created new review")

	return review, nil
}

// Approve одобряет отзыв — после этого он виден на сайте. Повторное одобрение не ошибка.
func (svc *ReviewService) Approve(id int) (*models.Review, error) {
	return svc.setApproved(id, true)
}

// Unapprove снимает одобрение — отзыв пропадает с сайта, но остается в админке
// (например, если одобрили по ошибке, не дочитав). Для неодобренного отзыва — не ошибка.
func (svc *ReviewService) Unapprove(id int) (*models.Review, error) {
	return svc.setApproved(id, false)
}

func (svc *ReviewService) setApproved(id int, approved bool) (*models.Review, error) {
	review, err := svc.repo.SetApproved(id, approved)
	if err != nil {
		return nil, err
	}

	svc.log.WithFields(map[string]interface{}{
		"review_id":   id,
		"is_approved": approved,
	}).Info("changed review approval")

	return review, nil
}

// Delete удаляет отзыв по ID (мягко — см. ReviewRepository.Delete).
func (svc *ReviewService) Delete(id int) error {
	if err := svc.repo.Delete(id); err != nil {
		return err
	}

	svc.log.WithField("review_id", id).Info("deleted review")
	return nil
}

// validateReview проверяет бизнес-правила, обязательные для отзыва.
func validateReview(r *models.Review) error {
	switch {
	case r.Name == "":
		return fmt.Errorf("%w: name is required", ErrInvalidReview)
	case len([]rune(r.Name)) > maxReviewNameLength:
		return fmt.Errorf("%w: name must be at most %d characters", ErrInvalidReview, maxReviewNameLength)
	case r.Phone == "":
		return fmt.Errorf("%w: phone is required", ErrInvalidReview)
	case !phonePattern.MatchString(r.Phone):
		return fmt.Errorf("%w: phone must be in international format, e.g. +79991234567", ErrInvalidReview)
	case r.Email != nil && len([]rune(*r.Email)) > maxReviewEmailLength:
		return fmt.Errorf("%w: email must be at most %d characters", ErrInvalidReview, maxReviewEmailLength)
	case r.Email != nil && !isValidEmail(*r.Email):
		return fmt.Errorf("%w: invalid email", ErrInvalidReview)
	case r.Rating < minReviewRating || r.Rating > maxReviewRating:
		return fmt.Errorf("%w: rating must be between %d and %d", ErrInvalidReview, minReviewRating, maxReviewRating)
	case r.Text == "":
		return fmt.Errorf("%w: text is required", ErrInvalidReview)
	case len([]rune(r.Text)) > maxReviewTextLength:
		return fmt.Errorf("%w: text must be at most %d characters", ErrInvalidReview, maxReviewTextLength)
	default:
		return nil
	}
}
