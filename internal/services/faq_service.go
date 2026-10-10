package services

import (
	"errors"
	"fmt"

	"nexera_back/internal/logger"
	"nexera_back/internal/models"
	"nexera_back/internal/repository"
)

// ErrInvalidFAQ оборачивает ошибки валидации данных FAQ (см. validateFAQ) —
// обработчик по нему отличает "клиент прислал некорректные данные" (400) от прочих ошибок.
var ErrInvalidFAQ = errors.New("invalid faq")

const (
	// maxFAQQuestionLength / maxFAQAnswerLength соответствуют faq.question VARCHAR(255)
	// и faq.answer VARCHAR(5000) — чтобы слишком длинный текст был 400 от сервиса,
	// а не 500 от Postgres.
	maxFAQQuestionLength = 255
	maxFAQAnswerLength   = 5000
)

// FAQService представляет сервис для работы с вопросами-ответами (FAQ) сайта.
type FAQService struct {
	repo repository.FAQRepository
	log  *logger.Logger
}

// NewFAQService создает новый сервис для работы с FAQ.
func NewFAQService(repo repository.FAQRepository, log *logger.Logger) *FAQService {
	return &FAQService{
		repo: repo,
		log:  log,
	}
}

// ListFAQFilter описывает публичный запрос списка FAQ: как клиент сайта может
// отсортировать и пролистать список.
type ListFAQFilter struct {
	Sort   repository.FAQSortField
	Order  repository.SortOrder
	Limit  int
	Offset int
}

// ListPublic возвращает записи FAQ, которые можно показать на сайте: только опубликованные
// (IsPublished) — это бизнес-правило видимости, и клиент не может переопределить его через фильтр.
func (svc *FAQService) ListPublic(filter ListFAQFilter) ([]*models.FAQ, error) {
	published := true
	return svc.repo.List(repository.FAQFilter{
		IsPublished: &published,
		Sort:        filter.Sort,
		Order:       filter.Order,
		Limit:       filter.Limit,
		Offset:      filter.Offset,
	})
}

// ListAdminFAQFilter описывает запрос списка FAQ из админ-панели: в отличие от
// ListFAQFilter, видимость не форсируется — админ видит и скрытые записи, а по флагу
// is_published может отфильтровать сам.
type ListAdminFAQFilter struct {
	IsPublished *bool
	Sort        repository.FAQSortField
	Order       repository.SortOrder
	Limit       int
	Offset      int
}

// List возвращает записи FAQ для админ-панели вне зависимости от публикации.
func (svc *FAQService) List(filter ListAdminFAQFilter) ([]*models.FAQ, error) {
	return svc.repo.List(repository.FAQFilter{
		IsPublished: filter.IsPublished,
		Sort:        filter.Sort,
		Order:       filter.Order,
		Limit:       filter.Limit,
		Offset:      filter.Offset,
	})
}

// GetByID возвращает запись FAQ по ID вне зависимости от публикации — для админ-панели.
func (svc *FAQService) GetByID(id int) (*models.FAQ, error) {
	return svc.repo.GetByID(id)
}

// GetPublicByID возвращает запись FAQ по ID, если она опубликована. Неопубликованная
// запись считается несуществующей — публичный API не должен подтверждать, что скрытый
// контент вообще есть.
func (svc *FAQService) GetPublicByID(id int) (*models.FAQ, error) {
	faq, err := svc.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if !faq.IsPublished {
		return nil, repository.ErrFAQNotFound
	}

	return faq, nil
}

// Create создает новую запись FAQ. Вызывается только из админ-панели (см. handlers/admin) —
// публичные эндпоинты только читают.
func (svc *FAQService) Create(req models.CreateFAQRequest) (*models.FAQ, error) {
	faq := &models.FAQ{
		Question:    req.Question,
		Answer:      req.Answer,
		Position:    req.Position,
		IsPublished: req.IsPublished,
	}

	if err := validateFAQ(faq); err != nil {
		return nil, err
	}

	if err := svc.repo.Create(faq); err != nil {
		return nil, err
	}

	svc.log.WithField("faq_id", faq.ID).Info("created new faq")

	return faq, nil
}

// Update частично обновляет запись FAQ: подгружает текущее состояние, накладывает на него
// только присланные в запросе поля (nil в запросе — поле остается прежним) и сохраняет
// итог целиком — тот же merge-then-validate-then-persist, что и BannerService.Update.
func (svc *FAQService) Update(id int, req models.UpdateFAQRequest) (*models.FAQ, error) {
	faq, err := svc.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if req.Question != nil {
		faq.Question = *req.Question
	}
	if req.Answer != nil {
		faq.Answer = *req.Answer
	}
	if req.Position != nil {
		faq.Position = *req.Position
	}
	if req.IsPublished != nil {
		faq.IsPublished = *req.IsPublished
	}

	if err := validateFAQ(faq); err != nil {
		return nil, err
	}

	if err := svc.repo.Update(faq); err != nil {
		return nil, err
	}

	svc.log.WithField("faq_id", faq.ID).Info("updated faq")

	return faq, nil
}

// Delete удаляет запись FAQ по ID.
func (svc *FAQService) Delete(id int) error {
	if err := svc.repo.Delete(id); err != nil {
		return err
	}

	svc.log.WithField("faq_id", id).Info("deleted faq")
	return nil
}

// validateFAQ проверяет бизнес-правила, обязательные для записи FAQ независимо от того,
// получила ли она текущий вид целиком при Create или слиянием при Update.
func validateFAQ(f *models.FAQ) error {
	switch {
	case f.Question == "":
		return fmt.Errorf("%w: question is required", ErrInvalidFAQ)
	case len([]rune(f.Question)) > maxFAQQuestionLength:
		return fmt.Errorf("%w: question must be at most %d characters", ErrInvalidFAQ, maxFAQQuestionLength)
	case f.Answer == "":
		return fmt.Errorf("%w: answer is required", ErrInvalidFAQ)
	case len([]rune(f.Answer)) > maxFAQAnswerLength:
		return fmt.Errorf("%w: answer must be at most %d characters", ErrInvalidFAQ, maxFAQAnswerLength)
	case f.Position < 0:
		return fmt.Errorf("%w: position must be non-negative", ErrInvalidFAQ)
	default:
		return nil
	}
}
