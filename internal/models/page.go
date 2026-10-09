package models

import "time"

// Page представляет страницу сайта, на которой можно показывать баннеры
// (например "home", "pricing", "about").
type Page struct {
	ID        int       `json:"id" db:"id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	Slug string `json:"slug" db:"slug"`
	Name string `json:"name" db:"name"`
}
