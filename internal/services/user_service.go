package services

import (
	"errors"
	"fmt"
	"net/mail"

	"nexera_back/internal/auth"
	"nexera_back/internal/logger"
	"nexera_back/internal/models"
	"nexera_back/internal/repository"
)

// ErrInvalidUser оборачивает ошибки валидации данных пользователя (см. validateUser,
// validatePassword) — обработчик по нему отличает "клиент прислал некорректные данные" (400)
// от прочих ошибок.
// ErrCannotDeleteSelf / ErrCannotChangeOwnRole — администратор не может удалить сам себя
// или сменить себе роль: иначе можно случайно остаться без единого администратора.
var (
	ErrInvalidUser         = errors.New("invalid user")
	ErrCannotDeleteSelf    = errors.New("cannot delete yourself")
	ErrCannotChangeOwnRole = errors.New("cannot change your own role")
)

const (
	// maxUserNameLength / maxUserEmailLength соответствуют VARCHAR(255) в таблице users.
	maxUserNameLength  = 255
	maxUserEmailLength = 255
	minPasswordLength  = 8
	// maxPasswordBytes — ограничение bcrypt: всё, что длиннее 72 байт, он не принимает.
	maxPasswordBytes = 72
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

// ListUsersFilter описывает запрос списка пользователей из админ-панели: что можно
// отфильтровать и как отсортировать.
type ListUsersFilter struct {
	RoleID   *int
	IsActive *bool
	Sort     repository.UserSortField
	Order    repository.SortOrder
	Limit    int
	Offset   int
}

// List возвращает пользователей админ-панели по фильтру. Мягко удаленные не возвращаются.
func (svc *UserService) List(filter ListUsersFilter) ([]*models.User, error) {
	return svc.repo.List(repository.UserFilter{
		RoleID:   filter.RoleID,
		IsActive: filter.IsActive,
		Sort:     filter.Sort,
		Order:    filter.Order,
		Limit:    filter.Limit,
		Offset:   filter.Offset,
	})
}

// GetByEmail возвращает пользователя вместе с его ролью по email.
func (svc *UserService) GetByEmail(email string) (*models.User, error) {
	return svc.repo.GetByEmail(email)
}

// GetByID возвращает пользователя вместе с его ролью по ID.
func (svc *UserService) GetByID(userID int) (*models.User, error) {
	return svc.repo.GetByID(userID)
}

// Create создает нового пользователя админ-панели. Пароль хешируется bcrypt'ом —
// в БД попадает только хеш. Новый пользователь всегда активен.
func (svc *UserService) Create(req models.CreateUserRequest) (*models.User, error) {
	user := &models.User{
		RoleID:   req.RoleID,
		Name:     req.Name,
		Email:    req.Email,
		IsActive: true,
	}

	if err := validateUser(user); err != nil {
		return nil, err
	}
	if err := validatePassword(req.Password); err != nil {
		return nil, err
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}
	user.PasswordHash = hash

	if err := svc.repo.Create(user); err != nil {
		return nil, err
	}

	svc.log.WithFields(map[string]interface{}{
		"user_id": user.ID,
		"role_id": user.RoleID,
	}).Info("created new user")

	// Перечитываем, чтобы вернуть пользователя вместе с ролью (repo.Create ее не заполняет).
	return svc.repo.GetByID(user.ID)
}

// Update частично обновляет пользователя: подгружает текущее состояние, накладывает на него
// только присланные в запросе поля (nil в запросе — поле остается прежним) и сохраняет
// итог целиком — тот же merge-then-validate-then-persist, что и BannerService.Update.
// actorID — ID администратора, выполняющего запрос: сменить роль самому себе нельзя.
func (svc *UserService) Update(actorID, id int, req models.UpdateUserRequest) (*models.User, error) {
	user, err := svc.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if actorID == id && req.RoleID != nil && *req.RoleID != user.RoleID {
		return nil, ErrCannotChangeOwnRole
	}

	if req.RoleID != nil {
		user.RoleID = *req.RoleID
	}
	if req.Name != nil {
		user.Name = *req.Name
	}
	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}

	if err := validateUser(user); err != nil {
		return nil, err
	}

	if req.Password != nil {
		if err := validatePassword(*req.Password); err != nil {
			return nil, err
		}
		hash, err := auth.HashPassword(*req.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}
		user.PasswordHash = hash
	}

	if err := svc.repo.Update(user); err != nil {
		return nil, err
	}

	svc.log.WithField("user_id", user.ID).Info("updated user")

	// Перечитываем, чтобы вернуть актуальную роль, если role_id поменялся.
	return svc.repo.GetByID(user.ID)
}

// Delete мягко удаляет пользователя по ID. actorID — ID администратора, выполняющего
// запрос: удалить самого себя нельзя.
func (svc *UserService) Delete(actorID, id int) error {
	if actorID == id {
		return ErrCannotDeleteSelf
	}

	if err := svc.repo.Delete(id); err != nil {
		return err
	}

	svc.log.WithField("user_id", id).Info("deleted user")
	return nil
}

// UpdateLastLogin обновляет отметку времени последнего входа пользователя.
func (svc *UserService) UpdateLastLogin(userID int) error {
	return svc.repo.UpdateLastLogin(userID)
}

// validateUser проверяет бизнес-правила, обязательные для пользователя независимо от того,
// получил ли он текущий вид целиком при Create или слиянием при Update. Пароль проверяется
// отдельно (validatePassword): в сущности хранится только хеш, а пароль в Update опционален.
func validateUser(u *models.User) error {
	switch {
	case u.Name == "":
		return fmt.Errorf("%w: name is required", ErrInvalidUser)
	case len([]rune(u.Name)) > maxUserNameLength:
		return fmt.Errorf("%w: name must be at most %d characters", ErrInvalidUser, maxUserNameLength)
	case u.Email == "":
		return fmt.Errorf("%w: email is required", ErrInvalidUser)
	case len([]rune(u.Email)) > maxUserEmailLength:
		return fmt.Errorf("%w: email must be at most %d characters", ErrInvalidUser, maxUserEmailLength)
	case !isValidEmail(u.Email):
		return fmt.Errorf("%w: invalid email", ErrInvalidUser)
	case u.RoleID <= 0:
		return fmt.Errorf("%w: role_id is required", ErrInvalidUser)
	default:
		return nil
	}
}

func validatePassword(password string) error {
	switch {
	case len([]rune(password)) < minPasswordLength:
		return fmt.Errorf("%w: password must be at least %d characters", ErrInvalidUser, minPasswordLength)
	case len(password) > maxPasswordBytes:
		return fmt.Errorf("%w: password must be at most %d bytes", ErrInvalidUser, maxPasswordBytes)
	default:
		return nil
	}
}

// isValidEmail принимает только "голый" адрес (user@example.com): mail.ParseAddress
// пропустил бы и "Имя <user@example.com>", поэтому дополнительно сверяем, что
// разобранный адрес совпадает с исходной строкой.
func isValidEmail(email string) bool {
	addr, err := mail.ParseAddress(email)
	return err == nil && addr.Address == email
}
