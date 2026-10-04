package handlers

import (
	"encoding/json"
	"fmt"
	"nexera_back/internal/logger"
	"nexera_back/internal/models"
	"nexera_back/internal/redis"
	"nexera_back/internal/services"
	"net/http"
	"strconv"
	"strings"
)

// CustomerHandler представляет обработчик клиентов
type CustomerHandler struct {
	customerService *services.CustomerService
	redisClient     *redis.Client
	log             *logger.Logger
}

// NewCustomerHandler создает новый обработчик клиентов
func NewCustomerHandler(
	customerService *services.CustomerService,
	redisClient *redis.Client,
	log *logger.Logger,
) *CustomerHandler {
	return &CustomerHandler{
		customerService: customerService,
		redisClient:     redisClient,
		log:             log,
	}
}

// GetCustomers получает список клиентов
// @Summary      Список клиентов
// @Description  Возвращает список клиентов с пагинацией
// @Tags         customers
// @Produce      json
// @Param        limit   query     int  false  "Количество записей (по умолчанию 50, максимум 100)"
// @Param        offset  query     int  false  "Смещение (по умолчанию 0)"
// @Success      200  {array}   models.Customer
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/customers [get]
func (h *CustomerHandler) GetCustomers(w http.ResponseWriter, r *http.Request) {
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

	customers, err := h.customerService.GetCustomers(limit, offset)
	if err != nil {
		h.log.WithError(err).Error("Error getting customers")
		writeErrorResponse(w, http.StatusInternalServerError, "Error getting customers")
		return
	}

	writeJSONResponse(w, http.StatusOK, customers)
}

// GetCustomer получает клиента по ID
// @Summary      Получить клиента
// @Description  Возвращает клиента по ID
// @Tags         customers
// @Produce      json
// @Param        id   path      int  true  "ID клиента"
// @Success      200  {object}  models.Customer
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/customers/{id} [get]
func (h *CustomerHandler) GetCustomer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	customerID, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || customerID <= 0 {
		writeErrorResponse(w, http.StatusBadRequest, "invalid customer id")
		return
	}

	// Попытка получить из кеша
	cacheKey := redis.GenerateKey(redis.KeyPrefixCustomer, strconv.Itoa(customerID))
	var customer models.Customer
	if err := h.redisClient.Get(r.Context(), cacheKey, &customer); err == nil {
		h.log.WithField("customer_id", customerID).Debug("Customer retrieved from cache")
		writeJSONResponse(w, http.StatusOK, &customer)
		return
	}

	// Получение из баз данных
	customerPtr, err := h.customerService.GetCustomer(customerID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			writeErrorResponse(w, http.StatusNotFound, "Car not found")
		} else {
			h.log.WithError(err).Error("Failed to get customer")
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to get customer")
		}
		return
	}

	// Кэширование клиента
	if err := h.redisClient.Set(r.Context(), cacheKey, customerPtr, defaultCacheTTL); err != nil {
		h.log.WithError(err).Error("Failed to set customer")
	}

	writeJSONResponse(w, http.StatusOK, customerPtr)
}

// CreateCustomer создает клиента
// @Summary      Создать клиента
// @Description  Создает нового клиента
// @Tags         customers
// @Accept       json
// @Produce      json
// @Param        request  body      models.CreateCustomerRequest  true  "Данные клиента"
// @Success      201  {object}  models.Customer
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /api/v1/customers [post]
func (h *CustomerHandler) CreateCustomer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorResponse(w, http.StatusMethodNotAllowed, "Method Not Allowed")
		return
	}

	var req models.CreateCustomerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.validateCreateCustomerRequest(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Создание клиента
	customer, err := h.customerService.CreateCustomer(&req)
	if err != nil {
		h.log.WithError(err).Error("Failed to create customer")
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to create customer")
		return
	}

	// Кеширование клиента в Redis
	cacheKey := redis.GenerateKey(redis.KeyPrefixCustomer, strconv.Itoa(customer.ID))
	if err := h.redisClient.Set(r.Context(), cacheKey, customer, defaultCacheTTL); err != nil {
		h.log.WithError(err).Error("Failed to cache customer")
	}

	writeJSONResponse(w, http.StatusCreated, customer)
}

// validateCreateCustomerRequest валидирует запрос на создание клиента
func (h *CustomerHandler) validateCreateCustomerRequest(req *models.CreateCustomerRequest) error {
	if req.Name == "" {
		return fmt.Errorf("courier name is required")
	}
	if req.Phone == "" {
		return fmt.Errorf("courier phone is required")
	}
	return nil
}
