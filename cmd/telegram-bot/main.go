package main

import (
	"os"
	"os/signal"
	"syscall"

	"nexera_back/internal/config"
	"nexera_back/internal/database"
	"nexera_back/internal/logger"
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

	telegramUserService := services.NewTelegramUserService(repository.NewTelegramUserRepository(db), log)

	bot, err := telegram.New(&cfg.Telegram, telegramUserService, log)
	if err != nil {
		log.WithError(err).Fatal("Failed to create telegram bot")
	}

	go bot.Start()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down telegram-bot service")

	bot.Stop()

	log.Info("telegram-bot service gracefully stopped")
}
