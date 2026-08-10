package expensecategory

type CreateExpenseCategoryInput struct {
	BusinessID  string
	ParentID    string
	Name        string
	Code        string
	Description string
	Active      bool
	SortOrder   int
	CreatedByID string
	CreatedBy   string
	UpdatedByID string
	UpdatedBy   string
}

type UpdateExpenseCategoryInput struct {
	ID          string
	BusinessID  string
	ParentID    string
	Name        string
	Code        string
	Description string
	Active      bool
	SortOrder   int
	UpdatedByID string
	UpdatedBy   string
}
