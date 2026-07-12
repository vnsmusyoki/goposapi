package models

type BusinessUnit struct {
	ID                string  `json:"id"`
	BusinessID        string  `json:"businessId"`
	Name              string  `json:"name"`
	ShortName         string  `json:"shortName"`
	AllowDecimal      bool    `json:"allowDecimal"`
	IsMultipleOfOther bool    `json:"isMultipleOfOther"`
	BaseUnitID        string  `json:"baseUnitId"`
	ConversionRate    float64 `json:"conversionRate"`
	CreatedByUserID   string  `json:"createdByUserId"`
	CreatedBy         string  `json:"createdBy"`
	CreatedAt         string  `json:"createdAt"`
	UpdatedAt         string  `json:"updatedAt"`
}
