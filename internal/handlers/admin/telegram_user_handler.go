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

// allowedTelegramUserSortFields — белый список значений query-параметра ?sort=.
var allowedTelegramUserSortFields = map[string]repository.TelegramUserSortField{
	"created_at": repository.TelegramUserSortCreatedAt,
	"name":       repository.TelegramUserSortName,
}

// TelegramUserHandler управляет пользователями telegram-бота из админ-панели: кто имеет
// доступ к боту и получает уведомления о заявках и отзывах. Публичных эндпоинтов нет,
// кеша тоже — данные читает только админка и сам бот.
type TelegramUserHandler struct {
	telegramUserService *services.TelegramUserService
	log                 *logger.Logger
}

// NewTelegramUserHandler создает новый обработчик пользователей бота админ-панели.
func NewTelegramUserHandler(telegramUserService *services.TelegramUserService, log *logger.Logger) *TelegramUserHandler {
	return &TelegramUserHandler{
		telegramUserService: telegramUserService,
		log:                 log,
	}
}

// GetTelegramUsers возвращает список пользователей бота с фильтрацией, сортировкой и пагинацией.
//
// limit/offset при некорректном значении молча заменяются значением по умолчанию,
// фильтры и сортировка — строже, некорректное значение — 400.
//
// @Summary      Список пользователей telegram-бота
// @Description  Возвращает всех пользователей бота, включая отключенных. Только для роли admin.
// @Tags         admin-telegram-users
// @Produce      json
// @Param        is_active  query     bool    false  "true — только активные, false — только отключенные"
// @Param        sort       query     string  false  "Поле сортировки: created_at (по умолчанию) или name"
// @Param        order      query     string  false  "Направление сортировки: asc (по умолчанию) или desc"
// @Param        limit      query     int     false  "Количество записей (по умолчанию 50, максимум 100)"
// @Param        offset     query     int     false  "Смещение (по умолчанию 0)"
// @Success      200  {array}   models.TelegramUser
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      403  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/telegram-users [get]
func (h *TelegramUserHandler) GetTelegramUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	query := r.URL.Query()

	filter := services.ListTelegramUsersFilter{
		Sort:  repository.TelegramUserSortCreatedAt,
		Order: repository.SortAsc,
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
		sortField, ok := allowedTelegramUserSortFields[sortStr]
		if !ok {
			httpresponse.WriteError(w, http.StatusBadRequest, "invalid sort, allowed: created_at, name")
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

	users, err := h.telegramUserService.List(filter)
	if err != nil {
		h.log.WithError(err).Error("failed to get telegram users")
		httpresponse.WriteError(w, http.StatusInternalServerError, "failed to get telegram users")
		return
	}

	httpresponse.WriteJSON(w, http.StatusOK, users)
}

// GetTelegramUser возвращает пользователя бота по ID.
// @Summary      Получить пользователя telegram-бота
// @Description  Возвращает пользователя бота по ID. Только для роли admin.
// @Tags         admin-telegram-users
// @Produce      json
// @Param        id   path      int  true  "ID пользователя бота"
// @Success      200  {object}  models.TelegramUser
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      403  {object}  httpresponse.ErrorResponse
// @Failure      404  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/telegram-users/{id} [get]
func (h *TelegramUserHandler) GetTelegramUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	userID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || userID <= 0 {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid telegram user id")
		return
	}

	user, err := h.telegramUserService.GetByID(userID)
	if err != nil {
		h.writeTelegramUserError(w, err)
		return
	}

	httpresponse.WriteJSON(w, http.StatusOK, user)
}

// CreateTelegramUser добавляет пользователя бота.
// @Summary      Добавить пользователя telegram-бота
// @Description  Добавляет пользователя бота по chat_id. Без is_active создается активным. Только для роли admin.
// @Tags         admin-telegram-users
// @Accept       json
// @Produce      json
// @Param        telegram_user  body      models.CreateTelegramUserRequest  true  "Данные пользователя бота"
// @Success      201  {object}  models.TelegramUser
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      403  {object}  httpresponse.ErrorResponse
// @Failure      409  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/telegram-users [post]
func (h *TelegramUserHandler) CreateTelegramUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	var req models.CreateTelegramUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.telegramUserService.Create(req)
	if err != nil {
		h.writeTelegramUserError(w, err)
		return
	}

	httpresponse.WriteJSON(w, http.StatusCreated, user)
}

// UpdateTelegramUser частично обновляет пользователя бота по ID.
// @Summary      Обновить пользователя telegram-бота
// @Description  Частично обновляет пользователя бота по ID (например, is_active=false отключает доступ и уведомления). Только для роли admin.
// @Tags         admin-telegram-users
// @Accept       json
// @Produce      json
// @Param        id             path      int                               true  "ID пользователя бота"
// @Param        telegram_user  body      models.UpdateTelegramUserRequest  true  "Поля для обновления"
// @Success      200  {object}  models.TelegramUser
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      403  {object}  httpresponse.ErrorResponse
// @Failure      404  {object}  httpresponse.ErrorResponse
// @Failure      409  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/telegram-users/{id} [patch]
func (h *TelegramUserHandler) UpdateTelegramUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	userID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || userID <= 0 {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid telegram user id")
		return
	}

	var req models.UpdateTelegramUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req == (models.UpdateTelegramUserRequest{}) {
		httpresponse.WriteError(w, http.StatusBadRequest, "at least one field is required")
		return
	}

	user, err := h.telegramUserService.Update(userID, req)
	if err != nil {
		h.writeTelegramUserError(w, err)
		return
	}

	httpresponse.WriteJSON(w, http.StatusOK, user)
}

// DeleteTelegramUser удаляет пользователя бота по ID.
// @Summary      Удалить пользователя telegram-бота
// @Description  Удаляет пользователя бота по ID: он сразу теряет доступ к боту. Только для роли admin.
// @Tags         admin-telegram-users
// @Param        id  path  int  true  "ID пользователя бота"
// @Success      204  "No Content"
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      403  {object}  httpresponse.ErrorResponse
// @Failure      404  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/telegram-users/{id} [delete]
func (h *TelegramUserHandler) DeleteTelegramUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	userID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || userID <= 0 {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid telegram user id")
		return
	}

	if err := h.telegramUserService.Delete(userID); err != nil {
		h.writeTelegramUserError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// writeTelegramUserError сопоставляет ошибки сервисного слоя с HTTP статус-кодами через
// errors.Is по сигнальным ошибкам — тот же подход, что writeBannerError в banner_handler.go.
func (h *TelegramUserHandler) writeTelegramUserError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrInvalidTelegramUser):
		httpresponse.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, repository.ErrTelegramUserNotFound):
		httpresponse.WriteError(w, http.StatusNotFound, "telegram user not found")
	case errors.Is(err, repository.ErrTelegramUserChatIDTaken):
		httpresponse.WriteError(w, http.StatusConflict, "telegram user with this chat_id already exists")
	default:
		h.log.WithError(err).Error("telegram user operation failed")
		httpresponse.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
