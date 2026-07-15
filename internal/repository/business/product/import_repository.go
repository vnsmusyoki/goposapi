package product

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type ProductImportRowData struct {
	Name                 string  `json:"name"`
	SKU                  string  `json:"sku"`
	Barcode              string  `json:"barcode"`
	ProductType          string  `json:"productType"`
	Unit                 string  `json:"unit"`
	Brand                string  `json:"brand"`
	CategoryCode         string  `json:"categoryCode"`
	SubCategoryCode      string  `json:"subCategoryCode"`
	LocationCode         string  `json:"locationCode"`
	ManageStock          bool    `json:"manageStock"`
	AlertQuantity        int     `json:"alertQuantity"`
	IsForSelling         bool    `json:"isForSelling"`
	TaxType              string  `json:"taxType"`
	TaxRate              float64 `json:"taxRate"`
	DefaultPurchasePrice float64 `json:"defaultPurchasePrice"`
	DefaultSellingPrice  float64 `json:"defaultSellingPrice"`
	Description          string  `json:"description"`
}

func CreateProductImportPreviewRepository(pool *pgxpool.Pool, businessID, fileName, createdBy string, rows []ProductImportBatchRow) (*ProductImportBatch, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	fileName = strings.TrimSpace(fileName)
	createdBy = strings.TrimSpace(createdBy)
	if businessID == "" {
		return nil, ErrBusinessNotResolved
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin product import tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	batch := ProductImportBatch{}
	if err := tx.QueryRow(ctx, `
		INSERT INTO product_import_batches (
			business_id,
			file_name,
			status,
			created_by
		)
		VALUES ($1::uuid, $2, 'previewed', NULLIF($3, '')::uuid)
		RETURNING id::text, business_id::text, file_name, status, COALESCE(created_by::text, ''), created_at::text, updated_at::text
	`, businessID, fileName, createdBy).Scan(&batch.ID, &batch.BusinessID, &batch.FileName, &batch.Status, &batch.CreatedBy, &batch.CreatedAt, &batch.UpdatedAt); err != nil {
		return nil, fmt.Errorf("create product import batch: %w", err)
	}

	for _, row := range rows {
		rowDataJSON, err := json.Marshal(row.RowData)
		if err != nil {
			return nil, fmt.Errorf("marshal product import row data: %w", err)
		}
		errorsJSON, err := json.Marshal(row.ValidationErrors)
		if err != nil {
			return nil, fmt.Errorf("marshal product import row errors: %w", err)
		}
		productInput, validationErrors, err := BuildProductImportProductInput(pool, row.RowData, businessID, createdBy)
		if err != nil {
			return nil, err
		}
		if len(validationErrors) > 0 {
			mergedErrors := append(append([]string{}, row.ValidationErrors...), validationErrors...)
			errorsJSON, err = json.Marshal(mergedErrors)
			if err != nil {
				return nil, fmt.Errorf("marshal product import row errors: %w", err)
			}
			row.Status = "invalid"
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO product_import_batch_rows (
				batch_id,
				row_number,
				name,
				sku,
				barcode,
				product_type,
				unit_id,
				brand_id,
				category_id,
				sub_category_id,
				is_for_selling,
				manage_stock,
				alert_quantity,
				tax_type,
				tax_rate,
				default_purchase_price,
				purchase_price_exclusive,
				purchase_price_inclusive,
				profit_margin,
				default_selling_price,
				description,
				brochure_name,
				brochure_url,
				currency_code,
				currency_symbol_placement,
				currency_precision,
				all_locations,
				has_warranty,
				warranty_duration,
				warranty_period,
				warranty_coverage,
				created_by,
				row_data,
				validation_errors,
				status
			)
			VALUES (
				$1::uuid,
				$2,
				$3,
				$4,
				$5,
				$6,
				NULLIF($7, '')::uuid,
				NULLIF($8, '')::uuid,
				NULLIF($9, '')::uuid,
				$10::integer,
				$11,
				$12,
				$13::integer,
				$14,
				$15::numeric(10,2),
				$16::numeric(14,4),
				$17::numeric(14,4),
				$18::numeric(14,4),
				$19::numeric(10,2),
				$20::numeric(14,4),
				$21,
				$22,
				$23,
				$24,
				$25,
				$26,
				$27,
				$28,
				$29,
				$30,
				$31,
				NULLIF($32, '')::uuid,
				$33::jsonb,
				$34::jsonb,
				$35
			)
		`, batch.ID, row.RowNumber,
			productInput.Name,
			productInput.SKU,
			productInput.Barcode,
			productInput.ProductType,
			productInput.UnitID,
			productInput.BrandID,
			productInput.CategoryID,
			productInput.SubCategoryID,
			productInput.IsForSelling,
			productInput.ManageStock,
			productInput.AlertQuantity,
			productInput.TaxType,
			productInput.TaxRate,
			productInput.DefaultPurchasePrice,
			productInput.PurchasePriceExclusive,
			productInput.PurchasePriceInclusive,
			productInput.ProfitMargin,
			productInput.DefaultSellingPrice,
			productInput.Description,
			productInput.BrochureName,
			productInput.BrochureURL,
			productInput.CurrencyCode,
			productInput.CurrencySymbolPlacement,
			productInput.CurrencyPrecision,
			productInput.AllLocations,
			productInput.HasWarranty,
			productInput.WarrantyDuration,
			productInput.WarrantyPeriod,
			productInput.WarrantyCoverage,
			productInput.CreatedBy,
			rowDataJSON, errorsJSON, row.Status); err != nil {
			return nil, fmt.Errorf("insert product import row: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit product import batch: %w", err)
	}

	return &batch, nil
}

func ListProductImportBatchRowsRepository(pool *pgxpool.Pool, businessID, batchID string) (*ProductImportBatch, []ProductImportBatchRow, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	batchID = strings.TrimSpace(batchID)
	if businessID == "" || batchID == "" {
		return nil, nil, ErrBusinessNotResolved
	}

	var batch ProductImportBatch
	if err := pool.QueryRow(ctx, `
		SELECT id::text, business_id::text, file_name, status, COALESCE(created_by::text, ''), created_at::text, updated_at::text
		FROM product_import_batches
		WHERE business_id = $1::uuid
		  AND id = $2::uuid
	`, businessID, batchID).Scan(&batch.ID, &batch.BusinessID, &batch.FileName, &batch.Status, &batch.CreatedBy, &batch.CreatedAt, &batch.UpdatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil, ErrProductNotFound
		}
		return nil, nil, fmt.Errorf("load product import batch: %w", err)
	}

	rows, err := pool.Query(ctx, `
		SELECT
			id::text,
			batch_id::text,
			row_number,
			row_data::text,
			COALESCE(validation_errors::text, '[]'),
			status,
			COALESCE(imported_product_id::text, ''),
			created_at::text,
			updated_at::text
		FROM product_import_batch_rows
		WHERE batch_id = $1::uuid
		ORDER BY row_number ASC, created_at ASC
	`, batch.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("list product import rows: %w", err)
	}
	defer rows.Close()

	items := make([]ProductImportBatchRow, 0)
	for rows.Next() {
		var item ProductImportBatchRow
		var rowDataJSON string
		var errorsJSON string
		var importedProductID string
		if err := rows.Scan(&item.ID, &item.BatchID, &item.RowNumber, &rowDataJSON, &errorsJSON, &item.Status, &importedProductID, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, nil, fmt.Errorf("scan product import row: %w", err)
		}
		item.ImportedProductID = importedProductID
		_ = json.Unmarshal([]byte(rowDataJSON), &item.RowData)
		_ = json.Unmarshal([]byte(errorsJSON), &item.ValidationErrors)
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate product import rows: %w", err)
	}

	return &batch, items, nil
}

func GetLatestProductImportBatchRepository(pool *pgxpool.Pool, businessID string) (*ProductImportBatch, []ProductImportBatchRow, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	if businessID == "" {
		return nil, nil, ErrBusinessNotResolved
	}

	var batchID string
	if err := pool.QueryRow(ctx, `
		SELECT uuid_id::text
		FROM product_import_batches
		WHERE business_id = $1::uuid
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	`, businessID).Scan(&batchID); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil, ErrProductNotFound
		}
		return nil, nil, fmt.Errorf("load latest product import batch: %w", err)
	}

	return ListProductImportBatchRowsRepository(pool, businessID, batchID)
}

func GetProductImportBatchRowRepository(pool *pgxpool.Pool, businessID, batchID, rowID string) (*ProductImportBatchRow, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	batchID = strings.TrimSpace(batchID)
	rowID = strings.TrimSpace(rowID)
	if businessID == "" || batchID == "" || rowID == "" {
		return nil, ErrBusinessNotResolved
	}

	row := pool.QueryRow(ctx, `
		SELECT
			r.id::text,
			r.batch_id::text,
			r.row_number,
			r.row_data::text,
			COALESCE(r.validation_errors::text, '[]'),
			r.status,
			COALESCE(r.imported_product_id::text, ''),
			r.created_at::text,
			r.updated_at::text
		FROM product_import_batch_rows r
		INNER JOIN product_import_batches b ON b.id = r.batch_id
		WHERE b.business_id = $1::uuid
		  AND b.id = $2::uuid
		  AND r.id = $3::uuid
	`, businessID, batchID, rowID)

	var item ProductImportBatchRow
	var rowDataJSON string
	var errorsJSON string
	if err := row.Scan(&item.ID, &item.BatchID, &item.RowNumber, &rowDataJSON, &errorsJSON, &item.Status, &item.ImportedProductID, &item.CreatedAt, &item.UpdatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrProductNotFound
		}
		return nil, fmt.Errorf("load product import row: %w", err)
	}
	_ = json.Unmarshal([]byte(rowDataJSON), &item.RowData)
	_ = json.Unmarshal([]byte(errorsJSON), &item.ValidationErrors)
	return &item, nil
}

func UpdateProductImportBatchRowStatusRepository(pool *pgxpool.Pool, businessID, batchID, rowID, status, importedProductID string, validationErrors []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	batchID = strings.TrimSpace(batchID)
	rowID = strings.TrimSpace(rowID)
	status = strings.TrimSpace(status)
	importedProductID = strings.TrimSpace(importedProductID)
	if businessID == "" || batchID == "" || rowID == "" {
		return ErrBusinessNotResolved
	}

	errorsJSON, err := json.Marshal(validationErrors)
	if err != nil {
		return fmt.Errorf("marshal product import validation errors: %w", err)
	}

	_, err = pool.Exec(ctx, `
		UPDATE product_import_batch_rows r
		SET status = $4,
		    imported_product_id = NULLIF($5, '')::uuid,
		    validation_errors = $6::jsonb,
		    updated_at = CURRENT_TIMESTAMP
		FROM product_import_batches b
		WHERE b.id = r.batch_id
		  AND b.business_id = $1::uuid
		  AND b.id = $2::uuid
		  AND r.id = $3::uuid
	`, businessID, batchID, rowID, status, importedProductID, errorsJSON)
	if err != nil {
		return fmt.Errorf("update product import row status: %w", err)
	}

	return nil
}

func UpdateProductImportBatchRowDataRepository(pool *pgxpool.Pool, businessID, batchID, rowID string, rowData map[string]string, createdBy string) (*ProductImportBatchRow, []string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	batchID = strings.TrimSpace(batchID)
	rowID = strings.TrimSpace(rowID)
	createdBy = strings.TrimSpace(createdBy)
	if businessID == "" || batchID == "" || rowID == "" {
		return nil, nil, ErrBusinessNotResolved
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("begin product import row update tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	row, err := getProductImportBatchRowTx(ctx, tx, businessID, batchID, rowID)
	if err != nil {
		return nil, nil, err
	}

	mergedData := make(map[string]string, len(row.RowData))
	for key, value := range row.RowData {
		mergedData[key] = value
	}
	for key, value := range rowData {
		mergedData[key] = strings.TrimSpace(value)
	}

	productInput, validationErrors, err := BuildProductImportProductInput(tx, mergedData, businessID, createdBy)
	if err != nil {
		return nil, nil, err
	}

	status := "valid"
	if len(validationErrors) > 0 {
		status = "invalid"
	}

	rowDataJSON, err := json.Marshal(mergedData)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal updated product import row data: %w", err)
	}
	errorsJSON, err := json.Marshal(validationErrors)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal updated product import row errors: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE product_import_batch_rows r
		SET row_data = $4::jsonb,
		    validation_errors = $5::jsonb,
		    name = $6,
		    sku = $7,
		    barcode = $8,
		    product_type = $9,
		    unit_id = NULLIF($10, '')::uuid,
		    brand_id = NULLIF($11, '')::uuid,
		    category_id = NULLIF($12, '')::uuid,
		    sub_category_id = $13::uuid,
		    is_for_selling = $14,
		    manage_stock = $15,
		    alert_quantity = $16::integer,
		    tax_type = $17,
		    tax_rate = $18::numeric(10,2),
		    default_purchase_price = $19::numeric(14,4),
		    purchase_price_exclusive = $20::numeric(14,4),
		    purchase_price_inclusive = $21::numeric(14,4),
		    profit_margin = $22::numeric(10,2),
		    default_selling_price = $23::numeric(14,4),
		    description = $24,
		    brochure_name = $25,
		    brochure_url = $26,
		    currency_code = $27,
		    currency_symbol_placement = $28,
		    currency_precision = $29,
		    all_locations = $30,
		    has_warranty = $31,
		    warranty_duration = NULLIF($32, ''),
		    warranty_period = NULLIF($33, ''),
		    warranty_coverage = NULLIF($34, ''),
		    created_by = NULLIF($35, '')::uuid,
		    status = $36,
		    imported_product_id = NULL,
		    updated_at = CURRENT_TIMESTAMP
		FROM product_import_batches b
		WHERE b.id = r.batch_id
		  AND b.business_id = $1::uuid
		  AND b.id = $2::uuid
		  AND r.id = $3::uuid
	`, businessID, batchID, rowID, rowDataJSON, errorsJSON,
		productInput.Name,
		productInput.SKU,
		productInput.Barcode,
		productInput.ProductType,
		productInput.UnitID,
		productInput.BrandID,
		productInput.CategoryID,
		productInput.SubCategoryID,
		productInput.IsForSelling,
		productInput.ManageStock,
		productInput.AlertQuantity,
		productInput.TaxType,
		productInput.TaxRate,
		productInput.DefaultPurchasePrice,
		productInput.PurchasePriceExclusive,
		productInput.PurchasePriceInclusive,
		productInput.ProfitMargin,
		productInput.DefaultSellingPrice,
		productInput.Description,
		productInput.BrochureName,
		productInput.BrochureURL,
		productInput.CurrencyCode,
		productInput.CurrencySymbolPlacement,
		productInput.CurrencyPrecision,
		productInput.AllLocations,
		productInput.HasWarranty,
		productInput.WarrantyDuration,
		productInput.WarrantyPeriod,
		productInput.WarrantyCoverage,
		productInput.CreatedBy,
		status); err != nil {
		return nil, nil, fmt.Errorf("update product import row data: %w", err)
	}

	updatedRow, err := getProductImportBatchRowTx(ctx, tx, businessID, batchID, rowID)
	if err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("commit product import row update tx: %w", err)
	}

	return updatedRow, validationErrors, nil
}

func getProductImportBatchRowTx(ctx context.Context, tx interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, businessID, batchID, rowID string) (*ProductImportBatchRow, error) {
	row := tx.QueryRow(ctx, `
		SELECT
			r.id::text,
			r.batch_id::text,
			r.row_number,
			r.row_data::text,
			COALESCE(r.validation_errors::text, '[]'),
			r.status,
			COALESCE(r.imported_product_id::text, ''),
			r.created_at::text,
			r.updated_at::text
		FROM product_import_batch_rows r
		INNER JOIN product_import_batches b ON b.id = r.batch_id
		WHERE b.business_id = $1::uuid
		  AND b.id = $2::uuid
		  AND r.id = $3::uuid
	`, businessID, batchID, rowID)

	var item ProductImportBatchRow
	var rowDataJSON string
	var errorsJSON string
	if err := row.Scan(&item.ID, &item.BatchID, &item.RowNumber, &rowDataJSON, &errorsJSON, &item.Status, &item.ImportedProductID, &item.CreatedAt, &item.UpdatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrProductNotFound
		}
		return nil, fmt.Errorf("load product import row: %w", err)
	}
	_ = json.Unmarshal([]byte(rowDataJSON), &item.RowData)
	_ = json.Unmarshal([]byte(errorsJSON), &item.ValidationErrors)
	return &item, nil
}

func BuildProductImportProductInput(q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, rowData map[string]string, businessID, createdBy string) (CreateProductInput, []string, error) {
	ctx := context.Background()
	return buildProductImportProductInputTx(ctx, q, rowData, businessID, createdBy)
}

func buildProductImportProductInputTx(ctx context.Context, tx interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, rowData map[string]string, businessID, createdBy string) (CreateProductInput, []string, error) {
	errs := make([]string, 0)
	normalize := func(key string) string {
		return strings.TrimSpace(rowData[key])
	}

	name := normalize("name")
	if name == "" {
		errs = append(errs, "Product name is required.")
	}
	productType := strings.ToLower(normalize("product_type"))
	if productType == "" {
		productType = "single"
	}
	if productType != "single" {
		errs = append(errs, "Only single products are supported for import.")
	}
	unitValue := normalize("unit")
	categoryCode := normalize("category_code")
	locationCode := normalize("location_code")
	if unitValue == "" {
		errs = append(errs, "Unit is required.")
	}
	if categoryCode == "" {
		errs = append(errs, "Category code is required.")
	}
	if locationCode == "" {
		errs = append(errs, "Location code is required.")
	}

	unitID, err := lookupBusinessUnitIDByNameOrShortNameTx(ctx, tx, businessID, unitValue)
	if err != nil {
		return CreateProductInput{}, errs, err
	}
	if unitID == "" {
		errs = append(errs, "Unit was not found.")
	}

	categoryID, err := lookupCategoryIDByCodeTx(ctx, tx, businessID, categoryCode)
	if err != nil {
		return CreateProductInput{}, errs, err
	}
	if categoryID == "" {
		errs = append(errs, "Category was not found.")
	}

	subCategoryCode := normalize("sub_category_code")
	subCategoryID := ""
	if subCategoryCode != "" {
		subCategoryID, err = lookupSubCategoryIDByCodeTx(ctx, tx, businessID, subCategoryCode, categoryID)
		if err != nil {
			return CreateProductInput{}, errs, err
		}
		if subCategoryID == "" {
			errs = append(errs, "Sub-category was not found.")
		}
	}

	brandValue := normalize("brand")
	brandID := ""
	if brandValue != "" {
		brandID, err = lookupBrandIDByNameTx(ctx, tx, businessID, brandValue)
		if err != nil {
			return CreateProductInput{}, errs, err
		}
		if brandID == "" {
			errs = append(errs, "Brand was not found.")
		}
	}

	locationID, err := lookupBusinessLocationIDByCodeTx(ctx, tx, businessID, locationCode)
	if err != nil {
		return CreateProductInput{}, errs, err
	}
	if locationID == "" {
		errs = append(errs, "Location was not found.")
	}

	alertQuantity := 2
	if raw := normalize("alert_quantity"); raw != "" {
		if parsed, err := parseIntString(raw); err == nil {
			alertQuantity = parsed
		}
	}
	isForSelling := true
	if raw := normalize("is_for_selling"); raw != "" {
		isForSelling = !strings.EqualFold(raw, "false")
	}
	manageStock := true
	if raw := normalize("manage_stock"); raw != "" {
		manageStock = !strings.EqualFold(raw, "false")
	}

	defaultPurchasePrice := parseFloatString(normalize("default_purchase_price"))
	defaultSellingPrice := parseFloatString(normalize("default_selling_price"))
	taxRate := parseFloatString(normalize("tax_rate"))
	taxType := strings.ToLower(normalize("tax_type"))
	if taxType == "" {
		taxType = "exclusive"
	}

	req := CreateProductInput{
		BusinessID:              businessID,
		Name:                    name,
		SKU:                     normalize("sku"),
		Barcode:                 normalize("barcode"),
		ProductType:             productType,
		UnitID:                  unitID,
		BrandID:                 brandID,
		CategoryID:              categoryID,
		SubCategoryID:           subCategoryID,
		LocationIDs:             []string{locationID},
		AllLocations:            false,
		ManageStock:             manageStock,
		AlertQuantity:           &alertQuantity,
		IsForSelling:            isForSelling,
		TaxType:                 taxType,
		TaxRate:                 taxRate,
		DefaultPurchasePrice:    &defaultPurchasePrice,
		DefaultSellingPrice:     &defaultSellingPrice,
		Description:             normalize("description"),
		CurrencyCode:            "USD",
		CurrencySymbolPlacement: "before",
		CurrencyPrecision:       2,
		CreatedBy:               createdBy,
	}

	if req.SKU == "" {
		req.SKU = ""
	}

	return req, errs, nil
}

func lookupBusinessUnitIDByNameOrShortNameTx(ctx context.Context, tx interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, businessID, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}

	var id string
	err := tx.QueryRow(ctx, `
		SELECT id::text
		FROM business_units
		WHERE business_id = $1::uuid
		  AND deleted_at IS NULL
		  AND (LOWER(name) = LOWER($2) OR LOWER(short_name) = LOWER($2))
		LIMIT 1
	`, businessID, value).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("lookup business unit: %w", err)
	}
	return id, nil
}

func lookupCategoryIDByCodeTx(ctx context.Context, tx interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, businessID, categoryCode string) (string, error) {
	categoryCode = strings.TrimSpace(categoryCode)
	if categoryCode == "" {
		return "", nil
	}

	var id string
	err := tx.QueryRow(ctx, `
		SELECT id::text
		FROM product_categories
		WHERE business_id = $1::uuid
		  AND deleted_at IS NULL
		  AND LOWER(category_code) = LOWER($2)
		LIMIT 1
	`, businessID, categoryCode).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("lookup category: %w", err)
	}
	return id, nil
}

func lookupSubCategoryIDByCodeTx(ctx context.Context, tx interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, businessID, subCategoryCode, categoryID string) (string, error) {
	subCategoryCode = strings.TrimSpace(subCategoryCode)
	if subCategoryCode == "" {
		return "", nil
	}

	var id string
	args := []any{businessID, subCategoryCode}
	query := `
		SELECT id::text
		FROM product_sub_categories
		WHERE business_id = $1::uuid
		  AND deleted_at IS NULL
		  AND LOWER(sub_category_code) = LOWER($2)
	`
	if strings.TrimSpace(categoryID) != "" {
		args = append(args, categoryID)
		query += fmt.Sprintf(" AND parent_category_id = $%d::uuid", len(args))
	}
	query += " LIMIT 1"
	err := tx.QueryRow(ctx, query, args...).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("lookup sub category: %w", err)
	}
	return id, nil
}

func lookupBrandIDByNameTx(ctx context.Context, tx interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, businessID, brandName string) (string, error) {
	brandName = strings.TrimSpace(brandName)
	if brandName == "" {
		return "", nil
	}

	var id string
	err := tx.QueryRow(ctx, `
		SELECT id::text
		FROM product_brands
		WHERE business_id = $1::uuid
		  AND deleted_at IS NULL
		  AND LOWER(name) = LOWER($2)
		LIMIT 1
	`, businessID, brandName).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("lookup brand: %w", err)
	}
	return id, nil
}

func lookupBusinessLocationIDByCodeTx(ctx context.Context, tx interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, businessID, locationCode string) (string, error) {
	locationCode = strings.TrimSpace(locationCode)
	if locationCode == "" {
		return "", nil
	}

	var id string
	err := tx.QueryRow(ctx, `
		SELECT id::text
		FROM business_locations
		WHERE business_id = $1::uuid
		  AND (
			  LOWER(COALESCE(location_code, location_id)) = LOWER($2)
			  OR LOWER(location_id) = LOWER($2)
		  )
		LIMIT 1
	`, businessID, locationCode).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("lookup business location: %w", err)
	}
	return id, nil
}

func parseFloatString(value string) float64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	parsed, _ := strconv.ParseFloat(value, 64)
	return parsed
}

func parseIntString(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}
