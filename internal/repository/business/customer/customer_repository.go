package customer

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"pos/internal/models"
)

type BusinessCustomerInput struct {
	BusinessID         string
	CreatedBy          string
	ContactID          string
	CustomerCode       string
	FirstName          string
	MiddleName         string
	LastName           string
	CompanyName        string
	Phone              string
	Email              string
	Address            string
	ShippingAddress    string
	TaxNumber          string
	OpeningBalance     float64
	PayTermsType       string
	PayTermsValue      int
	CreditLimit        float64
	CustomerGroup      string
	AdvanceBalance     float64
	TotalSaleDue       float64
	TotalSellReturnDue float64
	CustomField1       string
	CustomField2       string
	CustomField3       string
	CustomField4       string
	CustomField5       string
	Notes              string
	IsActive           bool
}

func ListBusinessCustomersRepository(pool *pgxpool.Pool, businessID string) ([]models.BusinessCustomer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	if businessID == "" {
		return nil, ErrBusinessNotResolved
	}

	rows, err := pool.Query(ctx, customerSelectQuery()+`
		WHERE business_id = $1
		  AND deleted_at IS NULL
		ORDER BY created_at DESC, company_name ASC, first_name ASC, last_name ASC, customer_code ASC
	`, businessID)
	if err != nil {
		return nil, fmt.Errorf("list business customers: %w", err)
	}
	defer rows.Close()

	customers := make([]models.BusinessCustomer, 0)
	for rows.Next() {
		customer, err := scanBusinessCustomer(rows)
		if err != nil {
			return nil, err
		}
		customers = append(customers, customer)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate business customers: %w", err)
	}

	return customers, nil
}

func CreateBusinessCustomerRepository(pool *pgxpool.Pool, req BusinessCustomerInput) (*models.BusinessCustomer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.CreatedBy = strings.TrimSpace(req.CreatedBy)
	req.ContactID = strings.TrimSpace(req.ContactID)
	req.CustomerCode = strings.TrimSpace(req.CustomerCode)
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.MiddleName = strings.TrimSpace(req.MiddleName)
	req.LastName = strings.TrimSpace(req.LastName)
	req.CompanyName = strings.TrimSpace(req.CompanyName)
	req.Phone = strings.TrimSpace(req.Phone)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Address = strings.TrimSpace(req.Address)
	req.ShippingAddress = strings.TrimSpace(req.ShippingAddress)
	req.TaxNumber = strings.TrimSpace(req.TaxNumber)
	req.PayTermsType = strings.ToLower(strings.TrimSpace(req.PayTermsType))
	req.CustomerGroup = strings.TrimSpace(req.CustomerGroup)
	req.CustomField1 = strings.TrimSpace(req.CustomField1)
	req.CustomField2 = strings.TrimSpace(req.CustomField2)
	req.CustomField3 = strings.TrimSpace(req.CustomField3)
	req.CustomField4 = strings.TrimSpace(req.CustomField4)
	req.CustomField5 = strings.TrimSpace(req.CustomField5)
	req.Notes = strings.TrimSpace(req.Notes)

	if req.BusinessID == "" {
		return nil, ErrBusinessNotResolved
	}
	if req.CustomerCode == "" {
		req.CustomerCode = generateBusinessCustomerCode(req.CompanyName, req.FirstName, req.LastName)
	}
	if req.ContactID == "" {
		req.ContactID = req.CustomerCode
	}
	if req.Phone == "" {
		return nil, ErrInvalidBusinessCustomerInput
	}

	exists, err := businessCustomerCodeExists(ctx, pool, req.BusinessID, req.CustomerCode, "")
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrBusinessCustomerCodeAlreadyExists
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO customers (
			business_id,
			contact_id,
			customer_code,
			first_name,
			middle_name,
			last_name,
			company_name,
			phone,
			email,
			address,
			shipping_address,
			tax_number,
			opening_balance,
			pay_terms_type,
			pay_terms_value,
			credit_limit,
			customer_group,
			advance_balance,
			total_sale_due,
			total_sell_return_due,
			custom_field_1,
			custom_field_2,
			custom_field_3,
			custom_field_4,
			custom_field_5,
			notes,
			is_active,
			created_by
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28
		)
	`, req.BusinessID, req.ContactID, req.CustomerCode, req.FirstName, req.MiddleName, req.LastName, req.CompanyName, req.Phone, req.Email, req.Address, nullString(req.ShippingAddress), req.TaxNumber, req.OpeningBalance, req.PayTermsType, req.PayTermsValue, req.CreditLimit, req.CustomerGroup, req.AdvanceBalance, req.TotalSaleDue, req.TotalSellReturnDue, req.CustomField1, req.CustomField2, req.CustomField3, req.CustomField4, req.CustomField5, req.Notes, req.IsActive, req.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("insert business customer: %w", err)
	}

	customer, err := GetBusinessCustomerRepository(pool, req.BusinessID, req.CustomerCode)
	if err != nil {
		return nil, err
	}

	return customer, nil
}

func GetBusinessCustomerRepository(pool *pgxpool.Pool, businessID, customerIDOrCode string) (*models.BusinessCustomer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	customerIDOrCode = strings.TrimSpace(customerIDOrCode)
	if businessID == "" || customerIDOrCode == "" {
		return nil, ErrBusinessNotResolved
	}

	row := pool.QueryRow(ctx, customerSelectQuery()+`
		WHERE business_id = $1
		  AND deleted_at IS NULL
		  AND (id::text = $2 OR customer_code = $2)
		LIMIT 1
	`, businessID, customerIDOrCode)

	customer, err := scanBusinessCustomer(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("business customer not found")
		}
		return nil, err
	}

	return &customer, nil
}

func UpdateBusinessCustomerRepository(pool *pgxpool.Pool, req BusinessCustomerInput, customerID string) (*models.BusinessCustomer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.CreatedBy = strings.TrimSpace(req.CreatedBy)
	req.ContactID = strings.TrimSpace(req.ContactID)
	req.CustomerCode = strings.TrimSpace(req.CustomerCode)
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.MiddleName = strings.TrimSpace(req.MiddleName)
	req.LastName = strings.TrimSpace(req.LastName)
	req.CompanyName = strings.TrimSpace(req.CompanyName)
	req.Phone = strings.TrimSpace(req.Phone)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Address = strings.TrimSpace(req.Address)
	req.ShippingAddress = strings.TrimSpace(req.ShippingAddress)
	req.TaxNumber = strings.TrimSpace(req.TaxNumber)
	req.PayTermsType = strings.ToLower(strings.TrimSpace(req.PayTermsType))
	req.CustomerGroup = strings.TrimSpace(req.CustomerGroup)
	req.CustomField1 = strings.TrimSpace(req.CustomField1)
	req.CustomField2 = strings.TrimSpace(req.CustomField2)
	req.CustomField3 = strings.TrimSpace(req.CustomField3)
	req.CustomField4 = strings.TrimSpace(req.CustomField4)
	req.CustomField5 = strings.TrimSpace(req.CustomField5)
	req.Notes = strings.TrimSpace(req.Notes)
	customerID = strings.TrimSpace(customerID)

	if req.BusinessID == "" || customerID == "" {
		return nil, ErrBusinessNotResolved
	}
	if req.CustomerCode == "" {
		req.CustomerCode = generateBusinessCustomerCode(req.CompanyName, req.FirstName, req.LastName)
	}
	if req.ContactID == "" {
		req.ContactID = req.CustomerCode
	}
	if req.Phone == "" {
		return nil, ErrInvalidBusinessCustomerInput
	}

	exists, err := businessCustomerCodeExists(ctx, pool, req.BusinessID, req.CustomerCode, customerID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrBusinessCustomerCodeAlreadyExists
	}

	_, err = pool.Exec(ctx, `
		UPDATE customers
		SET customer_code = $3,
		    contact_id = $4,
		    first_name = $5,
		    middle_name = $6,
		    last_name = $7,
		    company_name = $8,
		    phone = $9,
		    email = $10,
		    address = $11,
		    shipping_address = $12,
		    tax_number = $13,
		    opening_balance = $14,
		    pay_terms_type = $15,
		    pay_terms_value = $16,
		    credit_limit = $17,
		    customer_group = $18,
		    advance_balance = $19,
		    total_sale_due = $20,
		    total_sell_return_due = $21,
		    custom_field_1 = $22,
		    custom_field_2 = $23,
		    custom_field_3 = $24,
		    custom_field_4 = $25,
		    custom_field_5 = $26,
		    notes = $27,
		    is_active = $28,
		    updated_at = CURRENT_TIMESTAMP
		WHERE business_id = $1
		  AND id::text = $2
		  AND deleted_at IS NULL
	`, req.BusinessID, customerID, req.CustomerCode, req.ContactID, req.FirstName, req.MiddleName, req.LastName, req.CompanyName, req.Phone, req.Email, req.Address, nullString(req.ShippingAddress), req.TaxNumber, req.OpeningBalance, req.PayTermsType, req.PayTermsValue, req.CreditLimit, req.CustomerGroup, req.AdvanceBalance, req.TotalSaleDue, req.TotalSellReturnDue, req.CustomField1, req.CustomField2, req.CustomField3, req.CustomField4, req.CustomField5, req.Notes, req.IsActive)
	if err != nil {
		return nil, fmt.Errorf("update business customer: %w", err)
	}

	customer, err := GetBusinessCustomerRepository(pool, req.BusinessID, customerID)
	if err != nil {
		return nil, err
	}

	return customer, nil
}

func DeleteBusinessCustomerRepository(pool *pgxpool.Pool, businessID, customerID, deletedBy string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	customerID = strings.TrimSpace(customerID)
	deletedBy = strings.TrimSpace(deletedBy)
	if businessID == "" || customerID == "" {
		return ErrBusinessNotResolved
	}

	_, err := pool.Exec(ctx, `
		UPDATE customers
		SET deleted = TRUE,
		    deleted_at = CURRENT_TIMESTAMP,
		    deleted_by = NULLIF($3, '')::uuid,
		    is_active = FALSE,
		    updated_at = CURRENT_TIMESTAMP
		WHERE business_id = $1
		  AND id::text = $2
		  AND deleted_at IS NULL
	`, businessID, customerID, deletedBy)
	if err != nil {
		return fmt.Errorf("delete business customer: %w", err)
	}

	return nil
}

func businessCustomerCodeExists(ctx context.Context, pool *pgxpool.Pool, businessID, customerCode, excludeID string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM customers
			WHERE business_id = $1
			  AND customer_code = $2
			  AND deleted_at IS NULL
	`
	args := []any{businessID, customerCode}
	if strings.TrimSpace(excludeID) != "" {
		query += ` AND id::text <> $3`
		args = append(args, excludeID)
	}
	query += `)`

	var exists bool
	if err := pool.QueryRow(ctx, query, args...).Scan(&exists); err != nil {
		return false, fmt.Errorf("check business customer code: %w", err)
	}
	return exists, nil
}

func customerSelectQuery() string {
	return `
		SELECT
			id::text,
			business_id::text,
			COALESCE(contact_id, ''),
			customer_code,
			first_name,
			middle_name,
			last_name,
			company_name,
			phone,
			email,
			address,
			COALESCE(shipping_address, ''),
			tax_number,
			opening_balance,
			pay_terms_type,
			pay_terms_value,
			credit_limit,
			customer_group,
			advance_balance,
			total_sale_due,
			total_sell_return_due,
			custom_field_1,
			custom_field_2,
			custom_field_3,
			custom_field_4,
			custom_field_5,
			notes,
			is_active,
			COALESCE(created_by::text, ''),
			deleted,
			COALESCE(deleted_at::text, ''),
			COALESCE(deleted_by::text, ''),
			created_at::text,
			updated_at::text
		FROM customers`
}

func scanBusinessCustomer(scanner interface {
	Scan(dest ...any) error
}) (models.BusinessCustomer, error) {
	var customer models.BusinessCustomer
	if err := scanner.Scan(
		&customer.ID,
		&customer.BusinessID,
		&customer.ContactID,
		&customer.CustomerCode,
		&customer.FirstName,
		&customer.MiddleName,
		&customer.LastName,
		&customer.CompanyName,
		&customer.Phone,
		&customer.Email,
		&customer.Address,
		&customer.ShippingAddress,
		&customer.TaxNumber,
		&customer.OpeningBalance,
		&customer.PayTermsType,
		&customer.PayTermsValue,
		&customer.CreditLimit,
		&customer.CustomerGroup,
		&customer.AdvanceBalance,
		&customer.TotalSaleDue,
		&customer.TotalSellReturnDue,
		&customer.CustomField1,
		&customer.CustomField2,
		&customer.CustomField3,
		&customer.CustomField4,
		&customer.CustomField5,
		&customer.Notes,
		&customer.IsActive,
		&customer.CreatedBy,
		&customer.Deleted,
		&customer.DeletedAt,
		&customer.DeletedBy,
		&customer.CreatedAt,
		&customer.UpdatedAt,
	); err != nil {
		return models.BusinessCustomer{}, fmt.Errorf("scan business customer: %w", err)
	}

	customer.Name = buildBusinessCustomerName(customer.CompanyName, customer.FirstName, customer.MiddleName, customer.LastName)
	customer.DisplayName = customer.Name
	return customer, nil
}

func buildBusinessCustomerName(companyName, firstName, middleName, lastName string) string {
	companyName = strings.TrimSpace(companyName)
	if companyName != "" {
		return companyName
	}

	parts := []string{
		strings.TrimSpace(firstName),
		strings.TrimSpace(middleName),
		strings.TrimSpace(lastName),
	}
	name := strings.TrimSpace(strings.Join(filterNonEmpty(parts), " "))
	if name != "" {
		return name
	}

	return "Customer"
}

func filterNonEmpty(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			result = append(result, value)
		}
	}
	return result
}

func generateBusinessCustomerCode(companyName, firstName, lastName string) string {
	seed := strings.ToUpper(strings.TrimSpace(companyName + firstName + lastName))
	seed = strings.ReplaceAll(seed, " ", "")
	if len(seed) > 8 {
		seed = seed[:8]
	}
	if seed == "" {
		seed = "CUS"
	}

	var randomBytes [3]byte
	if _, err := rand.Read(randomBytes[:]); err != nil {
		return fmt.Sprintf("%s-%d", seed, time.Now().UnixNano())
	}

	return fmt.Sprintf("%s-%s", seed, strings.ToUpper(hex.EncodeToString(randomBytes[:])))
}

func nullString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}
