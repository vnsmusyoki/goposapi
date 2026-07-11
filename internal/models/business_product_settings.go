package models

type BusinessProductSettings struct {
	ID                    string `json:"id"`
	SKUPrefix             string `json:"skuPrefix"`
	EnableProductExpiry   bool   `json:"enableProductExpiry"`
	ExpiryTrackingMethod  string `json:"expiryTrackingMethod"`
	ExpirySellingBehavior string `json:"expirySellingBehavior"`
	StopSellingDaysBefore *int   `json:"stopSellingDaysBefore,omitempty"`
	EnableBrands          bool   `json:"enableBrands"`
	EnableCategories      bool   `json:"enableCategories"`
	EnableSubCategories   bool   `json:"enableSubCategories"`
	EnablePriceTaxInfo    bool   `json:"enablePriceTaxInfo"`
	DefaultUnit           string `json:"defaultUnit"`
	EnableSubUnits        bool   `json:"enableSubUnits"`
	EnableSecondaryUnit   bool   `json:"enableSecondaryUnit"`
	EnableRacks           bool   `json:"enableRacks"`
	EnableRow             bool   `json:"enableRow"`
	EnablePosition        bool   `json:"enablePosition"`
	EnableWarranty        bool   `json:"enableWarranty"`
	Message               string `json:"message"`
}
