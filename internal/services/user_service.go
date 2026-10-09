package services

import (
	"nexera_back/internal/logger"
	"nexera_back/internal/models"
	"nexera_back/internal/repository"
)

// UserService представляет сервис для работы с пользователями админ-панели.
// Сам не знает, откуда берутся данные (Postgres, мок для тестов и т.д.) — это
// скрыто за repository.UserRepository.
type UserService struct {
	repo repository.UserRepository
	log  *logger.Logger
}

// NewUserService создает новый сервис для работы с пользователями админ-панели
func NewUserService(repo repository.UserRepository, log *logger.Logger) *UserService {
	return &UserService{
		repo: repo,
		log:  log,
	}
}

// GetByEmail возвращает пользователя вместе с его ролью по email.
func (svc *UserService) GetByEmail(email string) (*models.User, error) {
	return svc.repo.GetByEmail(email)
}

// GetByID возвращает пользователя вместе с его ролью по ID.
func (svc *UserService) GetByID(userID int) (*models.User, error) {
	return svc.repo.GetByID(userID)
}

// UpdateLastLogin обновляет отметку времени последнего входа пользователя.
func (svc *UserService) UpdateLastLogin(userID int) error {
	return svc.repo.UpdateLastLogin(userID)
}
