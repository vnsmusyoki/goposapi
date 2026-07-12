package unit

type CreateBusinessUnitInput struct {
	BusinessID        string
	Name              string
	ShortName         string
	AllowDecimal      bool
	IsMultipleOfOther bool
	BaseUnitID        string
	ConversionRate    float64
	CreatedByUserID   string
	CreatedBy         string
}

type UpdateBusinessUnitInput struct {
	BusinessID        string
	ID                string
	Name              string
	ShortName         string
	AllowDecimal      bool
	IsMultipleOfOther bool
	BaseUnitID        string
	ConversionRate    float64
}
