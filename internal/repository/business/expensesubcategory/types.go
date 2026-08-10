package expensesubcategory

type CreateExpenseSubCategoryInput struct {
	BusinessID        string
	ExpenseCategoryID string
	Name              string
	Code              string
	Description       string
	Active            bool
	SortOrder         int
	CreatedByID       string
	CreatedBy         string
	UpdatedByID       string
	UpdatedBy         string
}

type UpdateExpenseSubCategoryInput struct {
	ID                string
	BusinessID        string
	ExpenseCategoryID string
	Name              string
	Code              string
	Description       string
	Active            bool
	SortOrder         int
	UpdatedByID       string
	UpdatedBy         string
}
