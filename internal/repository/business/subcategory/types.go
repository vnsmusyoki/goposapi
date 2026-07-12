package subcategory

type CreateSubCategoryInput struct {
	BusinessID       string
	ParentCategoryID string
	SubCategoryCode  string
	Name             string
	Description      string
	MetaTitle        string
	MetaDescription  string
	ImageURL         string
	Active           bool
	Featured         bool
	SortOrder        int
}

type UpdateSubCategoryInput struct {
	ID               string
	BusinessID       string
	ParentCategoryID string
	SubCategoryCode  string
	Name             string
	Description      string
	MetaTitle        string
	MetaDescription  string
	ImageURL         string
	Active           bool
	Featured         bool
	SortOrder        int
}
