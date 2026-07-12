package models

type BusinessSupplier struct {
	ID                     string  `json:"id"`
	BusinessID             string  `json:"businessId"`
	SupplierType           string  `json:"supplierType"`
	ContactID              string  `json:"contactId"`
	Prefix                 string  `json:"prefix"`
	FirstName              string  `json:"firstName"`
	MiddleName             string  `json:"middleName"`
	LastName               string  `json:"lastName"`
	BusinessName           string  `json:"businessName"`
	Mobile                 string  `json:"mobile"`
	AlternateContactNumber string  `json:"alternateContactNumber"`
	Landline               string  `json:"landline"`
	Email                  string  `json:"email"`
	TaxNumber              string  `json:"taxNumber"`
	OpeningBalance         float64 `json:"openingBalance"`
	PayTermsType           string  `json:"payTermsType"`
	PayTermsValue          int     `json:"payTermsValue"`
	AddressLine1           string  `json:"addressLine1"`
	AddressLine2           string  `json:"addressLine2"`
	City                   string  `json:"city"`
	State                  string  `json:"state"`
	Country                string  `json:"country"`
	ZipCode                string  `json:"zipCode"`
	Website                string  `json:"website"`
	Notes                  string  `json:"notes"`
	Status                 string  `json:"status"`
	Tier                   string  `json:"tier"`
	Rating                 float64 `json:"rating"`
	TotalPurchases         int     `json:"totalPurchases"`
	TotalAmount            float64 `json:"totalAmount"`
	OutstandingBalance     float64 `json:"outstandingBalance"`
	LeadTime               int     `json:"leadTime"`
	IsVerified             bool    `json:"isVerified"`
	IsFeatured             bool    `json:"isFeatured"`
	CreatedAt              string  `json:"createdAt"`
	UpdatedAt              string  `json:"updatedAt"`
}
