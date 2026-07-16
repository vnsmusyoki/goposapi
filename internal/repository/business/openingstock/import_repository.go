package openingstock

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

var ErrOpeningStockImportNotFound = errors.New("opening stock import not found")

type OpeningStockImportBatch struct {
	ID         string `json:"id"`
	BusinessID string `json:"businessId"`
	FileName   string `json:"fileName"`
	Status     string `json:"status"`
	CreatedBy  string `json:"createdBy"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
}

type OpeningStockImportBatchRow struct {
	ID                       string            `json:"id"`
	BatchID                  string            `json:"batchId"`
	RowNumber                int               `json:"rowNumber"`
	SKU                      string            `json:"sku"`
	ProductID                string            `json:"productId"`
	LocationID               string            `json:"locationId"`
	Quantity                 string            `json:"quantity"`
	UnitCostBeforeTax        string            `json:"unitCostBeforeTax"`
	LotNumber                string            `json:"lotNumber"`
	ExpiryDate               string            `json:"expiryDate"`
	RowData                  map[string]string `json:"rowData"`
	ValidationErrors         []string          `json:"validationErrors"`
	Status                   string            `json:"status"`
	ImportedInventoryBatchID string            `json:"importedInventoryBatchId"`
	CreatedAt                string            `json:"createdAt"`
	UpdatedAt                string            `json:"updatedAt"`
}

type OpeningStockImportInput struct {
	BusinessID        string
	CreatedBy         string
	SourceType        string
	SourceID          string
	ProductID         string
	ProductSKU        string
	ProductName       string
	LocationID        string
	LocationName      string
	LocationCode      string
	Quantity          float64
	UnitCostBeforeTax float64
	LotNumber         string
	ExpiryDate        *time.Time
}

type openingStockImportTx interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func CreateOpeningStockImportPreviewRepository(pool *pgxpool.Pool, businessID, fileName, createdBy string, rows []OpeningStockImportBatchRow) (*OpeningStockImportBatch, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	fileName = strings.TrimSpace(fileName)
	createdBy = strings.TrimSpace(createdBy)
	if businessID == "" {
		return nil, ErrOpeningStockImportNotFound
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin opening stock import tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	batch := OpeningStockImportBatch{}
	if err := tx.QueryRow(ctx, `
		INSERT INTO opening_stock_import_batches (
			business_id,
			file_name,
			status,
			created_by
		)
		VALUES ($1::uuid, $2, 'previewed', NULLIF($3, '')::uuid)
		RETURNING id::text, business_id::text, file_name, status, COALESCE(created_by::text, ''), created_at::text, updated_at::text
	`, businessID, fileName, createdBy).Scan(&batch.ID, &batch.BusinessID, &batch.FileName, &batch.Status, &batch.CreatedBy, &batch.CreatedAt, &batch.UpdatedAt); err != nil {
		return nil, fmt.Errorf("create opening stock import batch: %w", err)
	}

	for _, row := range rows {
		rowDataJSON, err := json.Marshal(row.RowData)
		if err != nil {
			return nil, fmt.Errorf("marshal opening stock import row data: %w", err)
		}
		input, validationErrors, err := BuildOpeningStockImportInput(tx, row.RowData, businessID, createdBy)
		if err != nil {
			return nil, err
		}

		mergedErrors := uniqueStrings(append(append([]string{}, row.ValidationErrors...), validationErrors...))
		status := row.Status
		if len(mergedErrors) > 0 {
			status = "invalid"
		} else {
			status = "valid"
		}

		errorsJSON, err := json.Marshal(mergedErrors)
		if err != nil {
			return nil, fmt.Errorf("marshal opening stock import row errors: %w", err)
		}

		if _, err := tx.Exec(ctx, `
		INSERT INTO opening_stock_import_batch_rows (
			batch_id,
			row_number,
			sku,
			product_id,
				location_id,
				quantity,
				unit_cost_before_tax,
				lot_number,
				expiry_date,
				row_data,
				validation_errors,
				status,
				created_by
			)
			VALUES (
				$1::uuid,
				$2,
				$3,
				NULLIF($4, '')::uuid,
				NULLIF($5, '')::uuid,
				$6,
				$7,
				$8,
				$9,
				$10::jsonb,
				$11::jsonb,
				$12,
				NULLIF($13, '')::uuid
			)
		`, batch.ID, row.RowNumber,
			input.ProductSKU,
			input.ProductID,
			input.LocationID,
			row.Value("quantity"),
			row.Value("unit_cost_before_tax"),
			row.Value("lot_number"),
			row.Value("expiry_date"),
			rowDataJSON, errorsJSON, status, createdBy); err != nil {
			return nil, fmt.Errorf("insert opening stock import row: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit opening stock import batch: %w", err)
	}

	return &batch, nil
}

func ListOpeningStockImportBatchRowsRepository(pool *pgxpool.Pool, businessID, batchID string) (*OpeningStockImportBatch, []OpeningStockImportBatchRow, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	batchID = strings.TrimSpace(batchID)
	if businessID == "" || batchID == "" {
		return nil, nil, ErrOpeningStockImportNotFound
	}

	var batch OpeningStockImportBatch
	if err := pool.QueryRow(ctx, `
		SELECT id::text, business_id::text, file_name, status, COALESCE(created_by::text, ''), created_at::text, updated_at::text
		FROM opening_stock_import_batches
		WHERE business_id = $1::uuid
		  AND id = $2::uuid
	`, businessID, batchID).Scan(&batch.ID, &batch.BusinessID, &batch.FileName, &batch.Status, &batch.CreatedBy, &batch.CreatedAt, &batch.UpdatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil, ErrOpeningStockImportNotFound
		}
		return nil, nil, fmt.Errorf("load opening stock import batch: %w", err)
	}

	rows, err := pool.Query(ctx, `
		SELECT
			id::text,
			batch_id::text,
			row_number,
			COALESCE(sku, ''),
			COALESCE(product_id::text, ''),
			COALESCE(location_id::text, ''),
			COALESCE(quantity, ''),
			COALESCE(unit_cost_before_tax, ''),
			COALESCE(lot_number, ''),
			COALESCE(expiry_date, ''),
			row_data::text,
			COALESCE(validation_errors::text, '[]'),
			status,
			COALESCE(imported_inventory_batch_id::text, ''),
			created_at::text,
			updated_at::text
		FROM opening_stock_import_batch_rows
		WHERE batch_id = $1::uuid
		ORDER BY row_number ASC, created_at ASC
	`, batch.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("list opening stock import rows: %w", err)
	}
	defer rows.Close()

	items := make([]OpeningStockImportBatchRow, 0)
	for rows.Next() {
		var item OpeningStockImportBatchRow
		var rowDataJSON string
		var errorsJSON string
		if err := rows.Scan(
			&item.ID,
			&item.BatchID,
			&item.RowNumber,
			&item.SKU,
			&item.ProductID,
			&item.LocationID,
			&item.Quantity,
			&item.UnitCostBeforeTax,
			&item.LotNumber,
			&item.ExpiryDate,
			&rowDataJSON,
			&errorsJSON,
			&item.Status,
			&item.ImportedInventoryBatchID,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, nil, fmt.Errorf("scan opening stock import row: %w", err)
		}
		_ = json.Unmarshal([]byte(rowDataJSON), &item.RowData)
		_ = json.Unmarshal([]byte(errorsJSON), &item.ValidationErrors)
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate opening stock import rows: %w", err)
	}

	return &batch, items, nil
}

func GetLatestOpeningStockImportBatchRepository(pool *pgxpool.Pool, businessID string) (*OpeningStockImportBatch, []OpeningStockImportBatchRow, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	if businessID == "" {
		return nil, nil, ErrOpeningStockImportNotFound
	}

	var batchID string
	if err := pool.QueryRow(ctx, `
		SELECT id::text
		FROM opening_stock_import_batches
		WHERE business_id = $1::uuid
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	`, businessID).Scan(&batchID); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil, ErrOpeningStockImportNotFound
		}
		return nil, nil, fmt.Errorf("load latest opening stock import batch: %w", err)
	}

	return ListOpeningStockImportBatchRowsRepository(pool, businessID, batchID)
}

func GetOpeningStockImportBatchRowRepository(pool *pgxpool.Pool, businessID, batchID, rowID string) (*OpeningStockImportBatchRow, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	batchID = strings.TrimSpace(batchID)
	rowID = strings.TrimSpace(rowID)
	if businessID == "" || batchID == "" || rowID == "" {
		return nil, ErrOpeningStockImportNotFound
	}

	row := pool.QueryRow(ctx, `
		SELECT
			r.id::text,
			r.batch_id::text,
			r.row_number,
			COALESCE(r.sku, ''),
			COALESCE(r.product_id::text, ''),
			COALESCE(r.location_id::text, ''),
			COALESCE(r.quantity, ''),
			COALESCE(r.unit_cost_before_tax, ''),
			COALESCE(r.lot_number, ''),
			COALESCE(r.expiry_date, ''),
			r.row_data::text,
			COALESCE(r.validation_errors::text, '[]'),
			r.status,
			COALESCE(r.imported_inventory_batch_id::text, ''),
			r.created_at::text,
			r.updated_at::text
		FROM opening_stock_import_batch_rows r
		INNER JOIN opening_stock_import_batches b ON b.id = r.batch_id
		WHERE b.business_id = $1::uuid
		  AND b.id = $2::uuid
		  AND r.id = $3::uuid
	`, businessID, batchID, rowID)

	var item OpeningStockImportBatchRow
	var rowDataJSON string
	var errorsJSON string
	if err := row.Scan(
		&item.ID,
		&item.BatchID,
		&item.RowNumber,
		&item.SKU,
		&item.ProductID,
		&item.LocationID,
		&item.Quantity,
		&item.UnitCostBeforeTax,
		&item.LotNumber,
		&item.ExpiryDate,
		&rowDataJSON,
		&errorsJSON,
		&item.Status,
		&item.ImportedInventoryBatchID,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrOpeningStockImportNotFound
		}
		return nil, fmt.Errorf("load opening stock import row: %w", err)
	}
	_ = json.Unmarshal([]byte(rowDataJSON), &item.RowData)
	_ = json.Unmarshal([]byte(errorsJSON), &item.ValidationErrors)
	return &item, nil
}

func UpdateOpeningStockImportBatchRowStatusRepository(pool *pgxpool.Pool, businessID, batchID, rowID, status, importedInventoryBatchID string, validationErrors []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	batchID = strings.TrimSpace(batchID)
	rowID = strings.TrimSpace(rowID)
	status = strings.TrimSpace(status)
	importedInventoryBatchID = strings.TrimSpace(importedInventoryBatchID)
	if businessID == "" || batchID == "" || rowID == "" {
		return ErrOpeningStockImportNotFound
	}

	errorsJSON, err := json.Marshal(validationErrors)
	if err != nil {
		return fmt.Errorf("marshal opening stock validation errors: %w", err)
	}

	_, err = pool.Exec(ctx, `
		UPDATE opening_stock_import_batch_rows r
		SET status = $4,
		    imported_inventory_batch_id = NULLIF($5, '')::uuid,
		    validation_errors = $6::jsonb,
		    updated_at = CURRENT_TIMESTAMP
		FROM opening_stock_import_batches b
		WHERE b.id = r.batch_id
		  AND b.business_id = $1::uuid
		  AND b.id = $2::uuid
		  AND r.id = $3::uuid
	`, businessID, batchID, rowID, status, importedInventoryBatchID, errorsJSON)
	if err != nil {
		return fmt.Errorf("update opening stock row status: %w", err)
	}

	return nil
}

func UpdateOpeningStockImportBatchRowDataRepository(pool *pgxpool.Pool, businessID, batchID, rowID string, rowData map[string]string, createdBy string) (*OpeningStockImportBatchRow, []string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	batchID = strings.TrimSpace(batchID)
	rowID = strings.TrimSpace(rowID)
	createdBy = strings.TrimSpace(createdBy)
	if businessID == "" || batchID == "" || rowID == "" {
		return nil, nil, ErrOpeningStockImportNotFound
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("begin opening stock row update tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	row, err := getOpeningStockImportBatchRowTx(ctx, tx, businessID, batchID, rowID)
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

	input, validationErrors, err := BuildOpeningStockImportInput(tx, mergedData, businessID, createdBy)
	if err != nil {
		return nil, nil, err
	}
	status := "valid"
	if len(validationErrors) > 0 {
		status = "invalid"
	}

	rowDataJSON, err := json.Marshal(mergedData)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal updated opening stock row data: %w", err)
	}
	errorsJSON, err := json.Marshal(validationErrors)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal updated opening stock row errors: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE opening_stock_import_batch_rows r
		SET row_data = $4::jsonb,
		    validation_errors = $5::jsonb,
		    sku = $6,
		    product_id = NULLIF($7, '')::uuid,
		    location_id = NULLIF($8, '')::uuid,
		    quantity = $9,
		    unit_cost_before_tax = $10,
		    lot_number = $11,
		    expiry_date = $12,
		    created_by = NULLIF($13, '')::uuid,
		    status = $14,
		    imported_inventory_batch_id = NULL,
		    updated_at = CURRENT_TIMESTAMP
		FROM opening_stock_import_batches b
		WHERE b.id = r.batch_id
		  AND b.business_id = $1::uuid
		  AND b.id = $2::uuid
		  AND r.id = $3::uuid
	`, businessID, batchID, rowID, rowDataJSON, errorsJSON,
		input.ProductSKU,
		input.ProductID,
		input.LocationID,
		row.Value("quantity"),
		row.Value("unit_cost_before_tax"),
		row.Value("lot_number"),
		row.Value("expiry_date"),
		createdBy,
		status); err != nil {
		return nil, nil, fmt.Errorf("update opening stock row data: %w", err)
	}

	updatedRow, err := getOpeningStockImportBatchRowTx(ctx, tx, businessID, batchID, rowID)
	if err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("commit opening stock row update tx: %w", err)
	}

	return updatedRow, validationErrors, nil
}

func BuildOpeningStockImportInput(q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, rowData map[string]string, businessID, createdBy string) (OpeningStockImportInput, []string, error) {
	ctx := context.Background()
	return buildOpeningStockImportInputTx(ctx, q, rowData, businessID, createdBy)
}

func buildOpeningStockImportInputTx(ctx context.Context, tx interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, rowData map[string]string, businessID, createdBy string) (OpeningStockImportInput, []string, error) {
	errs := make([]string, 0)
	normalize := func(key string) string {
		return strings.TrimSpace(rowData[key])
	}

	sku := normalize("sku")
	if sku == "" {
		errs = append(errs, "SKU is required.")
	}

	productID, productName, err := lookupProductBySKUTx(ctx, tx, businessID, sku)
	if err != nil {
		return OpeningStockImportInput{}, errs, err
	}
	if productID == "" {
		errs = append(errs, "Product SKU was not found.")
	}

	locationValue := normalize("location")
	locationID := ""
	locationName := ""
	locationCode := ""
	if locationValue == "" {
		locationID, locationName, locationCode, err = lookupFirstBusinessLocationTx(ctx, tx, businessID)
		if err != nil {
			return OpeningStockImportInput{}, errs, err
		}
		if locationID == "" {
			errs = append(errs, "No business location was found.")
		}
	} else {
		locationID, locationName, locationCode, err = lookupBusinessLocationByValueTx(ctx, tx, businessID, locationValue)
		if err != nil {
			return OpeningStockImportInput{}, errs, err
		}
		if locationID == "" {
			errs = append(errs, "Location was not found.")
		}
	}

	quantityText := normalize("quantity")
	if quantityText == "" {
		errs = append(errs, "Quantity is required.")
	}
	quantity, err := strconv.ParseFloat(quantityText, 64)
	if err != nil {
		if quantityText != "" {
			errs = append(errs, "Quantity must be a valid number.")
		}
	}
	if quantity <= 0 {
		errs = append(errs, "Quantity must be greater than zero.")
	}

	unitCostText := normalize("unit_cost_before_tax")
	if unitCostText == "" {
		errs = append(errs, "Unit cost before tax is required.")
	}
	unitCost, err := strconv.ParseFloat(unitCostText, 64)
	if err != nil {
		if unitCostText != "" {
			errs = append(errs, "Unit cost before tax must be a valid number.")
		}
	}
	if unitCost < 0 {
		errs = append(errs, "Unit cost before tax cannot be negative.")
	}

	lotNumber := normalize("lot_number")
	expiryText := normalize("expiry_date")
	var expiry *time.Time
	if expiryText != "" {
		parsedExpiry, err := time.Parse("01/02/2006", expiryText)
		if err != nil {
			errs = append(errs, "Expiry date must use mm/dd/yyyy format.")
		} else {
			expiry = &parsedExpiry
		}
	}

	return OpeningStockImportInput{
		BusinessID:        businessID,
		CreatedBy:         createdBy,
		SourceType:        "opening_stock_import",
		ProductID:         productID,
		ProductSKU:        sku,
		ProductName:       productName,
		LocationID:        locationID,
		LocationName:      locationName,
		LocationCode:      locationCode,
		Quantity:          quantity,
		UnitCostBeforeTax: unitCost,
		LotNumber:         lotNumber,
		ExpiryDate:        expiry,
	}, uniqueStrings(errs), nil
}

func ImportOpeningStockRepository(pool *pgxpool.Pool, input OpeningStockImportInput) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	input.BusinessID = strings.TrimSpace(input.BusinessID)
	input.CreatedBy = strings.TrimSpace(input.CreatedBy)
	input.ProductID = strings.TrimSpace(input.ProductID)
	input.SourceID = strings.TrimSpace(input.SourceID)
	input.SourceType = strings.TrimSpace(input.SourceType)
	input.LocationID = strings.TrimSpace(input.LocationID)
	input.ProductSKU = strings.TrimSpace(input.ProductSKU)
	input.ProductName = strings.TrimSpace(input.ProductName)
	input.LocationName = strings.TrimSpace(input.LocationName)
	input.LocationCode = strings.TrimSpace(input.LocationCode)
	input.LotNumber = strings.TrimSpace(input.LotNumber)

	if input.BusinessID == "" || input.ProductID == "" || input.LocationID == "" || input.Quantity <= 0 {
		return "", fmt.Errorf("invalid opening stock import input")
	}
	if input.SourceType == "" {
		input.SourceType = "opening_stock_import"
	}
	if input.SourceID == "" {
		return "", fmt.Errorf("opening stock source id is required")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin opening stock inventory tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	balance, err := getOrCreateInventoryBalanceTx(ctx, tx, input.BusinessID, input.ProductID, input.LocationID, input.Quantity)
	if err != nil {
		return "", err
	}

	batchID, err := insertOpeningStockInventoryBatchTx(ctx, tx, input)
	if err != nil {
		return "", err
	}

	note := fmt.Sprintf("Opening stock imported for SKU %s.", input.ProductSKU)
	if input.LocationName != "" {
		note = fmt.Sprintf("%s Location: %s.", note, input.LocationName)
	}

	if err := insertOpeningStockStockMovementTx(ctx, tx, openingStockMovementInput{
		BusinessID:         input.BusinessID,
		SourceType:         input.SourceType,
		SourceID:           input.SourceID,
		ReferenceNumber:    input.ProductSKU,
		LocationID:         input.LocationID,
		ProductID:          input.ProductID,
		InventoryBalanceID: balance.ID,
		InventoryBatchID:   batchID,
		MovementType:       "opening_stock",
		QuantityIn:         input.Quantity,
		QuantityOut:        0,
		UnitCost:           input.UnitCostBeforeTax,
		StockBefore:        balance.QuantityAvailable,
		StockAfter:         balance.QuantityAvailable + input.Quantity,
		Note:               note,
		PerformedBy:        input.CreatedBy,
	}); err != nil {
		return "", err
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit opening stock inventory tx: %w", err)
	}

	return batchID, nil
}

type openingStockBalanceSnapshot struct {
	ID                string
	QuantityAvailable float64
}

func getOrCreateInventoryBalanceTx(ctx context.Context, tx openingStockImportTx, businessID, productID, locationID string, delta float64) (openingStockBalanceSnapshot, error) {
	var balance openingStockBalanceSnapshot
	err := tx.QueryRow(ctx, `
		SELECT id::text, COALESCE(quantity_available, 0)
		FROM inventory_balances
		WHERE business_id = $1::uuid
		  AND product_id = $2::uuid
		  AND location_id = $3::uuid
		FOR UPDATE
	`, businessID, productID, locationID).Scan(&balance.ID, &balance.QuantityAvailable)
	if err == nil {
		nextQuantity := balance.QuantityAvailable + delta
		if nextQuantity < 0 {
			return openingStockBalanceSnapshot{}, fmt.Errorf("inventory balance cannot go below zero for product %s", productID)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE inventory_balances
			SET quantity_available = $4,
			    last_movement_at = CURRENT_TIMESTAMP,
			    updated_at = CURRENT_TIMESTAMP
			WHERE id = $1::uuid
			  AND business_id = $2::uuid
			  AND product_id = $3::uuid
		`, balance.ID, businessID, productID, nextQuantity); err != nil {
			return openingStockBalanceSnapshot{}, fmt.Errorf("update inventory balance: %w", err)
		}
		return openingStockBalanceSnapshot{ID: balance.ID, QuantityAvailable: balance.QuantityAvailable}, nil
	}
	if err != pgx.ErrNoRows {
		return openingStockBalanceSnapshot{}, fmt.Errorf("load inventory balance: %w", err)
	}
	if delta < 0 {
		return openingStockBalanceSnapshot{}, fmt.Errorf("inventory balance cannot go below zero for product %s", productID)
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO inventory_balances (
			business_id,
			product_id,
			location_id,
			quantity_available,
			quantity_reserved,
			last_movement_at,
			created_at,
			updated_at
		)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, 0, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id::text
	`, businessID, productID, locationID, delta).Scan(&balance.ID); err != nil {
		return openingStockBalanceSnapshot{}, fmt.Errorf("insert inventory balance: %w", err)
	}
	return openingStockBalanceSnapshot{ID: balance.ID, QuantityAvailable: 0}, nil
}

type openingStockMovementInput struct {
	BusinessID         string
	SourceType         string
	SourceID           string
	ReferenceNumber    string
	LocationID         string
	ProductID          string
	InventoryBalanceID string
	InventoryBatchID   string
	MovementType       string
	QuantityIn         float64
	QuantityOut        float64
	UnitCost           float64
	StockBefore        float64
	StockAfter         float64
	Note               string
	PerformedBy        string
}

func insertOpeningStockInventoryBatchTx(ctx context.Context, tx openingStockImportTx, req OpeningStockImportInput) (string, error) {
	var batchID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO inventory_batches (
			business_id,
			product_id,
			location_id,
			source_type,
			source_id,
			lot_number,
			batch_number,
			expiry_date,
			unit_cost,
			quantity_received,
			quantity_remaining,
			received_at,
			created_by,
			created_at,
			updated_at
		)
		VALUES (
			$1::uuid,
			$2::uuid,
			$3::uuid,
			$4,
			$5::uuid,
			$6,
			$7,
			$8,
			$9,
			$10,
			$10,
			CURRENT_TIMESTAMP,
			NULLIF($11, '')::uuid,
			CURRENT_TIMESTAMP,
			CURRENT_TIMESTAMP
		)
		RETURNING id::text
	`, req.BusinessID, req.ProductID, req.LocationID, req.SourceType, req.SourceID, req.LotNumber, req.ProductSKU, req.ExpiryDate, req.UnitCostBeforeTax, req.Quantity, req.CreatedBy).Scan(&batchID); err != nil {
		return "", fmt.Errorf("insert opening stock inventory batch: %w", err)
	}
	return batchID, nil
}

func insertOpeningStockStockMovementTx(ctx context.Context, tx openingStockImportTx, req openingStockMovementInput) error {
	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.SourceType = strings.TrimSpace(req.SourceType)
	req.SourceID = strings.TrimSpace(req.SourceID)
	req.ReferenceNumber = strings.TrimSpace(req.ReferenceNumber)
	req.LocationID = strings.TrimSpace(req.LocationID)
	req.ProductID = strings.TrimSpace(req.ProductID)
	req.InventoryBalanceID = strings.TrimSpace(req.InventoryBalanceID)
	req.InventoryBatchID = strings.TrimSpace(req.InventoryBatchID)
	req.MovementType = strings.TrimSpace(req.MovementType)
	req.Note = strings.TrimSpace(req.Note)
	req.PerformedBy = strings.TrimSpace(req.PerformedBy)

	if req.BusinessID == "" || req.LocationID == "" || req.ProductID == "" || req.MovementType == "" {
		return nil
	}

	_, err := tx.Exec(ctx, `
		INSERT INTO stock_movements (
			business_id,
			product_id,
			location_id,
			inventory_balance_id,
			inventory_batch_id,
			movement_type,
			source_type,
			source_id,
			reference_number,
			quantity_in,
			quantity_out,
			unit_cost,
			stock_before,
			stock_after,
			note,
			performed_by,
			occurred_at,
			created_at
		)
		VALUES (
			$1::uuid,
			$2::uuid,
			$3::uuid,
			NULLIF($4, '')::uuid,
			NULLIF($5, '')::uuid,
			$6,
			$7,
			$8::uuid,
			$9,
			$10,
			$11,
			$12,
			$13,
			$14,
			$15,
			NULLIF($16, '')::uuid,
			CURRENT_TIMESTAMP,
			CURRENT_TIMESTAMP
		)
	`, req.BusinessID, req.ProductID, req.LocationID, req.InventoryBalanceID, req.InventoryBatchID, req.MovementType, req.SourceType, req.SourceID, req.ReferenceNumber, req.QuantityIn, req.QuantityOut, req.UnitCost, req.StockBefore, req.StockAfter, req.Note, req.PerformedBy)
	if err != nil {
		return fmt.Errorf("insert opening stock movement: %w", err)
	}

	return nil
}

func getOpeningStockImportBatchRowTx(ctx context.Context, tx interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, businessID, batchID, rowID string) (*OpeningStockImportBatchRow, error) {
	row := tx.QueryRow(ctx, `
		SELECT
			r.id::text,
			r.batch_id::text,
			r.row_number,
			COALESCE(r.sku, ''),
			COALESCE(r.product_id::text, ''),
			COALESCE(r.location_id::text, ''),
			COALESCE(r.quantity, ''),
			COALESCE(r.unit_cost_before_tax, ''),
			COALESCE(r.lot_number, ''),
			COALESCE(r.expiry_date, ''),
			r.row_data::text,
			COALESCE(r.validation_errors::text, '[]'),
			r.status,
			COALESCE(r.imported_inventory_batch_id::text, ''),
			r.created_at::text,
			r.updated_at::text
		FROM opening_stock_import_batch_rows r
		INNER JOIN opening_stock_import_batches b ON b.id = r.batch_id
		WHERE b.business_id = $1::uuid
		  AND b.id = $2::uuid
		  AND r.id = $3::uuid
	`, businessID, batchID, rowID)

	var item OpeningStockImportBatchRow
	var rowDataJSON string
	var errorsJSON string
	if err := row.Scan(
		&item.ID,
		&item.BatchID,
		&item.RowNumber,
		&item.SKU,
		&item.ProductID,
		&item.LocationID,
		&item.Quantity,
		&item.UnitCostBeforeTax,
		&item.LotNumber,
		&item.ExpiryDate,
		&rowDataJSON,
		&errorsJSON,
		&item.Status,
		&item.ImportedInventoryBatchID,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrOpeningStockImportNotFound
		}
		return nil, fmt.Errorf("load opening stock import row: %w", err)
	}
	_ = json.Unmarshal([]byte(rowDataJSON), &item.RowData)
	_ = json.Unmarshal([]byte(errorsJSON), &item.ValidationErrors)
	return &item, nil
}

func lookupProductBySKUTx(ctx context.Context, tx interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, businessID, sku string) (string, string, error) {
	sku = strings.TrimSpace(sku)
	if sku == "" {
		return "", "", nil
	}

	var id, name string
	err := tx.QueryRow(ctx, `
		SELECT id::text, COALESCE(name, '')
		FROM products
		WHERE business_id = $1::uuid
		  AND deleted_at IS NULL
		  AND LOWER(COALESCE(sku, '')) = LOWER($2)
		LIMIT 1
	`, businessID, sku).Scan(&id, &name)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", "", nil
		}
		return "", "", fmt.Errorf("lookup product by sku: %w", err)
	}
	return id, name, nil
}

func lookupFirstBusinessLocationTx(ctx context.Context, tx interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, businessID string) (string, string, string, error) {
	var id, name, code string
	err := tx.QueryRow(ctx, `
		SELECT id::text, COALESCE(location_name, ''), COALESCE(location_code, location_id)
		FROM business_locations
		WHERE business_id = $1::uuid
		ORDER BY created_at ASC, location_name ASC
		LIMIT 1
	`, businessID).Scan(&id, &name, &code)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", "", "", nil
		}
		return "", "", "", fmt.Errorf("lookup first business location: %w", err)
	}
	return id, name, code, nil
}

func lookupBusinessLocationByValueTx(ctx context.Context, tx interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, businessID, value string) (string, string, string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", "", "", nil
	}

	var id, name, code string
	err := tx.QueryRow(ctx, `
		SELECT id::text, COALESCE(location_name, ''), COALESCE(location_code, location_id)
		FROM business_locations
		WHERE business_id = $1::uuid
		  AND (
			  LOWER(COALESCE(location_name, '')) = LOWER($2)
			  OR LOWER(COALESCE(location_code, location_id)) = LOWER($2)
			  OR LOWER(location_id) = LOWER($2)
		  )
		LIMIT 1
	`, businessID, value).Scan(&id, &name, &code)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", "", "", nil
		}
		return "", "", "", fmt.Errorf("lookup business location: %w", err)
	}
	return id, name, code, nil
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}

func (r OpeningStockImportBatchRow) Value(key string) string {
	if r.RowData == nil {
		return ""
	}
	return strings.TrimSpace(r.RowData[key])
}
