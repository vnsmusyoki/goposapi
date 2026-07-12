package brand

type CreateBrandInput struct {
	BusinessID       string
	Name             string
	ShortDescription string
	AddedByID        string
	AddedBy          string
}

type UpdateBrandInput struct {
	ID               string
	BusinessID       string
	Name             string
	ShortDescription string
}
