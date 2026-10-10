package models

import "time"

// Page представляет страницу сайта, на которой можно показывать баннеры
// (например "home", "pricing", "about").
type Page struct {
	ID        int       `json:"id" db:"id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	Slug string `json:"slug" db:"slug"`
	Name string `json:"name" db:"name"`

	// Banners заполняется только при получении одной страницы по ID (см.
	// PageService.GetPublicByIDWithBanners) — список баннеров, видимых прямо сейчас,
	// отсортированных по Position. При List() всегда nil, в БД не хранится.
	Banners []*Banner `json:"banners,omitempty" db:"-"`
}

// CreatePageRequest представляет запрос на создание страницы
type CreatePageRequest struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// UpdatePageRequest представляет PATCH-запрос на обновление страницы: nil-поле значит
// "не менять", присутствующее — новое значение.
type UpdatePageRequest struct {
	Slug *string `json:"slug"`
	Name *string `json:"name"`
}
