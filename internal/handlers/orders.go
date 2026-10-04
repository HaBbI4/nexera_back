package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"nexera_back/internal/logger"
	"nexera_back/internal/media"
	"nexera_back/internal/models"
	"nexera_back/internal/redis"
	"nexera_back/internal/services"
	"net/http"
	"strconv"
	"strings"
)

// maxMediaFilesPerRequest ограничивает количество файлов в одном запросе на загрузку медиа
const maxMediaFilesPerRequest = 10

// OrderHandler представляет обработчик заказов
type OrderHandler struct {
	orderService       *services.OrderService
	orderMediaService  *services.OrderMediaService
	redisClient        *redis.Client
	log                *logger.Logger
	maxMediaFileSizeMB int64
}

// NewOrderHandler создает новый обработчик заказов
func NewOrderHandler(
	orderService *services.OrderService,
	orderMediaService *services.OrderMediaService,
	redisClient *redis.Client,
	log *logger.Logger,
	maxMediaFileSizeMB int64,
) *OrderHandler {
	return &OrderHandler{
		orderService:       orderService,
		orderMediaService:  orderMediaService,
		redisClient:        redisClient,
		log:                log,
		maxMediaFileSizeMB: maxMediaFileSizeMB,
	}
}

// GetOrders получает список заказов
// @Summary      Список заказов
// @Description  Возвращает список заказов с автомобилем, клиентом, услугами и медиа
// @Tags         orders
// @Produce      json
// @Param        limit   query     int  false  "Количество записей (по умолчанию 50, максимум 100)"
// @Param        offset  query     int  false  "Смещение (по умолчанию 0)"
// @Success      200  {array}   models.Order
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/orders [get]
func (h *OrderHandler) GetOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	query := r.URL.Query()

	// Парсинг параметров
	limit := 50 // По умолчанию
	if limitStr := query.Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offset := 0
	if offsetStr := query.Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	orders, err := h.orderService.GetOrders(limit, offset)
	if err != nil {
		h.log.WithError(err).Error("Error getting orders")
		writeErrorResponse(w, http.StatusInternalServerError, "Error getting orders")
		return
	}

	for _, order := range orders {
		media, err := h.orderMediaService.GetMediaByOrderID(r.Context(), order.ID)
		if err != nil {
			h.log.WithError(err).Error("Failed to get order media")
			writeErrorResponse(w, http.StatusInternalServerError, "Error getting orders")
			return
		}
		order.Media = media
	}

	writeJSONResponse(w, http.StatusOK, orders)
}

// GetOrder получает заказ по ID
// @Summary      Получить заказ
// @Description  Возвращает заказ по ID вместе с автомобилем, клиентом, услугами и медиа
// @Tags         orders
// @Produce      json
// @Param        id   path      int  true  "ID заказа"
// @Success      200  {object}  models.Order
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/orders/{id} [get]
func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	orderID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || orderID <= 0 {
		writeErrorResponse(w, http.StatusBadRequest, "invalid order id")
		return
	}

	// Попытка получить из кеша
	cacheKey := redis.GenerateKey(redis.KeyPrefixOrder, strconv.Itoa(orderID))
	var order models.Order
	if err := h.redisClient.Get(r.Context(), cacheKey, &order); err == nil {
		h.log.WithField("order_id", orderID).Debug("Order retrieved from cache")
		writeJSONResponse(w, http.StatusOK, &order)
		return
	}

	// Получение из баз данных
	orderPtr, err := h.orderService.GetOrder(orderID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeErrorResponse(w, http.StatusNotFound, "Order not found")
		} else {
			h.log.WithError(err).Error("Failed to get order")
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to get order")
		}
		return
	}

	media, err := h.orderMediaService.GetMediaByOrderID(r.Context(), orderID)
	if err != nil {
		h.log.WithError(err).Error("Failed to get order media")
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to get order")
		return
	}
	orderPtr.Media = media

	// Кэширование заказа
	if err := h.redisClient.Set(r.Context(), cacheKey, orderPtr, defaultCacheTTL); err != nil {
		h.log.WithError(err).Error("Failed to set order")
	}

	writeJSONResponse(w, http.StatusOK, orderPtr)
}

// CreateOrder создает заказ
// @Summary      Создать заказ
// @Description  Создает заказ с услугами. В каждом item указывается либо service_id (существующая услуга), либо service_name (+ опционально service_description, service_is_active) для создания новой услуги на лету.
// @Tags         orders
// @Accept       json
// @Produce      json
// @Param        request  body      models.CreateOrderRequest  true  "Данные заказа"
// @Success      201  {object}  models.Order
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/orders [post]
func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Валидация запроса
	if err := h.validateCreateRequest(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Создание заказа
	order, err := h.orderService.CreateOrder(&req)
	if err != nil {
		h.log.WithError(err).Error("Failed to create order")
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to create order")
		return
	}

	// Кэширование заказа в Redis
	cacheKey := redis.GenerateKey(redis.KeyPrefixOrder, strconv.Itoa(order.ID))
	if err := h.redisClient.Set(r.Context(), cacheKey, order, defaultCacheTTL); err != nil {
		h.log.WithError(err).Error("Failed to set cache")
	}

	writeJSONResponse(w, http.StatusCreated, order)
}

// AddOrderMedia добавляет фото/видео к уже созданному заказу. Медиа необязательны —
// заказ можно создать и обслужить, ни разу не вызвав этот эндпоинт.
// @Summary      Добавить медиа к заказу
// @Description  Загружает фото/видео к уже созданному заказу (до 10 файлов за запрос)
// @Tags         orders
// @Accept       multipart/form-data
// @Produce      json
// @Param        id     path      int     true   "ID заказа"
// @Param        stage  formData  string  false  "Этап съемки: before или after (по умолчанию before)"
// @Param        files  formData  file    true   "Файлы (фото/видео)"
// @Success      201  {array}   models.OrderMedia
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      415  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/orders/{id}/media [post]
func (h *OrderHandler) AddOrderMedia(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	orderID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || orderID <= 0 {
		writeErrorResponse(w, http.StatusBadRequest, "invalid order id")
		return
	}

	maxRequestSize := h.maxMediaFileSizeMB * maxMediaFilesPerRequest * 1024 * 1024
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestSize)

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid multipart form or request too large")
		return
	}
	defer r.MultipartForm.RemoveAll()

	stage := r.FormValue("stage")
	if stage == "" {
		stage = "before"
	}
	if stage != "before" && stage != "after" {
		writeErrorResponse(w, http.StatusBadRequest, "stage must be 'before' or 'after'")
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		writeErrorResponse(w, http.StatusBadRequest, "at least one file is required")
		return
	}
	if len(files) > maxMediaFilesPerRequest {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("too many files, max %d per request", maxMediaFilesPerRequest))
		return
	}

	maxFileSizeBytes := h.maxMediaFileSizeMB * 1024 * 1024
	results := make([]*models.OrderMedia, 0, len(files))

	for _, fh := range files {
		if fh.Size > maxFileSizeBytes {
			writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("file %q exceeds max size of %dMB", fh.Filename, h.maxMediaFileSizeMB))
			return
		}

		result, err := h.addOneMedia(r, orderID, stage, fh)
		if err != nil {
			h.writeMediaError(w, err)
			return
		}

		results = append(results, result)
	}

	writeJSONResponse(w, http.StatusCreated, results)
}

func (h *OrderHandler) addOneMedia(r *http.Request, orderID int, stage string, fh *multipart.FileHeader) (*models.OrderMedia, error) {
	f, err := fh.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer f.Close()

	return h.orderMediaService.AddMedia(r.Context(), orderID, stage, fh.Size, f)
}

func (h *OrderHandler) writeMediaError(w http.ResponseWriter, err error) {
	var unsupportedFormat *media.ErrUnsupportedFormat

	switch {
	case errors.Is(err, services.ErrOrderNotFound):
		writeErrorResponse(w, http.StatusNotFound, "order not found")
	case errors.As(err, &unsupportedFormat):
		writeErrorResponse(w, http.StatusUnsupportedMediaType, err.Error())
	default:
		h.log.WithError(err).Error("failed to add order media")
		writeErrorResponse(w, http.StatusInternalServerError, "failed to add order media")
	}
}

// validateCreateOrderRequest валидирует запрос на создание заказа
func (h *OrderHandler) validateCreateRequest(req *models.CreateOrderRequest) error {
	if req.CustomerID <= 0 {
		return fmt.Errorf("customer is required")
	}
	if req.CarID <= 0 {
		return fmt.Errorf("car is required")
	}
	if len(req.Items) == 0 {
		return fmt.Errorf("order items are required")
	}

	for i, item := range req.Items {
		if item.Quantity <= 0 {
			return fmt.Errorf("item %d: quantity must be positive", i+1)
		}
		if item.Price < 0 {
			return fmt.Errorf("item %d: price cannot be negative", i+1)
		}
		if item.ServiceID <= 0 && item.ServiceName == "" {
			return fmt.Errorf("item %d: service_id or service_name is required", i+1)
		}
	}

	return nil
}
