package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"nexera_back/internal/httpresponse"
	"nexera_back/internal/logger"
	"nexera_back/internal/models"
	"nexera_back/internal/redis"
	"nexera_back/internal/repository"
	"nexera_back/internal/services"
)

var allowedContactInfoSortFields = map[string]repository.ContactInfoSortField{
	"position":   repository.ContactInfoSortPosition,
	"created_at": repository.ContactInfoSortCreatedAt,
}

// ContactInfoHandler управляет контактными данными из админ-панели.
type ContactInfoHandler struct {
	contactInfoService *services.ContactInfoService
	redisClient        *redis.Client
	log                *logger.Logger
}

// NewContactInfoHandler создает новый обработчик contact_info админ-панели.
func NewContactInfoHandler(contactInfoService *services.ContactInfoService, redisClient *redis.Client, log *logger.Logger) *ContactInfoHandler {
	return &ContactInfoHandler{
		contactInfoService: contactInfoService,
		redisClient:        redisClient,
		log:                log,
	}
}

// GetContactInfos возвращает список всех контактных записей для админ-панели.
//
// @Summary      Список контактов (админка)
// @Description  Возвращает все контактные записи, включая неопубликованные. Требует авторизации.
// @Tags         admin-contact-info
// @Produce      json
// @Param        is_published  query     bool    false  "true — только опубликованные, false — только скрытые"
// @Param        sort          query     string  false  "Поле сортировки: position (по умолчанию) или created_at"
// @Param        order         query     string  false  "Направление сортировки: asc (по умолчанию) или desc"
// @Param        limit         query     int     false  "Количество записей (по умолчанию 50, максимум 100)"
// @Param        offset        query     int     false  "Смещение (по умолчанию 0)"
// @Success      200  {array}   models.ContactInfo
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/contact-info [get]
func (h *ContactInfoHandler) GetContactInfos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	query := r.URL.Query()

	filter := services.ListAdminContactInfoFilter{
		Sort:  repository.ContactInfoSortPosition,
		Order: repository.SortAsc,
	}

	if isPublishedStr := query.Get("is_published"); isPublishedStr != "" {
		isPublished, err := strconv.ParseBool(isPublishedStr)
		if err != nil {
			httpresponse.WriteError(w, http.StatusBadRequest, "invalid is_published")
			return
		}
		filter.IsPublished = &isPublished
	}

	if sortStr := query.Get("sort"); sortStr != "" {
		sortField, ok := allowedContactInfoSortFields[sortStr]
		if !ok {
			httpresponse.WriteError(w, http.StatusBadRequest, "invalid sort, allowed: position, created_at")
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

	filter.Limit = 50
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

	items, err := h.contactInfoService.List(filter)
	if err != nil {
		h.log.WithError(err).Error("failed to get contact info")
		httpresponse.WriteError(w, http.StatusInternalServerError, "failed to get contact info")
		return
	}

	httpresponse.WriteJSON(w, http.StatusOK, items)
}

// GetContactInfo возвращает контактную запись по ID для админ-панели.
//
// @Summary      Получить контакт (админка)
// @Description  Возвращает контактную запись по ID, включая неопубликованную. Требует авторизации.
// @Tags         admin-contact-info
// @Produce      json
// @Param        id   path      int  true  "ID контактной записи"
// @Success      200  {object}  models.ContactInfo
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      404  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/contact-info/{id} [get]
func (h *ContactInfoHandler) GetContactInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	contactID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || contactID <= 0 {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid contact info id")
		return
	}

	info, err := h.contactInfoService.GetByID(contactID)
	if err != nil {
		h.writeContactInfoError(w, err)
		return
	}

	httpresponse.WriteJSON(w, http.StatusOK, info)
}

// CreateContactInfo создает контактную запись.
//
// @Summary      Создать контакт
// @Description  Создает новую контактную запись. Требует авторизации.
// @Tags         admin-contact-info
// @Accept       json
// @Produce      json
// @Param        contact  body      models.CreateContactInfoRequest  true  "Данные контакта"
// @Success      201  {object}  models.ContactInfo
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/contact-info [post]
func (h *ContactInfoHandler) CreateContactInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	var req models.CreateContactInfoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	info, err := h.contactInfoService.Create(req)
	if err != nil {
		h.writeContactInfoError(w, err)
		return
	}

	httpresponse.WriteJSON(w, http.StatusCreated, info)
}

// UpdateContactInfo частично обновляет контактную запись по ID.
//
// @Summary      Обновить контакт
// @Description  Частично обновляет контактную запись по ID. Требует авторизации.
// @Tags         admin-contact-info
// @Accept       json
// @Produce      json
// @Param        id       path      int                             true  "ID контактной записи"
// @Param        contact  body      models.UpdateContactInfoRequest  true  "Поля для обновления"
// @Success      200  {object}  models.ContactInfo
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      404  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/contact-info/{id} [patch]
func (h *ContactInfoHandler) UpdateContactInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	contactID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || contactID <= 0 {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid contact info id")
		return
	}

	var req models.UpdateContactInfoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req == (models.UpdateContactInfoRequest{}) {
		httpresponse.WriteError(w, http.StatusBadRequest, "at least one field is required")
		return
	}

	info, err := h.contactInfoService.Update(contactID, req)
	if err != nil {
		h.writeContactInfoError(w, err)
		return
	}

	h.invalidateContactInfoCache(r, contactID)

	httpresponse.WriteJSON(w, http.StatusOK, info)
}

// DeleteContactInfo удаляет контактную запись по ID.
//
// @Summary      Удалить контакт
// @Description  Удаляет контактную запись по ID. Требует авторизации.
// @Tags         admin-contact-info
// @Param        id  path  int  true  "ID контактной записи"
// @Success      204  "No Content"
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      404  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/contact-info/{id} [delete]
func (h *ContactInfoHandler) DeleteContactInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	contactID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || contactID <= 0 {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid contact info id")
		return
	}

	if err := h.contactInfoService.Delete(contactID); err != nil {
		h.writeContactInfoError(w, err)
		return
	}

	h.invalidateContactInfoCache(r, contactID)

	w.WriteHeader(http.StatusNoContent)
}

func (h *ContactInfoHandler) invalidateContactInfoCache(r *http.Request, contactID int) {
	cacheKey := redis.GenerateKey(redis.KeyPrefixContactInfo, strconv.Itoa(contactID))
	if err := h.redisClient.Delete(r.Context(), cacheKey); err != nil {
		h.log.WithError(err).Error("failed to invalidate contact info cache")
	}
}

func (h *ContactInfoHandler) writeContactInfoError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrInvalidContactInfo):
		httpresponse.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, repository.ErrContactInfoNotFound):
		httpresponse.WriteError(w, http.StatusNotFound, "contact info not found")
	default:
		h.log.WithError(err).Error("contact info operation failed")
		httpresponse.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
