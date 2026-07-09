package admin

type CreatePackageRequest struct {
	Name        string  `json:"name" binding:"required"`
	Slug        string  `json:"slug" binding:"required"`
	Description string  `json:"description" binding:"required"`
	Price       float64 `json:"price" binding:"required"`
	Currency    string  `json:"currency" binding:"required"`
	TrialDays   int     `json:"trial_days" binding:"required"`
	MaxUsers    int     `json:"max_users" binding:"required"`
	MaxBranches int     `json:"max_branches" binding:"required"`
	MaxProducts int     `json:"max_products" binding:"required"`
}

type CreateBusinessManagerInput struct {
	Username string
	Email    string
	Password string
	FullName string
	Phone    string
}

type CreateBusinessInput struct {
	Name               string
	BusinessEmail      string
	BusinessPhone      string
	RegistrationNumber string
	Industry           string
	OwnerName          string
	SubscriptionPlan   string
	ExistingManagerID  string
	Manager            *CreateBusinessManagerInput
}

type CreateBusinessResult struct {
	BusinessID   string
	BusinessName string
	ManagerID    string
	CreatedUser  bool
}
