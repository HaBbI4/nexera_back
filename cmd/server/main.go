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
	"nexera_back/internal/models"
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

	// Определение реального IP клиента за nginx — для ограничения частоты запросов
	clientIPResolver, err := middleware.NewClientIPResolver(cfg.Spam.TrustedProxies)
	if err != nil {
		log.WithError(err).Fatal("Invalid TRUSTED_PROXIES")
	}

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
	orderService := services.NewOrderService(db, log, kafkaProducer)
	orderMediaService := services.NewOrderMediaService(db, mediaStorage, log)
	userRepo := repository.NewUserRepository(db)
	userService := services.NewUserService(userRepo, log)
	roleRepo := repository.NewRoleRepository(db)
	roleService := services.NewRoleService(roleRepo, log)
	faqRepo := repository.NewFAQRepository(db)
	faqService := services.NewFAQService(faqRepo, log)
	leadRepo := repository.NewLeadRepository(db)
	leadService := services.NewLeadService(leadRepo, time.Duration(cfg.Spam.LeadDuplicateWindowMinutes)*time.Minute, log)
	serviceRepo := repository.NewServiceRepository(db)
	serviceService := services.NewServiceService(serviceRepo, log)
	reviewRepo := repository.NewReviewRepository(db)
	reviewService := services.NewReviewService(reviewRepo, time.Duration(cfg.Spam.ReviewDuplicateWindowMinutes)*time.Minute, log)
	bannerRepo := repository.NewBannerRepository(db)
	bannerService := services.NewBannerService(bannerRepo, log)
	pageRepo := repository.NewPageRepository(db)
	pageService := services.NewPageService(pageRepo, bannerService, log)
	telegramUserRepo := repository.NewTelegramUserRepository(db)
	telegramUserService := services.NewTelegramUserService(telegramUserRepo, log)

	// Инициализация handlers
	healthHandler := handlers.NewHealthHandler(db, redisClient)
	carHandler := handlers.NewCarHandler(carService, redisClient, log)
	customerHandler := handlers.NewCustomerHandler(customerService, redisClient, log)
	orderHandler := handlers.NewOrderHandler(orderService, orderMediaService, redisClient, log, cfg.Media.MaxFileSizeMB)
	jwtTTL := time.Duration(cfg.Auth.JWTTTLMinutes) * time.Minute
	authHandler := admin.NewAuthHandler(userService, cfg.Auth.JWTSecret, jwtTTL, log)
	bannerHandler := public.NewBannerHandler(bannerService, redisClient, log)
	adminBannerHandler := admin.NewBannerHandler(bannerService, redisClient, mediaStorage, cfg.Media.MaxFileSizeMB, log)
	pageHandler := public.NewPageHandler(pageService, redisClient, log)
	adminPageHandler := admin.NewPageHandler(pageService, redisClient, log)
	adminUserHandler := admin.NewUserHandler(userService, log)
	adminRoleHandler := admin.NewRoleHandler(roleService, log)
	faqHandler := public.NewFAQHandler(faqService, redisClient, log)
	adminFAQHandler := admin.NewFAQHandler(faqService, redisClient, log)
	leadHandler := public.NewLeadHandler(leadService, log)
	adminLeadHandler := admin.NewLeadHandler(leadService, log)
	serviceHandler := public.NewServiceHandler(serviceService, redisClient, log)
	adminServiceHandler := admin.NewServiceHandler(serviceService, redisClient, log)
	reviewHandler := public.NewReviewHandler(reviewService, log)
	adminReviewHandler := admin.NewReviewHandler(reviewService, log)
	adminTelegramUserHandler := admin.NewTelegramUserHandler(telegramUserService, log)

	// Настройки HTTP-роутера
	mux := setupRoutes(
		healthHandler,
		carHandler,
		customerHandler,
		orderHandler,
		authHandler,
		bannerHandler,
		adminBannerHandler,
		pageHandler,
		adminPageHandler,
		adminUserHandler,
		adminRoleHandler,
		faqHandler,
		adminFAQHandler,
		leadHandler,
		adminLeadHandler,
		serviceHandler,
		adminServiceHandler,
		reviewHandler,
		adminReviewHandler,
		adminTelegramUserHandler,
		redisClient,
		clientIPResolver,
		&cfg.Spam,
		cfg.Auth.JWTSecret,
		cfg.Media.Local.Path,
		log)

	server := http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler:      middleware.Preflight(mux),
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
// Хендлеры car/customer/order ниже относятся к старому (детейлинг) домену
// и доживают здесь временно — при переносе на баннеры/лиды/отзывы они переедут
// в internal/handlers/public и internal/handlers/admin по тому же принципу, что auth.
func setupRoutes(
	healthHandler *handlers.HealthHandler,
	carHandler *handlers.CarHandler,
	customerHandler *handlers.CustomerHandler,
	orderHandler *handlers.OrderHandler,
	authHandler *admin.AuthHandler,
	bannerHandler *public.BannerHandler,
	adminBannerHandler *admin.BannerHandler,
	pageHandler *public.PageHandler,
	adminPageHandler *admin.PageHandler,
	adminUserHandler *admin.UserHandler,
	adminRoleHandler *admin.RoleHandler,
	faqHandler *public.FAQHandler,
	adminFAQHandler *admin.FAQHandler,
	leadHandler *public.LeadHandler,
	adminLeadHandler *admin.LeadHandler,
	serviceHandler *public.ServiceHandler,
	adminServiceHandler *admin.ServiceHandler,
	reviewHandler *public.ReviewHandler,
	adminReviewHandler *admin.ReviewHandler,
	adminTelegramUserHandler *admin.TelegramUserHandler,
	redisClient *redis.Client,
	clientIPResolver *middleware.ClientIPResolver,
	spamCfg *config.SpamConfig,
	jwtSecret string,
	localMediaPath string,
	log *logger.Logger,
) *http.ServeMux {
	mux := http.NewServeMux()

	logging := middleware.Logging(log)
	requireAuth := middleware.RequireAuth(jwtSecret)
	requireAdmin := middleware.RequireRole(models.RoleCodeAdmin)

	// Ограничение частоты публичных форм с одного IP (счетчики в Redis)
	leadRateLimit := middleware.RateLimit(redisClient, middleware.RateLimitRule{
		Name:        "leads",
		MaxRequests: spamCfg.LeadRateLimit.MaxRequests,
		Window:      time.Duration(spamCfg.LeadRateLimit.WindowMinutes) * time.Minute,
	}, clientIPResolver, log)
	reviewRateLimit := middleware.RateLimit(redisClient, middleware.RateLimitRule{
		Name:        "reviews",
		MaxRequests: spamCfg.ReviewRateLimit.MaxRequests,
		Window:      time.Duration(spamCfg.ReviewRateLimit.WindowMinutes) * time.Minute,
	}, clientIPResolver, log)

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

	// Banner's admin endpoints (требуют авторизации; чтение — без фильтра видимости)
	mux.HandleFunc("GET /api/v1/admin/banners", middleware.Chain(adminBannerHandler.GetBanners, logging, middleware.CORS, requireAuth))
	mux.HandleFunc("GET /api/v1/admin/banners/{id}", middleware.Chain(adminBannerHandler.GetBanner, logging, middleware.CORS, requireAuth))
	mux.HandleFunc("POST /api/v1/admin/banners", middleware.Chain(adminBannerHandler.CreateBanner, logging, middleware.CORS, requireAuth))
	mux.HandleFunc("PATCH /api/v1/admin/banners/{id}", middleware.Chain(adminBannerHandler.UpdateBanner, logging, middleware.CORS, requireAuth))
	mux.HandleFunc("DELETE /api/v1/admin/banners/{id}", middleware.Chain(adminBannerHandler.DeleteBanner, logging, middleware.CORS, requireAuth))

	// Page's endpoints (публичные, без авторизации)
	mux.HandleFunc("GET /api/v1/pages", middleware.Chain(pageHandler.GetPages, logging, middleware.CORS))
	mux.HandleFunc("GET /api/v1/pages/{id}", middleware.Chain(pageHandler.GetPage, logging, middleware.CORS))

	// Page's admin endpoints (требуют авторизации)
	mux.HandleFunc("POST /api/v1/admin/pages", middleware.Chain(adminPageHandler.CreatePage, logging, middleware.CORS, requireAuth))
	mux.HandleFunc("PATCH /api/v1/admin/pages/{id}", middleware.Chain(adminPageHandler.UpdatePage, logging, middleware.CORS, requireAuth))
	mux.HandleFunc("DELETE /api/v1/admin/pages/{id}", middleware.Chain(adminPageHandler.DeletePage, logging, middleware.CORS, requireAuth))

	// FAQ endpoints (публичные, без авторизации)
	mux.HandleFunc("GET /api/v1/faq", middleware.Chain(faqHandler.GetFAQs, logging, middleware.CORS))
	mux.HandleFunc("GET /api/v1/faq/{id}", middleware.Chain(faqHandler.GetFAQ, logging, middleware.CORS))

	// FAQ admin endpoints (требуют авторизации; чтение — включая неопубликованные)
	mux.HandleFunc("GET /api/v1/admin/faq", middleware.Chain(adminFAQHandler.GetFAQs, logging, middleware.CORS, requireAuth))
	mux.HandleFunc("GET /api/v1/admin/faq/{id}", middleware.Chain(adminFAQHandler.GetFAQ, logging, middleware.CORS, requireAuth))
	mux.HandleFunc("POST /api/v1/admin/faq", middleware.Chain(adminFAQHandler.CreateFAQ, logging, middleware.CORS, requireAuth))
	mux.HandleFunc("PATCH /api/v1/admin/faq/{id}", middleware.Chain(adminFAQHandler.UpdateFAQ, logging, middleware.CORS, requireAuth))
	mux.HandleFunc("DELETE /api/v1/admin/faq/{id}", middleware.Chain(adminFAQHandler.DeleteFAQ, logging, middleware.CORS, requireAuth))

	// Lead's endpoints (публичные, без авторизации: клиент может только оставить заявку)
	mux.HandleFunc("POST /api/v1/leads", middleware.Chain(leadHandler.CreateLead, logging, middleware.CORS, leadRateLimit))

	// Lead's admin endpoints (требуют авторизации: просмотр и удаление заявок)
	mux.HandleFunc("GET /api/v1/admin/leads", middleware.Chain(adminLeadHandler.GetLeads, logging, middleware.CORS, requireAuth))
	mux.HandleFunc("GET /api/v1/admin/leads/{id}", middleware.Chain(adminLeadHandler.GetLead, logging, middleware.CORS, requireAuth))
	mux.HandleFunc("DELETE /api/v1/admin/leads/{id}", middleware.Chain(adminLeadHandler.DeleteLead, logging, middleware.CORS, requireAuth))

	// Service's endpoints (публичные, без авторизации)
	mux.HandleFunc("GET /api/v1/services", middleware.Chain(serviceHandler.GetServices, logging, middleware.CORS))
	mux.HandleFunc("GET /api/v1/services/{id}", middleware.Chain(serviceHandler.GetService, logging, middleware.CORS))

	// Service's admin endpoints (требуют авторизации; чтение — включая неопубликованные)
	mux.HandleFunc("GET /api/v1/admin/services", middleware.Chain(adminServiceHandler.GetServices, logging, middleware.CORS, requireAuth))
	mux.HandleFunc("GET /api/v1/admin/services/{id}", middleware.Chain(adminServiceHandler.GetService, logging, middleware.CORS, requireAuth))
	mux.HandleFunc("POST /api/v1/admin/services", middleware.Chain(adminServiceHandler.CreateService, logging, middleware.CORS, requireAuth))
	mux.HandleFunc("PATCH /api/v1/admin/services/{id}", middleware.Chain(adminServiceHandler.UpdateService, logging, middleware.CORS, requireAuth))
	mux.HandleFunc("DELETE /api/v1/admin/services/{id}", middleware.Chain(adminServiceHandler.DeleteService, logging, middleware.CORS, requireAuth))

	// Review's endpoints (публичные, без авторизации: только одобренные отзывы, без контактов клиента)
	mux.HandleFunc("GET /api/v1/reviews", middleware.Chain(reviewHandler.GetReviews, logging, middleware.CORS))
	mux.HandleFunc("GET /api/v1/reviews/summary", middleware.Chain(reviewHandler.GetReviewsSummary, logging, middleware.CORS))
	mux.HandleFunc("POST /api/v1/reviews", middleware.Chain(reviewHandler.CreateReview, logging, middleware.CORS, reviewRateLimit))

	// Review's admin endpoints (требуют авторизации: просмотр с контактами, модерация, удаление)
	mux.HandleFunc("GET /api/v1/admin/reviews", middleware.Chain(adminReviewHandler.GetReviews, logging, middleware.CORS, requireAuth))
	mux.HandleFunc("GET /api/v1/admin/reviews/{id}", middleware.Chain(adminReviewHandler.GetReview, logging, middleware.CORS, requireAuth))
	mux.HandleFunc("POST /api/v1/admin/reviews/{id}/approve", middleware.Chain(adminReviewHandler.ApproveReview, logging, middleware.CORS, requireAuth))
	mux.HandleFunc("POST /api/v1/admin/reviews/{id}/unapprove", middleware.Chain(adminReviewHandler.UnapproveReview, logging, middleware.CORS, requireAuth))
	mux.HandleFunc("DELETE /api/v1/admin/reviews/{id}", middleware.Chain(adminReviewHandler.DeleteReview, logging, middleware.CORS, requireAuth))

	// User's admin endpoints (только для роли admin)
	mux.HandleFunc("GET /api/v1/admin/users", middleware.Chain(adminUserHandler.GetUsers, logging, middleware.CORS, requireAuth, requireAdmin))
	mux.HandleFunc("GET /api/v1/admin/users/{id}", middleware.Chain(adminUserHandler.GetUser, logging, middleware.CORS, requireAuth, requireAdmin))
	mux.HandleFunc("POST /api/v1/admin/users", middleware.Chain(adminUserHandler.CreateUser, logging, middleware.CORS, requireAuth, requireAdmin))
	mux.HandleFunc("PATCH /api/v1/admin/users/{id}", middleware.Chain(adminUserHandler.UpdateUser, logging, middleware.CORS, requireAuth, requireAdmin))
	mux.HandleFunc("DELETE /api/v1/admin/users/{id}", middleware.Chain(adminUserHandler.DeleteUser, logging, middleware.CORS, requireAuth, requireAdmin))

	// Role's admin endpoints (только для роли admin)
	mux.HandleFunc("GET /api/v1/admin/roles", middleware.Chain(adminRoleHandler.GetRoles, logging, middleware.CORS, requireAuth, requireAdmin))
	mux.HandleFunc("GET /api/v1/admin/roles/{id}", middleware.Chain(adminRoleHandler.GetRole, logging, middleware.CORS, requireAuth, requireAdmin))
	mux.HandleFunc("POST /api/v1/admin/roles", middleware.Chain(adminRoleHandler.CreateRole, logging, middleware.CORS, requireAuth, requireAdmin))
	mux.HandleFunc("PATCH /api/v1/admin/roles/{id}", middleware.Chain(adminRoleHandler.UpdateRole, logging, middleware.CORS, requireAuth, requireAdmin))
	mux.HandleFunc("DELETE /api/v1/admin/roles/{id}", middleware.Chain(adminRoleHandler.DeleteRole, logging, middleware.CORS, requireAuth, requireAdmin))

	// Telegram user's admin endpoints (только для роли admin: пользователи бота получают
	// уведомления с контактами клиентов из заявок и отзывов)
	mux.HandleFunc("GET /api/v1/admin/telegram-users", middleware.Chain(adminTelegramUserHandler.GetTelegramUsers, logging, middleware.CORS, requireAuth, requireAdmin))
	mux.HandleFunc("GET /api/v1/admin/telegram-users/{id}", middleware.Chain(adminTelegramUserHandler.GetTelegramUser, logging, middleware.CORS, requireAuth, requireAdmin))
	mux.HandleFunc("POST /api/v1/admin/telegram-users", middleware.Chain(adminTelegramUserHandler.CreateTelegramUser, logging, middleware.CORS, requireAuth, requireAdmin))
	mux.HandleFunc("PATCH /api/v1/admin/telegram-users/{id}", middleware.Chain(adminTelegramUserHandler.UpdateTelegramUser, logging, middleware.CORS, requireAuth, requireAdmin))
	mux.HandleFunc("DELETE /api/v1/admin/telegram-users/{id}", middleware.Chain(adminTelegramUserHandler.DeleteTelegramUser, logging, middleware.CORS, requireAuth, requireAdmin))

	// Car's endpoints
	mux.HandleFunc("GET /api/v1/cars", middleware.Chain(carHandler.GetCars, logging, middleware.CORS))
	mux.HandleFunc("GET /api/v1/cars/{id}", middleware.Chain(carHandler.GetCar, logging, middleware.CORS))

	// Customer's endpoints
	mux.HandleFunc("GET /api/v1/customers", middleware.Chain(customerHandler.GetCustomers, logging, middleware.CORS))
	mux.HandleFunc("GET /api/v1/customers/{id}", middleware.Chain(customerHandler.GetCustomer, logging, middleware.CORS))
	mux.HandleFunc("POST /api/v1/customers", middleware.Chain(customerHandler.CreateCustomer, logging, middleware.CORS))

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
