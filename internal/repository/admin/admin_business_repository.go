package admin

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type businessUser struct {
	ID       string
	Email    string
	FullName string
}

type assignedModule struct {
	ModuleID    string
	SubModuleID sql.NullString
}

type queryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func CreateBusinessRepository(
	pool *pgxpool.Pool,
	req CreateBusinessInput,
) (*CreateBusinessResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	req.Name = strings.TrimSpace(req.Name)
	req.BusinessEmail = strings.ToLower(strings.TrimSpace(req.BusinessEmail))
	req.BusinessPhone = strings.TrimSpace(req.BusinessPhone)
	req.RegistrationNumber = strings.TrimSpace(req.RegistrationNumber)
	req.Industry = strings.TrimSpace(req.Industry)
	req.OwnerName = strings.TrimSpace(req.OwnerName)
	req.SubscriptionPlan = strings.ToLower(strings.TrimSpace(req.SubscriptionPlan))
	req.ExistingManagerID = strings.TrimSpace(req.ExistingManagerID)

	if req.SubscriptionPlan == "" {
		req.SubscriptionPlan = "free"
	}

	if req.Name == "" || req.BusinessEmail == "" {
		return nil, ErrInvalidBusinessInput
	}

	exists, err := businessEmailExists(ctx, tx, req.BusinessEmail)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrBusinessAlreadyExists
	}

	packageInfo, err := packageBySlug(ctx, tx, req.SubscriptionPlan)
	if err != nil {
		return nil, err
	}
	req.SubscriptionPlan = packageInfo.Slug

	businessRoleID, err := roleIDByCode(ctx, tx, "business")
	if err != nil {
		return nil, err
	}

	var manager businessUser
	createdUser := false

	if req.ExistingManagerID != "" {
		manager, err = loadUserByID(ctx, tx, req.ExistingManagerID)
		if err != nil {
			return nil, err
		}
		if err := ensureBusinessRole(ctx, tx, manager.ID, businessRoleID); err != nil {
			return nil, err
		}
	} else {
		if req.Manager == nil {
			return nil, ErrInvalidManagerInput
		}

		req.Manager.Username = strings.TrimSpace(req.Manager.Username)
		req.Manager.Email = strings.ToLower(strings.TrimSpace(req.Manager.Email))
		req.Manager.Password = strings.TrimSpace(req.Manager.Password)
		req.Manager.FullName = strings.TrimSpace(req.Manager.FullName)
		req.Manager.Phone = strings.TrimSpace(req.Manager.Phone)

		if req.Manager.Username == "" || req.Manager.Email == "" || req.Manager.Password == "" || req.Manager.FullName == "" {
			return nil, ErrInvalidManagerInput
		}

		if len(req.Manager.Password) < 8 {
			return nil, ErrInvalidManagerInput
		}

		managerExists, err := managerIdentityExists(ctx, tx, req.Manager.Username, req.Manager.Email)
		if err != nil {
			return nil, err
		}
		if managerExists {
			return nil, ErrManagerAlreadyExists
		}

		manager, err = createBusinessUser(ctx, tx, businessRoleID, req)
		if err != nil {
			return nil, err
		}
		createdUser = true
	}

	if req.OwnerName == "" {
		req.OwnerName = manager.FullName
	}

	businessID, err := createBusiness(ctx, tx, req)
	if err != nil {
		return nil, err
	}

	if err := createBusinessSubscription(ctx, tx, businessID, packageInfo); err != nil {
		return nil, err
	}

	if createdUser {
		if err := updateBusinessUserAssignment(ctx, tx, manager.ID, businessID); err != nil {
			return nil, err
		}
	}

	linked, err := businessManagerLinkExists(ctx, tx, businessID, manager.ID)
	if err != nil {
		return nil, err
	}
	if linked {
		return nil, ErrBusinessManagerAlreadyLinked
	}

	if err := linkBusinessManager(ctx, tx, businessID, manager.ID); err != nil {
		return nil, err
	}

	if err := assignBusinessModules(ctx, tx, businessID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit business registration: %w", err)
	}

	return &CreateBusinessResult{
		BusinessID:   businessID,
		BusinessName: req.Name,
		ManagerID:    manager.ID,
		CreatedUser:  createdUser,
	}, nil
}

func businessEmailExists(ctx context.Context, q queryer, email string) (bool, error) {
	var exists bool
	if err := q.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM businesses
			WHERE business_email = $1
		)
	`, email).Scan(&exists); err != nil {
		return false, fmt.Errorf("check business email: %w", err)
	}

	return exists, nil
}

func managerIdentityExists(ctx context.Context, q queryer, username, email string) (bool, error) {
	var exists bool
	if err := q.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM users
			WHERE username = $1
			   OR email = $2
		)
	`, username, email).Scan(&exists); err != nil {
		return false, fmt.Errorf("check manager identity: %w", err)
	}

	return exists, nil
}

func loadUserByID(ctx context.Context, q queryer, userID string) (businessUser, error) {
	var user businessUser
	if err := q.QueryRow(ctx, `
		SELECT id::text, email, full_name
		FROM users
		WHERE id = $1
		LIMIT 1
	`, userID).Scan(&user.ID, &user.Email, &user.FullName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return businessUser{}, ErrManagerNotFound
		}
		return businessUser{}, fmt.Errorf("load manager: %w", err)
	}

	return user, nil
}

func roleIDByCode(ctx context.Context, q queryer, code string) (string, error) {
	var roleID string
	if err := q.QueryRow(ctx, `
		SELECT id::text
		FROM roles
		WHERE code = $1
		LIMIT 1
	`, code).Scan(&roleID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("role %q not found", code)
		}
		return "", fmt.Errorf("load role %q: %w", code, err)
	}

	return roleID, nil
}

func packageBySlug(ctx context.Context, q queryer, slug string) (*BusinessPackageInfo, error) {
	var info BusinessPackageInfo
	var billingIntervalMonths sql.NullInt64

	if err := q.QueryRow(ctx, `
		SELECT
			p.id::text,
			p.slug,
			bi.code,
			bi.interval_months,
			COALESCE(p.trial_days, 0)
		FROM packages p
		JOIN billing_intervals bi ON bi.id = p.billing_interval_id
		WHERE p.slug = $1
		  AND p.is_active = TRUE
		LIMIT 1
	`, strings.ToLower(strings.TrimSpace(slug))).Scan(
		&info.ID,
		&info.Slug,
		&info.BillingIntervalCode,
		&billingIntervalMonths,
		&info.TrialDays,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPackageNotFound
		}
		return nil, fmt.Errorf("load package: %w", err)
	}

	if billingIntervalMonths.Valid {
		months := int(billingIntervalMonths.Int64)
		info.BillingIntervalMonths = &months
	}

	return &info, nil
}

func ensureBusinessRole(ctx context.Context, tx pgx.Tx, userID, roleID string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO user_roles (user_id, role_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, userID, roleID)
	if err != nil {
		return fmt.Errorf("assign business role: %w", err)
	}

	return nil
}

func createBusinessUser(ctx context.Context, tx pgx.Tx, roleID string, req CreateBusinessInput) (businessUser, error) {
	manager := req.Manager
	var user businessUser
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (
			business_id,
			store_id,
			username,
			password_hash,
			email,
			full_name,
			phone,
			role_id,
			is_active
		)
		VALUES (
			NULL,
			NULL,
			$1,
			crypt($2, gen_salt('bf')),
			$3,
			$4,
			$5,
			$6,
			TRUE
		)
		RETURNING id::text, email, full_name
	`, manager.Username, manager.Password, manager.Email, manager.FullName, nullableString(manager.Phone), roleID).Scan(&user.ID, &user.Email, &user.FullName); err != nil {
		return businessUser{}, fmt.Errorf("create manager user: %w", err)
	}

	if err := ensureBusinessRole(ctx, tx, user.ID, roleID); err != nil {
		return businessUser{}, err
	}

	return user, nil
}

func updateBusinessUserAssignment(ctx context.Context, tx pgx.Tx, userID, businessID string) error {
	_, err := tx.Exec(ctx, `
		UPDATE users
		SET business_id = $1
		WHERE id = $2
	`, businessID, userID)
	if err != nil {
		return fmt.Errorf("update manager business assignment: %w", err)
	}

	return nil
}

func createBusiness(ctx context.Context, tx pgx.Tx, req CreateBusinessInput) (string, error) {
	var businessID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO businesses (
			name,
			business_email,
			business_phone,
			registration_number,
			industry,
			owner_name,
			subscription_plan,
			is_active
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, TRUE)
		RETURNING id::text
	`, req.Name, req.BusinessEmail, nullableString(req.BusinessPhone), nullableString(req.RegistrationNumber), nullableString(req.Industry), nullableString(req.OwnerName), req.SubscriptionPlan).Scan(&businessID); err != nil {
		return "", fmt.Errorf("create business: %w", err)
	}

	return businessID, nil
}

func createBusinessSubscription(ctx context.Context, tx pgx.Tx, businessID string, pkg *BusinessPackageInfo) error {
	now := time.Now().UTC()
	var currentPeriodEnd time.Time
	if pkg.BillingIntervalCode == "lifetime" {
		currentPeriodEnd = now.AddDate(100, 0, 0)
	} else if pkg.BillingIntervalMonths != nil && *pkg.BillingIntervalMonths > 0 {
		currentPeriodEnd = now.AddDate(0, *pkg.BillingIntervalMonths, 0)
	} else {
		currentPeriodEnd = now.AddDate(0, 1, 0)
	}

	var trialEndsAt any
	if pkg.TrialDays > 0 {
		trialEndsAt = now.AddDate(0, 0, pkg.TrialDays)
	} else {
		trialEndsAt = nil
	}

	_, err := tx.Exec(ctx, `
		INSERT INTO business_subscriptions (
			business_id,
			package_id,
			status,
			current_period_end,
			trial_ends_at,
			auto_renew
		)
		VALUES ($1, $2, 'trialing', $3, $4, TRUE)
	`, businessID, pkg.ID, currentPeriodEnd, trialEndsAt)
	if err != nil {
		return fmt.Errorf("create business subscription: %w", err)
	}

	return nil
}

func businessManagerLinkExists(ctx context.Context, q queryer, businessID, userID string) (bool, error) {
	var exists bool
	if err := q.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM business_managers
			WHERE business_id = $1
			  AND user_id = $2
		)
	`, businessID, userID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check business manager link: %w", err)
	}

	return exists, nil
}

func linkBusinessManager(ctx context.Context, tx pgx.Tx, businessID, userID string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO business_managers (business_id, user_id)
		VALUES ($1, $2)
	`, businessID, userID)
	if err != nil {
		return fmt.Errorf("link business manager: %w", err)
	}

	return nil
}

func assignBusinessModules(ctx context.Context, tx pgx.Tx, businessID string) error {
	rows, err := tx.Query(ctx, `
		SELECT module_id::text, sub_module_id::text
		FROM (
			SELECT m.id AS module_id, NULL::uuid AS sub_module_id, m.sort_order AS module_order, 0 AS sub_order
			FROM modules m
			JOIN roles r ON r.id = m.role_id
			WHERE r.code = 'business'
			  AND m.is_active = TRUE

			UNION ALL

			SELECT m.id AS module_id, sm.id AS sub_module_id, m.sort_order AS module_order, sm.sort_order AS sub_order
			FROM modules m
			JOIN roles r ON r.id = m.role_id
			JOIN sub_modules sm ON sm.module_id = m.id
			WHERE r.code = 'business'
			  AND m.is_active = TRUE
			  AND sm.is_active = TRUE
		) AS assigned_modules
		ORDER BY module_order ASC, sub_order ASC, module_id ASC
	`)
	if err != nil {
		return fmt.Errorf("load business modules: %w", err)
	}
	modules := make([]assignedModule, 0)
	for rows.Next() {
		var module assignedModule
		if err := rows.Scan(&module.ModuleID, &module.SubModuleID); err != nil {
			rows.Close()
			return fmt.Errorf("scan business module: %w", err)
		}
		modules = append(modules, module)
	}

	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate business modules: %w", err)
	}
	rows.Close()

	for _, module := range modules {
		exists, err := businessModuleAssignmentExists(ctx, tx, businessID, module.ModuleID, module.SubModuleID)
		if err != nil {
			return err
		}
		if exists {
			continue
		}

		if err := insertBusinessModuleAssignment(ctx, tx, businessID, module.ModuleID, module.SubModuleID); err != nil {
			return err
		}
	}

	return nil
}

func businessModuleAssignmentExists(ctx context.Context, q queryer, businessID, moduleID string, subModuleID sql.NullString) (bool, error) {
	var exists bool
	if err := q.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM user_modules
			WHERE user_id IS NULL
			  AND business_id = $1
			  AND module_id = $2
			  AND (
				($3::uuid IS NULL AND sub_module_id IS NULL)
				OR sub_module_id = $3::uuid
			  )
		)
	`, businessID, moduleID, nullableUUIDArgument(subModuleID)).Scan(&exists); err != nil {
		return false, fmt.Errorf("check business module assignment: %w", err)
	}

	return exists, nil
}

func insertBusinessModuleAssignment(ctx context.Context, tx pgx.Tx, businessID, moduleID string, subModuleID sql.NullString) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO user_modules (user_id, module_id, business_id, sub_module_id)
		VALUES (NULL, $1, $2, $3)
	`, moduleID, businessID, nullableUUIDArgument(subModuleID))
	if err != nil {
		return fmt.Errorf("insert business module assignment: %w", err)
	}

	return nil
}

func nullableString(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}

	return trimmed
}

func nullableUUIDArgument(value sql.NullString) any {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return nil
	}

	return strings.TrimSpace(value.String)
}
