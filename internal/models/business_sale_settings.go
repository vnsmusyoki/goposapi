package models

type BusinessSaleSettings struct {
	ID                             string  `json:"id"`
	DefaultSaleDiscount            float64 `json:"defaultSaleDiscount"`
	DefaultSaleTax                 float64 `json:"defaultSaleTax"`
	SaleItemAdditionMethod         string  `json:"saleItemAdditionMethod"`
	EnableSaleOrder                bool    `json:"enableSaleOrder"`
	IsPayTermRequired              bool    `json:"isPayTermRequired"`
	SalePriceIsMinimumSellingPrice bool    `json:"salePriceIsMinimumSellingPrice"`
	EnableSaleCommissionAgent      bool    `json:"enableSaleCommissionAgent"`
	CommissionCalculationType      string  `json:"commissionCalculationType"`
	IsCommissionAgentRequired      bool    `json:"isCommissionAgentRequired"`
	Message                        string  `json:"message"`
}
