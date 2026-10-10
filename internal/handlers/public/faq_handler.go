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

// allowedFAQSortFields — белый список значений query-параметра ?sort=.
// Отдельный от repository.faqSortColumns: тот определяет SQL-колонки,
// этот — что разрешено принять от клиента (невалидное значение -> 400, а не тихий дефолт).
var allowedFAQSortFields = map[string]repository.FAQSortField{
	"position":   repository.FAQSortPosition,
	"created_at": repository.FAQSortCreatedAt,
}

// FAQHandler отдает вопросы-ответы (FAQ) сайта. Без авторизации — доступен любому клиенту.
type FAQHandler struct {
	faqService  *services.FAQService
	redisClient *redis.Client
	log         *logger.Logger
}

// NewFAQHandler создает новый обработчик FAQ.
func NewFAQHandler(faqService *services.FAQService, redisClient *redis.Client, log *logger.Logger) *FAQHandler {
	return &FAQHandler{
		faqService:  faqService,
		redisClient: redisClient,
		log:         log,
	}
}

// GetFAQs возвращает список опубликованных FAQ с сортировкой и пагинацией.
//
// Как и в GET /api/v1/banners: limit/offset при некорректном значении молча заменяются
// значением по умолчанию, а sort/order — строже, некорректное значение — это 400.
//
// @Summary      Список FAQ
// @Description  Возвращает опубликованные вопросы-ответы с сортировкой и пагинацией
// @Tags         faq
// @Produce      json
// @Param        sort    query     string  false  "Поле сортировки: position (по умолчанию) или created_at"
// @Param        order   query     string  false  "Направление сортировки: asc (по умолчанию) или desc"
// @Param        limit   query     int     false  "Количество записей (по умолчанию 50, максимум 100)"
// @Param        offset  query     int     false  "Смещение (по умолчанию 0)"
// @Success      200  {array}   models.FAQ
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/faq [get]
func (h *FAQHandler) GetFAQs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	query := r.URL.Query()

	filter := services.ListFAQFilter{
		Sort:  repository.FAQSortPosition,
		Order: repository.SortAsc,
	}

	if sortStr := query.Get("sort"); sortStr != "" {
		sortField, ok := allowedFAQSortFields[sortStr]
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

	faqs, err := h.faqService.ListPublic(filter)
	if err != nil {
		h.log.WithError(err).Error("failed to get faq")
		httpresponse.WriteError(w, http.StatusInternalServerError, "failed to get faq")
		return
	}

	httpresponse.WriteJSON(w, http.StatusOK, faqs)
}

// GetFAQ возвращает одну запись FAQ по ID, если она опубликована.
// @Summary      Получить FAQ
// @Description  Возвращает вопрос-ответ по ID, если он опубликован
// @Tags         faq
// @Produce      json
// @Param        id   path      int  true  "ID записи FAQ"
// @Success      200  {object}  models.FAQ
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      404  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/faq/{id} [get]
func (h *FAQHandler) GetFAQ(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	faqID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || faqID <= 0 {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid faq id")
		return
	}

	cacheKey := redis.GenerateKey(redis.KeyPrefixFAQ, strconv.Itoa(faqID))
	var faq models.FAQ
	if err := h.redisClient.Get(r.Context(), cacheKey, &faq); err == nil {
		h.log.WithField("faq_id", faqID).Debug("FAQ retrieved from cache")
		httpresponse.WriteJSON(w, http.StatusOK, &faq)
		return
	}

	faqPtr, err := h.faqService.GetPublicByID(faqID)
	if err != nil {
		if errors.Is(err, repository.ErrFAQNotFound) {
			httpresponse.WriteError(w, http.StatusNotFound, "faq not found")
		} else {
			h.log.WithError(err).Error("failed to get faq")
			httpresponse.WriteError(w, http.StatusInternalServerError, "failed to get faq")
		}
		return
	}

	if err := h.redisClient.Set(r.Context(), cacheKey, faqPtr, defaultCacheTTL); err != nil {
		h.log.WithError(err).Error("failed to cache faq")
	}

	httpresponse.WriteJSON(w, http.StatusOK, faqPtr)
}
