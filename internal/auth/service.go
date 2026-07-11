package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

const sessionCookieName = "pos_session"

type Service struct {
	db            *pgxpool.Pool
	cookieSecure  bool
	loginTTL      time.Duration
	rememberMeTTL time.Duration
}

type apiError struct {
	Status  int               `json:"-"`
	Message string            `json:"message"`
	Errors  map[string]string `json:"errors,omitempty"`
}

func (e *apiError) Error() string {
	return e.Message
}

func NewService(db *pgxpool.Pool, cookieSecure bool) *Service {
	return &Service{
		db:            db,
		cookieSecure:  cookieSecure,
		loginTTL:      8 * time.Hour,
		rememberMeTTL: 30 * 24 * time.Hour,
	}
}

func (s *Service) RegisterRoutes(group *gin.RouterGroup) {
	group.POST("/login", s.handleLogin)
	group.POST("/logout", s.handleLogout)
	group.GET("/me", s.handleMe)
	group.GET("/modules", s.handleModules)
}

func (s *Service) handleLogin(c *gin.Context) {
	var payload LoginRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, validationErrorResponse(err))
		return
	}

	payload.Email = strings.ToLower(strings.TrimSpace(payload.Email))
	payload.Password = strings.TrimSpace(payload.Password)

	if payload.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Validation failed",
			"errors":  map[string]string{"email": "Email is required."},
		})
		return
	}

	if payload.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Validation failed",
			"errors":  map[string]string{"password": "Password is required."},
		})
		return
	}

	result, err := s.login(c.Request.Context(), payload, c.GetHeader("User-Agent"), c.ClientIP())
	if err != nil {
		log.Printf("auth login failed for %s: %v", payload.Email, err)
		respondAuthError(c, err)
		return
	}

	http.SetCookie(c.Writer, buildSessionCookie(result.Token, result.ExpiresAt, s.cookieSecure))
	c.JSON(http.StatusOK, gin.H{
		"message":   "Logged in successfully",
		"user":      result.User,
		"expiresAt": result.ExpiresAt,
	})
}

func (s *Service) handleMe(c *gin.Context) {
	token, ok := readSessionCookie(c.Request)
	if !ok {
		http.SetCookie(c.Writer, clearSessionCookie(s.cookieSecure))
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "Session expired. Please log in again.",
		})
		return
	}

	user, expiresAt, err := s.currentUser(c.Request.Context(), token)
	if err != nil {
		respondAuthError(c, err)
		if errors.Is(err, errSessionMissing) || errors.Is(err, errSessionExpired) {
			http.SetCookie(c.Writer, clearSessionCookie(s.cookieSecure))
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":      user,
		"expiresAt": expiresAt,
	})
}

func (s *Service) handleModules(c *gin.Context) {
	token, ok := readSessionCookie(c.Request)
	if !ok {
		http.SetCookie(c.Writer, clearSessionCookie(s.cookieSecure))
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "Session expired. Please log in again.",
		})
		return
	}

	user, _, err := s.currentUser(c.Request.Context(), token)
	if err != nil {
		respondAuthError(c, err)
		if errors.Is(err, errSessionMissing) || errors.Is(err, errSessionExpired) {
			http.SetCookie(c.Writer, clearSessionCookie(s.cookieSecure))
		}
		return
	}

	modules, err := fetchNavigationModules(c.Request.Context(), s.db, user.ID, user.Roles)
	if err != nil {
		log.Printf("auth modules failed for user=%s: %v", user.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Unable to load modules.",
		})
		return
	}

	c.JSON(http.StatusOK, ModulesResponse{
		Modules: modules,
	})
}

func (s *Service) handleLogout(c *gin.Context) {
	token, ok := readSessionCookie(c.Request)
	if ok {
		_ = s.revokeSession(c.Request.Context(), token)
	}

	http.SetCookie(c.Writer, clearSessionCookie(s.cookieSecure))
	c.JSON(http.StatusOK, gin.H{
		"message": "Logged out successfully",
	})
}

func (s *Service) CurrentUserFromRequest(ctx context.Context, req *http.Request) (*UserResponse, time.Time, error) {
	token, ok := readSessionCookie(req)
	if !ok {
		return nil, time.Time{}, errSessionMissing
	}

	return s.currentUser(ctx, token)
}

type loginResult struct {
	Token     string
	User      UserResponse
	ExpiresAt time.Time
}

var (
	errInvalidCredentials = errors.New("invalid credentials")
	errInactiveUser       = errors.New("inactive user")
	errSessionMissing     = errors.New("session not found")
	errSessionExpired     = errors.New("session expired")
)

func (s *Service) login(ctx context.Context, payload LoginRequest, userAgent, ipAddress string) (*loginResult, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var user struct {
		ID               string
		Email            string
		FullName         string
		IsActive         bool
		BusinessID       string
		ActiveBusinessID string
	}

	row := tx.QueryRow(ctx, `
		SELECT id::text, email, full_name, is_active,
		       COALESCE(active_business_id::text, '') AS active_business_id,
		       COALESCE(business_id::text, '') AS business_id
		FROM users
		WHERE email = $1
		  AND password_hash = crypt($2, password_hash)
		LIMIT 1
	`, payload.Email, payload.Password)

	if err := row.Scan(&user.ID, &user.Email, &user.FullName, &user.IsActive, &user.ActiveBusinessID, &user.BusinessID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, &apiError{
				Status:  http.StatusUnauthorized,
				Message: "Invalid email or password.",
			}
		}

		return nil, fmt.Errorf("lookup user: %w", err)
	}

	if !user.IsActive {
		return nil, &apiError{
			Status:  http.StatusForbidden,
			Message: "Your account is inactive.",
		}
	}

	roles, err := fetchRoles(ctx, tx, user.ID)
	if err != nil {
		return nil, err
	}

	businessID := ""
	if primaryRoleCode(roles) == "business" {
		businessID, err = resolveBusinessLoginContextID(ctx, tx, user.ID, user.ActiveBusinessID)
		if err != nil {
			return nil, err
		}
		if businessID == "" {
			return nil, &apiError{
				Status:  http.StatusForbidden,
				Message: "You need to be linked to a business to continue.",
			}
		}

		if _, err := tx.Exec(ctx, `
			UPDATE users
			SET active_business_id = $1
			WHERE id = $2
		`, businessID, user.ID); err != nil {
			return nil, fmt.Errorf("update active business: %w", err)
		}
	} else {
		businessID, err = resolveBusinessContextID(ctx, tx, user.ID, user.ActiveBusinessID, user.BusinessID)
		if err != nil {
			return nil, err
		}
	}

	modules, err := fetchModules(ctx, tx, user.ID, roles, businessID)
	if err != nil {
		return nil, err
	}

	rememberMeTTL := s.loginTTL
	if payload.RememberMe {
		rememberMeTTL = s.rememberMeTTL
	}

	rawToken, expiresAt, err := s.createSession(ctx, tx, user.ID, userAgent, ipAddress, rememberMeTTL)
	if err != nil {
		return nil, err
	}

	if _, err := tx.Exec(ctx, `UPDATE users SET last_login_at = NOW() WHERE id = $1`, user.ID); err != nil {
		return nil, fmt.Errorf("update last login: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit login: %w", err)
	}

	return &loginResult{
		Token: rawToken,
		User: UserResponse{
			ID:               user.ID,
			Email:            user.Email,
			FullName:         user.FullName,
			IsActive:         user.IsActive,
			Roles:            roles,
			Modules:          modules,
			LandingPath:      landingPathForModules(modules),
			ActiveBusinessID: businessID,
		},
		ExpiresAt: expiresAt,
	}, nil
}

func (s *Service) currentUser(ctx context.Context, rawToken string) (*UserResponse, time.Time, error) {
	sessionHash := hashToken(rawToken)

	var user struct {
		ID               string
		Email            string
		FullName         string
		IsActive         bool
		BusinessID       string
		ActiveBusinessID string
	}
	var expiresAt time.Time

	row := s.db.QueryRow(ctx, `
		SELECT u.id::text, u.email, u.full_name, u.is_active, s.expires_at,
		       COALESCE(u.active_business_id::text, '') AS active_business_id,
		       COALESCE(u.business_id::text, '') AS business_id
		FROM user_sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.refresh_token_hash = $1
		  AND s.revoked_at IS NULL
		  AND s.expires_at > NOW()
		LIMIT 1
	`, sessionHash)

	if err := row.Scan(&user.ID, &user.Email, &user.FullName, &user.IsActive, &expiresAt, &user.ActiveBusinessID, &user.BusinessID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, time.Time{}, errSessionMissing
		}

		return nil, time.Time{}, fmt.Errorf("load session: %w", err)
	}

	if !user.IsActive {
		return nil, time.Time{}, errInactiveUser
	}

	roles, err := fetchRoles(ctx, s.db, user.ID)
	if err != nil {
		return nil, time.Time{}, err
	}

	businessID, err := resolveBusinessContextID(ctx, s.db, user.ID, user.ActiveBusinessID, user.BusinessID)
	if err != nil {
		return nil, time.Time{}, err
	}

	modules, err := fetchModules(ctx, s.db, user.ID, roles, businessID)
	if err != nil {
		return nil, time.Time{}, err
	}

	return &UserResponse{
		ID:               user.ID,
		Email:            user.Email,
		FullName:         user.FullName,
		IsActive:         user.IsActive,
		Roles:            roles,
		Modules:          modules,
		LandingPath:      landingPathForModules(modules),
		ActiveBusinessID: businessID,
	}, expiresAt, nil
}

func (s *Service) createSession(ctx context.Context, tx pgx.Tx, userID, userAgent, ipAddress string, ttl time.Duration) (string, time.Time, error) {
	rawToken, err := generateToken()
	if err != nil {
		return "", time.Time{}, err
	}

	expiresAt := time.Now().UTC().Add(ttl)
	_, err = tx.Exec(ctx, `
		INSERT INTO user_sessions (
			user_id,
			refresh_token_hash,
			user_agent,
			ip_address,
			expires_at
		) VALUES ($1, $2, $3, $4, $5)
	`, userID, hashToken(rawToken), nullableString(userAgent), nullableString(ipAddress), expiresAt)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("create session: %w", err)
	}

	return rawToken, expiresAt, nil
}

func (s *Service) revokeSession(ctx context.Context, rawToken string) error {
	sessionHash := hashToken(rawToken)
	_, err := s.db.Exec(ctx, `
		UPDATE user_sessions
		SET revoked_at = NOW()
		WHERE refresh_token_hash = $1
		  AND revoked_at IS NULL
	`, sessionHash)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}

	return nil
}

func fetchRoles(ctx context.Context, q queryer, userID string) ([]RoleResponse, error) {
	rows, err := q.Query(ctx, `
		SELECT r.id::text, r.code, r.name
		FROM user_roles ur
		JOIN roles r ON r.id = ur.role_id
		WHERE ur.user_id = $1
		  AND r.is_active = TRUE
		ORDER BY r.sort_order ASC, r.name ASC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("load roles: %w", err)
	}
	defer rows.Close()

	roles := make([]RoleResponse, 0)
	for rows.Next() {
		var role RoleResponse
		if err := rows.Scan(&role.ID, &role.Code, &role.Name); err != nil {
			return nil, fmt.Errorf("scan role: %w", err)
		}
		roles = append(roles, role)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate roles: %w", err)
	}

	return roles, nil
}

type navigationModuleBuilder struct {
	moduleItem *NavigationItemResponse
}

func fetchNavigationModules(ctx context.Context, q queryer, userID string, roles []RoleResponse) ([]NavigationGroupResponse, error) {
	scope := primaryRoleCode(roles)
	switch scope {
	case "business":
		businessID, err := resolveManagedBusinessID(ctx, q, userID)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(businessID) == "" {
			return []NavigationGroupResponse{}, nil
		}

		return loadNavigationModules(ctx, q, "business", businessID)
	default:
		return loadNavigationModules(ctx, q, "user", userID)
	}
}

func loadNavigationModules(ctx context.Context, q queryer, scope string, scopeID string) ([]NavigationGroupResponse, error) {
	query := `
		SELECT
			m.id::text AS module_id,
			m.name AS module_name,
			COALESCE(m.icon, '') AS module_icon,
			COALESCE(m.path, '') AS module_path,
			m.has_sub_modules AS module_has_sub_modules,
			m.sort_order AS module_sort_order,
			COALESCE(sm.id::text, '') AS sub_module_id,
			COALESCE(sm.name, '') AS item_name,
			COALESCE(sm.icon, '') AS item_icon,
			COALESCE(sm.url, '') AS item_path,
			COALESCE(sm.sort_order, 0) AS item_sort_order
		FROM user_modules um
		JOIN modules m ON m.id = um.module_id
		LEFT JOIN sub_modules sm ON sm.id = um.sub_module_id AND sm.is_active = TRUE
		WHERE %s = $1
		  AND m.is_active = TRUE
		  AND (um.sub_module_id IS NULL OR sm.id IS NOT NULL)
		ORDER BY m.sort_order ASC, m.name ASC, CASE WHEN um.sub_module_id IS NULL THEN 0 ELSE 1 END ASC, COALESCE(sm.sort_order, 0) ASC, COALESCE(sm.name, '') ASC
	`

	var rows pgx.Rows
	var err error

	switch scope {
	case "business":
		rows, err = q.Query(ctx, fmt.Sprintf(query, "um.business_id"), scopeID)
	default:
		rows, err = q.Query(ctx, fmt.Sprintf(query, "um.user_id"), scopeID)
	}
	if err != nil {
		return nil, fmt.Errorf("load navigation modules: %w", err)
	}
	defer rows.Close()

	builders := make(map[string]*navigationModuleBuilder)
	order := make([]string, 0)

	for rows.Next() {
		var (
			moduleID            string
			moduleName          string
			moduleIcon          string
			modulePath          string
			moduleHasSubModules bool
			moduleSortOrder     int
			subModuleID         string
			itemName            string
			itemIcon            string
			itemPath            string
			itemSortOrder       int
		)

		if err := rows.Scan(
			&moduleID,
			&moduleName,
			&moduleIcon,
			&modulePath,
			&moduleHasSubModules,
			&moduleSortOrder,
			&subModuleID,
			&itemName,
			&itemIcon,
			&itemPath,
			&itemSortOrder,
		); err != nil {
			return nil, fmt.Errorf("scan navigation module: %w", err)
		}

		builder, exists := builders[moduleID]
		if !exists {
			builder = &navigationModuleBuilder{
				moduleItem: &NavigationItemResponse{
					Name:          moduleName,
					Icon:          moduleIcon,
					Path:          strings.TrimSpace(modulePath),
					HasSubModules: moduleHasSubModules,
					Children:      []NavigationItemResponse{},
				},
			}
			builders[moduleID] = builder
			order = append(order, moduleID)
		}

		if strings.TrimSpace(subModuleID) == "" {
			if builder.moduleItem != nil {
				builder.moduleItem.Name = moduleName
				builder.moduleItem.Icon = moduleIcon
				builder.moduleItem.Path = strings.TrimSpace(modulePath)
				builder.moduleItem.HasSubModules = moduleHasSubModules
			}
			continue
		}

		path := strings.TrimSpace(itemPath)
		if path == "" {
			continue
		}

		if builder.moduleItem == nil {
			builder.moduleItem = &NavigationItemResponse{
				Name:          moduleName,
				Icon:          moduleIcon,
				Path:          strings.TrimSpace(modulePath),
				HasSubModules: moduleHasSubModules,
				Children:      []NavigationItemResponse{},
			}
		}

		builder.moduleItem.Children = append(builder.moduleItem.Children, NavigationItemResponse{
			Name: itemName,
			Icon: itemIcon,
			Path: path,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate navigation modules: %w", err)
	}

	groups := make([]NavigationGroupResponse, 0, len(order))
	for _, moduleID := range order {
		builder := builders[moduleID]
		if builder == nil || builder.moduleItem == nil {
			continue
		}

		if len(builder.moduleItem.Children) == 0 && strings.TrimSpace(builder.moduleItem.Path) == "" {
			continue
		}

		groups = append(groups, NavigationGroupResponse{
			Name:  builder.moduleItem.Name,
			Items: []NavigationItemResponse{*builder.moduleItem},
		})
	}

	return groups, nil
}

func resolveManagedBusinessID(ctx context.Context, q queryer, userID string) (string, error) {
	var businessID string
	if err := q.QueryRow(ctx, `
		SELECT COALESCE((
			SELECT bm.business_id::text
			FROM business_managers bm
			WHERE bm.user_id = $1
			ORDER BY bm.created_at ASC
			LIMIT 1
		), '')
	`, userID).Scan(&businessID); err != nil {
		return "", fmt.Errorf("resolve managed business: %w", err)
	}

	return strings.TrimSpace(businessID), nil
}

func fetchModules(ctx context.Context, q queryer, userID string, roles []RoleResponse, businessID string) ([]ModuleResponse, error) {
	if primaryRoleCode(roles) == "business" && strings.TrimSpace(businessID) != "" {
		modules, err := loadModules(ctx, q, `
			SELECT m.id::text, m.code, m.name, COALESCE(m.description, ''), COALESCE(m.icon, ''), COALESCE(m.path, ''), m.sort_order
			FROM user_modules um
			JOIN modules m ON m.id = um.module_id
			WHERE um.business_id = $1
			  AND m.is_active = TRUE
			ORDER BY m.sort_order ASC, m.name ASC
		`, businessID)
		if err != nil {
			return nil, err
		}

		if len(modules) > 0 {
			return attachSubmodules(ctx, q, modules)
		}
	}

	modules, err := loadModules(ctx, q, `
		SELECT m.id::text, m.code, m.name, COALESCE(m.description, ''), COALESCE(m.icon, ''), COALESCE(m.path, ''), m.sort_order
		FROM user_modules um
		JOIN modules m ON m.id = um.module_id
		WHERE um.user_id = $1
		  AND m.is_active = TRUE
		ORDER BY m.sort_order ASC, m.name ASC
	`, userID)
	if err != nil {
		return nil, err
	}

	if len(modules) > 0 {
		return attachSubmodules(ctx, q, modules)
	}

	roleIDs := make([]string, 0, len(roles))
	seenRoleIDs := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		roleID := strings.TrimSpace(role.ID)
		if roleID == "" {
			continue
		}
		if _, exists := seenRoleIDs[roleID]; exists {
			continue
		}
		seenRoleIDs[roleID] = struct{}{}
		roleIDs = append(roleIDs, roleID)
	}

	if len(roleIDs) == 0 {
		return []ModuleResponse{}, nil
	}

	rolePlaceholders := make([]string, 0, len(roleIDs))
	roleArgs := make([]any, 0, len(roleIDs))
	for i, roleID := range roleIDs {
		rolePlaceholders = append(rolePlaceholders, fmt.Sprintf("$%d", i+1))
		roleArgs = append(roleArgs, roleID)
	}

	modules, err = loadModules(ctx, q, fmt.Sprintf(`
		SELECT id::text, code, name, COALESCE(description, ''), COALESCE(icon, ''), COALESCE(path, ''), sort_order
		FROM modules
		WHERE role_id IN (%s)
		  AND is_active = TRUE
		ORDER BY sort_order ASC, name ASC
	`, strings.Join(rolePlaceholders, ", ")), roleArgs...)
	if err != nil {
		return nil, err
	}

	return attachSubmodules(ctx, q, modules)
}

func resolveBusinessContextID(ctx context.Context, q queryer, userID, activeBusinessID, businessID string) (string, error) {
	if trimmed := strings.TrimSpace(activeBusinessID); trimmed != "" {
		return trimmed, nil
	}

	if trimmed := strings.TrimSpace(businessID); trimmed != "" {
		return trimmed, nil
	}

	var resolved string
	if err := q.QueryRow(ctx, `
		SELECT COALESCE((
			SELECT bm.business_id::text
			FROM business_managers bm
			WHERE bm.user_id = $1
			ORDER BY bm.created_at ASC
			LIMIT 1
		), '')
	`, userID).Scan(&resolved); err != nil {
		return "", fmt.Errorf("resolve business context: %w", err)
	}

	return strings.TrimSpace(resolved), nil
}

func resolveBusinessLoginContextID(ctx context.Context, q queryer, userID, activeBusinessID string) (string, error) {
	if trimmed := strings.TrimSpace(activeBusinessID); trimmed != "" {
		return trimmed, nil
	}

	var count int
	var firstBusinessID string
	if err := q.QueryRow(ctx, `
		SELECT COUNT(*), COALESCE(MIN(bm.business_id::text), '')
		FROM business_managers bm
		WHERE bm.user_id = $1
	`, userID).Scan(&count, &firstBusinessID); err != nil {
		return "", fmt.Errorf("resolve business login context: %w", err)
	}

	switch count {
	case 0:
		return "", &apiError{
			Status:  http.StatusForbidden,
			Message: "You need to be linked to a business to continue.",
		}
	case 1:
		return strings.TrimSpace(firstBusinessID), nil
	default:
		return "", &apiError{
			Status:  http.StatusForbidden,
			Message: "Multiple businesses are linked to your account. Please select an active business first.",
		}
	}
}

func primaryRoleCode(roles []RoleResponse) string {
	if len(roles) == 0 {
		return ""
	}

	return strings.ToLower(strings.TrimSpace(roles[0].Code))
}

func landingPathForModules(modules []ModuleResponse) string {
	if len(modules) == 0 {
		return ""
	}

	return strings.TrimSpace(modules[0].Path)
}

func loadModules(ctx context.Context, q queryer, query string, args ...any) ([]ModuleResponse, error) {
	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("load modules: %w", err)
	}
	defer rows.Close()

	modules := make([]ModuleResponse, 0)
	for rows.Next() {
		var module ModuleResponse
		if err := rows.Scan(&module.ID, &module.Code, &module.Name, &module.Description, &module.Icon, &module.Path, &module.SortOrder); err != nil {
			return nil, fmt.Errorf("scan module: %w", err)
		}
		modules = append(modules, module)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate modules: %w", err)
	}

	return modules, nil
}

func attachSubmodules(ctx context.Context, q queryer, modules []ModuleResponse) ([]ModuleResponse, error) {
	for i := range modules {
		children, err := fetchSubmodules(ctx, q, modules[i].ID)
		if err != nil {
			return nil, err
		}
		modules[i].Children = children
	}

	return modules, nil
}

func fetchSubmodules(ctx context.Context, q queryer, moduleID string) ([]SubmoduleResponse, error) {
	rows, err := q.Query(ctx, `
		SELECT
			sm.id::text,
			sm.code,
			sm.name,
			COALESCE(sm.description, ''),
			COALESCE(sm.icon, ''),
			CASE
				WHEN COALESCE(m.path, '') = '' THEN '/' || sm.code
				ELSE RTRIM(m.path, '/') || '/' || sm.code
			END AS path,
			sm.sort_order
		FROM sub_modules sm
		JOIN modules m ON m.id = sm.module_id
		WHERE sm.module_id = $1
		  AND sm.is_active = TRUE
		ORDER BY sm.sort_order ASC, sm.name ASC
	`, moduleID)
	if err != nil {
		return nil, fmt.Errorf("load submodules: %w", err)
	}
	defer rows.Close()

	submodules := make([]SubmoduleResponse, 0)
	for rows.Next() {
		var submodule SubmoduleResponse
		if err := rows.Scan(&submodule.ID, &submodule.Code, &submodule.Name, &submodule.Description, &submodule.Icon, &submodule.Path, &submodule.SortOrder); err != nil {
			return nil, fmt.Errorf("scan submodule: %w", err)
		}
		submodules = append(submodules, submodule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate submodules: %w", err)
	}

	return submodules, nil
}

type queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func validationErrorResponse(err error) gin.H {
	errorsMap := map[string]string{}
	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		for _, fieldErr := range validationErrors {
			field := strings.ToLower(fieldErr.Field())
			switch fieldErr.Tag() {
			case "required":
				errorsMap[field] = fmt.Sprintf("%s is required.", fieldErr.Field())
			case "email":
				errorsMap[field] = "Enter a valid email address."
			case "min":
				errorsMap[field] = fmt.Sprintf("%s must be at least %s characters.", fieldErr.Field(), fieldErr.Param())
			default:
				errorsMap[field] = fmt.Sprintf("%s is invalid.", fieldErr.Field())
			}
		}
	}

	if len(errorsMap) == 0 {
		errorsMap["form"] = "Validation failed."
	}

	return gin.H{
		"message": "Validation failed",
		"errors":  errorsMap,
	}
}

func respondAuthError(c *gin.Context, err error) {
	var apiErr *apiError
	if errors.As(err, &apiErr) {
		c.JSON(apiErr.Status, gin.H{
			"message": apiErr.Message,
			"errors":  apiErr.Errors,
		})
		return
	}

	if errors.Is(err, errInvalidCredentials) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "Invalid email or password.",
		})
		return
	}

	if errors.Is(err, errInactiveUser) {
		c.JSON(http.StatusForbidden, gin.H{
			"message": "Your account is inactive.",
		})
		return
	}

	if errors.Is(err, errSessionMissing) || errors.Is(err, errSessionExpired) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "Session expired. Please sign in again.",
		})
		return
	}

	log.Printf("auth error: %v", err)
	c.JSON(http.StatusInternalServerError, gin.H{
		"message": "Something went wrong.",
	})
}

func generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func buildSessionCookie(token string, expiresAt time.Time, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
}

func clearSessionCookie(secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
}

func (s *Service) ClearSessionCookie() *http.Cookie {
	return clearSessionCookie(s.cookieSecure)
}

func readSessionCookie(req *http.Request) (string, bool) {
	cookie, err := req.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		return "", false
	}

	return cookie.Value, true
}

func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	return value
}
