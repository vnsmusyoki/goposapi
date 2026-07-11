package models

type BusinessPurchasesSettings struct {
	ID                                          string `json:"id"`
	EnableEditingProductPriceFromPurchaseScreen bool   `json:"enableEditingProductPriceFromPurchaseScreen"`
	EnablePurchaseStatus                        bool   `json:"enablePurchaseStatus"`
	EnableLotNumber                             bool   `json:"enableLotNumber"`
	EnablePurchaseOrder                         bool   `json:"enablePurchaseOrder"`
	Message                                     string `json:"message"`
}
