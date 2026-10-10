package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"nexera_back/internal/config"
	"nexera_back/internal/database"
	"nexera_back/internal/kafka"
	"nexera_back/internal/logger"
	"nexera_back/internal/models"
	"nexera_back/internal/repository"
	"nexera_back/internal/services"
	"nexera_back/internal/telegram"
)

func main() {
	cfg := config.Load()

	log := logger.New(&cfg.Logger)
	log.Info("Starting telegram-bot service")

	db, err := database.Connect(&cfg.Database, log)
	if err != nil {
		log.WithError(err).Fatal("Failed to connect to database")
	}
	defer db.Close()

	// orderService создается без Kafka producer — этот сервис только читает заказы,
	// публикация событий (создание заказа) — зона ответственности cmd/server.
	orderService := services.NewOrderService(db, log, nil)
	telegramUserService := services.NewTelegramUserService(repository.NewTelegramUserRepository(db), log)

	bot, err := telegram.New(&cfg.Telegram, orderService, telegramUserService, log)
	if err != nil {
		log.WithError(err).Fatal("Failed to create telegram bot")
	}

	consumer, err := kafka.NewConsumer(&cfg.Kafka, log)
	if err != nil {
		log.WithError(err).Fatal("Failed to create Kafka consumer")
	}

	consumer.RegisterHandler(models.EventTypeOrderCreated, func(ctx context.Context, event *models.Event) error {
		raw, err := json.Marshal(event.Data)
		if err != nil {
			return fmt.Errorf("failed to marshal order.created payload: %w", err)
		}

		var payload models.OrderCreatedEvent
		if err := json.Unmarshal(raw, &payload); err != nil {
			return fmt.Errorf("failed to unmarshal order.created payload: %w", err)
		}

		order, err := orderService.GetOrder(payload.OrderID)
		if err != nil {
			return err
		}

		return bot.NotifyOrderCreated(order)
	})

	if err := consumer.Start(); err != nil {
		log.WithError(err).Fatal("Failed to start Kafka consumer")
	}

	go bot.Start()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down telegram-bot service")

	bot.Stop()
	if err := consumer.Stop(); err != nil {
		log.WithError(err).Error("Failed to stop Kafka consumer")
	}

	log.Info("telegram-bot service gracefully stopped")
}
