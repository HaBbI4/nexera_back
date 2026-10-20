package services

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"nexera_back/internal/logger"
	"nexera_back/internal/models"
	"nexera_back/internal/repository"
)

// ErrInvalidContactInfo оборачивает ошибки валидации данных contact_info
// (см. validateContactInfo) — обработчик по нему отличает "клиент прислал
// некорректные данные" (400) от прочих ошибок.
var ErrInvalidContactInfo = errors.New("invalid contact info")

const (
	// maxContactInfoValueLength / maxContactInfoLabelLength соответствуют
	// contact_info.value VARCHAR(100) и contact_info.label VARCHAR(100).
	maxContactInfoValueLength = 100
	maxContactInfoLabelLength = 100
)

// telegramHandlePattern — @username или username (5–32 символа, латиница/цифры/_).
var telegramHandlePattern = regexp.MustCompile(`^@?[A-Za-z0-9_]{5,32}$`)

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

// ListPublic возвращает только опубликованные контактные записи —
// бизнес-правило видимости, клиент не может его обойти.
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

// List возвращает записи contact_info для админ-панели вне зависимости от публикации.
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
// Неопубликованная запись считается несуществующей.
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

// Create создает новую запись contact_info. Вызывается только из админ-панели.
func (svc *ContactInfoService) Create(req models.CreateContactInfoRequest) (*models.ContactInfo, error) {
	value := strings.TrimSpace(req.Value)
	info := &models.ContactInfo{
		Type:        req.Type,
		Value:       &value,
		Label:       normalizeOptionalLabel(req.Label),
		Position:    req.Position,
		IsPublished: req.IsPublished,
	}
	normalizeContactInfoValue(info)

	if err := validateContactInfo(info); err != nil {
		return nil, err
	}

	if err := svc.repo.Create(info); err != nil {
		return nil, err
	}

	svc.log.WithField("contact_info_id", info.ID).Info("created new contact info")

	return info, nil
}

// Update частично обновляет запись contact_info: merge-then-validate-then-persist,
// как у FAQ/Banner. Пустой label ("") нормализуется в NULL.
func (svc *ContactInfoService) Update(id int, req models.UpdateContactInfoRequest) (*models.ContactInfo, error) {
	info, err := svc.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if req.Type != nil {
		info.Type = *req.Type
	}
	if req.Value != nil {
		trimmed := strings.TrimSpace(*req.Value)
		info.Value = &trimmed
	}
	if req.Label != nil {
		info.Label = normalizeOptionalLabel(req.Label)
	}
	if req.Position != nil {
		info.Position = *req.Position
	}
	if req.IsPublished != nil {
		info.IsPublished = *req.IsPublished
	}

	normalizeContactInfoValue(info)

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

// normalizeOptionalLabel превращает пустую строку в nil, чтобы в JSON не уходило "label": ""
// и в БД писался NULL, а не ”.
func normalizeOptionalLabel(label *string) *string {
	if label == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*label)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// normalizeContactInfoValue приводит value к каноническому виду перед валидацией
// (для телефона — убирает пробелы/скобки/дефисы, как у лидов).
func normalizeContactInfoValue(c *models.ContactInfo) {
	if c.Value == nil {
		return
	}
	switch c.Type {
	case models.ContactInfoTypePhone, models.ContactInfoTypeWhatsapp:
		// WhatsApp часто хранят как E.164; если это URL — не трогаем.
		if strings.HasPrefix(*c.Value, "http://") || strings.HasPrefix(*c.Value, "https://") {
			return
		}
		normalized := phoneFormatting.Replace(strings.TrimSpace(*c.Value))
		c.Value = &normalized
	}
}

// validateContactInfo проверяет бизнес-правила записи contact_info.
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
	}

	if err := validateContactInfoValueByType(c.Type, value); err != nil {
		return err
	}
	return nil
}

func validateContactInfoValueByType(t models.ContactInfoType, value string) error {
	switch t {
	case models.ContactInfoTypeEmail:
		if !isValidEmail(value) {
			return fmt.Errorf("%w: invalid email", ErrInvalidContactInfo)
		}
	case models.ContactInfoTypePhone:
		if !phonePattern.MatchString(value) {
			return fmt.Errorf("%w: phone must be E.164 (+79991234567)", ErrInvalidContactInfo)
		}
	case models.ContactInfoTypeTelegram:
		if !isValidTelegramValue(value) {
			return fmt.Errorf("%w: invalid telegram handle or URL", ErrInvalidContactInfo)
		}
	case models.ContactInfoTypeWhatsapp:
		if !isValidWhatsAppValue(value) {
			return fmt.Errorf("%w: whatsapp must be E.164 phone or wa.me URL", ErrInvalidContactInfo)
		}
	case models.ContactInfoTypeVK:
		if !isValidVKValue(value) {
			return fmt.Errorf("%w: invalid vk username or URL", ErrInvalidContactInfo)
		}
	}
	return nil
}

func isValidTelegramValue(value string) bool {
	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "https://t.me/") || strings.HasPrefix(lower, "http://t.me/") || strings.HasPrefix(lower, "t.me/") {
		u := value
		if strings.HasPrefix(lower, "t.me/") {
			u = "https://" + value
		}
		parsed, err := url.Parse(u)
		if err != nil || parsed.Host != "t.me" && !strings.EqualFold(parsed.Host, "t.me") {
			// host check: Parse may keep case
			if err != nil {
				return false
			}
			if !strings.EqualFold(parsed.Hostname(), "t.me") {
				return false
			}
		}
		path := strings.Trim(parsed.Path, "/")
		return telegramHandlePattern.MatchString(path)
	}
	return telegramHandlePattern.MatchString(value)
}

func isValidWhatsAppValue(value string) bool {
	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "https://wa.me/") || strings.HasPrefix(lower, "http://wa.me/") || strings.HasPrefix(lower, "wa.me/") {
		u := value
		if strings.HasPrefix(lower, "wa.me/") {
			u = "https://" + value
		}
		parsed, err := url.Parse(u)
		if err != nil || !strings.EqualFold(parsed.Hostname(), "wa.me") {
			return false
		}
		digits := strings.Trim(parsed.Path, "/")
		return regexp.MustCompile(`^[1-9][0-9]{6,14}$`).MatchString(digits)
	}
	return phonePattern.MatchString(value)
}

func isValidVKValue(value string) bool {
	lower := strings.ToLower(value)
	if strings.Contains(lower, "vk.com/") {
		u := value
		if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
			u = "https://" + value
		}
		parsed, err := url.Parse(u)
		if err != nil {
			return false
		}
		host := strings.ToLower(parsed.Hostname())
		if host != "vk.com" && host != "www.vk.com" && host != "m.vk.com" {
			return false
		}
		return strings.Trim(parsed.Path, "/") != ""
	}
	// screen name
	return regexp.MustCompile(`^[A-Za-z0-9_.]{2,50}$`).MatchString(value)
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
