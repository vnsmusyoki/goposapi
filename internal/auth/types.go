package auth

import "time"

type LoginRequest struct {
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required,min=8"`
	RememberMe bool   `json:"rememberMe"`
}

type RoleResponse struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type SubmoduleResponse struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`
	Path        string `json:"path"`
	SortOrder   int    `json:"sortOrder"`
}

type ModuleResponse struct {
	ID          string              `json:"id"`
	Code        string              `json:"code"`
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Icon        string              `json:"icon,omitempty"`
	Path        string              `json:"path"`
	SortOrder   int                 `json:"sortOrder"`
	Children    []SubmoduleResponse `json:"children,omitempty"`
}

type UserResponse struct {
	ID       string           `json:"id"`
	Email    string           `json:"email"`
	FullName string           `json:"fullName"`
	IsActive bool             `json:"isActive"`
	Roles    []RoleResponse   `json:"roles"`
	Modules  []ModuleResponse `json:"modules"`
}

type SessionResponse struct {
	User      UserResponse `json:"user"`
	ExpiresAt time.Time    `json:"expiresAt"`
}
