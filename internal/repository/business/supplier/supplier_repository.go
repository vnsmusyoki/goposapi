package supplier

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

func ListBusinessSuppliersRepository(pool *pgxpool.Pool, businessID string) ([]models.BusinessSupplier, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	if businessID == "" {
		return nil, ErrBusinessNotResolved
	}

	rows, err := pool.Query(ctx, supplierSelectQuery()+`
		WHERE business_id = $1
		ORDER BY created_at DESC, business_name ASC, first_name ASC, last_name ASC
	`, businessID)
	if err != nil {
		return nil, fmt.Errorf("list business suppliers: %w", err)
	}
	defer rows.Close()

	suppliers := make([]models.BusinessSupplier, 0)
	for rows.Next() {
		supplier, err := scanBusinessSupplier(rows)
		if err != nil {
			return nil, err
		}
		suppliers = append(suppliers, supplier)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate business suppliers: %w", err)
	}

	return suppliers, nil
}

func CreateBusinessSupplierRepository(pool *pgxpool.Pool, req BusinessSupplierInput) (*models.BusinessSupplier, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.SupplierType = strings.ToLower(strings.TrimSpace(req.SupplierType))
	req.ContactID = strings.TrimSpace(req.ContactID)
	req.Prefix = strings.TrimSpace(req.Prefix)
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.MiddleName = strings.TrimSpace(req.MiddleName)
	req.LastName = strings.TrimSpace(req.LastName)
	req.BusinessName = strings.TrimSpace(req.BusinessName)
	req.Mobile = strings.TrimSpace(req.Mobile)
	req.AlternateContactNumber = strings.TrimSpace(req.AlternateContactNumber)
	req.Landline = strings.TrimSpace(req.Landline)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.TaxNumber = strings.TrimSpace(req.TaxNumber)
	req.PayTermsType = strings.ToLower(strings.TrimSpace(req.PayTermsType))
	req.AddressLine1 = strings.TrimSpace(req.AddressLine1)
	req.AddressLine2 = strings.TrimSpace(req.AddressLine2)
	req.City = strings.TrimSpace(req.City)
	req.State = strings.TrimSpace(req.State)
	req.Country = strings.TrimSpace(req.Country)
	req.ZipCode = strings.TrimSpace(req.ZipCode)
	req.Website = strings.TrimSpace(req.Website)
	req.Notes = strings.TrimSpace(req.Notes)

	if req.BusinessID == "" {
		return nil, ErrBusinessNotResolved
	}

	if req.ContactID == "" {
		req.ContactID = generateBusinessSupplierContactID()
	}

	exists, err := businessSupplierContactExists(ctx, pool, req.BusinessID, req.ContactID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrBusinessSupplierContactIDAlreadyExists
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO business_suppliers (
			business_id,
			supplier_type,
			contact_id,
			prefix,
			first_name,
			middle_name,
			last_name,
			business_name,
			mobile,
			alternate_contact_number,
			landline,
			email,
			tax_number,
			opening_balance,
			pay_terms_type,
			pay_terms_value,
			address_line_1,
			address_line_2,
			city,
			state,
			country,
			zip_code,
			website,
			notes
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24
		)
	`, req.BusinessID, req.SupplierType, req.ContactID, req.Prefix, req.FirstName, req.MiddleName, req.LastName, req.BusinessName, req.Mobile, req.AlternateContactNumber, req.Landline, req.Email, req.TaxNumber, req.OpeningBalance, req.PayTermsType, req.PayTermsValue, req.AddressLine1, req.AddressLine2, req.City, req.State, req.Country, req.ZipCode, req.Website, req.Notes)
	if err != nil {
		return nil, fmt.Errorf("insert business supplier: %w", err)
	}

	supplier, err := GetBusinessSupplierRepository(pool, req.BusinessID, req.ContactID)
	if err != nil {
		return nil, err
	}

	return supplier, nil
}

func GetBusinessSupplierRepository(pool *pgxpool.Pool, businessID, contactID string) (*models.BusinessSupplier, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	contactID = strings.TrimSpace(contactID)
	if businessID == "" || contactID == "" {
		return nil, ErrBusinessNotResolved
	}

	row := pool.QueryRow(ctx, supplierSelectQuery()+`
		WHERE business_id = $1
		  AND contact_id = $2
		LIMIT 1
	`, businessID, contactID)

	supplier, err := scanBusinessSupplier(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("business supplier not found")
		}
		return nil, err
	}

	return &supplier, nil
}

func businessSupplierContactExists(ctx context.Context, pool *pgxpool.Pool, businessID, contactID string) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM business_suppliers
			WHERE business_id = $1
			  AND contact_id = $2
		)
	`, businessID, contactID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check business supplier contact id: %w", err)
	}
	return exists, nil
}

func supplierSelectQuery() string {
	return `
		SELECT
			id::text,
			business_id::text,
			supplier_type,
			contact_id,
			prefix,
			first_name,
			middle_name,
			last_name,
			business_name,
			mobile,
			alternate_contact_number,
			landline,
			email,
			tax_number,
			opening_balance,
			pay_terms_type,
			pay_terms_value,
			address_line_1,
			address_line_2,
			city,
			state,
			country,
			zip_code,
			website,
			notes,
			status,
			tier,
			rating,
			total_purchases,
			total_amount,
			outstanding_balance,
			lead_time,
			is_verified,
			is_featured,
			created_at::text,
			updated_at::text
		FROM business_suppliers`
}

func scanBusinessSupplier(scanner interface {
	Scan(dest ...any) error
}) (models.BusinessSupplier, error) {
	var supplier models.BusinessSupplier
	if err := scanner.Scan(
		&supplier.ID,
		&supplier.BusinessID,
		&supplier.SupplierType,
		&supplier.ContactID,
		&supplier.Prefix,
		&supplier.FirstName,
		&supplier.MiddleName,
		&supplier.LastName,
		&supplier.BusinessName,
		&supplier.Mobile,
		&supplier.AlternateContactNumber,
		&supplier.Landline,
		&supplier.Email,
		&supplier.TaxNumber,
		&supplier.OpeningBalance,
		&supplier.PayTermsType,
		&supplier.PayTermsValue,
		&supplier.AddressLine1,
		&supplier.AddressLine2,
		&supplier.City,
		&supplier.State,
		&supplier.Country,
		&supplier.ZipCode,
		&supplier.Website,
		&supplier.Notes,
		&supplier.Status,
		&supplier.Tier,
		&supplier.Rating,
		&supplier.TotalPurchases,
		&supplier.TotalAmount,
		&supplier.OutstandingBalance,
		&supplier.LeadTime,
		&supplier.IsVerified,
		&supplier.IsFeatured,
		&supplier.CreatedAt,
		&supplier.UpdatedAt,
	); err != nil {
		return models.BusinessSupplier{}, fmt.Errorf("scan business supplier: %w", err)
	}

	return supplier, nil
}

func generateBusinessSupplierContactID() string {
	var randomBytes [5]byte
	if _, err := rand.Read(randomBytes[:]); err != nil {
		return fmt.Sprintf("SUP-%d", time.Now().UnixNano())
	}

	return fmt.Sprintf("SUP-%s", strings.ToUpper(hex.EncodeToString(randomBytes[:])))
}
