package models

type ExpenseCategory struct {
	ID          string `json:"id" db:"id"`
	BusinessID  string `json:"business_id" db:"business_id"`
	ParentID    string `json:"parent_id" db:"parent_id"`
	ParentName  string `json:"parent_name" db:"parent_name"`
	Name        string `json:"name" db:"name"`
	Code        string `json:"code" db:"code"`
	Description string `json:"description" db:"description"`
	Active      bool   `json:"active" db:"active"`
	SortOrder   int    `json:"sort_order" db:"sort_order"`
	CreatedByID string `json:"created_by_id" db:"created_by_id"`
	CreatedBy   string `json:"created_by" db:"created_by"`
	AddedAt     string `json:"added_at" db:"added_at"`
	UpdatedAt   string `json:"updated_at" db:"updated_at"`
}
