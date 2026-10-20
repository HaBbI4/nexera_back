package public

import (
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

// ContactInfoHandler отдает контактные данные компании. Без авторизации.
type ContactInfoHandler struct {
	contactInfoService *services.ContactInfoService
	redisClient        *redis.Client
	log                *logger.Logger
}

// NewContactInfoHandler создает новый обработчик contact_info.
func NewContactInfoHandler(contactInfoService *services.ContactInfoService, redisClient *redis.Client, log *logger.Logger) *ContactInfoHandler {
	return &ContactInfoHandler{
		contactInfoService: contactInfoService,
		redisClient:        redisClient,
		log:                log,
	}
}

// GetContactInfos возвращает список опубликованных контактных записей.
//
// @Summary      Список контактов
// @Description  Возвращает опубликованные контактные данные с сортировкой и пагинацией
// @Tags         contact-info
// @Produce      json
// @Param        sort    query     string  false  "Поле сортировки: position (по умолчанию) или created_at"
// @Param        order   query     string  false  "Направление сортировки: asc (по умолчанию) или desc"
// @Param        limit   query     int     false  "Количество записей (по умолчанию 50, максимум 100)"
// @Param        offset  query     int     false  "Смещение (по умолчанию 0)"
// @Success      200  {array}   models.ContactInfo
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/contact-info [get]
func (h *ContactInfoHandler) GetContactInfos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	query := r.URL.Query()

	filter := services.ListContactInfoFilter{
		Sort:  repository.ContactInfoSortPosition,
		Order: repository.SortAsc,
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

	items, err := h.contactInfoService.ListPublic(filter)
	if err != nil {
		h.log.WithError(err).Error("failed to get contact info")
		httpresponse.WriteError(w, http.StatusInternalServerError, "failed to get contact info")
		return
	}

	httpresponse.WriteJSON(w, http.StatusOK, items)
}

// GetContactInfo возвращает одну опубликованную контактную запись по ID.
//
// @Summary      Получить контакт
// @Description  Возвращает контактную запись по ID, если она опубликована
// @Tags         contact-info
// @Produce      json
// @Param        id   path      int  true  "ID контактной записи"
// @Success      200  {object}  models.ContactInfo
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      404  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/contact-info/{id} [get]
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

	cacheKey := redis.GenerateKey(redis.KeyPrefixContactInfo, strconv.Itoa(contactID))
	var info models.ContactInfo
	if err := h.redisClient.Get(r.Context(), cacheKey, &info); err == nil {
		h.log.WithField("contact_info_id", contactID).Debug("contact info retrieved from cache")
		httpresponse.WriteJSON(w, http.StatusOK, &info)
		return
	}

	infoPtr, err := h.contactInfoService.GetPublicByID(contactID)
	if err != nil {
		if errors.Is(err, repository.ErrContactInfoNotFound) {
			httpresponse.WriteError(w, http.StatusNotFound, "contact info not found")
		} else {
			h.log.WithError(err).Error("failed to get contact info")
			httpresponse.WriteError(w, http.StatusInternalServerError, "failed to get contact info")
		}
		return
	}

	if err := h.redisClient.Set(r.Context(), cacheKey, infoPtr, defaultCacheTTL); err != nil {
		h.log.WithError(err).Error("failed to cache contact info")
	}

	httpresponse.WriteJSON(w, http.StatusOK, infoPtr)
}
