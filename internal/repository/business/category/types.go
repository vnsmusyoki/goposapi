package category

type CreateCategoryInput struct {
	BusinessID      string
	CategoryCode    string
	Name            string
	Description     string
	MetaTitle       string
	MetaDescription string
	ImageURL        string
	Active          bool
	Featured        bool
	SortOrder       int
}

type UpdateCategoryInput struct {
	ID              string
	BusinessID      string
	CategoryCode    string
	Name            string
	Description     string
	MetaTitle       string
	MetaDescription string
	ImageURL        string
	Active          bool
	Featured        bool
	SortOrder       int
}
