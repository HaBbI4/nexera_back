package services

import (
	"errors"
	"fmt"

	"nexera_back/internal/logger"
	"nexera_back/internal/models"
	"nexera_back/internal/repository"
)

// ErrInvalidService оборачивает ошибки валидации данных услуги (см. validateService) —
// обработчик по нему отличает "клиент прислал некорректные данные" (400) от прочих ошибок.
var ErrInvalidService = errors.New("invalid service")

const (
	// maxServiceNameLength / maxServiceDescriptionLength соответствуют services.name
	// VARCHAR(100) и services.description VARCHAR(5000).
	maxServiceNameLength        = 100
	maxServiceDescriptionLength = 5000
)

// ServiceService представляет сервис для работы с услугами компании, отображаемыми на сайте.
type ServiceService struct {
	repo repository.ServiceRepository
	log  *logger.Logger
}

// NewServiceService создает новый сервис для работы с услугами.
func NewServiceService(repo repository.ServiceRepository, log *logger.Logger) *ServiceService {
	return &ServiceService{
		repo: repo,
		log:  log,
	}
}

// ListServicesFilter описывает публичный запрос списка услуг: что клиент сайта
// может отфильтровать и как отсортировать.
type ListServicesFilter struct {
	PriceFrom *int
	PriceTo   *int
	Sort      repository.ServiceSortField
	Order     repository.SortOrder
	Limit     int
	Offset    int
}

// ListPublic возвращает услуги, которые можно показать на сайте: только опубликованные
// (IsPublished) — это бизнес-правило видимости, и клиент не может переопределить его через фильтр.
func (svc *ServiceService) ListPublic(filter ListServicesFilter) ([]*models.Service, error) {
	published := true
	return svc.repo.List(repository.ServiceFilter{
		IsPublished: &published,
		PriceFrom:   filter.PriceFrom,
		PriceTo:     filter.PriceTo,
		Sort:        filter.Sort,
		Order:       filter.Order,
		Limit:       filter.Limit,
		Offset:      filter.Offset,
	})
}

// ListAdminServicesFilter описывает запрос списка услуг из админ-панели: в отличие от
// ListServicesFilter, видимость не форсируется — админ видит и скрытые услуги, а по флагу
// is_published может отфильтровать сам.
type ListAdminServicesFilter struct {
	IsPublished *bool
	PriceFrom   *int
	PriceTo     *int
	Sort        repository.ServiceSortField
	Order       repository.SortOrder
	Limit       int
	Offset      int
}

// List возвращает услуги для админ-панели вне зависимости от публикации.
func (svc *ServiceService) List(filter ListAdminServicesFilter) ([]*models.Service, error) {
	return svc.repo.List(repository.ServiceFilter{
		IsPublished: filter.IsPublished,
		PriceFrom:   filter.PriceFrom,
		PriceTo:     filter.PriceTo,
		Sort:        filter.Sort,
		Order:       filter.Order,
		Limit:       filter.Limit,
		Offset:      filter.Offset,
	})
}

// GetByID возвращает услугу по ID вне зависимости от публикации — для админ-панели.
func (svc *ServiceService) GetByID(id int) (*models.Service, error) {
	return svc.repo.GetByID(id)
}

// GetPublicByID возвращает услугу по ID, если она опубликована. Неопубликованная услуга
// считается несуществующей — публичный API не должен подтверждать, что скрытый контент
// вообще есть.
func (svc *ServiceService) GetPublicByID(id int) (*models.Service, error) {
	service, err := svc.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if !service.IsPublished {
		return nil, repository.ErrServiceNotFound
	}

	return service, nil
}

// Create создает новую услугу. Вызывается только из админ-панели (см. handlers/admin) —
// публичные эндпоинты только читают.
func (svc *ServiceService) Create(req models.CreateServiceRequest) (*models.Service, error) {
	service := &models.Service{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Position:    req.Position,
		IsPublished: req.IsPublished,
	}

	if err := validateService(service); err != nil {
		return nil, err
	}

	if err := svc.repo.Create(service); err != nil {
		return nil, err
	}

	svc.log.WithField("service_id", service.ID).Info("created new service")

	return service, nil
}

// Update частично обновляет услугу: подгружает текущее состояние, накладывает на него
// только присланные в запросе поля (nil в запросе — поле остается прежним) и сохраняет
// итог целиком — тот же merge-then-validate-then-persist, что и BannerService.Update.
func (svc *ServiceService) Update(id int, req models.UpdateServiceRequest) (*models.Service, error) {
	service, err := svc.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		service.Name = *req.Name
	}
	if req.Description != nil {
		service.Description = req.Description
	}
	if req.Price != nil {
		service.Price = req.Price
	}
	if req.Position != nil {
		service.Position = *req.Position
	}
	if req.IsPublished != nil {
		service.IsPublished = *req.IsPublished
	}

	if err := validateService(service); err != nil {
		return nil, err
	}

	if err := svc.repo.Update(service); err != nil {
		return nil, err
	}

	svc.log.WithField("service_id", service.ID).Info("updated service")

	return service, nil
}

// Delete удаляет услугу по ID. Заявки на нее остаются, но теряют ссылку на услугу
// (leads.service_id -> services(id) ON DELETE SET NULL) — это ограничение БД, не сервиса.
func (svc *ServiceService) Delete(id int) error {
	if err := svc.repo.Delete(id); err != nil {
		return err
	}

	svc.log.WithField("service_id", id).Info("deleted service")
	return nil
}

// validateService проверяет бизнес-правила, обязательные для услуги независимо от того,
// получила ли она текущий вид целиком при Create или слиянием при Update.
func validateService(s *models.Service) error {
	switch {
	case s.Name == "":
		return fmt.Errorf("%w: name is required", ErrInvalidService)
	case len([]rune(s.Name)) > maxServiceNameLength:
		return fmt.Errorf("%w: name must be at most %d characters", ErrInvalidService, maxServiceNameLength)
	case s.Description != nil && len([]rune(*s.Description)) > maxServiceDescriptionLength:
		return fmt.Errorf("%w: description must be at most %d characters", ErrInvalidService, maxServiceDescriptionLength)
	case s.Price != nil && *s.Price < 0:
		return fmt.Errorf("%w: price must be non-negative", ErrInvalidService)
	case s.Position < 0:
		return fmt.Errorf("%w: position must be non-negative", ErrInvalidService)
	default:
		return nil
	}
}
