package location

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"pos/internal/models"
)

func CreateBusinessLocationRepository(
	pool *pgxpool.Pool,
	req CreateBusinessLocationInput,
) (*models.BusinessLocation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.LocationID = strings.TrimSpace(req.LocationID)
	req.LocationCode = strings.TrimSpace(req.LocationCode)
	req.LocationName = strings.TrimSpace(req.LocationName)
	req.Landmark = strings.TrimSpace(req.Landmark)
	req.ExactAddress = strings.TrimSpace(req.ExactAddress)
	req.City = strings.TrimSpace(req.City)
	req.ZipCode = strings.TrimSpace(req.ZipCode)
	req.State = strings.TrimSpace(req.State)
	req.Country = strings.TrimSpace(req.Country)
	req.Mobile = strings.TrimSpace(req.Mobile)
	req.AlternateContactNumber = strings.TrimSpace(req.AlternateContactNumber)
	req.Email = strings.TrimSpace(req.Email)
	req.Website = strings.TrimSpace(req.Website)
	req.InvoiceScheme = strings.TrimSpace(req.InvoiceScheme)
	req.PosInvoiceLayout = strings.TrimSpace(req.PosInvoiceLayout)
	req.SaleInvoiceLayout = strings.TrimSpace(req.SaleInvoiceLayout)
	req.DefaultSellingPriceGroup = strings.TrimSpace(req.DefaultSellingPriceGroup)
	req.KraPin = strings.TrimSpace(req.KraPin)
	req.TaxJurisdiction = strings.TrimSpace(req.TaxJurisdiction)
	req.VatNumber = strings.TrimSpace(req.VatNumber)
	req.DefaultTaxType = strings.TrimSpace(req.DefaultTaxType)
	req.TaxNote = strings.TrimSpace(req.TaxNote)
	req.Environment = strings.TrimSpace(req.Environment)
	req.IntegrationType = strings.TrimSpace(req.IntegrationType)
	req.KraBranchID = strings.TrimSpace(req.KraBranchID)
	req.DeviceSerialNumber = strings.TrimSpace(req.DeviceSerialNumber)
	req.CmcKey = strings.TrimSpace(req.CmcKey)

	if req.BusinessID == "" || req.LocationID == "" || req.LocationName == "" || req.Mobile == "" || req.KraPin == "" {
		return nil, ErrInvalidBusinessLocationInput
	}
	if req.LocationCode == "" {
		req.LocationCode = req.LocationID
	}

	if req.Country == "" {
		req.Country = "Kenya"
	}
	if req.InvoiceScheme == "" {
		req.InvoiceScheme = "default"
	}
	if req.PosInvoiceLayout == "" {
		req.PosInvoiceLayout = "default"
	}
	if req.SaleInvoiceLayout == "" {
		req.SaleInvoiceLayout = "default"
	}
	if req.DefaultSellingPriceGroup == "" {
		req.DefaultSellingPriceGroup = "retail"
	}
	if req.TaxJurisdiction == "" {
		req.TaxJurisdiction = "Kenya"
	}
	if req.Environment == "" {
		req.Environment = "sandbox"
	}
	if req.IntegrationType == "" {
		req.IntegrationType = "OSCU"
	}

	var exists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM business_locations
			WHERE business_id = $1
			  AND LOWER(location_id) = LOWER($2)
		)
	`, req.BusinessID, req.LocationID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check duplicate business location id: %w", err)
	}
	if exists {
		return nil, ErrBusinessLocationAlreadyExists
	}

	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM business_locations
			WHERE business_id = $1
			  AND LOWER(location_name) = LOWER($2)
		)
	`, req.BusinessID, req.LocationName).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check duplicate business location name: %w", err)
	}
	if exists {
		return nil, ErrBusinessLocationAlreadyExists
	}

	paymentMethodsJSON, err := json.Marshal(req.PaymentMethods)
	if err != nil {
		return nil, fmt.Errorf("marshal payment methods: %w", err)
	}

	var location models.BusinessLocation
	var latitude sql.NullFloat64
	var longitude sql.NullFloat64
	var rawPaymentMethods string

	err = pool.QueryRow(ctx, `
		INSERT INTO business_locations (
			business_id,
			location_id,
			location_code,
			location_name,
			landmark,
			exact_address,
			city,
			zip_code,
			state,
			country,
			latitude,
			longitude,
			mobile,
			alternate_contact_number,
			email,
			website,
			invoice_scheme,
			pos_invoice_layout,
			sale_invoice_layout,
			default_selling_price_group,
			payment_methods,
			kra_pin,
			tax_jurisdiction,
			is_vat_registered,
			vat_number,
			default_tax_type,
			prices_include_tax,
			issue_tax_invoices,
			tax_note,
			etims_enabled,
			environment,
			integration_type,
			is_head_office_branch,
			kra_branch_id,
			device_serial_number,
			cmc_key,
			auto_submit_invoices,
			allow_offline_sales,
			retry_failed_invoices,
			print_qr_code,
			print_fiscal_details
		)
		VALUES (
			$1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''), NULLIF($7, ''), NULLIF($8, ''), NULLIF($9, ''), $10,
			$11, $12, $13, NULLIF($14, ''), NULLIF($15, ''), NULLIF($16, ''), $17, $18, $19, $20, $21::jsonb,
			$22, $23, $24, NULLIF($25, ''), NULLIF($26, ''), $27, $28, NULLIF($29, ''), $30, $31, $32,
			$33, NULLIF($34, ''), NULLIF($35, ''), NULLIF($36, ''), $37, $38, $39, $40, $41
		)
		RETURNING
			id::text,
			business_id::text,
			location_id,
			location_name,
			COALESCE(landmark, ''),
			COALESCE(exact_address, ''),
			COALESCE(city, ''),
			COALESCE(zip_code, ''),
			COALESCE(state, ''),
			country,
			latitude,
			longitude,
			mobile,
			COALESCE(alternate_contact_number, ''),
			COALESCE(email, ''),
			COALESCE(website, ''),
			invoice_scheme,
			pos_invoice_layout,
			sale_invoice_layout,
			default_selling_price_group,
			COALESCE(payment_methods::text, '[]'),
			kra_pin,
			tax_jurisdiction,
			is_vat_registered,
			COALESCE(vat_number, ''),
			COALESCE(default_tax_type, ''),
			prices_include_tax,
			issue_tax_invoices,
			COALESCE(tax_note, ''),
			etims_enabled,
			environment,
			integration_type,
			is_head_office_branch,
			COALESCE(kra_branch_id, ''),
			COALESCE(device_serial_number, ''),
			COALESCE(cmc_key, ''),
			auto_submit_invoices,
			allow_offline_sales,
			retry_failed_invoices,
			print_qr_code,
			print_fiscal_details,
			created_at::text,
			updated_at::text
	`,
		req.BusinessID,
		req.LocationID,
		req.LocationCode,
		req.LocationName,
		nullIfBlank(req.Landmark),
		nullIfBlank(req.ExactAddress),
		nullIfBlank(req.City),
		nullIfBlank(req.ZipCode),
		nullIfBlank(req.State),
		req.Country,
		req.Latitude,
		req.Longitude,
		req.Mobile,
		nullIfBlank(req.AlternateContactNumber),
		nullIfBlank(req.Email),
		nullIfBlank(req.Website),
		req.InvoiceScheme,
		req.PosInvoiceLayout,
		req.SaleInvoiceLayout,
		req.DefaultSellingPriceGroup,
		paymentMethodsJSON,
		req.KraPin,
		req.TaxJurisdiction,
		req.IsVatRegistered,
		nullIfBlank(req.VatNumber),
		nullIfBlank(req.DefaultTaxType),
		req.PricesIncludeTax,
		req.IssueTaxInvoices,
		nullIfBlank(req.TaxNote),
		req.EtimsEnabled,
		req.Environment,
		req.IntegrationType,
		req.IsHeadOfficeBranch,
		nullIfBlank(req.KraBranchID),
		nullIfBlank(req.DeviceSerialNumber),
		nullIfBlank(req.CmcKey),
		req.AutoSubmitInvoices,
		req.AllowOfflineSales,
		req.RetryFailedInvoices,
		req.PrintQrCode,
		req.PrintFiscalDetails,
	).Scan(
		&location.ID,
		&location.BusinessID,
		&location.LocationID,
		&location.LocationCode,
		&location.LocationName,
		&location.Landmark,
		&location.ExactAddress,
		&location.City,
		&location.ZipCode,
		&location.State,
		&location.Country,
		&latitude,
		&longitude,
		&location.Mobile,
		&location.AlternateContactNumber,
		&location.Email,
		&location.Website,
		&location.InvoiceScheme,
		&location.PosInvoiceLayout,
		&location.SaleInvoiceLayout,
		&location.DefaultSellingPriceGroup,
		&rawPaymentMethods,
		&location.KraPin,
		&location.TaxJurisdiction,
		&location.IsVatRegistered,
		&location.VatNumber,
		&location.DefaultTaxType,
		&location.PricesIncludeTax,
		&location.IssueTaxInvoices,
		&location.TaxNote,
		&location.EtimsEnabled,
		&location.Environment,
		&location.IntegrationType,
		&location.IsHeadOfficeBranch,
		&location.KraBranchID,
		&location.DeviceSerialNumber,
		&location.CmcKey,
		&location.AutoSubmitInvoices,
		&location.AllowOfflineSales,
		&location.RetryFailedInvoices,
		&location.PrintQrCode,
		&location.PrintFiscalDetails,
		&location.CreatedAt,
		&location.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create business location: %w", err)
	}

	if latitude.Valid {
		value := latitude.Float64
		location.Latitude = &value
	}
	if longitude.Valid {
		value := longitude.Float64
		location.Longitude = &value
	}
	if err := json.Unmarshal([]byte(rawPaymentMethods), &location.PaymentMethods); err != nil {
		location.PaymentMethods = append([]string(nil), req.PaymentMethods...)
	}

	return &location, nil
}

func nullIfBlank(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.TrimSpace(value)
}
