package services

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"nexera_back/internal/logger"
	"nexera_back/internal/models"
	"nexera_back/internal/repository"
)

// ErrInvalidLead оборачивает ошибки валидации данных заявки (см. validateLead) —
// обработчик по нему отличает "клиент прислал некорректные данные" (400) от прочих ошибок.
var ErrInvalidLead = errors.New("invalid lead")

// leadPhonePattern повторяет CHECK (phone ~ '^\+[1-9][0-9]{6,14}$') из миграции
// 008_create_leads (формат E.164) — чтобы некорректный телефон был 400 от сервиса,
// а не 500 от Postgres.
var leadPhonePattern = regexp.MustCompile(`^\+[1-9][0-9]{6,14}$`)

// leadPhoneFormatting — символы, которыми люди обычно разделяют цифры телефона в форме
// ("+7 (999) 123-45-67"); перед проверкой формата они вырезаются.
var leadPhoneFormatting = strings.NewReplacer(" ", "", "-", "", "(", "", ")", "")

const (
	// defaultLeadSource соответствует DEFAULT 'contact_form' колонки leads.source.
	defaultLeadSource = "contact_form"

	// Ограничения длины соответствуют колонкам таблицы leads.
	maxLeadNameLength    = 100
	maxLeadEmailLength   = 255
	maxLeadMessageLength = 5000
	maxLeadSourceLength  = 100
)

// LeadService представляет сервис для работы с заявками клиентов.
type LeadService struct {
	repo repository.LeadRepository
	log  *logger.Logger
}

// NewLeadService создает новый сервис для работы с заявками.
func NewLeadService(repo repository.LeadRepository, log *logger.Logger) *LeadService {
	return &LeadService{
		repo: repo,
		log:  log,
	}
}

// ListLeadsFilter описывает запрос списка заявок из админ-панели: что можно
// отфильтровать и как отсортировать.
type ListLeadsFilter struct {
	ServiceID   *int
	Source      *string
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	Sort        repository.LeadSortField
	Order       repository.SortOrder
	Limit       int
	Offset      int
}

// List возвращает заявки по фильтру. Вызывается только из админ-панели.
func (svc *LeadService) List(filter ListLeadsFilter) ([]*models.Lead, error) {
	return svc.repo.List(repository.LeadFilter{
		ServiceID:   filter.ServiceID,
		Source:      filter.Source,
		CreatedFrom: filter.CreatedFrom,
		CreatedTo:   filter.CreatedTo,
		Sort:        filter.Sort,
		Order:       filter.Order,
		Limit:       filter.Limit,
		Offset:      filter.Offset,
	})
}

// GetByID возвращает заявку по ID. Вызывается только из админ-панели.
func (svc *LeadService) GetByID(id int) (*models.Lead, error) {
	return svc.repo.GetByID(id)
}

// Create создает заявку клиента. Вызывается из публичного API (см. handlers/public) —
// клиенту сайта авторизация не нужна.
//
// Перед валидацией данные нормализуются так, как их обычно присылает форма на сайте:
// пробелы по краям обрезаются, из телефона убираются пробелы/скобки/дефисы, пустые
// необязательные поля (email, message) считаются не заполненными, а пустой source
// заменяется на contact_form (как DEFAULT в БД).
func (svc *LeadService) Create(req models.CreateLeadRequest) (*models.Lead, error) {
	lead := &models.Lead{
		ServiceID: req.ServiceID,
		Name:      strings.TrimSpace(req.Name),
		Phone:     leadPhoneFormatting.Replace(strings.TrimSpace(req.Phone)),
		Email:     trimmedOrNil(req.Email),
		Message:   trimmedOrNil(req.Message),
		Source:    strings.TrimSpace(req.Source),
	}
	if lead.Source == "" {
		lead.Source = defaultLeadSource
	}

	if err := validateLead(lead); err != nil {
		return nil, err
	}

	if err := svc.repo.Create(lead); err != nil {
		return nil, err
	}

	svc.log.WithFields(map[string]interface{}{
		"lead_id": lead.ID,
		"source":  lead.Source,
	}).Info("created new lead")

	return lead, nil
}

// Delete удаляет заявку по ID. Вызывается только из админ-панели.
func (svc *LeadService) Delete(id int) error {
	if err := svc.repo.Delete(id); err != nil {
		return err
	}

	svc.log.WithField("lead_id", id).Info("deleted lead")
	return nil
}

// validateLead проверяет бизнес-правила, обязательные для заявки.
func validateLead(l *models.Lead) error {
	switch {
	case l.Name == "":
		return fmt.Errorf("%w: name is required", ErrInvalidLead)
	case len([]rune(l.Name)) > maxLeadNameLength:
		return fmt.Errorf("%w: name must be at most %d characters", ErrInvalidLead, maxLeadNameLength)
	case l.Phone == "":
		return fmt.Errorf("%w: phone is required", ErrInvalidLead)
	case !leadPhonePattern.MatchString(l.Phone):
		return fmt.Errorf("%w: phone must be in international format, e.g. +79991234567", ErrInvalidLead)
	case l.Email != nil && len([]rune(*l.Email)) > maxLeadEmailLength:
		return fmt.Errorf("%w: email must be at most %d characters", ErrInvalidLead, maxLeadEmailLength)
	case l.Email != nil && !isValidEmail(*l.Email):
		return fmt.Errorf("%w: invalid email", ErrInvalidLead)
	case l.Message != nil && len([]rune(*l.Message)) > maxLeadMessageLength:
		return fmt.Errorf("%w: message must be at most %d characters", ErrInvalidLead, maxLeadMessageLength)
	case len([]rune(l.Source)) > maxLeadSourceLength:
		return fmt.Errorf("%w: source must be at most %d characters", ErrInvalidLead, maxLeadSourceLength)
	case l.ServiceID != nil && *l.ServiceID <= 0:
		return fmt.Errorf("%w: invalid service_id", ErrInvalidLead)
	default:
		return nil
	}
}

// trimmedOrNil обрезает пробелы по краям; пустая после обрезки строка — nil
// (необязательное поле формы, которое клиент оставил пустым).
func trimmedOrNil(s *string) *string {
	if s == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*s)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
