package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"nexera_back/internal/httpresponse"
	"nexera_back/internal/logger"
	"nexera_back/internal/models"
	"nexera_back/internal/repository"
	"nexera_back/internal/services"
)

// RoleHandler управляет ролями пользователей админ-панели: список, просмотр, создание,
// обновление, удаление. Доступен только администраторам (RequireRole(models.RoleCodeAdmin)).
// Кеша нет: эти данные читаются только из админки, а не публичным сайтом.
type RoleHandler struct {
	roleService *services.RoleService
	log         *logger.Logger
}

// NewRoleHandler создает новый обработчик ролей админ-панели.
func NewRoleHandler(roleService *services.RoleService, log *logger.Logger) *RoleHandler {
	return &RoleHandler{
		roleService: roleService,
		log:         log,
	}
}

// GetRoles возвращает все роли. Ролей единицы, поэтому без фильтрации и пагинации —
// как GET /api/v1/pages.
// @Summary      Список ролей
// @Description  Возвращает все роли пользователей админ-панели. Требует роли admin.
// @Tags         admin-roles
// @Produce      json
// @Success      200  {array}   models.Role
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      403  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/roles [get]
func (h *RoleHandler) GetRoles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	roles, err := h.roleService.List()
	if err != nil {
		h.log.WithError(err).Error("failed to get roles")
		httpresponse.WriteError(w, http.StatusInternalServerError, "failed to get roles")
		return
	}

	httpresponse.WriteJSON(w, http.StatusOK, roles)
}

// GetRole возвращает роль по ID.
// @Summary      Получить роль
// @Description  Возвращает роль по ID. Требует роли admin.
// @Tags         admin-roles
// @Produce      json
// @Param        id   path      int  true  "ID роли"
// @Success      200  {object}  models.Role
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      403  {object}  httpresponse.ErrorResponse
// @Failure      404  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/roles/{id} [get]
func (h *RoleHandler) GetRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	roleID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || roleID <= 0 {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid role id")
		return
	}

	role, err := h.roleService.GetByID(roleID)
	if err != nil {
		h.writeRoleError(w, err)
		return
	}

	httpresponse.WriteJSON(w, http.StatusOK, role)
}

// CreateRole создает роль.
// @Summary      Создать роль
// @Description  Создает новую роль. code — латиница в нижнем регистре, цифры и "_", начинается с буквы, 2–50 символов. Требует роли admin.
// @Tags         admin-roles
// @Accept       json
// @Produce      json
// @Param        role  body      models.CreateRoleRequest  true  "Данные роли"
// @Success      201  {object}  models.Role
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      403  {object}  httpresponse.ErrorResponse
// @Failure      409  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/roles [post]
func (h *RoleHandler) CreateRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	var req models.CreateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	role, err := h.roleService.Create(req)
	if err != nil {
		h.writeRoleError(w, err)
		return
	}

	httpresponse.WriteJSON(w, http.StatusCreated, role)
}

// UpdateRole частично обновляет роль по ID.
// @Summary      Обновить роль
// @Description  Частично обновляет роль по ID. Code роли admin менять нельзя. Требует роли admin.
// @Tags         admin-roles
// @Accept       json
// @Produce      json
// @Param        id    path      int                       true  "ID роли"
// @Param        role  body      models.UpdateRoleRequest  true  "Поля для обновления"
// @Success      200  {object}  models.Role
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      403  {object}  httpresponse.ErrorResponse
// @Failure      404  {object}  httpresponse.ErrorResponse
// @Failure      409  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/roles/{id} [patch]
func (h *RoleHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	roleID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || roleID <= 0 {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid role id")
		return
	}

	var req models.UpdateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req == (models.UpdateRoleRequest{}) {
		httpresponse.WriteError(w, http.StatusBadRequest, "at least one field is required")
		return
	}

	role, err := h.roleService.Update(roleID, req)
	if err != nil {
		h.writeRoleError(w, err)
		return
	}

	httpresponse.WriteJSON(w, http.StatusOK, role)
}

// DeleteRole удаляет роль по ID. Роль, назначенную хотя бы одному пользователю, удалить
// нельзя (users.role_id -> roles(id) ON DELETE RESTRICT) — 409.
// @Summary      Удалить роль
// @Description  Удаляет роль по ID. Роль, назначенную пользователям (в т.ч. удаленным), удалить нельзя. Требует роли admin.
// @Tags         admin-roles
// @Param        id  path  int  true  "ID роли"
// @Success      204  "No Content"
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      403  {object}  httpresponse.ErrorResponse
// @Failure      404  {object}  httpresponse.ErrorResponse
// @Failure      409  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/roles/{id} [delete]
func (h *RoleHandler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	roleID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || roleID <= 0 {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid role id")
		return
	}

	if err := h.roleService.Delete(roleID); err != nil {
		h.writeRoleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// writeRoleError сопоставляет ошибки сервисного слоя с HTTP статус-кодами через errors.Is
// по сигнальным ошибкам — тот же подход, что writeBannerError в banner_handler.go.
func (h *RoleHandler) writeRoleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrInvalidRole):
		httpresponse.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, repository.ErrRoleNotFound):
		httpresponse.WriteError(w, http.StatusNotFound, "role not found")
	case errors.Is(err, repository.ErrRoleCodeTaken):
		httpresponse.WriteError(w, http.StatusConflict, "role code already taken")
	case errors.Is(err, repository.ErrRoleInUse):
		httpresponse.WriteError(w, http.StatusConflict, "role is assigned to users")
	default:
		h.log.WithError(err).Error("role operation failed")
		httpresponse.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
