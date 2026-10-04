package telegram

import (
	"fmt"
	"time"

	"nexera_back/internal/config"
	"nexera_back/internal/logger"
	"nexera_back/internal/models"
	"nexera_back/internal/services"

	tele "gopkg.in/telebot.v3"
)

const (
	ordersPerPage = 20

	btnOrderDetailUnique = "order_detail"
	btnOrdersListUnique  = "orders_list"
)

// Bot оборачивает telebot-инстанс и связывает его с бизнес-логикой (заказы, авторизация)
type Bot struct {
	bot          *tele.Bot
	orderService *services.OrderService
	userService  *services.TelegramUserService
	log          *logger.Logger
}

// New создает бота, настраивает middleware авторизации и регистрирует обработчики команд
func New(
	cfg *config.TelegramConfig,
	orderService *services.OrderService,
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
		bot:          tb,
		orderService: orderService,
		userService:  userService,
		log:          log,
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
	b.bot.Handle("/orders", b.handleOrders)
	b.bot.Handle(&tele.Btn{Unique: btnOrderDetailUnique}, b.handleOrderDetail)
	b.bot.Handle(&tele.Btn{Unique: btnOrdersListUnique}, b.handleOrdersCallback)
}

func (b *Bot) handleStart(c tele.Context) error {
	return c.Send("👋 Добро пожаловать! Команда /orders покажет активные заказы.")
}

func (b *Bot) handleOrders(c tele.Context) error {
	text, markup, err := b.buildOrdersList()
	if err != nil {
		b.log.WithError(err).Error("Failed to load active orders")
		return c.Send("Не удалось загрузить заказы, попробуйте позже.")
	}
	return c.Send(text, markup)
}

// handleOrdersCallback обрабатывает кнопку "‹ Назад к списку" на карточке заказа
func (b *Bot) handleOrdersCallback(c tele.Context) error {
	text, markup, err := b.buildOrdersList()
	if err != nil {
		b.log.WithError(err).Error("Failed to load active orders")
		return c.Respond(&tele.CallbackResponse{Text: "Не удалось загрузить заказы"})
	}
	return c.Edit(text, markup)
}

func (b *Bot) buildOrdersList() (string, *tele.ReplyMarkup, error) {
	orders, err := b.orderService.GetActiveOrders(ordersPerPage, 0)
	if err != nil {
		return "", nil, err
	}

	markup := &tele.ReplyMarkup{}
	if len(orders) == 0 {
		return "Активных заказов нет.", markup, nil
	}

	rows := make([]tele.Row, 0, len(orders))
	for _, order := range orders {
		btn := markup.Data(formatOrderListItem(order), btnOrderDetailUnique, fmt.Sprintf("%d", order.ID))
		rows = append(rows, markup.Row(btn))
	}
	markup.Inline(rows...)

	return fmt.Sprintf("Активные заказы (%d):", len(orders)), markup, nil
}

func (b *Bot) handleOrderDetail(c tele.Context) error {
	var orderID int
	if _, err := fmt.Sscanf(c.Data(), "%d", &orderID); err != nil {
		return c.Respond(&tele.CallbackResponse{Text: "Некорректный заказ"})
	}

	order, err := b.orderService.GetOrder(orderID)
	if err != nil {
		b.log.WithError(err).WithField("order_id", orderID).Error("Failed to load order")
		return c.Respond(&tele.CallbackResponse{Text: "Заказ не найден"})
	}

	backMarkup := &tele.ReplyMarkup{}
	backBtn := backMarkup.Data("‹ Назад к списку", btnOrdersListUnique)
	backMarkup.Inline(backMarkup.Row(backBtn))

	if err := c.Edit(formatOrderDetail(order), backMarkup); err != nil {
		return err
	}
	return c.Respond()
}

// NotifyOrderCreated рассылает уведомление о новом заказе всем активным пользователям бота.
// Вызывается обработчиком события Kafka order.created.
func (b *Bot) NotifyOrderCreated(order *models.Order) error {
	users, err := b.userService.GetActiveUsers()
	if err != nil {
		return fmt.Errorf("failed to get active telegram users: %w", err)
	}

	if len(users) == 0 {
		b.log.Warn("No active telegram users to notify about new order")
		return nil
	}

	text := formatOrderCreatedNotification(order)
	for _, user := range users {
		recipient := &tele.User{ID: user.ChatID}
		if _, err := b.bot.Send(recipient, text); err != nil {
			b.log.WithError(err).WithField("chat_id", user.ChatID).Error("Failed to send order notification")
		}
	}

	return nil
}
