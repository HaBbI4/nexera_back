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

// allowedFAQSortFields — белый список значений query-параметра ?sort= для админского
// списка FAQ (тот же набор, что в публичном internal/handlers/public).
var allowedFAQSortFields = map[string]repository.FAQSortField{
	"position":   repository.FAQSortPosition,
	"created_at": repository.FAQSortCreatedAt,
}

// FAQHandler управляет вопросами-ответами (FAQ) из админ-панели: просмотр, создание,
// обновление, удаление. В отличие от публичного internal/handlers/public.FAQHandler,
// чтение здесь показывает все записи, включая неопубликованные, и без кеша. Изображений
// у FAQ нет, поэтому тело запроса — обычный application/json, как у страниц.
type FAQHandler struct {
	faqService  *services.FAQService
	redisClient *redis.Client
	log         *logger.Logger
}

// NewFAQHandler создает новый обработчик FAQ админ-панели.
func NewFAQHandler(faqService *services.FAQService, redisClient *redis.Client, log *logger.Logger) *FAQHandler {
	return &FAQHandler{
		faqService:  faqService,
		redisClient: redisClient,
		log:         log,
	}
}

// GetFAQs возвращает список всех записей FAQ для админ-панели с фильтрацией, сортировкой
// и пагинацией — вне зависимости от публикации.
//
// Как и в публичном GET /api/v1/faq: limit/offset при некорректном значении молча
// заменяются значением по умолчанию, а фильтры и сортировка — строже, некорректное значение — 400.
//
// @Summary      Список FAQ (админка)
// @Description  Возвращает все вопросы-ответы, включая неопубликованные. Требует авторизации.
// @Tags         admin-faq
// @Produce      json
// @Param        is_published  query     bool    false  "true — только опубликованные, false — только скрытые"
// @Param        sort          query     string  false  "Поле сортировки: position (по умолчанию) или created_at"
// @Param        order         query     string  false  "Направление сортировки: asc (по умолчанию) или desc"
// @Param        limit         query     int     false  "Количество записей (по умолчанию 50, максимум 100)"
// @Param        offset        query     int     false  "Смещение (по умолчанию 0)"
// @Success      200  {array}   models.FAQ
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/faq [get]
func (h *FAQHandler) GetFAQs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	query := r.URL.Query()

	filter := services.ListAdminFAQFilter{
		Sort:  repository.FAQSortPosition,
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

	faqs, err := h.faqService.List(filter)
	if err != nil {
		h.log.WithError(err).Error("failed to get faq")
		httpresponse.WriteError(w, http.StatusInternalServerError, "failed to get faq")
		return
	}

	httpresponse.WriteJSON(w, http.StatusOK, faqs)
}

// GetFAQ возвращает запись FAQ по ID для админ-панели — вне зависимости от публикации.
// @Summary      Получить FAQ (админка)
// @Description  Возвращает вопрос-ответ по ID, включая неопубликованный. Требует авторизации.
// @Tags         admin-faq
// @Produce      json
// @Param        id   path      int  true  "ID записи FAQ"
// @Success      200  {object}  models.FAQ
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      404  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/faq/{id} [get]
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

	faq, err := h.faqService.GetByID(faqID)
	if err != nil {
		h.writeFAQError(w, err)
		return
	}

	httpresponse.WriteJSON(w, http.StatusOK, faq)
}

// CreateFAQ создает запись FAQ.
// @Summary      Создать FAQ
// @Description  Создает новый вопрос-ответ. Требует авторизации.
// @Tags         admin-faq
// @Accept       json
// @Produce      json
// @Param        faq  body      models.CreateFAQRequest  true  "Данные FAQ"
// @Success      201  {object}  models.FAQ
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/faq [post]
func (h *FAQHandler) CreateFAQ(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	var req models.CreateFAQRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	faq, err := h.faqService.Create(req)
	if err != nil {
		h.writeFAQError(w, err)
		return
	}

	httpresponse.WriteJSON(w, http.StatusCreated, faq)
}

// UpdateFAQ частично обновляет запись FAQ по ID.
// @Summary      Обновить FAQ
// @Description  Частично обновляет вопрос-ответ по ID. Требует авторизации.
// @Tags         admin-faq
// @Accept       json
// @Produce      json
// @Param        id   path      int                      true  "ID записи FAQ"
// @Param        faq  body      models.UpdateFAQRequest  true  "Поля для обновления"
// @Success      200  {object}  models.FAQ
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      404  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/faq/{id} [patch]
func (h *FAQHandler) UpdateFAQ(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	faqID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || faqID <= 0 {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid faq id")
		return
	}

	var req models.UpdateFAQRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req == (models.UpdateFAQRequest{}) {
		httpresponse.WriteError(w, http.StatusBadRequest, "at least one field is required")
		return
	}

	faq, err := h.faqService.Update(faqID, req)
	if err != nil {
		h.writeFAQError(w, err)
		return
	}

	// Инвалидация кеша: следующий публичный GET /faq/{id} перечитает из БД.
	h.invalidateFAQCache(r, faqID)

	httpresponse.WriteJSON(w, http.StatusOK, faq)
}

// DeleteFAQ удаляет запись FAQ по ID.
// @Summary      Удалить FAQ
// @Description  Удаляет вопрос-ответ по ID. Требует авторизации.
// @Tags         admin-faq
// @Param        id  path  int  true  "ID записи FAQ"
// @Success      204  "No Content"
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      404  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/faq/{id} [delete]
func (h *FAQHandler) DeleteFAQ(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	faqID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || faqID <= 0 {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid faq id")
		return
	}

	if err := h.faqService.Delete(faqID); err != nil {
		h.writeFAQError(w, err)
		return
	}

	h.invalidateFAQCache(r, faqID)

	w.WriteHeader(http.StatusNoContent)
}

// invalidateFAQCache сбрасывает кеш публичного GET /api/v1/faq/{id}.
func (h *FAQHandler) invalidateFAQCache(r *http.Request, faqID int) {
	cacheKey := redis.GenerateKey(redis.KeyPrefixFAQ, strconv.Itoa(faqID))
	if err := h.redisClient.Delete(r.Context(), cacheKey); err != nil {
		h.log.WithError(err).Error("failed to invalidate faq cache")
	}
}

// writeFAQError сопоставляет ошибки сервисного слоя с HTTP статус-кодами через errors.Is
// по сигнальным ошибкам — тот же подход, что writeBannerError в banner_handler.go.
func (h *FAQHandler) writeFAQError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrInvalidFAQ):
		httpresponse.WriteError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, repository.ErrFAQNotFound):
		httpresponse.WriteError(w, http.StatusNotFound, "faq not found")
	default:
		h.log.WithError(err).Error("faq operation failed")
		httpresponse.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
