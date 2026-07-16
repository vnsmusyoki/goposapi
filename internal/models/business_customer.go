package models

type BusinessCustomer struct {
	ID                 string  `json:"id"`
	BusinessID         string  `json:"businessId"`
	ContactID          string  `json:"contactId"`
	CustomerCode       string  `json:"customerCode"`
	FirstName          string  `json:"firstName"`
	MiddleName         string  `json:"middleName"`
	LastName           string  `json:"lastName"`
	CompanyName        string  `json:"companyName"`
	Phone              string  `json:"phone"`
	Email              string  `json:"email"`
	Address            string  `json:"address"`
	ShippingAddress    string  `json:"shippingAddress"`
	TaxNumber          string  `json:"taxNumber"`
	OpeningBalance     float64 `json:"openingBalance"`
	PayTermsType       string  `json:"payTermsType"`
	PayTermsValue      int     `json:"payTermsValue"`
	CreditLimit        float64 `json:"creditLimit"`
	CustomerGroup      string  `json:"customerGroup"`
	AdvanceBalance     float64 `json:"advanceBalance"`
	TotalSaleDue       float64 `json:"totalSaleDue"`
	TotalSellReturnDue float64 `json:"totalSellReturnDue"`
	CustomField1       string  `json:"customField1"`
	CustomField2       string  `json:"customField2"`
	CustomField3       string  `json:"customField3"`
	CustomField4       string  `json:"customField4"`
	CustomField5       string  `json:"customField5"`
	Notes              string  `json:"notes"`
	IsActive           bool    `json:"isActive"`
	CreatedBy          string  `json:"createdBy"`
	Deleted            bool    `json:"deleted"`
	DeletedAt          string  `json:"deletedAt"`
	DeletedBy          string  `json:"deletedBy"`
	CreatedAt          string  `json:"createdAt"`
	UpdatedAt          string  `json:"updatedAt"`
	Name               string  `json:"name"`
	DisplayName        string  `json:"displayName"`
}
