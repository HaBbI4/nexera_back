package models

import "time"

// Car представляет модель
type Car struct {
	ID         int       `json:"id" db:"id"`
	CustomerID int       `json:"customer_id" db:"customer_id"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`

	Brand        string  `json:"brand" db:"brand"`
	Model        string  `json:"model" db:"model"`
	LicensePlate string  `json:"license_plate" db:"license_plate"`
	VIN          string  `json:"vin" db:"vin"`
	Comment      *string `json:"comment,omitempty" db:"comment"`
	Year         *int    `json:"year,omitempty" db:"year"`
}
