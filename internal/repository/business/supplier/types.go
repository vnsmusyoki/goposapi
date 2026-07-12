package supplier

type BusinessSupplierInput struct {
	BusinessID             string
	SupplierType           string
	ContactID              string
	Prefix                 string
	FirstName              string
	MiddleName             string
	LastName               string
	BusinessName           string
	Mobile                 string
	AlternateContactNumber string
	Landline               string
	Email                  string
	TaxNumber              string
	OpeningBalance         float64
	PayTermsType           string
	PayTermsValue          int
	AddressLine1           string
	AddressLine2           string
	City                   string
	State                  string
	Country                string
	ZipCode                string
	Website                string
	Notes                  string
}
