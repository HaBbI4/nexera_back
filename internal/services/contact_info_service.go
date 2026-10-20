package services

import (
	"errors"
	"fmt"

	"nexera_back/internal/logger"
	"nexera_back/internal/models"
	"nexera_back/internal/repository"
)

// ErrInvalidContactInfo оборачивает ошибки валидации данных contact_info.
var ErrInvalidContactInfo = errors.New("invalid contact info")

const (
	maxContactInfoValueLength = 100
	maxContactInfoLabelLength = 100
)

// ContactInfoService представляет сервис для работы с контактными данными компании.
type ContactInfoService struct {
	repo repository.ContactInfoRepository
	log  *logger.Logger
}

// NewContactInfoService создает новый сервис для работы с contact_info.
func NewContactInfoService(repo repository.ContactInfoRepository, log *logger.Logger) *ContactInfoService {
	return &ContactInfoService{
		repo: repo,
		log:  log,
	}
}

// ListContactInfoFilter описывает публичный запрос списка contact_info.
type ListContactInfoFilter struct {
	Sort   repository.ContactInfoSortField
	Order  repository.SortOrder
	Limit  int
	Offset int
}

// ListPublic возвращает только опубликованные контактные записи.
func (svc *ContactInfoService) ListPublic(filter ListContactInfoFilter) ([]*models.ContactInfo, error) {
	published := true
	return svc.repo.List(repository.ContactInfoFilter{
		IsPublished: &published,
		Sort:        filter.Sort,
		Order:       filter.Order,
		Limit:       filter.Limit,
		Offset:      filter.Offset,
	})
}

// ListAdminContactInfoFilter описывает запрос списка contact_info из админ-панели.
type ListAdminContactInfoFilter struct {
	IsPublished *bool
	Sort        repository.ContactInfoSortField
	Order       repository.SortOrder
	Limit       int
	Offset      int
}

// List возвращает записи contact_info для админ-панели.
func (svc *ContactInfoService) List(filter ListAdminContactInfoFilter) ([]*models.ContactInfo, error) {
	return svc.repo.List(repository.ContactInfoFilter{
		IsPublished: filter.IsPublished,
		Sort:        filter.Sort,
		Order:       filter.Order,
		Limit:       filter.Limit,
		Offset:      filter.Offset,
	})
}

// GetByID возвращает запись contact_info по ID для админ-панели.
func (svc *ContactInfoService) GetByID(id int) (*models.ContactInfo, error) {
	return svc.repo.GetByID(id)
}

// GetPublicByID возвращает опубликованную запись contact_info по ID.
func (svc *ContactInfoService) GetPublicByID(id int) (*models.ContactInfo, error) {
	info, err := svc.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if !info.IsPublished {
		return nil, repository.ErrContactInfoNotFound
	}

	return info, nil
}

// Create создает новую запись contact_info.
func (svc *ContactInfoService) Create(req models.CreateContactInfoRequest) (*models.ContactInfo, error) {
	value := req.Value
	info := &models.ContactInfo{
		Type:        req.Type,
		Value:       &value,
		Label:       req.Label,
		Position:    req.Position,
		IsPublished: req.IsPublished,
	}

	if err := validateContactInfo(info); err != nil {
		return nil, err
	}

	if err := svc.repo.Create(info); err != nil {
		return nil, err
	}

	svc.log.WithField("contact_info_id", info.ID).Info("created new contact info")

	return info, nil
}

// Update частично обновляет запись contact_info.
func (svc *ContactInfoService) Update(id int, req models.UpdateContactInfoRequest) (*models.ContactInfo, error) {
	info, err := svc.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if req.Type != nil {
		info.Type = *req.Type
	}
	if req.Value != nil {
		info.Value = req.Value
	}
	if req.Label != nil {
		info.Label = req.Label
	}
	if req.Position != nil {
		info.Position = *req.Position
	}
	if req.IsPublished != nil {
		info.IsPublished = *req.IsPublished
	}

	if err := validateContactInfo(info); err != nil {
		return nil, err
	}

	if err := svc.repo.Update(info); err != nil {
		return nil, err
	}

	svc.log.WithField("contact_info_id", info.ID).Info("updated contact info")

	return info, nil
}

// Delete удаляет запись contact_info по ID.
func (svc *ContactInfoService) Delete(id int) error {
	if err := svc.repo.Delete(id); err != nil {
		return err
	}

	svc.log.WithField("contact_info_id", id).Info("deleted contact info")
	return nil
}

func validateContactInfo(c *models.ContactInfo) error {
	if !isValidContactInfoType(c.Type) {
		return fmt.Errorf("%w: type must be one of phone, email, telegram, whatsapp, vk", ErrInvalidContactInfo)
	}

	value := ""
	if c.Value != nil {
		value = *c.Value
	}

	switch {
	case value == "":
		return fmt.Errorf("%w: value is required", ErrInvalidContactInfo)
	case len([]rune(value)) > maxContactInfoValueLength:
		return fmt.Errorf("%w: value must be at most %d characters", ErrInvalidContactInfo, maxContactInfoValueLength)
	case c.Label != nil && len([]rune(*c.Label)) > maxContactInfoLabelLength:
		return fmt.Errorf("%w: label must be at most %d characters", ErrInvalidContactInfo, maxContactInfoLabelLength)
	case c.Position < 0:
		return fmt.Errorf("%w: position must be non-negative", ErrInvalidContactInfo)
	default:
		return nil
	}
}

func isValidContactInfoType(t models.ContactInfoType) bool {
	switch t {
	case models.ContactInfoTypePhone,
		models.ContactInfoTypeEmail,
		models.ContactInfoTypeTelegram,
		models.ContactInfoTypeWhatsapp,
		models.ContactInfoTypeVK:
		return true
	default:
		return false
	}
}
