package models

type Business struct {
	Id          int    `json:"id" db:"id"`
	Name        string `json:"name" db:"name"`
	Email       string `json:"email" db:"email"`
	PhoneNumber string `json:"phone_number" db:"phone_number"`
	Address     string `json:"address" db:"address"`
	City        string `json:"city" db:"city"`
	State       string `json:"state" db:"state"`
	Country     string `json:"country" db:"country"`
	ZipCode     string `json:"zip_code" db:"zip_code"`
	PackageID   int    `json:"package_id" db:"package_id"`
	IsActive    bool   `json:"is_active" db:"is_active"`
	CreatedAt   string `json:"created_at" db:"created_at"`
	UpdatedAt   string `json:"updated_at" db:"updated_at"`
}