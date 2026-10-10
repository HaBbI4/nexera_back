package services

import (
	"errors"
	"fmt"

	"nexera_back/internal/logger"
	"nexera_back/internal/models"
	"nexera_back/internal/repository"
)

// ErrInvalidTelegramUser оборачивает ошибки валидации пользователя бота (см. validateTelegramUser).
var ErrInvalidTelegramUser = errors.New("invalid telegram user")

// maxTelegramUserNameLength соответствует telegram_users.name VARCHAR(255).
const maxTelegramUserNameLength = 255

// TelegramUserService представляет сервис для работы с пользователями telegram-бота:
// управление из админ-панели и проверка доступа/рассылка в самом боте.
type TelegramUserService struct {
	repo repository.TelegramUserRepository
	log  *logger.Logger
}

// NewTelegramUserService создает новый сервис для работы с пользователями telegram-бота.
func NewTelegramUserService(repo repository.TelegramUserRepository, log *logger.Logger) *TelegramUserService {
	return &TelegramUserService{
		repo: repo,
		log:  log,
	}
}

// ListTelegramUsersFilter описывает запрос списка пользователей бота из админ-панели.
type ListTelegramUsersFilter struct {
	IsActive *bool
	Sort     repository.TelegramUserSortField
	Order    repository.SortOrder
	Limit    int
	Offset   int
}

// List возвращает пользователей бота для админ-панели, включая отключенных.
func (svc *TelegramUserService) List(filter ListTelegramUsersFilter) ([]*models.TelegramUser, error) {
	return svc.repo.List(repository.TelegramUserFilter{
		IsActive: filter.IsActive,
		Sort:     filter.Sort,
		Order:    filter.Order,
		Limit:    filter.Limit,
		Offset:   filter.Offset,
	})
}

// GetActiveUsers возвращает всех активных пользователей бота (получателей уведомлений).
func (svc *TelegramUserService) GetActiveUsers() ([]*models.TelegramUser, error) {
	active := true
	return svc.repo.List(repository.TelegramUserFilter{IsActive: &active})
}

// GetByID возвращает пользователя бота по ID — для админ-панели.
func (svc *TelegramUserService) GetByID(id int) (*models.TelegramUser, error) {
	return svc.repo.GetByID(id)
}

// GetByChatID возвращает пользователя бота по chat_id. Используется middleware авторизации
// бота: доступ разрешен, только если запись найдена и активна.
func (svc *TelegramUserService) GetByChatID(chatID int64) (*models.TelegramUser, error) {
	return svc.repo.GetByChatID(chatID)
}

// Create добавляет пользователя бота. Без is_active в запросе пользователь создается
// активным — как DEFAULT TRUE в таблице.
func (svc *TelegramUserService) Create(req models.CreateTelegramUserRequest) (*models.TelegramUser, error) {
	user := &models.TelegramUser{
		ChatID:   req.ChatID,
		Name:     req.Name,
		IsActive: true,
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}

	if err := validateTelegramUser(user); err != nil {
		return nil, err
	}

	if err := svc.repo.Create(user); err != nil {
		return nil, err
	}

	svc.log.WithField("telegram_user_id", user.ID).Info("created new telegram user")

	return user, nil
}

// Update частично обновляет пользователя бота: merge-then-validate-then-persist,
// как BannerService.Update.
func (svc *TelegramUserService) Update(id int, req models.UpdateTelegramUserRequest) (*models.TelegramUser, error) {
	user, err := svc.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if req.ChatID != nil {
		user.ChatID = *req.ChatID
	}
	if req.Name != nil {
		user.Name = *req.Name
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}

	if err := validateTelegramUser(user); err != nil {
		return nil, err
	}

	if err := svc.repo.Update(user); err != nil {
		return nil, err
	}

	svc.log.WithField("telegram_user_id", user.ID).Info("updated telegram user")

	return user, nil
}

// Delete удаляет пользователя бота по ID.
func (svc *TelegramUserService) Delete(id int) error {
	if err := svc.repo.Delete(id); err != nil {
		return err
	}

	svc.log.WithField("telegram_user_id", id).Info("deleted telegram user")
	return nil
}

// validateTelegramUser проверяет бизнес-правила пользователя бота — одинаково для Create и Update.
// chat_id может быть отрицательным (группы и каналы в Telegram), но не нулевым.
func validateTelegramUser(u *models.TelegramUser) error {
	switch {
	case u.ChatID == 0:
		return fmt.Errorf("%w: chat_id is required", ErrInvalidTelegramUser)
	case u.Name == "":
		return fmt.Errorf("%w: name is required", ErrInvalidTelegramUser)
	case len([]rune(u.Name)) > maxTelegramUserNameLength:
		return fmt.Errorf("%w: name must be at most %d characters", ErrInvalidTelegramUser, maxTelegramUserNameLength)
	default:
		return nil
	}
}
