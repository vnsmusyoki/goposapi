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

func ListBusinessLocationsRepository(pool *pgxpool.Pool, businessID string) ([]models.BusinessLocation, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	if businessID == "" {
		return nil, ErrBusinessNotResolved
	}

	rows, err := pool.Query(ctx, `
		SELECT
			id::text,
			business_id::text,
			location_id,
			COALESCE(location_code, location_id),
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
		FROM business_locations
		WHERE business_id = $1
		ORDER BY created_at DESC, location_name ASC
	`, businessID)
	if err != nil {
		return nil, fmt.Errorf("list business locations: %w", err)
	}
	defer rows.Close()

	locations := make([]models.BusinessLocation, 0)
	for rows.Next() {
		var location models.BusinessLocation
		var latitude sql.NullFloat64
		var longitude sql.NullFloat64
		var paymentMethodsJSON string

		if err := rows.Scan(
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
			&paymentMethodsJSON,
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
		); err != nil {
			return nil, fmt.Errorf("scan business location: %w", err)
		}

		if latitude.Valid {
			value := latitude.Float64
			location.Latitude = &value
		}
		if longitude.Valid {
			value := longitude.Float64
			location.Longitude = &value
		}
		if err := json.Unmarshal([]byte(paymentMethodsJSON), &location.PaymentMethods); err != nil {
			location.PaymentMethods = nil
		}

		locations = append(locations, location)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate business locations: %w", err)
	}

	return locations, nil
}
