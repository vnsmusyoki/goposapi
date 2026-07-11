package admin

type CreatePackageRequest struct {
	Name                string  `json:"name" binding:"required"`
	Slug                string  `json:"slug" binding:"required"`
	Description         string  `json:"description" binding:"required"`
	Price               float64 `json:"price" binding:"required"`
	Currency            string  `json:"currency" binding:"required"`
	BillingIntervalCode string  `json:"billing_interval_code" binding:"required"`
	TrialDays           int     `json:"trial_days" binding:"required"`
	MaxUsers            int     `json:"max_users" binding:"required"`
	MaxBranches         int     `json:"max_branches" binding:"required"`
	MaxProducts         int     `json:"max_products" binding:"required"`
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

type BusinessPackageInfo struct {
	ID                    string
	Slug                  string
	BillingIntervalCode   string
	BillingIntervalMonths *int
	TrialDays             int
}

type RoleCatalogItem struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type ModuleCatalogSubmodule struct {
	ID          string
	Code        string
	Name        string
	URL         string
	Icon        string
	Description string
	AccessLevel int
	SortOrder   int
	Active      bool
}

type ModuleCatalogModule struct {
	ID            string
	Code          string
	Name          string
	Description   string
	Icon          string
	Path          string
	HasSubModules bool
	AccessLevel   int
	RoleCode      string
	RoleName      string
	SortOrder     int
	Active        bool
	Submodules    []ModuleCatalogSubmodule
}

type ModuleCatalogGroup struct {
	Key     string
	Label   string
	Modules []ModuleCatalogModule
}

type CreateModuleRequest struct {
	RoleID        string
	Code          string
	Name          string
	Description   string
	Icon          string
	Path          string
	HasSubModules bool
	AccessLevel   int
	SortOrder     int
	Active        bool
}

type CreateSubmoduleRequest struct {
	ModuleID    string
	Name        string
	Description string
	Icon        string
	URL         string
	AccessLevel int
	SortOrder   int
	Active      bool
}

type UpdateSubmoduleRequest struct {
	ModuleID    string
	Name        string
	Description string
	Icon        string
	URL         string
	AccessLevel int
	SortOrder   int
	Active      bool
}
