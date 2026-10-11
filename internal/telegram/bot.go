package telegram

import (
	"fmt"
	"time"

	"nexera_back/internal/config"
	"nexera_back/internal/logger"
	"nexera_back/internal/services"

	tele "gopkg.in/telebot.v3"
)

// Bot оборачивает telebot-инстанс и связывает его с бизнес-логикой (авторизация через telegram_users)
type Bot struct {
	bot         *tele.Bot
	userService *services.TelegramUserService
	log         *logger.Logger
}

// New создает бота, настраивает middleware авторизации и регистрирует обработчики команд
func New(
	cfg *config.TelegramConfig,
	userService *services.TelegramUserService,
	log *logger.Logger,
) (*Bot, error) {
	if cfg.BotToken == "" {
		return nil, fmt.Errorf("telegram bot token is not set (TELEGRAM_BOT_TOKEN)")
	}

	tb, err := tele.NewBot(tele.Settings{
		Token:  cfg.BotToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
		OnError: func(err error, c tele.Context) {
			log.WithError(err).Error("telegram bot handler error")
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	b := &Bot{
		bot:         tb,
		userService: userService,
		log:         log,
	}

	tb.Use(b.authMiddleware)
	b.registerHandlers()

	return b, nil
}

// Start блокирующе запускает long polling. Предполагается запуск в отдельной горутине.
func (b *Bot) Start() {
	b.log.Info("Telegram bot started")
	b.bot.Start()
}

// Stop останавливает long polling
func (b *Bot) Stop() {
	b.bot.Stop()
}

// authMiddleware пропускает только активных пользователей из telegram_users
func (b *Bot) authMiddleware(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		sender := c.Sender()
		if sender == nil {
			return nil
		}

		user, err := b.userService.GetByChatID(sender.ID)
		if err != nil || !user.IsActive {
			b.log.WithField("chat_id", sender.ID).Warn("Rejected unauthorized telegram access")
			return c.Send("⛔ Доступ запрещен. Обратитесь к владельцу системы.")
		}

		return next(c)
	}
}

func (b *Bot) registerHandlers() {
	b.bot.Handle("/start", b.handleStart)
}

func (b *Bot) handleStart(c tele.Context) error {
	return c.Send("👋 Добро пожаловать! Сюда будут приходить уведомления о новых заявках и отзывах.")
}
