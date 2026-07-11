package models

type BusinessContactSettings struct {
	ID                 string   `json:"id"`
	DefaultCreditLimit *float64 `json:"defaultCreditLimit,omitempty"`
	Message            string   `json:"message"`
}
