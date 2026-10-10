package public

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

// maxReviewBodyBytes ограничивает размер тела публичного POST /reviews: эндпоинт без
// авторизации, а самый длинный отзыв (text до 5000 символов) укладывается в ~20KB.
const maxReviewBodyBytes = 64 << 10 // 64KB

// allowedReviewSortFields — белый список значений query-параметра ?sort=.
// Отдельный от repository.reviewSortColumns: тот определяет SQL-колонки,
// этот — что разрешено принять от клиента (невалидное значение -> 400, а не тихий дефолт).
var allowedReviewSortFields = map[string]repository.ReviewSortField{
	"created_at": repository.ReviewSortCreatedAt,
	"rating":     repository.ReviewSortRating,
}

// ReviewHandler отдает одобренные отзывы на сайт и принимает новые отзывы от клиентов.
// Без авторизации — доступен любому клиенту. Отзывы наружу уходят только как
// models.PublicReview — без телефона и email; полный отзыв с контактами — только в админке,
// см. internal/handlers/admin.ReviewHandler. Кеша нет: GET по ID у отзывов нет,
// а списки в проекте не кешируются.
type ReviewHandler struct {
	reviewService *services.ReviewService
	log           *logger.Logger
}

// NewReviewHandler создает новый обработчик отзывов.
func NewReviewHandler(reviewService *services.ReviewService, log *logger.Logger) *ReviewHandler {
	return &ReviewHandler{
		reviewService: reviewService,
		log:           log,
	}
}

// GetReviews возвращает список одобренных отзывов с фильтрацией, сортировкой и пагинацией.
//
// Как и в GET /api/v1/banners: limit/offset при некорректном значении молча заменяются
// значением по умолчанию, а фильтры и сортировка — строже, некорректное значение — это 400.
// По умолчанию сначала новые отзывы (created_at desc).
//
// @Summary      Список отзывов
// @Description  Возвращает одобренные отзывы клиентов (без контактных данных) с фильтрацией по минимальной оценке, сортировкой и пагинацией.
// @Tags         reviews
// @Produce      json
// @Param        rating_min  query     int     false  "Оценка не ниже (1-5)"
// @Param        sort        query     string  false  "Поле сортировки: created_at (по умолчанию) или rating"
// @Param        order       query     string  false  "Направление сортировки: desc (по умолчанию) или asc"
// @Param        limit       query     int     false  "Количество записей (по умолчанию 50, максимум 100)"
// @Param        offset      query     int     false  "Смещение (по умолчанию 0)"
// @Success      200  {array}   models.PublicReview
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/reviews [get]
func (h *ReviewHandler) GetReviews(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	query := r.URL.Query()

	filter := services.ListPublicReviewsFilter{
		Sort:  repository.ReviewSortCreatedAt,
		Order: repository.SortDesc,
	}

	if ratingMinStr := query.Get("rating_min"); ratingMinStr != "" {
		ratingMin, err := strconv.Atoi(ratingMinStr)
		if err != nil || ratingMin < 1 || ratingMin > 5 {
			httpresponse.WriteError(w, http.StatusBadRequest, "invalid rating_min, allowed: 1-5")
			return
		}
		filter.RatingMin = &ratingMin
	}

	if sortStr := query.Get("sort"); sortStr != "" {
		sortField, ok := allowedReviewSortFields[sortStr]
		if !ok {
			httpresponse.WriteError(w, http.StatusBadRequest, "invalid sort, allowed: created_at, rating")
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

	reviews, err := h.reviewService.ListPublic(filter)
	if err != nil {
		h.log.WithError(err).Error("failed to get reviews")
		httpresponse.WriteError(w, http.StatusInternalServerError, "failed to get reviews")
		return
	}

	httpresponse.WriteJSON(w, http.StatusOK, reviews)
}

// GetReviewsSummary возвращает количество и средний рейтинг одобренных отзывов.
// @Summary      Сводка по отзывам
// @Description  Возвращает количество одобренных отзывов и их среднюю оценку (округлена до 0.1; 0, если отзывов нет) — для блока "4.8 из 5, 37 отзывов".
// @Tags         reviews
// @Produce      json
// @Success      200  {object}  models.ReviewSummary
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/reviews/summary [get]
func (h *ReviewHandler) GetReviewsSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	summary, err := h.reviewService.GetPublicSummary()
	if err != nil {
		h.log.WithError(err).Error("failed to get reviews summary")
		httpresponse.WriteError(w, http.StatusInternalServerError, "failed to get reviews summary")
		return
	}

	httpresponse.WriteJSON(w, http.StatusOK, summary)
}

// CreateReview создает отзыв клиента. Отзыв появится на сайте только после одобрения
// в админке; в ответе — публичное представление отзыва, как его потом увидят на сайте.
// @Summary      Оставить отзыв
// @Description  Создает отзыв клиента с сайта. Отзыв публикуется только после модерации. Телефон — в международном формате (+79991234567), пробелы/скобки/дефисы допускаются. Оценка — от 1 до 5.
// @Tags         reviews
// @Accept       json
// @Produce      json
// @Param        review  body      models.CreateReviewRequest  true  "Данные отзыва"
// @Success      201  {object}  models.PublicReview
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      429  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/reviews [post]
func (h *ReviewHandler) CreateReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxReviewBodyBytes)

	var req models.CreateReviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	review, err := h.reviewService.Create(req)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrSpamDetected):
			// Бот получает обычный успешный ответ и не узнает, что отзыв не сохранен.
			httpresponse.WriteJSON(w, http.StatusCreated, struct{}{})
		case errors.Is(err, services.ErrDuplicateSubmission):
			httpresponse.WriteError(w, http.StatusTooManyRequests, httpresponse.TooManyRequestsMessage)
		case errors.Is(err, services.ErrInvalidReview):
			httpresponse.WriteError(w, http.StatusBadRequest, err.Error())
		default:
			h.log.WithError(err).Error("failed to create review")
			httpresponse.WriteError(w, http.StatusInternalServerError, "failed to create review")
		}
		return
	}

	httpresponse.WriteJSON(w, http.StatusCreated, review.ToPublic())
}
