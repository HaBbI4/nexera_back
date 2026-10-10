package admin

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"nexera_back/internal/httpresponse"
	"nexera_back/internal/logger"
	"nexera_back/internal/models"
	"nexera_back/internal/repository"
	"nexera_back/internal/services"
)

// allowedReviewSortFields — белый список значений query-параметра ?sort= для админского
// списка отзывов (тот же набор, что в публичном internal/handlers/public).
var allowedReviewSortFields = map[string]repository.ReviewSortField{
	"created_at": repository.ReviewSortCreatedAt,
	"rating":     repository.ReviewSortRating,
}

// ReviewHandler дает админ-панели просматривать отзывы (вместе с контактами клиента),
// модерировать (одобрять и снимать одобрение) и удалять их. Создания и редактирования
// здесь нет: отзывы оставляют клиенты через публичный POST /api/v1/reviews
// (internal/handlers/public.ReviewHandler), а текст клиента админ не правит.
// Кеша нет: публичного GET по ID у отзывов нет, инвалидировать нечего.
type ReviewHandler struct {
	reviewService *services.ReviewService
	log           *logger.Logger
}

// NewReviewHandler создает новый обработчик отзывов админ-панели.
func NewReviewHandler(reviewService *services.ReviewService, log *logger.Logger) *ReviewHandler {
	return &ReviewHandler{
		reviewService: reviewService,
		log:           log,
	}
}

// GetReviews возвращает список отзывов для админ-панели с фильтрацией, сортировкой и
// пагинацией — вне зависимости от одобрения, вместе с телефоном и email клиента.
//
// Как и в GET /api/v1/admin/leads: limit/offset при некорректном значении молча заменяются
// значением по умолчанию, а фильтры и сортировка — строже, некорректное значение — это 400.
// По умолчанию сначала новые отзывы (created_at desc).
//
// @Summary      Список отзывов (админка)
// @Description  Возвращает отзывы клиентов с контактными данными, включая неодобренные (очередь модерации — ?is_approved=false). Удаленные отзывы не возвращаются. Требует авторизации.
// @Tags         admin-reviews
// @Produce      json
// @Param        is_approved   query     bool    false  "true — только одобренные, false — только ожидающие модерации"
// @Param        rating_min    query     int     false  "Оценка не ниже (1-5)"
// @Param        created_from  query     string  false  "Создан не раньше, RFC3339"
// @Param        created_to    query     string  false  "Создан не позже, RFC3339"
// @Param        sort          query     string  false  "Поле сортировки: created_at (по умолчанию) или rating"
// @Param        order         query     string  false  "Направление сортировки: desc (по умолчанию) или asc"
// @Param        limit         query     int     false  "Количество записей (по умолчанию 50, максимум 100)"
// @Param        offset        query     int     false  "Смещение (по умолчанию 0)"
// @Success      200  {array}   models.Review
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/reviews [get]
func (h *ReviewHandler) GetReviews(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	query := r.URL.Query()

	filter := services.ListAdminReviewsFilter{
		Sort:  repository.ReviewSortCreatedAt,
		Order: repository.SortDesc,
	}

	if isApprovedStr := query.Get("is_approved"); isApprovedStr != "" {
		isApproved, err := strconv.ParseBool(isApprovedStr)
		if err != nil {
			httpresponse.WriteError(w, http.StatusBadRequest, "invalid is_approved")
			return
		}
		filter.IsApproved = &isApproved
	}

	if ratingMinStr := query.Get("rating_min"); ratingMinStr != "" {
		ratingMin, err := strconv.Atoi(ratingMinStr)
		if err != nil || ratingMin < 1 || ratingMin > 5 {
			httpresponse.WriteError(w, http.StatusBadRequest, "invalid rating_min, allowed: 1-5")
			return
		}
		filter.RatingMin = &ratingMin
	}

	if createdFromStr := query.Get("created_from"); createdFromStr != "" {
		createdFrom, err := time.Parse(time.RFC3339, createdFromStr)
		if err != nil {
			httpresponse.WriteError(w, http.StatusBadRequest, "invalid created_from, expected RFC3339")
			return
		}
		filter.CreatedFrom = &createdFrom
	}

	if createdToStr := query.Get("created_to"); createdToStr != "" {
		createdTo, err := time.Parse(time.RFC3339, createdToStr)
		if err != nil {
			httpresponse.WriteError(w, http.StatusBadRequest, "invalid created_to, expected RFC3339")
			return
		}
		filter.CreatedTo = &createdTo
	}

	if filter.CreatedFrom != nil && filter.CreatedTo != nil && filter.CreatedTo.Before(*filter.CreatedFrom) {
		httpresponse.WriteError(w, http.StatusBadRequest, "created_to must not be before created_from")
		return
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

	reviews, err := h.reviewService.List(filter)
	if err != nil {
		h.log.WithError(err).Error("failed to get reviews")
		httpresponse.WriteError(w, http.StatusInternalServerError, "failed to get reviews")
		return
	}

	httpresponse.WriteJSON(w, http.StatusOK, reviews)
}

// GetReview возвращает отзыв по ID для админ-панели — вне зависимости от одобрения.
// @Summary      Получить отзыв (админка)
// @Description  Возвращает отзыв по ID вместе с контактными данными клиента, включая неодобренный. Требует авторизации.
// @Tags         admin-reviews
// @Produce      json
// @Param        id   path      int  true  "ID отзыва"
// @Success      200  {object}  models.Review
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      404  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/reviews/{id} [get]
func (h *ReviewHandler) GetReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	reviewID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || reviewID <= 0 {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid review id")
		return
	}

	review, err := h.reviewService.GetByID(reviewID)
	if err != nil {
		h.writeReviewError(w, err)
		return
	}

	httpresponse.WriteJSON(w, http.StatusOK, review)
}

// ApproveReview одобряет отзыв — после этого он виден на сайте.
// @Summary      Одобрить отзыв
// @Description  Одобряет отзыв по ID — он появляется на сайте. Повторное одобрение не ошибка. Требует авторизации.
// @Tags         admin-reviews
// @Produce      json
// @Param        id   path      int  true  "ID отзыва"
// @Success      200  {object}  models.Review
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      404  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/reviews/{id}/approve [post]
func (h *ReviewHandler) ApproveReview(w http.ResponseWriter, r *http.Request) {
	h.setApproved(w, r, h.reviewService.Approve)
}

// UnapproveReview снимает одобрение с отзыва — он пропадает с сайта, но остается в админке.
// @Summary      Снять одобрение с отзыва
// @Description  Снимает одобрение с отзыва по ID — он пропадает с сайта, но не удаляется. Для неодобренного отзыва не ошибка. Требует авторизации.
// @Tags         admin-reviews
// @Produce      json
// @Param        id   path      int  true  "ID отзыва"
// @Success      200  {object}  models.Review
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      404  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/reviews/{id}/unapprove [post]
func (h *ReviewHandler) UnapproveReview(w http.ResponseWriter, r *http.Request) {
	h.setApproved(w, r, h.reviewService.Unapprove)
}

// setApproved — общая часть ApproveReview/UnapproveReview: они отличаются только
// вызываемым методом сервиса.
func (h *ReviewHandler) setApproved(w http.ResponseWriter, r *http.Request, action func(id int) (*models.Review, error)) {
	if r.Method != http.MethodPost {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	reviewID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || reviewID <= 0 {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid review id")
		return
	}

	review, err := action(reviewID)
	if err != nil {
		h.writeReviewError(w, err)
		return
	}

	httpresponse.WriteJSON(w, http.StatusOK, review)
}

// DeleteReview удаляет отзыв по ID.
// @Summary      Удалить отзыв
// @Description  Удаляет отзыв по ID: он пропадает и с сайта, и из админки. Требует авторизации.
// @Tags         admin-reviews
// @Param        id  path  int  true  "ID отзыва"
// @Success      204  "No Content"
// @Failure      400  {object}  httpresponse.ErrorResponse
// @Failure      401  {object}  httpresponse.ErrorResponse
// @Failure      404  {object}  httpresponse.ErrorResponse
// @Failure      500  {object}  httpresponse.ErrorResponse
// @Router       /api/v1/admin/reviews/{id} [delete]
func (h *ReviewHandler) DeleteReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		httpresponse.WriteError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	reviewID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || reviewID <= 0 {
		httpresponse.WriteError(w, http.StatusBadRequest, "invalid review id")
		return
	}

	if err := h.reviewService.Delete(reviewID); err != nil {
		h.writeReviewError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// writeReviewError сопоставляет ошибки сервисного слоя с HTTP статус-кодами через errors.Is
// по сигнальным ошибкам — тот же подход, что writeBannerError в banner_handler.go.
func (h *ReviewHandler) writeReviewError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrReviewNotFound):
		httpresponse.WriteError(w, http.StatusNotFound, "review not found")
	default:
		h.log.WithError(err).Error("review operation failed")
		httpresponse.WriteError(w, http.StatusInternalServerError, "internal server error")
	}
}
