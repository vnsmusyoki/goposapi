package settings

type UpdateBusinessProductSettingsInput struct {
	BusinessID            string
	SKUPrefix             string
	EnableProductExpiry   bool
	ExpiryTrackingMethod  string
	ExpirySellingBehavior string
	StopSellingDaysBefore *int
	EnableBrands          bool
	EnableCategories      bool
	EnableSubCategories   bool
	EnablePriceTaxInfo    bool
	DefaultUnit           string
	EnableSubUnits        bool
	EnableSecondaryUnit   bool
	EnableRacks           bool
	EnableRow             bool
	EnablePosition        bool
	EnableWarranty        bool
}
