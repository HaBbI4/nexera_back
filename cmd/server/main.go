package main

import (
	"context"
	"fmt"
	"nexera_back/internal/config"
	"nexera_back/internal/database"
	_ "nexera_back/internal/docs"
	"nexera_back/internal/handlers"
	"nexera_back/internal/kafka"
	"nexera_back/internal/logger"
	"nexera_back/internal/media"
	"nexera_back/internal/redis"
	"nexera_back/internal/services"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// @title           Detailing CRM API
// @version         1.0
// @description     API для системы учета и записи автомобилей на детейлинг услуги.
// @description     Позволяет управлять клиентами, автомобилями, услугами и заказами.

// @contact.name   API Support
// @contact.email  911gtrss@gmail.com

// @license.name  Proprietary

// @schemes http https
func main() {
	// Загрузка конфига
	cfg := config.Load()

	// Инициализация логирования
	log := logger.New(&cfg.Logger)
	log.Info("Starting system")

	// Подключение к базе данных
	db, err := database.Connect(&cfg.Database, log)
	if err != nil {
		log.WithError(err).Fatal("Failed to connect to database")
	}
	defer db.Close()

	// Подключение к Reids
	redisClient, err := redis.Connect(&cfg.Redis, log)
	if err != nil {
		log.WithError(err).Fatal("Failed to connect to redis")
	}
	defer redisClient.Close()

	// Подключение хранилища медиа (local/s3 — выбирается через MEDIA_DRIVER)
	mediaStorage, err := media.NewStorage(context.Background(), &cfg.Media)
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize media storage")
	}

	// Kafka producer — публикует события о заказах (например, для telegram-bot)
	kafkaProducer, err := kafka.NewProducer(&cfg.Kafka, log)
	if err != nil {
		log.WithError(err).Fatal("Failed to create Kafka producer")
	}
	defer kafkaProducer.Close()

	// Инициализация сервисов
	carService := services.NewCarService(db, log)
	customerService := services.NewCustomerService(db, log)
	serviceService := services.NewServiceService(db, log)
	orderService := services.NewOrderService(db, log, kafkaProducer)
	orderMediaService := services.NewOrderMediaService(db, mediaStorage, log)

	// Инициализация handlers
	healthHandler := handlers.NewHealthHandler(db, redisClient)
	carHandler := handlers.NewCarHandler(carService, redisClient, log)
	customerHandler := handlers.NewCustomerHandler(customerService, redisClient, log)
	serviceHandler := handlers.NewServiceHandler(serviceService, redisClient, log)
	orderHandler := handlers.NewOrderHandler(orderService, orderMediaService, redisClient, log, cfg.Media.MaxFileSizeMB)

	// Настройки HTTP-роутера
	mux := setupRoutes(
		healthHandler,
		carHandler,
		customerHandler,
		serviceHandler,
		orderHandler,
		cfg.Media.Local.Path)

	server := http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	// Сервер в Go-рутине
	go func() {
		log.WithField("Address", server.Addr).Info("Starting server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.WithError(err).Fatal("Failed to start server")
		}
	}()

	// Ожидание сигнала завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.WithError(err).Fatal("Failed to shutdown server")
	}

	log.Info("Server gracefully stopped")
}

// setupRoutes настраивает маршруты HTTP сервера
func setupRoutes(
	healthHandler *handlers.HealthHandler,
	carHandler *handlers.CarHandler,
	customerHandler *handlers.CustomerHandler,
	serviceHandler *handlers.ServiceHandler,
	orderHandler *handlers.OrderHandler,
	localMediaPath string,
) *http.ServeMux {
	mux := http.NewServeMux()

	// Health check endpoints
	mux.HandleFunc("GET /health", corsMiddleware(healthHandler.Health))
	mux.HandleFunc("GET /health/readiness", corsMiddleware(healthHandler.Readiness))
	mux.HandleFunc("GET /health/liveness", corsMiddleware(healthHandler.Liveness))

	// Car's endpoints
	mux.HandleFunc("GET /api/v1/cars", corsMiddleware(carHandler.GetCars))
	mux.HandleFunc("GET /api/v1/cars/{id}", corsMiddleware(carHandler.GetCar))

	// Customer's endpoints
	mux.HandleFunc("GET /api/v1/customers", corsMiddleware(customerHandler.GetCustomers))
	mux.HandleFunc("GET /api/v1/customers/{id}", corsMiddleware(customerHandler.GetCustomer))
	mux.HandleFunc("POST /api/v1/customers", corsMiddleware(customerHandler.CreateCustomer))

	// Service's endpoints
	mux.HandleFunc("POST /api/v1/services", corsMiddleware(serviceHandler.CreateService))

	// Order's endpoints
	mux.HandleFunc("GET /api/v1/orders", corsMiddleware(orderHandler.GetOrders))
	mux.HandleFunc("GET /api/v1/orders/{id}", corsMiddleware(orderHandler.GetOrder))
	mux.HandleFunc("POST /api/v1/orders", corsMiddleware(orderHandler.CreateOrder))
	mux.HandleFunc("POST /api/v1/orders/{id}/media", corsMiddleware(orderHandler.AddOrderMedia))

	// Раздача локально сохранённых медиа-файлов (используется при MEDIA_DRIVER=local)
	mux.HandleFunc("GET /media/{path...}", corsMiddleware(handlers.NewLocalMediaHandler(localMediaPath)))

	// Swagger UI: интерактивная документация API
	mux.Handle("GET /swagger/", httpSwagger.WrapHandler)

	return mux
}

// corsMiddleware и другие helper функции
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	fmt.Fprintf(w, `{"error": "%s", "message": "%s"}`, http.StatusText(statusCode), message)
}
