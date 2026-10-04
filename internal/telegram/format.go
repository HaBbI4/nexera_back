package telegram

import (
	"fmt"
	"strings"

	"nexera_back/internal/models"
)

// statusLabels переводит статус заказа в человекочитаемый вид для сообщений бота
var statusLabels = map[models.OrderStatus]string{
	models.OrderStatusCreated:    "🆕 Создан",
	models.OrderStatusScheduled:  "📅 Запланирован",
	models.OrderStatusInProgress: "🔧 В работе",
	models.OrderStatusCompleted:  "✅ Завершен",
	models.OrderStatusCancelled:  "❌ Отменен",
}

func statusLabel(status models.OrderStatus) string {
	if label, ok := statusLabels[status]; ok {
		return label
	}
	return string(status)
}

// carLabel собирает "Brand Model (LicensePlate)" с учетом отсутствующих данных
func carLabel(car *models.Car) string {
	if car == nil {
		return "неизвестный автомобиль"
	}
	label := fmt.Sprintf("%s %s", car.Brand, car.Model)
	if car.LicensePlate != "" {
		label += fmt.Sprintf(" (%s)", car.LicensePlate)
	}
	return label
}

// formatOrderListItem форматирует одну строку заказа для списка активных заказов
func formatOrderListItem(order *models.Order) string {
	return fmt.Sprintf("#%d · %s · %s — %.0f ₽",
		order.ID, carLabel(order.Car), statusLabel(order.Status), order.TotalAmount)
}

// formatOrderDetail форматирует полную карточку заказа: клиент, авто, услуги, сумма
func formatOrderDetail(order *models.Order) string {
	var b strings.Builder

	fmt.Fprintf(&b, "📋 Заказ #%d\n", order.ID)
	fmt.Fprintf(&b, "Статус: %s\n", statusLabel(order.Status))
	fmt.Fprintf(&b, "Запланирован: %s\n\n", order.ScheduledAt.Format("02.01.2006 15:04"))

	if order.Customer != nil {
		fmt.Fprintf(&b, "👤 Клиент: %s\n", order.Customer.Name)
		if order.Customer.Phone != "" {
			fmt.Fprintf(&b, "📞 %s\n", order.Customer.Phone)
		}
	}

	if order.Car != nil {
		fmt.Fprintf(&b, "🚗 Авто: %s %s", order.Car.Brand, order.Car.Model)
		if order.Car.Year != nil {
			fmt.Fprintf(&b, " (%d)", *order.Car.Year)
		}
		b.WriteString("\n")
		if order.Car.LicensePlate != "" {
			fmt.Fprintf(&b, "Гос. номер: %s\n", order.Car.LicensePlate)
		}
		if order.Car.VIN != "" {
			fmt.Fprintf(&b, "VIN: %s\n", order.Car.VIN)
		}
	}

	if order.Comment != nil && *order.Comment != "" {
		fmt.Fprintf(&b, "\n💬 %s\n", *order.Comment)
	}

	if len(order.Items) > 0 {
		b.WriteString("\n🛠 Услуги:\n")
		for _, item := range order.Items {
			fmt.Fprintf(&b, "• %s × %d — %.0f ₽\n", item.ServiceName, item.Quantity, item.Price*float64(item.Quantity))
		}
	}

	fmt.Fprintf(&b, "\n💰 Итого: %.0f ₽", order.TotalAmount)

	return b.String()
}

// formatOrderCreatedNotification форматирует уведомление о новом заказе (событие Kafka)
func formatOrderCreatedNotification(order *models.Order) string {
	var b strings.Builder

	b.WriteString("🆕 Новый заказ!\n\n")
	fmt.Fprintf(&b, "%s\n", formatOrderDetail(order))

	return b.String()
}
