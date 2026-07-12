package models

type SubCategory struct {
	ID                 string `json:"id" db:"id"`
	BusinessID         string `json:"business_id" db:"business_id"`
	ParentCategoryID   string `json:"parent_category_id" db:"parent_category_id"`
	ParentCategoryName string `json:"parent_category_name" db:"parent_category_name"`
	SubCategoryCode    string `json:"sub_category_code" db:"sub_category_code"`
	Name               string `json:"name" db:"name"`
	Description        string `json:"description" db:"description"`
	MetaTitle          string `json:"meta_title" db:"meta_title"`
	MetaDescription    string `json:"meta_description" db:"meta_description"`
	ImageURL           string `json:"image_url" db:"image_url"`
	Active             bool   `json:"active" db:"active"`
	Featured           bool   `json:"featured" db:"featured"`
	SortOrder          int    `json:"sort_order" db:"sort_order"`
	CreatedAt          string `json:"created_at" db:"created_at"`
	UpdatedAt          string `json:"updated_at" db:"updated_at"`
}
