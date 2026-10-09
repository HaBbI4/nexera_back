package main

import (
	"context"
	"fmt"
	"nexera_back/internal/config"
	"nexera_back/internal/database"
	_ "nexera_back/internal/docs"
	"nexera_back/internal/handlers"
	"nexera_back/internal/handlers/admin"
	"nexera_back/internal/handlers/public"
	"nexera_back/internal/kafka"
	"nexera_back/internal/logger"
	"nexera_back/internal/media"
	"nexera_back/internal/middleware"
	"nexera_back/internal/redis"
	"nexera_back/internal/repository"
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

	if cfg.Auth.JWTSecret == "" {
		log.Fatal("JWT_SECRET is required")
	}

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
	userRepo := repository.NewUserRepository(db)
	userService := services.NewUserService(userRepo, log)
	bannerRepo := repository.NewBannerRepository(db)
	bannerService := services.NewBannerService(bannerRepo, log)

	// Инициализация handlers
	healthHandler := handlers.NewHealthHandler(db, redisClient)
	carHandler := handlers.NewCarHandler(carService, redisClient, log)
	customerHandler := handlers.NewCustomerHandler(customerService, redisClient, log)
	serviceHandler := handlers.NewServiceHandler(serviceService, redisClient, log)
	orderHandler := handlers.NewOrderHandler(orderService, orderMediaService, redisClient, log, cfg.Media.MaxFileSizeMB)
	jwtTTL := time.Duration(cfg.Auth.JWTTTLMinutes) * time.Minute
	authHandler := admin.NewAuthHandler(userService, cfg.Auth.JWTSecret, jwtTTL, log)
	bannerHandler := public.NewBannerHandler(bannerService, redisClient, log)
	adminBannerHandler := admin.NewBannerHandler(bannerService, redisClient, log)

	// Настройки HTTP-роутера
	mux := setupRoutes(
		healthHandler,
		carHandler,
		customerHandler,
		serviceHandler,
		orderHandler,
		authHandler,
		bannerHandler,
		adminBannerHandler,
		cfg.Auth.JWTSecret,
		cfg.Media.Local.Path,
		log)

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

// setupRoutes настраивает маршруты HTTP сервера.
//
// Маршруты разделены на два контура:
//   - публичные /api/v1/*        — читает/создает любой клиент сайта, только CORS
//   - админские /api/v1/admin/*  — требуют валидный JWT (middleware.RequireAuth),
//     часть — еще и конкретную роль (middleware.RequireRole)
//
// Хендлеры car/customer/service/order ниже относятся к старому (детейлинг) домену
// и доживают здесь временно — при переносе на баннеры/лиды/отзывы они переедут
// в internal/handlers/public и internal/handlers/admin по тому же принципу, что auth.
func setupRoutes(
	healthHandler *handlers.HealthHandler,
	carHandler *handlers.CarHandler,
	customerHandler *handlers.CustomerHandler,
	serviceHandler *handlers.ServiceHandler,
	orderHandler *handlers.OrderHandler,
	authHandler *admin.AuthHandler,
	bannerHandler *public.BannerHandler,
	adminBannerHandler *admin.BannerHandler,
	jwtSecret string,
	localMediaPath string,
	log *logger.Logger,
) *http.ServeMux {
	mux := http.NewServeMux()

	logging := middleware.Logging(log)
	requireAuth := middleware.RequireAuth(jwtSecret)

	// Health check endpoints
	mux.HandleFunc("GET /health", middleware.Chain(healthHandler.Health, logging, middleware.CORS))
	mux.HandleFunc("GET /health/readiness", middleware.Chain(healthHandler.Readiness, logging, middleware.CORS))
	mux.HandleFunc("GET /health/liveness", middleware.Chain(healthHandler.Liveness, logging, middleware.CORS))

	// Admin auth endpoints
	mux.HandleFunc("POST /api/v1/admin/auth/login", middleware.Chain(authHandler.Login, logging, middleware.CORS))
	mux.HandleFunc("GET /api/v1/admin/auth/me", middleware.Chain(authHandler.Me, logging, middleware.CORS, requireAuth))

	// Banner's endpoints (публичные, без авторизации)
	mux.HandleFunc("GET /api/v1/banners", middleware.Chain(bannerHandler.GetBanners, logging, middleware.CORS))
	mux.HandleFunc("GET /api/v1/banners/{id}", middleware.Chain(bannerHandler.GetBanner, logging, middleware.CORS))

	// Banner's admin endpoints (требуют авторизации)
	mux.HandleFunc("POST /api/v1/admin/banners", middleware.Chain(adminBannerHandler.CreateBanner, logging, middleware.CORS, requireAuth))
	mux.HandleFunc("PATCH /api/v1/admin/banners/{id}", middleware.Chain(adminBannerHandler.UpdateBanner, logging, middleware.CORS, requireAuth))
	mux.HandleFunc("DELETE /api/v1/admin/banners/{id}", middleware.Chain(adminBannerHandler.DeleteBanner, logging, middleware.CORS, requireAuth))

	// Car's endpoints
	mux.HandleFunc("GET /api/v1/cars", middleware.Chain(carHandler.GetCars, logging, middleware.CORS))
	mux.HandleFunc("GET /api/v1/cars/{id}", middleware.Chain(carHandler.GetCar, logging, middleware.CORS))

	// Customer's endpoints
	mux.HandleFunc("GET /api/v1/customers", middleware.Chain(customerHandler.GetCustomers, logging, middleware.CORS))
	mux.HandleFunc("GET /api/v1/customers/{id}", middleware.Chain(customerHandler.GetCustomer, logging, middleware.CORS))
	mux.HandleFunc("POST /api/v1/customers", middleware.Chain(customerHandler.CreateCustomer, logging, middleware.CORS))

	// Service's endpoints
	mux.HandleFunc("POST /api/v1/services", middleware.Chain(serviceHandler.CreateService, logging, middleware.CORS))

	// Order's endpoints
	mux.HandleFunc("GET /api/v1/orders", middleware.Chain(orderHandler.GetOrders, logging, middleware.CORS))
	mux.HandleFunc("GET /api/v1/orders/{id}", middleware.Chain(orderHandler.GetOrder, logging, middleware.CORS))
	mux.HandleFunc("POST /api/v1/orders", middleware.Chain(orderHandler.CreateOrder, logging, middleware.CORS))
	mux.HandleFunc("POST /api/v1/orders/{id}/media", middleware.Chain(orderHandler.AddOrderMedia, logging, middleware.CORS))

	// Раздача локально сохранённых медиа-файлов (используется при MEDIA_DRIVER=local)
	mux.HandleFunc("GET /media/{path...}", middleware.Chain(handlers.NewLocalMediaHandler(localMediaPath), logging, middleware.CORS))

	// Swagger UI: интерактивная документация API
	mux.Handle("GET /swagger/", httpSwagger.WrapHandler)

	return mux
}
