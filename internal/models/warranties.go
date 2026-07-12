package models

type Warranty struct {
	ID            string `json:"id" db:"id"`
	BusinessID    string `json:"business_id" db:"business_id"`
	Name          string `json:"name" db:"name"`
	Description   string `json:"description" db:"description"`
	DurationValue int    `json:"duration_value" db:"duration_value"`
	DurationUnit  string `json:"duration_unit" db:"duration_unit"`
	AddedByID     string `json:"added_by_id" db:"added_by_id"`
	AddedBy       string `json:"added_by" db:"added_by"`
	AddedAt       string `json:"added_at" db:"added_at"`
	UpdatedAt     string `json:"updated_at" db:"updated_at"`
}
