package models

import "time"

// Banner представляет баннер сайта-визитки. Показывается на одной странице (PageID),
// порядок отображения среди баннеров одной страницы задается полем Position.
type Banner struct {
	ID        int       `json:"id" db:"id"`
	PageID    int       `json:"page_id" db:"page_id"`
	Position  int       `json:"position" db:"position"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	Title       string     `json:"title" db:"title"`
	SubTitle	*string    `json:"subtitle" db:"subtitle"`
	Description *string    `json:"description,omitempty" db:"description"`
	LinkTitle   *string    `json:"link_title,omitempty" db:"link_title"`
	LinkURL     *string    `json:"link_url,omitempty" db:"link_url"`
	ImageURL    string     `json:"image_url" db:"image_url"`
	StartsAt    *time.Time `json:"starts_at,omitempty" db:"starts_at"`
	EndsAt      *time.Time `json:"ends_at,omitempty" db:"ends_at"`

	Page *Page `json:"page,omitempty"`
}

// CreateBannerRequest представляет запрос на создание баннера
type CreateBannerRequest struct {
	PageID      int        `json:"page_id"`
	SubTitle	*string    `json:"subtitle"`
	Title       string     `json:"title"`
	Description *string    `json:"description"`
	LinkTitle   *string    `json:"link_title"`
	LinkURL     *string    `json:"link_url"`
	ImageURL    string     `json:"image_url"`
	Position    int        `json:"position"`
	StartsAt    *time.Time `json:"starts_at"`
	EndsAt      *time.Time `json:"ends_at"`
	IsActive    bool       `json:"is_active"`
}

// UpdateBannerRequest представляет PATCH-запрос на обновление баннера: nil-поле значит
// "не менять", присутствующее — новое значение. Так через это поле нельзя явно сбросить
// необязательное поле (subtitle, description, link_title, link_url, starts_at, ends_at)
// обратно в NULL, только задать ему новое значение — этого достаточно для админки сайта-визитки.
type UpdateBannerRequest struct {
	PageID      *int       `json:"page_id"`
	Title       *string    `json:"title"`
	SubTitle    *string    `json:"subtitle"`
	Description *string    `json:"description"`
	LinkTitle   *string    `json:"link_title"`
	LinkURL     *string    `json:"link_url"`
	ImageURL    *string    `json:"image_url"`
	Position    *int       `json:"position"`
	StartsAt    *time.Time `json:"starts_at"`
	EndsAt      *time.Time `json:"ends_at"`
	IsActive    *bool      `json:"is_active"`
}
