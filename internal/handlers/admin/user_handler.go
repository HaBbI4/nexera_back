package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"nexera_back/internal/httpresponse"
	"nexera_back/internal/logger"
	"nexera_back/internal/middleware"
	"nexera_back/internal/models"
	"nexera_back/internal/repository"
	"nexera_back/internal/services"
)

// allowedUserSortFields — белый список значений query-параметра ?sort=.
// Отдельный от repository.userSortColumns: тот определяет SQL-колонки,
// этот — что разрешено принять от клиента (невалидное значение -> 400, а не тихий дефолт).
var allowedUserSortFields = map[string]repository.UserSortField{
	"created_at": repository.UserSortCreatedAt,
	"name":       repository.UserSortName,
	"email":      repository.UserSortEmail,
}

// UserHandler управляет пользователями админ-панели: список, просмотр, создание,
// обновление, удаление. Доступен только администраторам (RequireRole(models.RoleCodeAdmin)).
// Кеша нет: эти данные читаются только из админки, а не публичным сайтом.
type UserHandler struct {
	userService *services.UserService
	log         *logger.Logger
}

// NewUserHandler создает новый обработчик пользователей админ-панели.
func NewUserHandler(userService *services.UserService, log *logger.Logger) *UserHandler {
	return &UserHandler{
		userService: userService,
		log:         log,
	}
}

// GetUsers возвращает список пользователей с фильтрацией, сортировкой и пагинацией.
//
// Как и в GET /api/v1/banners: limit/offset при некорректном значении молча заменяются
// значением по умолчанию, а фильтры и сортировка (role_id/is_active/sort/order) — строже,
// некорректное значение — это 400.
//
// @Summary      Список пользователей
// @Description  Возвращает пользователей админ-панели (без удаленных) с фильтрацией по роли и активности. Требует роли admin.
// @Tags         admin-users
// @Produce      json
// @Param        role_id    query     int     false  "ID роли — вернуть только пользователей с этой ролью"
// @Param        is_active  query     bool    false  "true — только активные, false — только отключенные"
// @Param        sort       query     string  false  "Поле сортировки: created_at (по умолчанию), name или email"
// @Param        order      query     string  false  "Направление сортировки: asc (по умолчанию) или desc"
// @Param        limit      query     int     false  "Количество записей (по умолчанию 50, максимум 100)"
// @Param        offset     query     int     false  "Смещение (по умолчанию 0)"
// @Success      200  {array}   models.User
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      403  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/users [get]
func (h *UserHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	query := r.URL.Query()

	filter := services.ListUsersFilter{
		Sort:  repository.UserSortCreatedAt,
		Order: repository.SortAsc,
	}

	if roleIDStr := query.Get("role_id"); roleIDStr != "" {
		roleID, err := strconv.Atoi(roleIDStr)
		if err != nil || roleID <= 0 {
			httpresponse.WriteError(w, http.StatusBadRequest, "invalid role_id")
			return
		}
		filter.RoleID = &roleID
	}

	if isActiveStr := query.Get("is_active"); isActiveStr != "" {
		isActive, err := strconv.ParseBool(isActiveStr)
		if err != nil {
			httpresponse.WriteError(w, http.StatusBadRequest, "invalid is_active")
			return
		}
		filter.IsActive = &isActive
	}

	if sortStr := query.Get("sort"); sortStr != "" {
		sortField, ok := allowedUserSortFields[sortStr]
		if !ok {
			httpresponse.WriteError(w, http.StatusBadRequest, "invalid sort, allowed: created_at, name, email")
			return
		}
		filter.Sort = sortField
	}

	if orderStr := query.Get("order"); orderStr != "" {
		switch repository.SortOrder(orderStr) {
		case repository.SortAsc, repository.SortDesc:
			filter.Order = repository.SortOrder(orderStr)
		default:
			httpresponse.WriteError(w, http.StatusBadRequest, "invalid order, allowed: asc, desc")
			return
		}
	}

	// Пагинация — по тому же правилу, что в остальных списочных эндпоинтах проекта:
	// некорректное значение не ошибка, а молчаливый возврат к дефолту.
	filter.Limit = 50 // По умолчанию
	if limitStr := query.Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			filter.Limit = l
		}
	}

	if offsetStr := query.Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			filter.Offset = o
		}
	}

	users, err := h.userService.List(filter)
	if err != nil {
		h.log.WithError(err).Error("failed to get users")
		httpresponse.WriteError(w, http.StatusInternalServerError, "failed to get users")
		return
	}

	httpresponse.WriteJSON(w, http.StatusOK, users)
}

// GetUser возвращает пользователя по ID вместе с его ролью.
// @Summary      Получить пользователя
// @Description  Возвращает пользователя админ-панели по ID. Требует роли admin.
// @Tags         admin-users
// @Produce      json
// @Param        id   path      int  true  "ID пользователя"
// @Success      200  {object}  models.User
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      403  {object}  httpresponse.ErrorResponse
// @Failure      404  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/users/{id} [get]
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	userID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || userID <= 0 {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	user, err := h.userService.GetByID(userID)
	if err != nil {
		h.writeUserError(w, err)
		return
	}

	httpresponse.WriteJSON(w, http.StatusOK, user)
}

// CreateUser создает пользователя админ-панели.
// @Summary      Создать пользователя
// @Description  Создает нового пользователя админ-панели. Пароль хранится только в виде bcrypt-хеша. Требует роли admin.
// @Tags         admin-users
// @Accept       json
// @Produce      json
// @Param        user  body      models.CreateUserRequest  true  "Данные пользователя"
// @Success      201  {object}  models.User
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      403  {object}  httpresponse.ErrorResponse
// @Failure      409  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/users [post]
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.userService.Create(req)
	if err != nil {
		h.writeUserError(w, err)
		return
	}

	httpresponse.WriteJSON(w, http.StatusCreated, user)
}

// UpdateUser частично обновляет пользователя по ID.
// @Summary      Обновить пользователя
// @Description  Частично обновляет пользователя по ID. Сменить роль самому себе нельзя. Требует роли admin.
// @Tags         admin-users
// @Accept       json
// @Produce      json
// @Param        id    path      int                       true  "ID пользователя"
// @Param        user  body      models.UpdateUserRequest  true  "Поля для обновления"
// @Success      200  {object}  models.User
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      403  {object}  httpresponse.ErrorResponse
// @Failure      404  {object}  httpresponse.ErrorResponse
// @Failure      409  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/users/{id} [patch]
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	authUser, ok := middleware.UserFromContext(r.Context())
	if !ok {
		httpresponse.WriteError(w, http.StatusUnauthorized, "missing bearer token")
		return
	}

	userID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || userID <= 0 {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req == (models.UpdateUserRequest{}) {
		httpresponse.WriteError(w, http.StatusBadRequest, "at least one field is required")
		return
	}

	user, err := h.userService.Update(authUser.ID, userID, req)
	if err != nil {
		h.writeUserError(w, err)
		return
	}

	httpresponse.WriteJSON(w, http.StatusOK, user)
}

// DeleteUser мягко удаляет пользователя по ID (проставляет deleted_at).
// @Summary      Удалить пользователя
// @Description  Мягко удаляет пользователя по ID. Удалить самого себя нельзя. Требует роли admin.
// @Tags         admin-users
// @Param        id  path  int  true  "ID пользователя"
// @Success      204  "No Content"
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      403  {object}  httpresponse.ErrorResponse
// @Failure      404  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/users/{id} [delete]
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	authUser, ok := middleware.UserFromContext(r.Context())
	if !ok {
		httpresponse.WriteError(w, http.StatusUnauthorized, "missing bearer token")
		return
	}

	userID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || userID <= 0 {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	if err := h.userService.Delete(authUser.ID, userID); err != nil {
		h.writeUserError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// writeUserError сопоставляет ошибки сервисного слоя с HTTP статус-кодами через errors.Is
// по сигнальным ошибкам — тот же подход, что writeBannerError в banner_handler.go.
func (h *UserHandler) writeUserError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrInvalidUser):
		httpresponse.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, services.ErrCannotDeleteSelf), errors.Is(err, services.ErrCannotChangeOwnRole):
		httpresponse.WriteError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, repository.ErrUserNotFound):
		httpresponse.WriteError(w, http.StatusNotFound, "user not found")
	case errors.Is(err, repository.ErrUserEmailTaken):
		httpresponse.WriteError(w, http.StatusConflict, "user email already taken")
	case errors.Is(err, repository.ErrRoleNotFound):
		httpresponse.WriteError(w, http.StatusBadRequest, "role not found")
	default:
		h.log.WithError(err).Error("user operation failed")
		httpresponse.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
