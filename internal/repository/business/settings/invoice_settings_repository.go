package settings

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"pos/internal/models"
)

func ListBusinessInvoiceSettingsRepository(pool *pgxpool.Pool, businessID string) ([]models.BusinessInvoiceSettings, error) {
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
			COALESCE(name, ''),
			COALESCE(code, ''),
			COALESCE(product_label, ''),
			COALESCE(quantity_label, ''),
			COALESCE(unit_price_label, ''),
			COALESCE(sub_total_label, ''),
			COALESCE(category_hsn_code_label, ''),
			COALESCE(total_quantity_label, ''),
			COALESCE(item_discount_label, ''),
			COALESCE(discounted_unit_price_label, ''),
			COALESCE(subheading_line_1, ''),
			COALESCE(subheading_line_2, ''),
			COALESCE(subheading_line_3, ''),
			COALESCE(subheading_line_4, ''),
			COALESCE(subheading_line_5, ''),
			COALESCE(design, ''),
			COALESCE(paper_size, ''),
			COALESCE(is_default, FALSE),
			COALESCE(show_logo, FALSE),
			COALESCE(show_business_details, FALSE),
			COALESCE(show_customer_details, FALSE),
			COALESCE(show_items_sku, FALSE),
			COALESCE(show_brand, FALSE),
			COALESCE(show_sale_description, FALSE),
			COALESCE(show_qr_code, FALSE),
			COALESCE(show_product_expiry, FALSE),
			COALESCE(show_lot_number, FALSE),
			COALESCE(show_product_image, FALSE),
			COALESCE(show_warranty_name, FALSE),
			COALESCE(show_warranty_expiry_date, FALSE),
			COALESCE(show_warranty_description, FALSE),
			COALESCE(show_tax_breakdown, FALSE),
			COALESCE(show_discounts, FALSE),
			COALESCE(show_barcode, FALSE),
			COALESCE(barcode_total_due_label, ''),
			COALESCE(show_total_balance_due, FALSE),
			COALESCE(barcode_change_return_label, ''),
			COALESCE(hide_all_prices, FALSE),
			COALESCE(show_total_in_words, FALSE),
			COALESCE(barcode_word_format, ''),
			COALESCE(barcode_tax_summary_label, ''),
			COALESCE(header_alignment, ''),
			COALESCE(logo_url, ''),
			COALESCE(qr_show_labels, FALSE),
			COALESCE(qr_show_business_name, FALSE),
			COALESCE(qr_show_business_location_address, FALSE),
			COALESCE(qr_show_invoice_no, FALSE),
			COALESCE(qr_show_subtotal, FALSE),
			COALESCE(qr_show_total_amount_with_tax, FALSE),
			COALESCE(qr_show_total_tax, FALSE),
			COALESCE(qr_show_customer_name, FALSE),
			COALESCE(qr_show_invoice_url, FALSE),
			COALESCE(qr_show_invoice_date_time, FALSE),
			COALESCE(qr_show_business_tax1, FALSE),
			COALESCE(invoice_note, ''),
			created_at,
			updated_at
		FROM business_invoice_settings
		WHERE business_id = $1
		ORDER BY is_default DESC, created_at DESC
	`, businessID)
	if err != nil {
		return nil, fmt.Errorf("load business invoice settings: %w", err)
	}
	defer rows.Close()

	settings := make([]models.BusinessInvoiceSettings, 0)
	for rows.Next() {
		var item models.BusinessInvoiceSettings
		var createdAt time.Time
		var updatedAt time.Time

		if err := rows.Scan(
			&item.ID,
			&item.BusinessID,
			&item.Name,
			&item.Code,
			&item.ProductLabel,
			&item.QuantityLabel,
			&item.UnitPriceLabel,
			&item.SubTotalLabel,
			&item.CategoryHsnCodeLabel,
			&item.TotalQuantityLabel,
			&item.ItemDiscountLabel,
			&item.DiscountedUnitPriceLabel,
			&item.SubheadingLine1,
			&item.SubheadingLine2,
			&item.SubheadingLine3,
			&item.SubheadingLine4,
			&item.SubheadingLine5,
			&item.Design,
			&item.PaperSize,
			&item.IsDefault,
			&item.ShowLogo,
			&item.ShowBusinessDetails,
			&item.ShowCustomerDetails,
			&item.ShowItemsSku,
			&item.ShowBrand,
			&item.ShowSaleDescription,
			&item.ShowQrCode,
			&item.ShowProductExpiry,
			&item.ShowLotNumber,
			&item.ShowProductImage,
			&item.ShowWarrantyName,
			&item.ShowWarrantyExpiryDate,
			&item.ShowWarrantyDescription,
			&item.ShowTaxBreakdown,
			&item.ShowDiscounts,
			&item.ShowBarcode,
			&item.BarcodeTotalDueLabel,
			&item.ShowTotalBalanceDue,
			&item.BarcodeChangeReturnLabel,
			&item.HideAllPrices,
			&item.ShowTotalInWords,
			&item.BarcodeWordFormat,
			&item.BarcodeTaxSummaryLabel,
			&item.HeaderAlignment,
			&item.LogoURL,
			&item.QrShowLabels,
			&item.QrShowBusinessName,
			&item.QrShowBusinessLocationAddress,
			&item.QrShowInvoiceNo,
			&item.QrShowSubtotal,
			&item.QrShowTotalAmountWithTax,
			&item.QrShowTotalTax,
			&item.QrShowCustomerName,
			&item.QrShowInvoiceUrl,
			&item.QrShowInvoiceDateTime,
			&item.QrShowBusinessTax1,
			&item.InvoiceNote,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan business invoice settings: %w", err)
		}

		item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		item.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
		settings = append(settings, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate business invoice settings: %w", err)
	}

	return settings, nil
}

func GetBusinessInvoiceSettingRepository(pool *pgxpool.Pool, businessID, id string) (*models.BusinessInvoiceSettings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	id = strings.TrimSpace(id)
	if businessID == "" || id == "" {
		return nil, ErrBusinessNotResolved
	}

	var item models.BusinessInvoiceSettings
	var createdAt time.Time
	var updatedAt time.Time

	err := pool.QueryRow(ctx, `
		SELECT
			id::text,
			business_id::text,
			COALESCE(name, ''),
			COALESCE(code, ''),
			COALESCE(product_label, ''),
			COALESCE(quantity_label, ''),
			COALESCE(unit_price_label, ''),
			COALESCE(sub_total_label, ''),
			COALESCE(category_hsn_code_label, ''),
			COALESCE(total_quantity_label, ''),
			COALESCE(item_discount_label, ''),
			COALESCE(discounted_unit_price_label, ''),
			COALESCE(subheading_line_1, ''),
			COALESCE(subheading_line_2, ''),
			COALESCE(subheading_line_3, ''),
			COALESCE(subheading_line_4, ''),
			COALESCE(subheading_line_5, ''),
			COALESCE(design, ''),
			COALESCE(paper_size, ''),
			COALESCE(is_default, FALSE),
			COALESCE(show_logo, FALSE),
			COALESCE(show_business_details, FALSE),
			COALESCE(show_customer_details, FALSE),
			COALESCE(show_items_sku, FALSE),
			COALESCE(show_brand, FALSE),
			COALESCE(show_sale_description, FALSE),
			COALESCE(show_qr_code, FALSE),
			COALESCE(show_product_expiry, FALSE),
			COALESCE(show_lot_number, FALSE),
			COALESCE(show_product_image, FALSE),
			COALESCE(show_warranty_name, FALSE),
			COALESCE(show_warranty_expiry_date, FALSE),
			COALESCE(show_warranty_description, FALSE),
			COALESCE(show_tax_breakdown, FALSE),
			COALESCE(show_discounts, FALSE),
			COALESCE(show_barcode, FALSE),
			COALESCE(barcode_total_due_label, ''),
			COALESCE(show_total_balance_due, FALSE),
			COALESCE(barcode_change_return_label, ''),
			COALESCE(hide_all_prices, FALSE),
			COALESCE(show_total_in_words, FALSE),
			COALESCE(barcode_word_format, ''),
			COALESCE(barcode_tax_summary_label, ''),
			COALESCE(header_alignment, ''),
			COALESCE(logo_url, ''),
			COALESCE(qr_show_labels, FALSE),
			COALESCE(qr_show_business_name, FALSE),
			COALESCE(qr_show_business_location_address, FALSE),
			COALESCE(qr_show_invoice_no, FALSE),
			COALESCE(qr_show_subtotal, FALSE),
			COALESCE(qr_show_total_amount_with_tax, FALSE),
			COALESCE(qr_show_total_tax, FALSE),
			COALESCE(qr_show_customer_name, FALSE),
			COALESCE(qr_show_invoice_url, FALSE),
			COALESCE(qr_show_invoice_date_time, FALSE),
			COALESCE(qr_show_business_tax1, FALSE),
			COALESCE(invoice_note, ''),
			created_at,
			updated_at
		FROM business_invoice_settings
		WHERE business_id = $1 AND id = $2
		LIMIT 1
	`, businessID, id).Scan(
		&item.ID,
		&item.BusinessID,
		&item.Name,
		&item.Code,
		&item.ProductLabel,
		&item.QuantityLabel,
		&item.UnitPriceLabel,
		&item.SubTotalLabel,
		&item.CategoryHsnCodeLabel,
		&item.TotalQuantityLabel,
		&item.ItemDiscountLabel,
		&item.DiscountedUnitPriceLabel,
		&item.SubheadingLine1,
		&item.SubheadingLine2,
		&item.SubheadingLine3,
		&item.SubheadingLine4,
		&item.SubheadingLine5,
		&item.Design,
		&item.PaperSize,
		&item.IsDefault,
		&item.ShowLogo,
		&item.ShowBusinessDetails,
		&item.ShowCustomerDetails,
		&item.ShowItemsSku,
		&item.ShowBrand,
		&item.ShowSaleDescription,
		&item.ShowQrCode,
		&item.ShowProductExpiry,
		&item.ShowLotNumber,
		&item.ShowProductImage,
		&item.ShowWarrantyName,
		&item.ShowWarrantyExpiryDate,
		&item.ShowWarrantyDescription,
		&item.ShowTaxBreakdown,
		&item.ShowDiscounts,
		&item.ShowBarcode,
		&item.BarcodeTotalDueLabel,
		&item.ShowTotalBalanceDue,
		&item.BarcodeChangeReturnLabel,
		&item.HideAllPrices,
		&item.ShowTotalInWords,
		&item.BarcodeWordFormat,
		&item.BarcodeTaxSummaryLabel,
		&item.HeaderAlignment,
		&item.LogoURL,
		&item.QrShowLabels,
		&item.QrShowBusinessName,
		&item.QrShowBusinessLocationAddress,
		&item.QrShowInvoiceNo,
		&item.QrShowSubtotal,
		&item.QrShowTotalAmountWithTax,
		&item.QrShowTotalTax,
		&item.QrShowCustomerName,
		&item.QrShowInvoiceUrl,
		&item.QrShowInvoiceDateTime,
		&item.QrShowBusinessTax1,
		&item.InvoiceNote,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if errorsIsNoRows(err) {
			return nil, ErrBusinessInvoiceSettingsNotFound
		}
		return nil, fmt.Errorf("load business invoice setting: %w", err)
	}

	item.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	item.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)

	return &item, nil
}

func CreateBusinessInvoiceSettingsRepository(pool *pgxpool.Pool, req CreateBusinessInvoiceSettingsInput) (*models.BusinessInvoiceSettings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.Name = strings.TrimSpace(req.Name)
	req.Code = strings.TrimSpace(req.Code)
	req.ProductLabel = strings.TrimSpace(req.ProductLabel)
	req.QuantityLabel = strings.TrimSpace(req.QuantityLabel)
	req.UnitPriceLabel = strings.TrimSpace(req.UnitPriceLabel)
	req.SubTotalLabel = strings.TrimSpace(req.SubTotalLabel)
	req.CategoryHsnCodeLabel = strings.TrimSpace(req.CategoryHsnCodeLabel)
	req.TotalQuantityLabel = strings.TrimSpace(req.TotalQuantityLabel)
	req.ItemDiscountLabel = strings.TrimSpace(req.ItemDiscountLabel)
	req.DiscountedUnitPriceLabel = strings.TrimSpace(req.DiscountedUnitPriceLabel)
	req.SubheadingLine1 = strings.TrimSpace(req.SubheadingLine1)
	req.SubheadingLine2 = strings.TrimSpace(req.SubheadingLine2)
	req.SubheadingLine3 = strings.TrimSpace(req.SubheadingLine3)
	req.SubheadingLine4 = strings.TrimSpace(req.SubheadingLine4)
	req.SubheadingLine5 = strings.TrimSpace(req.SubheadingLine5)
	req.Design = strings.TrimSpace(req.Design)
	req.PaperSize = strings.TrimSpace(req.PaperSize)
	req.BarcodeTotalDueLabel = strings.TrimSpace(req.BarcodeTotalDueLabel)
	req.BarcodeChangeReturnLabel = strings.TrimSpace(req.BarcodeChangeReturnLabel)
	req.BarcodeWordFormat = strings.TrimSpace(req.BarcodeWordFormat)
	req.BarcodeTaxSummaryLabel = strings.TrimSpace(req.BarcodeTaxSummaryLabel)
	req.HeaderAlignment = strings.TrimSpace(req.HeaderAlignment)
	req.LogoURL = strings.TrimSpace(req.LogoURL)
	req.InvoiceNote = strings.TrimSpace(req.InvoiceNote)

	if req.Code == "" {
		req.Code = slugifyInvoiceSettingsCode(req.Name)
	}

	if req.LogoURL != "" {
		var err error
		req.LogoURL, err = normalizeBusinessInvoiceSettingsLogoURL(req.LogoURL)
		if err != nil {
			return nil, err
		}
	}

	if req.BusinessID == "" || req.Name == "" || req.Code == "" {
		return nil, ErrInvalidBusinessInvoiceSettingsInput
	}

	if !allowedInvoiceLayoutDesigns[req.Design] {
		req.Design = "classic"
	}
	if !allowedInvoicePaperSizes[req.PaperSize] {
		req.PaperSize = "a4"
	}
	if !allowedInvoiceHeaderAlignments[req.HeaderAlignment] {
		req.HeaderAlignment = "center"
	}
	if !allowedInvoiceBarcodeWordFormats[req.BarcodeWordFormat] {
		req.BarcodeWordFormat = "international"
	}
	if req.ProductLabel == "" {
		req.ProductLabel = "Product"
	}
	if req.QuantityLabel == "" {
		req.QuantityLabel = "Qty"
	}
	if req.UnitPriceLabel == "" {
		req.UnitPriceLabel = "Unit Price"
	}
	if req.SubTotalLabel == "" {
		req.SubTotalLabel = "Subtotal"
	}
	if req.CategoryHsnCodeLabel == "" {
		req.CategoryHsnCodeLabel = "Category / HSN Code"
	}
	if req.TotalQuantityLabel == "" {
		req.TotalQuantityLabel = "Total Quantity"
	}
	if req.ItemDiscountLabel == "" {
		req.ItemDiscountLabel = "Item Discount"
	}
	if req.DiscountedUnitPriceLabel == "" {
		req.DiscountedUnitPriceLabel = "Discounted Unit Price"
	}
	if req.BarcodeTotalDueLabel == "" {
		req.BarcodeTotalDueLabel = "Due"
	}
	if req.BarcodeChangeReturnLabel == "" {
		req.BarcodeChangeReturnLabel = "Change return label"
	}
	if req.BarcodeTaxSummaryLabel == "" {
		req.BarcodeTaxSummaryLabel = "Tax summary label"
	}
	if req.InvoiceNote == "" {
		req.InvoiceNote = "Payment is due upon receipt unless otherwise agreed."
	}

	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin business invoice settings create tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var duplicateID string
	err = tx.QueryRow(ctx, `
		SELECT id::text
		FROM business_invoice_settings
		WHERE business_id = $1 AND code = $2
		LIMIT 1
	`, req.BusinessID, req.Code).Scan(&duplicateID)
	if err == nil {
		return nil, ErrBusinessInvoiceSettingsDuplicateCode
	}
	if !errorsIsNoRows(err) {
		return nil, fmt.Errorf("check business invoice settings duplicate code: %w", err)
	}

	if req.IsDefault {
		if _, err := tx.Exec(ctx, `
			UPDATE business_invoice_settings
			SET is_default = FALSE, updated_at = CURRENT_TIMESTAMP
			WHERE business_id = $1
		`, req.BusinessID); err != nil {
			return nil, fmt.Errorf("reset business invoice settings defaults: %w", err)
		}
	}

	var createdID string
	var createdAt time.Time
	var updatedAt time.Time
	err = tx.QueryRow(ctx, `
		INSERT INTO business_invoice_settings (
			business_id,
			name,
			code,
			product_label,
			quantity_label,
			unit_price_label,
			sub_total_label,
			category_hsn_code_label,
			total_quantity_label,
			item_discount_label,
			discounted_unit_price_label,
			subheading_line_1,
			subheading_line_2,
			subheading_line_3,
			subheading_line_4,
			subheading_line_5,
			design,
			paper_size,
			is_default,
			show_logo,
			show_business_details,
			show_customer_details,
			show_items_sku,
			show_brand,
			show_sale_description,
			show_qr_code,
			show_product_expiry,
			show_lot_number,
			show_product_image,
			show_warranty_name,
			show_warranty_expiry_date,
			show_warranty_description,
			show_tax_breakdown,
			show_discounts,
			show_barcode,
			barcode_total_due_label,
			show_total_balance_due,
			barcode_change_return_label,
			hide_all_prices,
			show_total_in_words,
			barcode_word_format,
			barcode_tax_summary_label,
			header_alignment,
			logo_url,
			qr_show_labels,
			qr_show_business_name,
			qr_show_business_location_address,
			qr_show_invoice_no,
			qr_show_subtotal,
			qr_show_total_amount_with_tax,
			qr_show_total_tax,
			qr_show_customer_name,
			qr_show_invoice_url,
			qr_show_invoice_date_time,
			qr_show_business_tax1,
			invoice_note
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37,$38,$39,$40,$41,$42,$43,$44,$45,$46,$47,$48,$49,$50,$51,$52,$53,$54,$55,$56
		)
		RETURNING
			id::text,
			created_at,
			updated_at
	`, req.BusinessID, req.Name, req.Code, req.ProductLabel, req.QuantityLabel, req.UnitPriceLabel, req.SubTotalLabel, req.CategoryHsnCodeLabel, req.TotalQuantityLabel, req.ItemDiscountLabel, req.DiscountedUnitPriceLabel, req.SubheadingLine1, req.SubheadingLine2, req.SubheadingLine3, req.SubheadingLine4, req.SubheadingLine5, req.Design, req.PaperSize, req.IsDefault, req.ShowLogo, req.ShowBusinessDetails, req.ShowCustomerDetails, req.ShowItemsSku, req.ShowBrand, req.ShowSaleDescription, req.ShowQrCode, req.ShowProductExpiry, req.ShowLotNumber, req.ShowProductImage, req.ShowWarrantyName, req.ShowWarrantyExpiryDate, req.ShowWarrantyDescription, req.ShowTaxBreakdown, req.ShowDiscounts, req.ShowBarcode, req.BarcodeTotalDueLabel, req.ShowTotalBalanceDue, req.BarcodeChangeReturnLabel, req.HideAllPrices, req.ShowTotalInWords, req.BarcodeWordFormat, req.BarcodeTaxSummaryLabel, req.HeaderAlignment, req.LogoURL, req.QrShowLabels, req.QrShowBusinessName, req.QrShowBusinessLocationAddress, req.QrShowInvoiceNo, req.QrShowSubtotal, req.QrShowTotalAmountWithTax, req.QrShowTotalTax, req.QrShowCustomerName, req.QrShowInvoiceUrl, req.QrShowInvoiceDateTime, req.QrShowBusinessTax1, req.InvoiceNote).Scan(&createdID, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("insert business invoice settings: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit business invoice settings create tx: %w", err)
	}

	return GetBusinessInvoiceSettingRepository(pool, req.BusinessID, createdID)
}

func UpdateBusinessInvoiceSettingsRepository(pool *pgxpool.Pool, req UpdateBusinessInvoiceSettingsInput) (*models.BusinessInvoiceSettings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req.ID = strings.TrimSpace(req.ID)
	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.Name = strings.TrimSpace(req.Name)
	req.Code = strings.TrimSpace(req.Code)
	req.ProductLabel = strings.TrimSpace(req.ProductLabel)
	req.QuantityLabel = strings.TrimSpace(req.QuantityLabel)
	req.UnitPriceLabel = strings.TrimSpace(req.UnitPriceLabel)
	req.SubTotalLabel = strings.TrimSpace(req.SubTotalLabel)
	req.CategoryHsnCodeLabel = strings.TrimSpace(req.CategoryHsnCodeLabel)
	req.TotalQuantityLabel = strings.TrimSpace(req.TotalQuantityLabel)
	req.ItemDiscountLabel = strings.TrimSpace(req.ItemDiscountLabel)
	req.DiscountedUnitPriceLabel = strings.TrimSpace(req.DiscountedUnitPriceLabel)
	req.SubheadingLine1 = strings.TrimSpace(req.SubheadingLine1)
	req.SubheadingLine2 = strings.TrimSpace(req.SubheadingLine2)
	req.SubheadingLine3 = strings.TrimSpace(req.SubheadingLine3)
	req.SubheadingLine4 = strings.TrimSpace(req.SubheadingLine4)
	req.SubheadingLine5 = strings.TrimSpace(req.SubheadingLine5)
	req.Design = strings.TrimSpace(req.Design)
	req.PaperSize = strings.TrimSpace(req.PaperSize)
	req.BarcodeTotalDueLabel = strings.TrimSpace(req.BarcodeTotalDueLabel)
	req.BarcodeChangeReturnLabel = strings.TrimSpace(req.BarcodeChangeReturnLabel)
	req.BarcodeWordFormat = strings.TrimSpace(req.BarcodeWordFormat)
	req.BarcodeTaxSummaryLabel = strings.TrimSpace(req.BarcodeTaxSummaryLabel)
	req.HeaderAlignment = strings.TrimSpace(req.HeaderAlignment)
	req.LogoURL = strings.TrimSpace(req.LogoURL)
	req.InvoiceNote = strings.TrimSpace(req.InvoiceNote)

	if req.Code == "" {
		req.Code = slugifyInvoiceSettingsCode(req.Name)
	}

	if req.LogoURL != "" {
		var err error
		req.LogoURL, err = normalizeBusinessInvoiceSettingsLogoURL(req.LogoURL)
		if err != nil {
			return nil, err
		}
	}

	if req.BusinessID == "" || req.ID == "" || req.Name == "" || req.Code == "" {
		return nil, ErrInvalidBusinessInvoiceSettingsInput
	}

	if !allowedInvoiceLayoutDesigns[req.Design] {
		req.Design = "classic"
	}
	if !allowedInvoicePaperSizes[req.PaperSize] {
		req.PaperSize = "a4"
	}
	if !allowedInvoiceHeaderAlignments[req.HeaderAlignment] {
		req.HeaderAlignment = "center"
	}
	if !allowedInvoiceBarcodeWordFormats[req.BarcodeWordFormat] {
		req.BarcodeWordFormat = "international"
	}
	if req.ProductLabel == "" {
		req.ProductLabel = "Product"
	}
	if req.QuantityLabel == "" {
		req.QuantityLabel = "Qty"
	}
	if req.UnitPriceLabel == "" {
		req.UnitPriceLabel = "Unit Price"
	}
	if req.SubTotalLabel == "" {
		req.SubTotalLabel = "Subtotal"
	}
	if req.CategoryHsnCodeLabel == "" {
		req.CategoryHsnCodeLabel = "Category / HSN Code"
	}
	if req.TotalQuantityLabel == "" {
		req.TotalQuantityLabel = "Total Quantity"
	}
	if req.ItemDiscountLabel == "" {
		req.ItemDiscountLabel = "Item Discount"
	}
	if req.DiscountedUnitPriceLabel == "" {
		req.DiscountedUnitPriceLabel = "Discounted Unit Price"
	}
	if req.BarcodeTotalDueLabel == "" {
		req.BarcodeTotalDueLabel = "Due"
	}
	if req.BarcodeChangeReturnLabel == "" {
		req.BarcodeChangeReturnLabel = "Change return label"
	}
	if req.BarcodeTaxSummaryLabel == "" {
		req.BarcodeTaxSummaryLabel = "Tax summary label"
	}
	if req.InvoiceNote == "" {
		req.InvoiceNote = "Payment is due upon receipt unless otherwise agreed."
	}

	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin business invoice settings update tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var duplicateID string
	err = tx.QueryRow(ctx, `
		SELECT id::text
		FROM business_invoice_settings
		WHERE business_id = $1 AND code = $2 AND id <> $3
		LIMIT 1
	`, req.BusinessID, req.Code, req.ID).Scan(&duplicateID)
	if err == nil {
		return nil, ErrBusinessInvoiceSettingsDuplicateCode
	}
	if !errorsIsNoRows(err) {
		return nil, fmt.Errorf("check business invoice settings duplicate code: %w", err)
	}

	if req.IsDefault {
		if _, err := tx.Exec(ctx, `
			UPDATE business_invoice_settings
			SET is_default = FALSE, updated_at = CURRENT_TIMESTAMP
			WHERE business_id = $1 AND id <> $2
		`, req.BusinessID, req.ID); err != nil {
			return nil, fmt.Errorf("reset business invoice settings defaults: %w", err)
		}
	}

	var createdAt time.Time
	var updatedAt time.Time
	err = tx.QueryRow(ctx, `
		UPDATE business_invoice_settings
		SET name = $2,
			code = $3,
			product_label = $4,
			quantity_label = $5,
			unit_price_label = $6,
			sub_total_label = $7,
			category_hsn_code_label = $8,
			total_quantity_label = $9,
			item_discount_label = $10,
			discounted_unit_price_label = $11,
			subheading_line_1 = $12,
			subheading_line_2 = $13,
			subheading_line_3 = $14,
			subheading_line_4 = $15,
			subheading_line_5 = $16,
			design = $17,
			paper_size = $18,
			is_default = $19,
			show_logo = $20,
			show_business_details = $21,
			show_customer_details = $22,
			show_items_sku = $23,
			show_brand = $24,
			show_sale_description = $25,
			show_qr_code = $26,
			show_product_expiry = $27,
			show_lot_number = $28,
			show_product_image = $29,
			show_warranty_name = $30,
			show_warranty_expiry_date = $31,
			show_warranty_description = $32,
			show_tax_breakdown = $33,
			show_discounts = $34,
			show_barcode = $35,
			barcode_total_due_label = $36,
			show_total_balance_due = $37,
			barcode_change_return_label = $38,
			hide_all_prices = $39,
			show_total_in_words = $40,
			barcode_word_format = $41,
			barcode_tax_summary_label = $42,
			header_alignment = $43,
			logo_url = $44,
			qr_show_labels = $45,
			qr_show_business_name = $46,
			qr_show_business_location_address = $47,
			qr_show_invoice_no = $48,
			qr_show_subtotal = $49,
			qr_show_total_amount_with_tax = $50,
			qr_show_total_tax = $51,
			qr_show_customer_name = $52,
			qr_show_invoice_url = $53,
			qr_show_invoice_date_time = $54,
			qr_show_business_tax1 = $55,
			invoice_note = $56,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND business_id = $57
		RETURNING created_at, updated_at
	`, req.ID, req.Name, req.Code, req.ProductLabel, req.QuantityLabel, req.UnitPriceLabel, req.SubTotalLabel, req.CategoryHsnCodeLabel, req.TotalQuantityLabel, req.ItemDiscountLabel, req.DiscountedUnitPriceLabel, req.SubheadingLine1, req.SubheadingLine2, req.SubheadingLine3, req.SubheadingLine4, req.SubheadingLine5, req.Design, req.PaperSize, req.IsDefault, req.ShowLogo, req.ShowBusinessDetails, req.ShowCustomerDetails, req.ShowItemsSku, req.ShowBrand, req.ShowSaleDescription, req.ShowQrCode, req.ShowProductExpiry, req.ShowLotNumber, req.ShowProductImage, req.ShowWarrantyName, req.ShowWarrantyExpiryDate, req.ShowWarrantyDescription, req.ShowTaxBreakdown, req.ShowDiscounts, req.ShowBarcode, req.BarcodeTotalDueLabel, req.ShowTotalBalanceDue, req.BarcodeChangeReturnLabel, req.HideAllPrices, req.ShowTotalInWords, req.BarcodeWordFormat, req.BarcodeTaxSummaryLabel, req.HeaderAlignment, req.LogoURL, req.QrShowLabels, req.QrShowBusinessName, req.QrShowBusinessLocationAddress, req.QrShowInvoiceNo, req.QrShowSubtotal, req.QrShowTotalAmountWithTax, req.QrShowTotalTax, req.QrShowCustomerName, req.QrShowInvoiceUrl, req.QrShowInvoiceDateTime, req.QrShowBusinessTax1, req.InvoiceNote, req.BusinessID).Scan(&createdAt, &updatedAt)
	if err != nil {
		if errorsIsNoRows(err) {
			return nil, ErrBusinessInvoiceSettingsNotFound
		}
		return nil, fmt.Errorf("update business invoice settings: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit business invoice settings update tx: %w", err)
	}

	return GetBusinessInvoiceSettingRepository(pool, req.BusinessID, req.ID)
}

func slugifyInvoiceSettingsCode(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "-")
	var builder strings.Builder
	lastDash := false

	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				builder.WriteRune('-')
				lastDash = true
			}
		}
	}

	return strings.Trim(builder.String(), "-")
}

func errorsIsNoRows(err error) bool {
	return err == pgx.ErrNoRows
}

var allowedInvoiceLayoutDesigns = map[string]bool{
	"classic": true,
	"modern":  true,
	"minimal": true,
	"compact": true,
}

var allowedInvoicePaperSizes = map[string]bool{
	"a4":      true,
	"thermal": true,
}

var allowedInvoiceHeaderAlignments = map[string]bool{
	"left":   true,
	"center": true,
	"right":  true,
}

var allowedInvoiceBarcodeWordFormats = map[string]bool{
	"international": true,
	"indian":        true,
}
