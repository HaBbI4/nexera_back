package models

import "time"

// ContactInfoType представляет тип контакта компании
type ContactInfoType string

const (
	ContactInfoTypePhone    ContactInfoType = "phone"
	ContactInfoTypeEmail    ContactInfoType = "email"
	ContactInfoTypeTelegram ContactInfoType = "telegram"
	ContactInfoTypeWhatsapp ContactInfoType = "whatsapp"
	ContactInfoTypeVK       ContactInfoType = "vk"
)

// ContactInfo представляет контактные данные компании (телефон, email, мессенджеры и т.д.),
// отображаемые на сайте. Таблица управляется исключительно через админ-панель.
type ContactInfo struct {
	ID          int       `json:"id" db:"id"`
	Position    int       `json:"position" db:"position"`
	IsPublished bool      `json:"is_published" db:"is_published"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`

	Type  ContactInfoType `json:"type" db:"type"`
	Value *string         `json:"value,omitempty" db:"value"`
	Label *string         `json:"label,omitempty" db:"label"`
}
