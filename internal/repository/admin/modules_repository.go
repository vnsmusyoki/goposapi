package admin

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

type moduleRow struct {
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
}

type submoduleRow struct {
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

func ListModulesRepository(pool *pgxpool.Pool) ([]ModuleCatalogGroup, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := pool.Query(ctx, `
		SELECT
			m.id::text,
			m.code,
			m.name,
			COALESCE(m.description, ''),
			COALESCE(m.icon, ''),
			COALESCE(m.path, ''),
			m.has_sub_modules,
			m.access_level,
			COALESCE(r.code, ''),
			COALESCE(r.name, ''),
			m.sort_order,
			m.is_active
		FROM modules m
		JOIN roles r ON r.id = m.role_id
		ORDER BY r.code ASC, m.sort_order ASC, m.name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list modules: %w", err)
	}
	defer rows.Close()

	modulesByRole := make(map[string][]moduleRow)
	roleLabels := make(map[string]string)
	roleOrder := make([]string, 0)

	for rows.Next() {
		var row moduleRow
		if err := rows.Scan(
			&row.ID,
			&row.Code,
			&row.Name,
			&row.Description,
			&row.Icon,
			&row.Path,
			&row.HasSubModules,
			&row.AccessLevel,
			&row.RoleCode,
			&row.RoleName,
			&row.SortOrder,
			&row.Active,
		); err != nil {
			return nil, fmt.Errorf("scan module: %w", err)
		}

		roleCode := strings.ToLower(strings.TrimSpace(row.RoleCode))
		if roleCode == "" {
			roleCode = "unknown"
		}
		if _, exists := modulesByRole[roleCode]; !exists {
			roleOrder = append(roleOrder, roleCode)
		}
		modulesByRole[roleCode] = append(modulesByRole[roleCode], row)
		if strings.TrimSpace(row.RoleName) != "" {
			roleLabels[roleCode] = row.RoleName
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate modules: %w", err)
	}

	tabs := make([]ModuleCatalogGroup, 0, len(modulesByRole))
	for _, roleCode := range roleOrder {
		roleModules := modulesByRole[roleCode]
		sort.SliceStable(roleModules, func(i, j int) bool {
			if roleModules[i].SortOrder == roleModules[j].SortOrder {
				return roleModules[i].Name < roleModules[j].Name
			}
			return roleModules[i].SortOrder < roleModules[j].SortOrder
		})

		catalogModules := make([]ModuleCatalogModule, 0, len(roleModules))
		for _, module := range roleModules {
			submodules, err := listSubmodulesForModule(ctx, pool, module.ID)
			if err != nil {
				return nil, err
			}

			catalogModules = append(catalogModules, ModuleCatalogModule{
				ID:            module.ID,
				Code:          module.Code,
				Name:          module.Name,
				Description:   module.Description,
				Icon:          module.Icon,
				Path:          module.Path,
				HasSubModules: module.HasSubModules,
				AccessLevel:   module.AccessLevel,
				RoleCode:      roleCode,
				RoleName:      roleLabels[roleCode],
				SortOrder:     module.SortOrder,
				Active:        module.Active,
				Submodules:    submodules,
			})
		}

		tabs = append(tabs, ModuleCatalogGroup{
			Key:     roleCode,
			Label:   roleLabels[roleCode],
			Modules: catalogModules,
		})
	}

	return tabs, nil
}

func CreateModuleRepository(pool *pgxpool.Pool, req CreateModuleRequest) (*ModuleCatalogModule, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req.RoleID = strings.TrimSpace(req.RoleID)
	req.Code = normalizeModuleCode(req.Code)
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	req.Icon = strings.TrimSpace(req.Icon)
	req.Path = normalizeModulePath(req.Path)
	req.AccessLevel = normalizeAccessLevel(req.AccessLevel)
	if req.SortOrder < 0 {
		req.SortOrder = 0
	}

	if req.RoleID == "" || req.Code == "" || req.Name == "" || req.Path == "" {
		return nil, ErrInvalidBusinessInput
	}

	var exists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM modules
			WHERE code = $1
		)
	`, req.Code).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check module code: %w", err)
	}
	if exists {
		return nil, ErrModuleAlreadyExists
	}

	var roleCode string
	var roleName string
	if err := pool.QueryRow(ctx, `
		SELECT r.code, r.name
		FROM roles r
		WHERE r.id = $1
		LIMIT 1
	`, req.RoleID).Scan(&roleCode, &roleName); err != nil {
		return nil, fmt.Errorf("load role by id: %w", err)
	}
	roleCode = strings.ToLower(strings.TrimSpace(roleCode))
	if roleCode == "" {
		return nil, ErrInvalidBusinessInput
	}

	var moduleID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO modules (
			code,
			name,
			description,
			icon,
			path,
			has_sub_modules,
			access_level,
			role_id,
			sort_order,
			is_active
		)
		VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), $5, $6, $7, $8, $9, $10)
		RETURNING id::text
	`, req.Code, req.Name, nullIfBlank(req.Description), nullIfBlank(req.Icon), req.Path, req.HasSubModules, req.AccessLevel, req.RoleID, req.SortOrder, req.Active).Scan(&moduleID); err != nil {
		return nil, fmt.Errorf("insert module: %w", err)
	}

	return &ModuleCatalogModule{
		ID:            moduleID,
		Code:          req.Code,
		Name:          req.Name,
		Description:   req.Description,
		Icon:          req.Icon,
		Path:          req.Path,
		HasSubModules: req.HasSubModules,
		AccessLevel:   req.AccessLevel,
		RoleCode:      roleCode,
		RoleName:      roleName,
		SortOrder:     req.SortOrder,
		Active:        req.Active,
		Submodules:    []ModuleCatalogSubmodule{},
	}, nil
}

func CreateSubmoduleRepository(pool *pgxpool.Pool, req CreateSubmoduleRequest) (*ModuleCatalogSubmodule, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req.ModuleID = strings.TrimSpace(req.ModuleID)
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	req.Icon = strings.TrimSpace(req.Icon)
	req.URL = normalizeModulePath(req.URL)
	req.AccessLevel = normalizeAccessLevel(req.AccessLevel)
	if req.SortOrder < 0 {
		req.SortOrder = 0
	}

	if req.ModuleID == "" || req.Name == "" || req.URL == "" || req.Icon == "" {
		return nil, ErrInvalidBusinessInput
	}

	var moduleRoleCode string
	if err := pool.QueryRow(ctx, `
		SELECT COALESCE(r.code, '')
		FROM modules m
		JOIN roles r ON r.id = m.role_id
		WHERE m.id = $1
		LIMIT 1
	`, req.ModuleID).Scan(&moduleRoleCode); err != nil {
		return nil, fmt.Errorf("resolve parent module role: %w", err)
	}
	moduleRoleCode = strings.ToLower(strings.TrimSpace(moduleRoleCode))
	if moduleRoleCode == "" {
		return nil, ErrInvalidBusinessInput
	}

	var sameNameExists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM sub_modules
			WHERE role_id = (
				SELECT role_id
				FROM modules
				WHERE id = $1
				LIMIT 1
			)
			  AND lower(name) = lower($2)
		)
	`, req.ModuleID, req.Name).Scan(&sameNameExists); err != nil {
		return nil, fmt.Errorf("check duplicate submodule name: %w", err)
	}
	if sameNameExists {
		return nil, ErrSubmoduleAlreadyExists
	}

	baseCode := normalizeModuleCode(req.Name)
	if baseCode == "" {
		return nil, ErrInvalidBusinessInput
	}
	code := buildSubmoduleCode(moduleRoleCode, baseCode)
	code = truncateSubmoduleCode(code)
	for suffix := 2; ; suffix++ {
		var codeExists bool
		if err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM sub_modules
				WHERE code = $1
			)
		`, code).Scan(&codeExists); err != nil {
			return nil, fmt.Errorf("check submodule code: %w", err)
		}
		if !codeExists {
			break
		}

		code = truncateSubmoduleCode(fmt.Sprintf("%s-%d", buildSubmoduleCode(moduleRoleCode, baseCode), suffix))
	}

	var submoduleID string
	if err := pool.QueryRow(ctx, `
		INSERT INTO sub_modules (
			module_id,
			role_id,
			url,
			code,
			name,
			description,
			icon,
			access_level,
			sort_order,
			is_active
		)
		VALUES ($1, (
			SELECT role_id
			FROM modules
			WHERE id = $1
			LIMIT 1
		), $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''), $7, $8, $9)
		RETURNING id::text
	`, req.ModuleID, req.URL, code, req.Name, nullIfBlank(req.Description), nullIfBlank(req.Icon), req.AccessLevel, req.SortOrder, req.Active).Scan(&submoduleID); err != nil {
		return nil, fmt.Errorf("insert submodule: %w", err)
	}

	return &ModuleCatalogSubmodule{
		ID:          submoduleID,
		Code:        code,
		Name:        req.Name,
		URL:         req.URL,
		Icon:        req.Icon,
		Description: req.Description,
		AccessLevel: req.AccessLevel,
		SortOrder:   req.SortOrder,
		Active:      req.Active,
	}, nil
}

func UpdateSubmoduleRepository(pool *pgxpool.Pool, submoduleID string, req UpdateSubmoduleRequest) (*ModuleCatalogSubmodule, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	submoduleID = strings.TrimSpace(submoduleID)
	req.ModuleID = strings.TrimSpace(req.ModuleID)
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	req.Icon = strings.TrimSpace(req.Icon)
	req.URL = normalizeModulePath(req.URL)
	req.AccessLevel = normalizeAccessLevel(req.AccessLevel)
	if req.SortOrder < 0 {
		req.SortOrder = 0
	}

	if submoduleID == "" || req.ModuleID == "" || req.Name == "" || req.URL == "" || req.Icon == "" {
		return nil, ErrInvalidBusinessInput
	}

	var existing struct {
		ID       string
		RoleID   string
		RoleCode string
	}
	if err := pool.QueryRow(ctx, `
		SELECT
			sm.id::text,
			m.role_id::text,
			COALESCE(r.code, '')
		FROM sub_modules sm
		JOIN modules m ON m.id = sm.module_id
		JOIN roles r ON r.id = m.role_id
		WHERE sm.id = $1
		LIMIT 1
	`, submoduleID).Scan(&existing.ID, &existing.RoleID, &existing.RoleCode); err != nil {
		return nil, ErrSubmoduleNotFound
	}

	var targetRoleCode string
	if err := pool.QueryRow(ctx, `
		SELECT COALESCE(r.code, '')
		FROM modules m
		JOIN roles r ON r.id = m.role_id
		WHERE m.id = $1
		LIMIT 1
	`, req.ModuleID).Scan(&targetRoleCode); err != nil {
		return nil, ErrInvalidBusinessInput
	}
	targetRoleCode = strings.ToLower(strings.TrimSpace(targetRoleCode))
	if targetRoleCode == "" {
		return nil, ErrInvalidBusinessInput
	}

	var sameNameExists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM sub_modules sm
			JOIN modules m ON m.id = sm.module_id
			WHERE m.role_id = (
				SELECT role_id
				FROM modules
				WHERE id = $1
				LIMIT 1
			)
			  AND lower(sm.name) = lower($2)
			  AND sm.id <> $3
		)
	`, req.ModuleID, req.Name, submoduleID).Scan(&sameNameExists); err != nil {
		return nil, fmt.Errorf("check duplicate submodule name: %w", err)
	}
	if sameNameExists {
		return nil, ErrSubmoduleAlreadyExists
	}

	baseCode := normalizeModuleCode(req.Name)
	if baseCode == "" {
		return nil, ErrInvalidBusinessInput
	}
	code := buildSubmoduleCode(targetRoleCode, baseCode)
	code = truncateSubmoduleCode(code)
	for suffix := 2; ; suffix++ {
		var codeExists bool
		if err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM sub_modules
				WHERE code = $1
				  AND id <> $2
			)
		`, code, submoduleID).Scan(&codeExists); err != nil {
			return nil, fmt.Errorf("check submodule code: %w", err)
		}
		if !codeExists {
			break
		}

		code = truncateSubmoduleCode(fmt.Sprintf("%s-%d", buildSubmoduleCode(targetRoleCode, baseCode), suffix))
	}

	var updatedID string
	if err := pool.QueryRow(ctx, `
		UPDATE sub_modules
		SET
			module_id = $2,
			role_id = (
				SELECT role_id
				FROM modules
				WHERE id = $2
				LIMIT 1
			),
			url = $3,
			code = $4,
			name = $5,
			description = NULLIF($6, ''),
			icon = NULLIF($7, ''),
			access_level = $8,
			sort_order = $9,
			is_active = $10
		WHERE id = $1
		RETURNING id::text
	`, submoduleID, req.ModuleID, req.URL, code, req.Name, nullIfBlank(req.Description), nullIfBlank(req.Icon), req.AccessLevel, req.SortOrder, req.Active).Scan(&updatedID); err != nil {
		return nil, fmt.Errorf("update submodule: %w", err)
	}

	return &ModuleCatalogSubmodule{
		ID:          updatedID,
		Code:        code,
		Name:        req.Name,
		URL:         req.URL,
		Icon:        req.Icon,
		Description: req.Description,
		AccessLevel: req.AccessLevel,
		SortOrder:   req.SortOrder,
		Active:      req.Active,
	}, nil
}

func ReorderSubmodulesRepository(pool *pgxpool.Pool, moduleID string, orderedSubmoduleIDs []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	moduleID = strings.TrimSpace(moduleID)
	if moduleID == "" {
		return ErrInvalidBusinessInput
	}

	seen := make(map[string]struct{}, len(orderedSubmoduleIDs))
	cleanedIDs := make([]string, 0, len(orderedSubmoduleIDs))
	for _, id := range orderedSubmoduleIDs {
		cleanedID := strings.TrimSpace(id)
		if cleanedID == "" {
			return ErrInvalidBusinessInput
		}
		if _, exists := seen[cleanedID]; exists {
			return ErrInvalidBusinessInput
		}
		seen[cleanedID] = struct{}{}
		cleanedIDs = append(cleanedIDs, cleanedID)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin reorder submodules: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var moduleExists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM modules
			WHERE id = $1
		)
	`, moduleID).Scan(&moduleExists); err != nil {
		return fmt.Errorf("verify module exists: %w", err)
	}
	if !moduleExists {
		return ErrModuleNotFound
	}

	existingIDs := make([]string, 0)
	rows, err := tx.Query(ctx, `
		SELECT id::text
		FROM sub_modules
		WHERE module_id = $1
		ORDER BY sort_order ASC, name ASC
	`, moduleID)
	if err != nil {
		return fmt.Errorf("load existing submodule ids: %w", err)
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return fmt.Errorf("scan existing submodule ids: %w", err)
		}
		existingIDs = append(existingIDs, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate existing submodule ids: %w", err)
	}
	rows.Close()

	if len(existingIDs) != len(cleanedIDs) {
		return ErrInvalidBusinessInput
	}

	existingSet := make(map[string]struct{}, len(existingIDs))
	for _, id := range existingIDs {
		existingSet[id] = struct{}{}
	}
	for _, id := range cleanedIDs {
		if _, ok := existingSet[id]; !ok {
			return ErrInvalidBusinessInput
		}
	}

	for index, id := range cleanedIDs {
		if _, err := tx.Exec(ctx, `
			UPDATE sub_modules
			SET sort_order = $1
			WHERE id = $2
			  AND module_id = $3
		`, index+1, id, moduleID); err != nil {
			return fmt.Errorf("update submodule sort order: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit reorder submodules: %w", err)
	}

	return nil
}

func ReorderModulesRepository(pool *pgxpool.Pool, roleCode string, orderedModuleIDs []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	roleCode = strings.ToLower(strings.TrimSpace(roleCode))
	if roleCode == "" {
		return ErrInvalidBusinessInput
	}

	seen := make(map[string]struct{}, len(orderedModuleIDs))
	cleanedIDs := make([]string, 0, len(orderedModuleIDs))
	for _, id := range orderedModuleIDs {
		cleanedID := strings.TrimSpace(id)
		if cleanedID == "" {
			return ErrInvalidBusinessInput
		}
		if _, exists := seen[cleanedID]; exists {
			return ErrInvalidBusinessInput
		}
		seen[cleanedID] = struct{}{}
		cleanedIDs = append(cleanedIDs, cleanedID)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin reorder modules: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var roleID string
	if err := tx.QueryRow(ctx, `
		SELECT id::text
		FROM roles
		WHERE lower(code) = $1
		LIMIT 1
	`, roleCode).Scan(&roleID); err != nil {
		return ErrRoleNotFound
	}

	existingIDs := make([]string, 0)
	rows, err := tx.Query(ctx, `
		SELECT id::text
		FROM modules
		WHERE role_id = $1
		ORDER BY sort_order ASC, name ASC
	`, roleID)
	if err != nil {
		return fmt.Errorf("load existing module ids: %w", err)
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return fmt.Errorf("scan existing module ids: %w", err)
		}
		existingIDs = append(existingIDs, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate existing module ids: %w", err)
	}
	rows.Close()

	if len(existingIDs) != len(cleanedIDs) {
		return ErrInvalidBusinessInput
	}

	existingSet := make(map[string]struct{}, len(existingIDs))
	for _, id := range existingIDs {
		existingSet[id] = struct{}{}
	}
	for _, id := range cleanedIDs {
		if _, ok := existingSet[id]; !ok {
			return ErrInvalidBusinessInput
		}
	}

	for index, id := range cleanedIDs {
		if _, err := tx.Exec(ctx, `
			UPDATE modules
			SET sort_order = $1
			WHERE id = $2
			  AND role_id = $3
		`, index+1, id, roleID); err != nil {
			return fmt.Errorf("update module sort order: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit reorder modules: %w", err)
	}

	return nil
}

func listSubmodulesForModule(ctx context.Context, pool *pgxpool.Pool, moduleID string) ([]ModuleCatalogSubmodule, error) {
	rows, err := pool.Query(ctx, `
		SELECT
			sm.id::text,
			sm.code,
			sm.name,
			COALESCE(sm.url, ''),
			COALESCE(sm.icon, ''),
			COALESCE(sm.description, ''),
			sm.access_level,
			sm.sort_order,
			sm.is_active
		FROM sub_modules sm
		WHERE sm.module_id = $1
		ORDER BY sm.sort_order ASC, sm.name ASC
	`, moduleID)
	if err != nil {
		return nil, fmt.Errorf("list submodules: %w", err)
	}
	defer rows.Close()

	submodules := make([]ModuleCatalogSubmodule, 0)
	for rows.Next() {
		var row submoduleRow
		if err := rows.Scan(&row.ID, &row.Code, &row.Name, &row.URL, &row.Icon, &row.Description, &row.AccessLevel, &row.SortOrder, &row.Active); err != nil {
			return nil, fmt.Errorf("scan submodule: %w", err)
		}

		submodules = append(submodules, ModuleCatalogSubmodule{
			ID:          row.ID,
			Code:        row.Code,
			Name:        row.Name,
			URL:         row.URL,
			Icon:        row.Icon,
			Description: row.Description,
			AccessLevel: row.AccessLevel,
			SortOrder:   row.SortOrder,
			Active:      row.Active,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate submodules: %w", err)
	}

	return submodules, nil
}

func normalizeModuleCode(value string) string {
	code := strings.ToLower(strings.TrimSpace(value))
	code = strings.ReplaceAll(code, " ", "-")
	code = strings.ReplaceAll(code, "/", "-")
	code = strings.ReplaceAll(code, "_", "-")
	code = strings.ReplaceAll(code, "--", "-")
	return strings.Trim(code, "-")
}

func normalizeModulePath(value string) string {
	path := strings.TrimSpace(value)
	if path == "" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	path = strings.ReplaceAll(path, " ", "-")
	return path
}

func normalizeAccessLevel(value int) int {
	if value != 2 {
		return 1
	}
	return 2
}

func buildSubmoduleCode(roleCode, baseCode string) string {
	rolePrefix := strings.TrimSpace(strings.ToLower(roleCode))
	if rolePrefix == "" {
		rolePrefix = "module"
	}
	return normalizeModuleCode(fmt.Sprintf("%s-%s", rolePrefix, baseCode))
}

func truncateSubmoduleCode(value string) string {
	const maxCodeLength = 100
	value = strings.TrimSpace(value)
	if len(value) <= maxCodeLength {
		return value
	}

	return strings.TrimRight(value[:maxCodeLength], "-_")
}

func roleLabelFromCode(code string) string {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "admin":
		return "Admin"
	case "business":
		return "Business"
	case "cashier":
		return "Cashier"
	default:
		return strings.Title(strings.TrimSpace(code))
	}
}

func nullIfBlank(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
