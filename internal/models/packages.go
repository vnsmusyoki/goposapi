package models

type Package struct {
	Id          string  `json:"id" db:"id"`
	Name        string  `json:"name" db:"name"`
	Slug        string  `json:"slug" db:"slug"`
	Description string  `json:"description" db:"description"`
	Price       float64 `json:"price" db:"price"`
	Currency    string  `json:"currency" db:"currency"`
	BillingIntervalID string `json:"billing_interval_id" db:"billing_interval_id"`
	TrialDays   int     `json:"trial_days" db:"trial_days"`
	MaxUsers    int     `json:"max_users" db:"max_users"`
	MaxBranches int     `json:"max_branches" db:"max_branches"`
	MaxProducts int     `json:"max_products" db:"max_products"`
	IsActive    bool    `json:"is_active" db:"is_active"`
	IsFeatured  bool    `json:"is_featured" db:"is_featured"`
	SortOrder   int     `json:"sort_order" db:"sort_order"`
	CreatedAt   string  `json:"created_at" db:"created_at"`
	UpdatedAt   string  `json:"updated_at" db:"updated_at"`
}
