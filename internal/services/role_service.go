package services

import (
	"errors"
	"fmt"
	"regexp"

	"nexera_back/internal/logger"
	"nexera_back/internal/models"
	"nexera_back/internal/repository"
)

// ErrInvalidRole оборачивает ошибки валидации данных роли (см. validateRole) —
// обработчик по нему отличает "клиент прислал некорректные данные" (400) от прочих ошибок.
var ErrInvalidRole = errors.New("invalid role")

// roleCodePattern повторяет CHECK (code ~ '^[a-z][a-z0-9_]{1,49}$') из миграции
// 002_create_users — чтобы некорректный code был 400 от сервиса, а не 500 от Postgres.
var roleCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,49}$`)

// maxRoleNameLength соответствует roles.name VARCHAR(100).
const maxRoleNameLength = 100

// RoleService представляет сервис для работы с ролями пользователей админ-панели.
type RoleService struct {
	repo repository.RoleRepository
	log  *logger.Logger
}

// NewRoleService создает новый сервис для работы с ролями.
func NewRoleService(repo repository.RoleRepository, log *logger.Logger) *RoleService {
	return &RoleService{
		repo: repo,
		log:  log,
	}
}

// List возвращает все роли.
func (svc *RoleService) List() ([]*models.Role, error) {
	return svc.repo.List()
}

// GetByID возвращает роль по ID.
func (svc *RoleService) GetByID(id int) (*models.Role, error) {
	return svc.repo.GetByID(id)
}

// Create создает новую роль.
func (svc *RoleService) Create(req models.CreateRoleRequest) (*models.Role, error) {
	role := &models.Role{
		Name: req.Name,
		Code: req.Code,
	}

	if err := validateRole(role); err != nil {
		return nil, err
	}

	if err := svc.repo.Create(role); err != nil {
		return nil, err
	}

	svc.log.WithField("role_id", role.ID).Info("created new role")

	return role, nil
}

// Update частично обновляет роль: подгружает текущее состояние, накладывает на него
// только присланные в запросе поля (nil в запросе — поле остается прежним) и сохраняет
// итог целиком — тот же merge-then-validate-then-persist, что и BannerService.Update.
//
// Code роли администратора менять нельзя: доступ к /api/v1/admin/users и /roles проверяется
// по нему (middleware.RequireRole(models.RoleCodeAdmin)), и после переименования ни один
// администратор, войдя заново, не смог бы управлять пользователями и ролями.
func (svc *RoleService) Update(id int, req models.UpdateRoleRequest) (*models.Role, error) {
	role, err := svc.repo.GetByID(id)
	if err != nil {
		return nil, err
	}

	if req.Code != nil && role.Code == models.RoleCodeAdmin && *req.Code != models.RoleCodeAdmin {
		return nil, fmt.Errorf("%w: code of the %q role cannot be changed", ErrInvalidRole, models.RoleCodeAdmin)
	}

	if req.Name != nil {
		role.Name = *req.Name
	}
	if req.Code != nil {
		role.Code = *req.Code
	}

	if err := validateRole(role); err != nil {
		return nil, err
	}

	if err := svc.repo.Update(role); err != nil {
		return nil, err
	}

	svc.log.WithField("role_id", role.ID).Info("updated role")

	return role, nil
}

// Delete удаляет роль по ID. Роль, назначенную пользователям, удалить нельзя
// (users.role_id -> roles(id) ON DELETE RESTRICT, см. repository.ErrRoleInUse).
func (svc *RoleService) Delete(id int) error {
	if err := svc.repo.Delete(id); err != nil {
		return err
	}

	svc.log.WithField("role_id", id).Info("deleted role")
	return nil
}

// validateRole проверяет бизнес-правила, обязательные для роли независимо от того,
// получила ли она текущий вид целиком при Create или слиянием при Update.
func validateRole(r *models.Role) error {
	switch {
	case r.Name == "":
		return fmt.Errorf("%w: name is required", ErrInvalidRole)
	case len([]rune(r.Name)) > maxRoleNameLength:
		return fmt.Errorf("%w: name must be at most %d characters", ErrInvalidRole, maxRoleNameLength)
	case r.Code == "":
		return fmt.Errorf("%w: code is required", ErrInvalidRole)
	case !roleCodePattern.MatchString(r.Code):
		return fmt.Errorf("%w: code must match %s", ErrInvalidRole, roleCodePattern.String())
	default:
		return nil
	}
}
