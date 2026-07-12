package models

type Brand struct {
	ID               string `json:"id" db:"id"`
	BusinessID       string `json:"business_id" db:"business_id"`
	Name             string `json:"name" db:"name"`
	ShortDescription string `json:"short_description" db:"short_description"`
	AddedByID        string `json:"added_by_id" db:"added_by_id"`
	AddedBy          string `json:"added_by" db:"added_by"`
	AddedAt          string `json:"added_at" db:"added_at"`
	UpdatedAt        string `json:"updated_at" db:"updated_at"`
}
