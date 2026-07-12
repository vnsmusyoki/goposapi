package warranty

type CreateWarrantyInput struct {
	BusinessID    string
	Name          string
	Description   string
	DurationValue int
	DurationUnit  string
	AddedByID     string
	AddedBy       string
}

type UpdateWarrantyInput struct {
	ID            string
	BusinessID    string
	Name          string
	Description   string
	DurationValue int
	DurationUnit  string
}
