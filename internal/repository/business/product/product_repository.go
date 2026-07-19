package product

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"pos/internal/models"
)

func CreateProductRepository(pool *pgxpool.Pool, req CreateProductInput) (*models.Product, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.Name = strings.TrimSpace(req.Name)
	req.SKU = strings.TrimSpace(req.SKU)
	req.Barcode = strings.TrimSpace(req.Barcode)
	req.ProductType = strings.TrimSpace(strings.ToLower(req.ProductType))
	req.UnitID = strings.TrimSpace(req.UnitID)
	req.BrandID = strings.TrimSpace(req.BrandID)
	req.CategoryID = strings.TrimSpace(req.CategoryID)
	req.SubCategoryID = strings.TrimSpace(req.SubCategoryID)
	req.Description = strings.TrimSpace(req.Description)
	req.WarrantyDuration = strings.TrimSpace(req.WarrantyDuration)
	req.WarrantyPeriod = strings.TrimSpace(req.WarrantyPeriod)
	req.WarrantyCoverage = strings.TrimSpace(req.WarrantyCoverage)
	req.BrochureName = strings.TrimSpace(req.BrochureName)
	req.BrochureURL = strings.TrimSpace(req.BrochureURL)
	req.CurrencyCode = strings.TrimSpace(req.CurrencyCode)
	req.CurrencySymbolPlacement = strings.TrimSpace(req.CurrencySymbolPlacement)
	req.CreatedBy = strings.TrimSpace(req.CreatedBy)

	if req.BusinessID == "" || req.Name == "" || req.ProductType == "" || req.UnitID == "" || req.CategoryID == "" {
		return nil, ErrInvalidProductInput
	}

	if req.CurrencyCode == "" {
		req.CurrencyCode = "USD"
	}
	if req.CurrencySymbolPlacement != "after" {
		req.CurrencySymbolPlacement = "before"
	}
	if req.CurrencyPrecision < 0 {
		req.CurrencyPrecision = 2
	}

	switch req.ProductType {
	case "single", "combo", "variable":
	default:
		return nil, ErrInvalidProductInput
	}

	if !req.AllLocations && len(req.LocationIDs) == 0 {
		return nil, ErrInvalidProductInput
	}

	if req.ManageStock && req.AlertQuantity != nil && *req.AlertQuantity < 2 {
		return nil, ErrInvalidProductInput
	}

	if req.ProductType == "single" {
		if req.DefaultPurchasePrice == nil || req.DefaultSellingPrice == nil {
			return nil, ErrInvalidProductInput
		}
	}

	if req.ProductType == "combo" && len(req.ComboItems) == 0 {
		return nil, ErrInvalidProductInput
	}

	if req.ProductType == "variable" && len(req.Variants) == 0 {
		return nil, ErrInvalidProductInput
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin product tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	sku := req.SKU
	if sku == "" {
		generatedSKU, err := generateProductSKU(ctx, tx, req.BusinessID)
		if err != nil {
			return nil, err
		}
		sku = generatedSKU
	} else {
		var exists bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM products
				WHERE business_id = $1
				  AND LOWER(sku) = LOWER($2)
				  AND deleted_at IS NULL
			)
		`, req.BusinessID, sku).Scan(&exists); err != nil {
			return nil, fmt.Errorf("check product sku duplicate: %w", err)
		}
		if exists {
			return nil, ErrProductAlreadyExists
		}
	}

	var product models.Product
	var productSKU sql.NullString
	err = tx.QueryRow(ctx, `
		INSERT INTO products (
			business_id,
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
			created_by
		)
		VALUES (
			$1,$2,NULLIF($3, ''),$4,$5,
			NULLIF($6, '')::uuid,
			NULLIF($7, '')::uuid,
			NULLIF($8, '')::uuid,
			NULLIF($9, '')::uuid,
			$10,$11,$12,$13,$14,
			$15,$16,$17,$18,$19,
			NULLIF($20, ''),
			NULLIF($21, ''),
			NULLIF($22, ''),
			$23,$24,$25,$26,$27,$28,$29,$30,NULLIF($31, '')::uuid
		)
		RETURNING id::text, name, sku, product_type
	`,
		req.BusinessID,
		req.Name,
		sku,
		req.Barcode,
		req.ProductType,
		req.UnitID,
		req.BrandID,
		req.CategoryID,
		req.SubCategoryID,
		req.IsForSelling,
		req.ManageStock,
		req.AlertQuantity,
		req.TaxType,
		req.TaxRate,
		req.DefaultPurchasePrice,
		req.PurchasePriceExclusive,
		req.PurchasePriceInclusive,
		req.ProfitMargin,
		req.DefaultSellingPrice,
		req.Description,
		req.BrochureName,
		req.BrochureURL,
		req.CurrencyCode,
		req.CurrencySymbolPlacement,
		req.CurrencyPrecision,
		req.AllLocations,
		req.HasWarranty,
		req.WarrantyDuration,
		req.WarrantyPeriod,
		req.WarrantyCoverage,
		req.CreatedBy,
	).Scan(&product.ID, &product.Name, &productSKU, &product.ProductType)
	if err != nil {
		log.Printf("create product: insert failed business_id=%s sku=%q err=%v", req.BusinessID, sku, err)
		return nil, fmt.Errorf("create product: %w", err)
	}
	product.SKU = models.StringPtrFromNullString(productSKU)

	if len(req.SubUnitIDs) > 0 {
		for _, unitID := range req.SubUnitIDs {
			unitID = strings.TrimSpace(unitID)
			if unitID == "" {
				continue
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO product_sub_units (product_id, unit_id)
				VALUES ($1::uuid, $2::uuid)
			`, product.ID, unitID); err != nil {
				return nil, fmt.Errorf("create product sub units: %w", err)
			}
		}
	}

	for idx, locationID := range req.LocationIDs {
		locationID = strings.TrimSpace(locationID)
		if locationID == "" {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO product_locations (product_id, location_id, is_default)
			VALUES ($1::uuid, $2::uuid, $3)
		`, product.ID, locationID, idx == 0); err != nil {
			return nil, fmt.Errorf("create product locations: %w", err)
		}
	}

	for idx, image := range req.Images {
		if strings.TrimSpace(image.URL) == "" {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO product_images (product_id, image_url, image_name, is_primary, sort_order)
			VALUES ($1::uuid, $2, $3, $4, $5)
		`, product.ID, strings.TrimSpace(image.URL), strings.TrimSpace(image.Name), image.IsPrimary, idx); err != nil {
			return nil, fmt.Errorf("create product images: %w", err)
		}
	}

	if req.ProductType == "combo" {
		for idx, item := range req.ComboItems {
			item.ProductID = strings.TrimSpace(item.ProductID)
			if item.ProductID == "" {
				return nil, ErrInvalidComboProduct
			}

			var itemType string
			if err := tx.QueryRow(ctx, `
				SELECT product_type
				FROM products
				WHERE business_id = $1
				  AND id::text = $2
				  AND deleted_at IS NULL
			`, req.BusinessID, item.ProductID).Scan(&itemType); err != nil {
				if err == pgx.ErrNoRows {
					return nil, ErrInvalidComboProduct
				}
				return nil, fmt.Errorf("validate combo item: %w", err)
			}
			if itemType != "single" {
				return nil, ErrInvalidComboProduct
			}

			if _, err := tx.Exec(ctx, `
				INSERT INTO product_combo_items (
					business_id,
					combo_product_id,
					item_product_id,
					item_name,
					item_sku,
					item_unit,
					quantity,
					price_each,
					subtotal,
					sort_order
				)
				VALUES ($1, $2::uuid, $3::uuid, $4, $5, $6, $7, $8, $9, $10)
			`, req.BusinessID, product.ID, item.ProductID, item.ProductName, item.SKU, item.Unit, item.Quantity, item.PriceEach, item.Subtotal, idx); err != nil {
				return nil, fmt.Errorf("create product combo items: %w", err)
			}
		}
	}

	if req.ProductType == "variable" {
		for _, variant := range req.Variants {
			variant.Name = strings.TrimSpace(variant.Name)
			variant.SKU = strings.TrimSpace(variant.SKU)
			if variant.Name == "" || variant.SKU == "" {
				return nil, ErrInvalidProductInput
			}

			var reorderLevel any
			if variant.ReorderLevel != nil {
				reorderLevel = *variant.ReorderLevel
			}

			if _, err := tx.Exec(ctx, `
				INSERT INTO product_variants (
					business_id,
					product_id,
					name,
					sku,
					barcode,
					cost,
					selling,
					stock,
					show_optional_fields,
					weight,
					length,
					width,
					height,
					image_name,
					image_url,
					reorder_level,
					expiry_date,
					supplier_code
				)
				VALUES (
					$1, $2::uuid, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, NULLIF($17, '')::date, $18
				)
			`, req.BusinessID, product.ID, variant.Name, variant.SKU, variant.Barcode, variant.Cost, variant.Selling, variant.Stock, variant.ShowOptionalFields, nullIfBlank(variant.Weight), nullIfBlank(variant.Length), nullIfBlank(variant.Width), nullIfBlank(variant.Height), nullIfBlank(variant.ImageName), nullIfBlank(variant.ImageURL), reorderLevel, nullIfBlank(variant.ExpiryDate), nullIfBlank(variant.SupplierCode)); err != nil {
				return nil, fmt.Errorf("create product variants: %w", err)
			}
		}
	}

	if err := insertProductPrices(ctx, tx, req.BusinessID, product.ID, productPricesWithRetailFallback(req.ProductPrices, req.DefaultSellingPrice), nil, req.CreatedBy); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit product tx: %w", err)
	}

	return &product, nil
}

func GetProductByIDRepository(pool *pgxpool.Pool, businessID, productID string) (*ProductDetail, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	productID = strings.TrimSpace(productID)
	if businessID == "" || productID == "" {
		return nil, ErrBusinessNotResolved
	}

	var detail ProductDetail
	var sku sql.NullString
	var alertQuantity sql.NullInt64
	var taxRate float64
	var defaultPurchasePrice, purchasePriceExclusive, profitAmount, purchasePriceInclusive, profitMargin, defaultSellingPrice float64
	var description sql.NullString
	var warrantyDuration sql.NullString
	var warrantyPeriod sql.NullString
	var warrantyCoverage sql.NullString
	var brochureName sql.NullString
	var brochureURL sql.NullString
	var currencyCode sql.NullString
	var currencySymbolPlacement sql.NullString
	var currencyPrecision sql.NullInt64
	var deletedAt sql.NullTime
	var imageURL sql.NullString

	row := pool.QueryRow(ctx, `
		SELECT
			p.id::text,
			p.name,
			p.sku,
			COALESCE(p.barcode, ''),
			p.product_type,
			COALESCE(p.unit_id::text, ''),
			COALESCE(u.name, ''),
			COALESCE(p.brand_id::text, ''),
			COALESCE(b.name, ''),
			COALESCE(p.category_id::text, ''),
			COALESCE(c.name, ''),
			COALESCE(p.sub_category_id::text, ''),
			COALESCE(sc.name, ''),
			COALESCE(array_agg(DISTINCT pl.location_id::text) FILTER (WHERE pl.location_id IS NOT NULL), '{}'::text[]),
			COALESCE(array_agg(DISTINCT bl.location_name) FILTER (WHERE bl.location_name IS NOT NULL), '{}'::text[]),
			p.all_locations,
			p.manage_stock,
			p.alert_quantity,
			p.is_for_selling,
			p.tax_type,
			COALESCE(p.tax_rate, 0),
			COALESCE(p.default_purchase_price, 0),
			COALESCE(p.purchase_price_exclusive, 0),
			COALESCE(p.profit_amount, 0),
			COALESCE(p.purchase_price_inclusive, 0),
			COALESCE(p.profit_margin, 0),
			COALESCE(p.default_selling_price, 0),
			p.description,
			p.has_warranty,
			p.warranty_duration,
			p.warranty_period,
			p.warranty_coverage,
			p.brochure_name,
			p.brochure_url,
			p.currency_code,
			p.currency_symbol_placement,
			p.currency_precision,
			COALESCE(pi.image_url, ''),
			p.created_at::text,
			p.updated_at::text,
			CASE WHEN p.deleted_at IS NULL THEN 'active' ELSE 'inactive' END,
			p.deleted_at
		FROM products p
		LEFT JOIN business_units u ON u.id = p.unit_id
		LEFT JOIN product_brands b ON b.id = p.brand_id
		LEFT JOIN product_categories c ON c.id = p.category_id
		LEFT JOIN product_sub_categories sc ON sc.uuid_id = p.sub_category_id
		LEFT JOIN product_locations pl ON pl.product_id = p.id AND pl.deleted_at IS NULL
		LEFT JOIN business_locations bl ON bl.id = pl.location_id
		LEFT JOIN LATERAL (
			SELECT image_url
			FROM product_images
			WHERE product_id = p.id
			  AND deleted_at IS NULL
			ORDER BY is_primary DESC, sort_order ASC, created_at ASC
			LIMIT 1
		) pi ON TRUE
		WHERE p.business_id = $1
		  AND p.id::text = $2
		GROUP BY
			p.id, u.name, b.name, c.name, sc.name, pi.image_url
	`, businessID, productID)
	if err := row.Scan(
		&detail.ID,
		&detail.Name,
		&sku,
		&detail.Barcode,
		&detail.ProductType,
		&detail.UnitID,
		&detail.UnitName,
		&detail.BrandID,
		&detail.BrandName,
		&detail.CategoryID,
		&detail.CategoryName,
		&detail.SubCategoryID,
		&detail.SubCategoryName,
		&detail.LocationIDs,
		&detail.LocationNames,
		&detail.AllLocations,
		&detail.ManageStock,
		&alertQuantity,
		&detail.IsForSelling,
		&detail.TaxType,
		&taxRate,
		&defaultPurchasePrice,
		&purchasePriceExclusive,
		&profitAmount,
		&purchasePriceInclusive,
		&profitMargin,
		&defaultSellingPrice,
		&description,
		&detail.HasWarranty,
		&warrantyDuration,
		&warrantyPeriod,
		&warrantyCoverage,
		&brochureName,
		&brochureURL,
		&currencyCode,
		&currencySymbolPlacement,
		&currencyPrecision,
		&imageURL,
		&detail.CreatedAt,
		&detail.UpdatedAt,
		&detail.Status,
		&deletedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrProductNotFound
		}
		return nil, fmt.Errorf("get product: %w", err)
	}

	detail.SKU = models.StringPtrFromNullString(sku)
	detail.AlertQuantity = int(alertQuantity.Int64)
	detail.TaxRate = taxRate
	detail.DefaultPurchasePrice = defaultPurchasePrice
	detail.PurchasePriceExclusive = purchasePriceExclusive
	detail.ProfitAmount = profitAmount
	detail.PurchasePriceInclusive = purchasePriceInclusive
	detail.ProfitMargin = profitMargin
	detail.DefaultSellingPrice = defaultSellingPrice
	detail.Description = description.String
	detail.WarrantyDuration = warrantyDuration.String
	detail.WarrantyPeriod = warrantyPeriod.String
	detail.WarrantyCoverage = warrantyCoverage.String
	detail.BrochureName = brochureName.String
	detail.BrochureURL = brochureURL.String
	detail.CurrencyCode = currencyCode.String
	detail.CurrencySymbolPlacement = currencySymbolPlacement.String
	detail.CurrencyPrecision = int(currencyPrecision.Int64)
	detail.ImageURL = imageURL.String

	if detail.CurrencyCode == "" {
		detail.CurrencyCode = "USD"
	}
	if detail.CurrencySymbolPlacement == "" {
		detail.CurrencySymbolPlacement = "before"
	}
	if detail.CurrencyPrecision < 0 {
		detail.CurrencyPrecision = 2
	}
	if !alertQuantity.Valid {
		detail.AlertQuantity = 0
	}
	if deletedAt.Valid {
		detail.Status = "inactive"
	}

	rows, err := pool.Query(ctx, `
		SELECT
			psu.unit_id::text,
			COALESCE(bu.name, '')
		FROM product_sub_units psu
		LEFT JOIN business_units bu ON bu.id = psu.unit_id
		WHERE psu.product_id = $1
		  AND psu.deleted_at IS NULL
		ORDER BY psu.created_at ASC
	`, productID)
	if err != nil {
		return nil, fmt.Errorf("list product sub units: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item ProductSubUnitItem
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return nil, fmt.Errorf("scan product sub unit: %w", err)
		}
		detail.SubUnitIDs = append(detail.SubUnitIDs, item.ID)
		detail.SubUnits = append(detail.SubUnits, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate product sub units: %w", err)
	}

	imageRows, err := pool.Query(ctx, `
		SELECT
			id::text,
			COALESCE(image_name, ''),
			image_url,
			is_primary
		FROM product_images
		WHERE product_id = $1
		  AND deleted_at IS NULL
		ORDER BY is_primary DESC, sort_order ASC, created_at ASC
	`, productID)
	if err != nil {
		return nil, fmt.Errorf("list product images: %w", err)
	}
	defer imageRows.Close()
	for imageRows.Next() {
		var item ProductImageItem
		if err := imageRows.Scan(&item.ID, &item.Name, &item.URL, &item.IsPrimary); err != nil {
			return nil, fmt.Errorf("scan product image: %w", err)
		}
		detail.Images = append(detail.Images, item)
	}
	if err := imageRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate product images: %w", err)
	}

	comboRows, err := pool.Query(ctx, `
		SELECT
			id::text,
			item_product_id::text,
			item_name,
			item_sku,
			item_unit,
			quantity,
			price_each,
			subtotal
		FROM product_combo_items
		WHERE combo_product_id = $1
		  AND deleted_at IS NULL
		ORDER BY sort_order ASC, created_at ASC
	`, productID)
	if err != nil {
		return nil, fmt.Errorf("list product combo items: %w", err)
	}
	defer comboRows.Close()
	for comboRows.Next() {
		var item ProductComboItemItem
		if err := comboRows.Scan(&item.ID, &item.ProductID, &item.ProductName, &item.SKU, &item.Unit, &item.Quantity, &item.PriceEach, &item.Subtotal); err != nil {
			return nil, fmt.Errorf("scan product combo item: %w", err)
		}
		detail.ComboItems = append(detail.ComboItems, item)
	}
	if err := comboRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate product combo items: %w", err)
	}

	variantRows, err := pool.Query(ctx, `
		SELECT
			id::text,
			name,
			sku,
			COALESCE(barcode, ''),
			cost,
			selling,
			stock,
			show_optional_fields,
			COALESCE(weight, ''),
			COALESCE(length, ''),
			COALESCE(width, ''),
			COALESCE(height, ''),
			COALESCE(image_name, ''),
			COALESCE(image_url, ''),
			reorder_level,
			COALESCE(expiry_date::text, ''),
			COALESCE(supplier_code, '')
		FROM product_variants
		WHERE product_id = $1
		  AND deleted_at IS NULL
		ORDER BY created_at ASC
	`, productID)
	if err != nil {
		return nil, fmt.Errorf("list product variants: %w", err)
	}
	defer variantRows.Close()
	for variantRows.Next() {
		var item ProductVariantItem
		var reorderLevel sql.NullInt64
		if err := variantRows.Scan(
			&item.ID,
			&item.Name,
			&item.SKU,
			&item.Barcode,
			&item.Cost,
			&item.Selling,
			&item.Stock,
			&item.ShowOptionalFields,
			&item.Weight,
			&item.Length,
			&item.Width,
			&item.Height,
			&item.ImageName,
			&item.ImageURL,
			&reorderLevel,
			&item.ExpiryDate,
			&item.SupplierCode,
		); err != nil {
			return nil, fmt.Errorf("scan product variant: %w", err)
		}
		if reorderLevel.Valid {
			value := int(reorderLevel.Int64)
			item.ReorderLevel = &value
		}
		detail.Variants = append(detail.Variants, item)
	}
	if err := variantRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate product variants: %w", err)
	}

	priceRows, err := pool.Query(ctx, `
		SELECT
			id::text,
			price_type,
			min_quantity,
			price,
			COALESCE(location_id::text, ''),
			COALESCE(customer_group, ''),
			COALESCE(starts_at::text, ''),
			COALESCE(ends_at::text, ''),
			active,
			priority
		FROM product_prices
		WHERE business_id = $1::uuid
		  AND product_id = $2::uuid
		ORDER BY priority ASC, min_quantity ASC, created_at ASC
	`, businessID, productID)
	if err != nil {
		return nil, fmt.Errorf("list product prices: %w", err)
	}
	defer priceRows.Close()
	for priceRows.Next() {
		var item ProductPriceItem
		if err := priceRows.Scan(
			&item.ID,
			&item.PriceType,
			&item.MinQuantity,
			&item.Price,
			&item.LocationID,
			&item.CustomerGroup,
			&item.StartsAt,
			&item.EndsAt,
			&item.Active,
			&item.Priority,
		); err != nil {
			return nil, fmt.Errorf("scan product price: %w", err)
		}
		detail.ProductPrices = append(detail.ProductPrices, item)
	}
	if err := priceRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate product prices: %w", err)
	}

	return &detail, nil
}

func UpdateProductRepository(pool *pgxpool.Pool, productID string, req CreateProductInput, actorID string) (*models.Product, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	productID = strings.TrimSpace(productID)
	actorID = strings.TrimSpace(actorID)
	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.Name = strings.TrimSpace(req.Name)
	req.SKU = strings.TrimSpace(req.SKU)
	req.Barcode = strings.TrimSpace(req.Barcode)
	req.ProductType = strings.TrimSpace(strings.ToLower(req.ProductType))
	req.UnitID = strings.TrimSpace(req.UnitID)
	req.BrandID = strings.TrimSpace(req.BrandID)
	req.CategoryID = strings.TrimSpace(req.CategoryID)
	req.SubCategoryID = strings.TrimSpace(req.SubCategoryID)
	req.Description = strings.TrimSpace(req.Description)
	req.WarrantyDuration = strings.TrimSpace(req.WarrantyDuration)
	req.WarrantyPeriod = strings.TrimSpace(req.WarrantyPeriod)
	req.WarrantyCoverage = strings.TrimSpace(req.WarrantyCoverage)
	req.BrochureName = strings.TrimSpace(req.BrochureName)
	req.BrochureURL = strings.TrimSpace(req.BrochureURL)
	req.CurrencyCode = strings.TrimSpace(req.CurrencyCode)
	req.CurrencySymbolPlacement = strings.TrimSpace(req.CurrencySymbolPlacement)
	req.CreatedBy = strings.TrimSpace(req.CreatedBy)

	if req.BusinessID == "" || productID == "" || req.Name == "" || req.ProductType == "" || req.UnitID == "" || req.CategoryID == "" {
		return nil, ErrInvalidProductInput
	}
	if req.CurrencyCode == "" {
		req.CurrencyCode = "USD"
	}
	if req.CurrencySymbolPlacement != "after" {
		req.CurrencySymbolPlacement = "before"
	}
	if req.CurrencyPrecision < 0 {
		req.CurrencyPrecision = 2
	}

	switch req.ProductType {
	case "single", "combo", "variable":
	default:
		return nil, ErrInvalidProductInput
	}
	if !req.AllLocations && len(req.LocationIDs) == 0 {
		return nil, ErrInvalidProductInput
	}
	if req.ManageStock && req.AlertQuantity != nil && *req.AlertQuantity < 2 {
		return nil, ErrInvalidProductInput
	}
	if req.ProductType == "single" {
		if req.DefaultPurchasePrice == nil || req.DefaultSellingPrice == nil {
			return nil, ErrInvalidProductInput
		}
	}
	if req.ProductType == "combo" && len(req.ComboItems) == 0 {
		return nil, ErrInvalidProductInput
	}
	if req.ProductType == "variable" && len(req.Variants) == 0 {
		return nil, ErrInvalidProductInput
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin product update tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var existingID string
	if err := tx.QueryRow(ctx, `
		SELECT id::text
		FROM products
		WHERE business_id = $1
		  AND id::text = $2
		  AND deleted_at IS NULL
	`, req.BusinessID, productID).Scan(&existingID); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrProductNotFound
		}
		return nil, fmt.Errorf("load product for update: %w", err)
	}

	existingPriceRules, err := loadExistingProductPriceRules(ctx, tx, req.BusinessID, productID)
	if err != nil {
		return nil, err
	}
	nextPriceRules := productPricesWithRetailFallback(req.ProductPrices, req.DefaultSellingPrice)

	if req.SKU != "" {
		var exists bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM products
				WHERE business_id = $1
				  AND LOWER(COALESCE(sku, '')) = LOWER($2)
				  AND id::text <> $3
				  AND deleted_at IS NULL
			)
		`, req.BusinessID, req.SKU, productID).Scan(&exists); err != nil {
			return nil, fmt.Errorf("check product sku duplicate: %w", err)
		}
		if exists {
			return nil, ErrProductAlreadyExists
		}
	}

	_, err = tx.Exec(ctx, `
		UPDATE products
		SET
			name = $3,
			sku = NULLIF($4, ''),
			barcode = $5,
			product_type = $6,
			unit_id = NULLIF($7, '')::uuid,
			brand_id = NULLIF($8, '')::uuid,
			category_id = NULLIF($9, '')::uuid,
			sub_category_id = NULLIF($10, '')::uuid,
			is_for_selling = $11,
			manage_stock = $12,
			alert_quantity = $13,
			tax_type = $14,
			tax_rate = $15,
			default_purchase_price = $16,
			purchase_price_exclusive = $17,
			purchase_price_inclusive = $18,
			profit_margin = $19,
			default_selling_price = $20,
			description = NULLIF($21, ''),
			brochure_name = NULLIF($22, ''),
			brochure_url = NULLIF($23, ''),
			currency_code = $24,
			currency_symbol_placement = $25,
			currency_precision = $26,
			all_locations = $27,
			has_warranty = $28,
			warranty_duration = NULLIF($29, ''),
			warranty_period = NULLIF($30, ''),
			warranty_coverage = NULLIF($31, '')
		WHERE business_id = $1
		  AND id::text = $2
		  AND deleted_at IS NULL
	`, req.BusinessID, productID, req.Name, req.SKU, req.Barcode, req.ProductType, req.UnitID, req.BrandID, req.CategoryID, req.SubCategoryID, req.IsForSelling, req.ManageStock, req.AlertQuantity, req.TaxType, req.TaxRate, req.DefaultPurchasePrice, req.PurchasePriceExclusive, req.PurchasePriceInclusive, req.ProfitMargin, req.DefaultSellingPrice, req.Description, req.BrochureName, req.BrochureURL, req.CurrencyCode, req.CurrencySymbolPlacement, req.CurrencyPrecision, req.AllLocations, req.HasWarranty, req.WarrantyDuration, req.WarrantyPeriod, req.WarrantyCoverage)
	if err != nil {
		return nil, fmt.Errorf("update product: %w", err)
	}

	_, err = tx.Exec(ctx, `
		UPDATE product_sub_units
		SET deleted = TRUE, deleted_at = NOW(), deleted_by = NULLIF($2, '')::uuid
		WHERE product_id = $1
		  AND deleted_at IS NULL
	`, productID, actorID)
	if err != nil {
		return nil, fmt.Errorf("clear product sub units: %w", err)
	}
	_, err = tx.Exec(ctx, `
		UPDATE product_locations
		SET deleted = TRUE, deleted_at = NOW(), deleted_by = NULLIF($2, '')::uuid
		WHERE product_id = $1
		  AND deleted_at IS NULL
	`, productID, actorID)
	if err != nil {
		return nil, fmt.Errorf("clear product locations: %w", err)
	}
	_, err = tx.Exec(ctx, `
		UPDATE product_images
		SET deleted = TRUE, deleted_at = NOW(), deleted_by = NULLIF($2, '')::uuid
		WHERE product_id = $1
		  AND deleted_at IS NULL
	`, productID, actorID)
	if err != nil {
		return nil, fmt.Errorf("clear product images: %w", err)
	}
	_, err = tx.Exec(ctx, `
		UPDATE product_combo_items
		SET deleted = TRUE, deleted_at = NOW(), deleted_by = NULLIF($2, '')::uuid
		WHERE combo_product_id = $1
		  AND deleted_at IS NULL
	`, productID, actorID)
	if err != nil {
		return nil, fmt.Errorf("clear product combo items: %w", err)
	}
	_, err = tx.Exec(ctx, `
		UPDATE product_variants
		SET deleted = TRUE, deleted_at = NOW(), deleted_by = NULLIF($2, '')::uuid
		WHERE product_id = $1
		  AND deleted_at IS NULL
	`, productID, actorID)
	if err != nil {
		return nil, fmt.Errorf("clear product variants: %w", err)
	}
	if err := recordRemovedProductPriceRules(ctx, tx, req.BusinessID, productID, existingPriceRules, nextPriceRules, actorID); err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `
		DELETE FROM product_prices
		WHERE business_id = $1
		  AND product_id = $2::uuid
	`, req.BusinessID, productID)
	if err != nil {
		return nil, fmt.Errorf("clear product prices: %w", err)
	}

	if len(req.SubUnitIDs) > 0 {
		for _, unitID := range req.SubUnitIDs {
			unitID = strings.TrimSpace(unitID)
			if unitID == "" {
				continue
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO product_sub_units (product_id, unit_id)
				VALUES ($1::uuid, $2::uuid)
			`, productID, unitID); err != nil {
				return nil, fmt.Errorf("update product sub units: %w", err)
			}
		}
	}

	for idx, locationID := range req.LocationIDs {
		locationID = strings.TrimSpace(locationID)
		if locationID == "" {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO product_locations (product_id, location_id, is_default)
			VALUES ($1::uuid, $2::uuid, $3)
		`, productID, locationID, idx == 0); err != nil {
			return nil, fmt.Errorf("update product locations: %w", err)
		}
	}

	for idx, image := range req.Images {
		if strings.TrimSpace(image.URL) == "" {
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO product_images (product_id, image_url, image_name, is_primary, sort_order)
			VALUES ($1::uuid, $2, $3, $4, $5)
		`, productID, strings.TrimSpace(image.URL), strings.TrimSpace(image.Name), image.IsPrimary, idx); err != nil {
			return nil, fmt.Errorf("update product images: %w", err)
		}
	}

	if req.ProductType == "combo" {
		for idx, item := range req.ComboItems {
			item.ProductID = strings.TrimSpace(item.ProductID)
			if item.ProductID == "" {
				return nil, ErrInvalidComboProduct
			}

			var itemType string
			if err := tx.QueryRow(ctx, `
				SELECT product_type
				FROM products
				WHERE business_id = $1
				  AND id::text = $2
				  AND deleted_at IS NULL
			`, req.BusinessID, item.ProductID).Scan(&itemType); err != nil {
				if err == pgx.ErrNoRows {
					return nil, ErrInvalidComboProduct
				}
				return nil, fmt.Errorf("validate combo item: %w", err)
			}
			if itemType != "single" {
				return nil, ErrInvalidComboProduct
			}

			if _, err := tx.Exec(ctx, `
				INSERT INTO product_combo_items (
					business_id,
					combo_product_id,
					item_product_id,
					item_name,
					item_sku,
					item_unit,
					quantity,
					price_each,
					subtotal,
					sort_order
				)
				VALUES ($1, $2::uuid, $3::uuid, $4, $5, $6, $7, $8, $9, $10)
			`, req.BusinessID, productID, item.ProductID, item.ProductName, item.SKU, item.Unit, item.Quantity, item.PriceEach, item.Subtotal, idx); err != nil {
				return nil, fmt.Errorf("update product combo items: %w", err)
			}
		}
	}

	if req.ProductType == "variable" {
		for _, variant := range req.Variants {
			variant.Name = strings.TrimSpace(variant.Name)
			variant.SKU = strings.TrimSpace(variant.SKU)
			if variant.Name == "" || variant.SKU == "" {
				return nil, ErrInvalidProductInput
			}

			var reorderLevel any
			if variant.ReorderLevel != nil {
				reorderLevel = *variant.ReorderLevel
			}

			if _, err := tx.Exec(ctx, `
				INSERT INTO product_variants (
					business_id,
					product_id,
					name,
					sku,
					barcode,
					cost,
					selling,
					stock,
					show_optional_fields,
					weight,
					length,
					width,
					height,
					image_name,
					image_url,
					reorder_level,
					expiry_date,
					supplier_code
				)
				VALUES (
					$1, $2::uuid, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, NULLIF($17, '')::date, $18
				)
			`, req.BusinessID, productID, variant.Name, variant.SKU, variant.Barcode, variant.Cost, variant.Selling, variant.Stock, variant.ShowOptionalFields, nullIfBlank(variant.Weight), nullIfBlank(variant.Length), nullIfBlank(variant.Width), nullIfBlank(variant.Height), nullIfBlank(variant.ImageName), nullIfBlank(variant.ImageURL), reorderLevel, nullIfBlank(variant.ExpiryDate), nullIfBlank(variant.SupplierCode)); err != nil {
				return nil, fmt.Errorf("update product variants: %w", err)
			}
		}
	}

	if err := insertProductPrices(ctx, tx, req.BusinessID, productID, nextPriceRules, existingPriceRules, actorID); err != nil {
		return nil, err
	}

	var product models.Product
	var productSKU sql.NullString
	if err := tx.QueryRow(ctx, `
		SELECT id::text, name, sku, product_type
		FROM products
		WHERE business_id = $1
		  AND id::text = $2
		  AND deleted_at IS NULL
	`, req.BusinessID, productID).Scan(&product.ID, &product.Name, &productSKU, &product.ProductType); err != nil {
		return nil, fmt.Errorf("load updated product: %w", err)
	}
	product.SKU = models.StringPtrFromNullString(productSKU)

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit product update tx: %w", err)
	}

	return &product, nil
}

func productPricesWithRetailFallback(prices []CreateProductPriceInput, defaultSellingPrice *float64) []CreateProductPriceInput {
	if defaultSellingPrice == nil {
		return prices
	}

	hasRetail := false
	for _, price := range prices {
		if strings.EqualFold(strings.TrimSpace(price.PriceType), "retail") {
			hasRetail = true
			break
		}
	}
	if hasRetail {
		return prices
	}

	return append([]CreateProductPriceInput{{
		PriceType:   "retail",
		MinQuantity: 1,
		Price:       *defaultSellingPrice,
		Active:      true,
		Priority:    100,
	}}, prices...)
}

type existingProductPriceRule struct {
	ID            string
	PriceType     string
	MinQuantity   float64
	Price         float64
	LocationID    string
	CustomerGroup string
	StartsAt      string
	EndsAt        string
	Active        bool
	Priority      int
}

func loadExistingProductPriceRules(ctx context.Context, tx pgx.Tx, businessID, productID string) (map[string]existingProductPriceRule, error) {
	rows, err := tx.Query(ctx, `
		SELECT
			id::text,
			price_type,
			min_quantity,
			price,
			COALESCE(location_id::text, ''),
			COALESCE(customer_group, ''),
			COALESCE(starts_at::text, ''),
			COALESCE(ends_at::text, ''),
			active,
			priority
		FROM product_prices
		WHERE business_id = $1::uuid
		  AND product_id = $2::uuid
	`, businessID, productID)
	if err != nil {
		return nil, fmt.Errorf("load product price rules for update: %w", err)
	}
	defer rows.Close()

	rules := make(map[string]existingProductPriceRule)
	for rows.Next() {
		var rule existingProductPriceRule
		if err := rows.Scan(
			&rule.ID,
			&rule.PriceType,
			&rule.MinQuantity,
			&rule.Price,
			&rule.LocationID,
			&rule.CustomerGroup,
			&rule.StartsAt,
			&rule.EndsAt,
			&rule.Active,
			&rule.Priority,
		); err != nil {
			return nil, fmt.Errorf("scan product price rule for update: %w", err)
		}
		rules[productPriceRuleKey(rule.PriceType, rule.MinQuantity, rule.LocationID, rule.CustomerGroup)] = rule
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate product price rules for update: %w", err)
	}

	return rules, nil
}

func insertProductPrices(ctx context.Context, tx pgx.Tx, businessID, productID string, prices []CreateProductPriceInput, existingRules map[string]existingProductPriceRule, actorID string) error {
	for _, price := range prices {
		priceType := strings.TrimSpace(strings.ToLower(price.PriceType))
		switch priceType {
		case "retail", "wholesale", "tier", "location", "promotion", "customer_group":
		default:
			return ErrInvalidProductInput
		}

		if price.MinQuantity <= 0 {
			price.MinQuantity = 1
		}
		if price.Price < 0 {
			return ErrInvalidProductInput
		}
		if price.Priority == 0 {
			price.Priority = 100
		}
		price.LocationID = strings.TrimSpace(price.LocationID)
		price.CustomerGroup = strings.TrimSpace(price.CustomerGroup)
		price.StartsAt = strings.TrimSpace(price.StartsAt)
		price.EndsAt = strings.TrimSpace(price.EndsAt)

		var productPriceID string
		if err := tx.QueryRow(ctx, `
			INSERT INTO product_prices (
				business_id,
				product_id,
				location_id,
				customer_group,
				price_type,
				min_quantity,
				price,
				starts_at,
				ends_at,
				active,
				priority
			)
			VALUES (
				$1::uuid,
				$2::uuid,
				NULLIF($3, '')::uuid,
				NULLIF($4, ''),
				$5,
				$6,
				$7,
				NULLIF($8, '')::timestamptz,
				NULLIF($9, '')::timestamptz,
				$10,
				$11
			)
			RETURNING id::text
		`,
			businessID,
			productID,
			price.LocationID,
			price.CustomerGroup,
			priceType,
			price.MinQuantity,
			price.Price,
			price.StartsAt,
			price.EndsAt,
			price.Active,
			price.Priority,
		).Scan(&productPriceID); err != nil {
			return fmt.Errorf("insert product prices: %w", err)
		}

		action := "created"
		var oldPrice any
		if existingRules != nil {
			if existingRule, exists := existingRules[productPriceRuleKey(priceType, price.MinQuantity, price.LocationID, price.CustomerGroup)]; exists {
				if !productPriceRuleChanged(existingRule, priceType, price) {
					continue
				}
				action = productPriceChangeAction(existingRule.Active, price.Active)
				oldPrice = existingRule.Price
			}
		}

		if err := insertProductPriceRuleHistory(ctx, tx, businessID, productID, productPriceID, action, priceType, price, oldPrice, actorID); err != nil {
			return err
		}
	}

	return nil
}

func recordRemovedProductPriceRules(ctx context.Context, tx pgx.Tx, businessID, productID string, existingRules map[string]existingProductPriceRule, nextRules []CreateProductPriceInput, actorID string) error {
	nextKeys := make(map[string]struct{}, len(nextRules))
	for _, rule := range nextRules {
		priceType := strings.TrimSpace(strings.ToLower(rule.PriceType))
		minQuantity := rule.MinQuantity
		if minQuantity <= 0 {
			minQuantity = 1
		}
		nextKeys[productPriceRuleKey(priceType, minQuantity, strings.TrimSpace(rule.LocationID), strings.TrimSpace(rule.CustomerGroup))] = struct{}{}
	}

	for key, existingRule := range existingRules {
		if _, exists := nextKeys[key]; exists {
			continue
		}
		price := CreateProductPriceInput{
			PriceType:     existingRule.PriceType,
			MinQuantity:   existingRule.MinQuantity,
			Price:         existingRule.Price,
			LocationID:    existingRule.LocationID,
			CustomerGroup: existingRule.CustomerGroup,
			StartsAt:      existingRule.StartsAt,
			EndsAt:        existingRule.EndsAt,
			Active:        false,
			Priority:      existingRule.Priority,
		}
		if err := insertProductPriceRuleHistory(ctx, tx, businessID, productID, existingRule.ID, "deleted", existingRule.PriceType, price, existingRule.Price, actorID); err != nil {
			return err
		}
	}

	return nil
}

func insertProductPriceRuleHistory(ctx context.Context, tx pgx.Tx, businessID, productID, productPriceID, action, priceType string, price CreateProductPriceInput, oldPrice any, actorID string) error {
	if _, err := tx.Exec(ctx, `
			INSERT INTO product_price_rule_history (
				business_id,
				product_id,
				product_price_id,
				action,
				price_type,
				min_quantity,
				old_price,
				new_price,
				location_id,
				customer_group,
				starts_at,
				ends_at,
				active,
				priority,
				reason,
				changed_by
			)
			VALUES (
				$1::uuid,
				$2::uuid,
				$3::uuid,
				$4,
				$5,
				$6,
				$7,
				$8,
				NULLIF($9, '')::uuid,
				NULLIF($10, ''),
				NULLIF($11, '')::timestamptz,
				NULLIF($12, '')::timestamptz,
				$13,
				$14,
				NULL,
				NULLIF($15, '')::uuid
			)
		`,
		businessID,
		productID,
		productPriceID,
		action,
		priceType,
		price.MinQuantity,
		oldPrice,
		price.Price,
		strings.TrimSpace(price.LocationID),
		strings.TrimSpace(price.CustomerGroup),
		strings.TrimSpace(price.StartsAt),
		strings.TrimSpace(price.EndsAt),
		price.Active,
		price.Priority,
		strings.TrimSpace(actorID),
	); err != nil {
		return fmt.Errorf("insert product price rule history: %w", err)
	}

	return nil
}

func productPriceRuleKey(priceType string, minQuantity float64, locationID, customerGroup string) string {
	return fmt.Sprintf(
		"%s|%.4f|%s|%s",
		strings.TrimSpace(strings.ToLower(priceType)),
		minQuantity,
		strings.TrimSpace(strings.ToLower(locationID)),
		strings.TrimSpace(strings.ToLower(customerGroup)),
	)
}

func productPriceRuleChanged(existingRule existingProductPriceRule, priceType string, nextRule CreateProductPriceInput) bool {
	return strings.TrimSpace(strings.ToLower(existingRule.PriceType)) != priceType ||
		existingRule.MinQuantity != nextRule.MinQuantity ||
		existingRule.Price != nextRule.Price ||
		strings.TrimSpace(existingRule.LocationID) != strings.TrimSpace(nextRule.LocationID) ||
		strings.TrimSpace(existingRule.CustomerGroup) != strings.TrimSpace(nextRule.CustomerGroup) ||
		strings.TrimSpace(existingRule.StartsAt) != strings.TrimSpace(nextRule.StartsAt) ||
		strings.TrimSpace(existingRule.EndsAt) != strings.TrimSpace(nextRule.EndsAt) ||
		existingRule.Active != nextRule.Active ||
		existingRule.Priority != nextRule.Priority
}

func productPriceChangeAction(wasActive, isActive bool) string {
	if wasActive && !isActive {
		return "deactivated"
	}
	if !wasActive && isActive {
		return "reactivated"
	}
	return "updated"
}

func SearchProductsRepository(pool *pgxpool.Pool, businessID, query, productType string) ([]models.ProductSearchItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	query = strings.TrimSpace(query)
	productType = strings.TrimSpace(strings.ToLower(productType))
	if businessID == "" {
		return nil, ErrBusinessNotResolved
	}

	args := []any{businessID}
	where := `
		WHERE p.business_id = $1
		  AND p.deleted_at IS NULL
	`

	if query != "" {
		args = append(args, "%"+strings.ToLower(query)+"%")
		where += fmt.Sprintf(`
		  AND (
			LOWER(p.name) LIKE $%d
			OR LOWER(p.sku) LIKE $%d
		  )
		`, len(args), len(args))
	}

	if productType != "" {
		args = append(args, productType)
		where += fmt.Sprintf(" AND p.product_type = $%d", len(args))
	}

	rows, err := pool.Query(ctx, fmt.Sprintf(`
		SELECT
			p.id::text,
			p.name,
			p.sku,
			COALESCE(u.name, '') AS unit_name,
			COALESCE(p.default_selling_price, 0) AS selling_price,
			COALESCE(stock.current_stock, 0) AS current_stock,
			COALESCE(p.tax_type, 'exclusive') AS tax_type,
			COALESCE(p.tax_rate, 0) AS tax_rate,
			COALESCE(p.default_purchase_price, 0) AS default_purchase_price,
			COALESCE(p.purchase_price_exclusive, 0) AS purchase_price_exclusive,
			COALESCE(p.purchase_price_inclusive, 0) AS purchase_price_inclusive,
			p.product_type
		FROM products p
		LEFT JOIN business_units u ON u.id = p.unit_id
		LEFT JOIN LATERAL (
			SELECT COALESCE(ROUND(SUM(ib.quantity_available)), 0)::int AS current_stock
			FROM inventory_balances ib
			WHERE ib.business_id = p.business_id
			  AND ib.product_id = p.id
		) stock ON TRUE
		%s
		ORDER BY p.created_at DESC, p.name ASC
		LIMIT 20
	`, where), args...)
	if err != nil {
		return nil, fmt.Errorf("search products: %w", err)
	}
	defer rows.Close()

	items := make([]models.ProductSearchItem, 0)
	for rows.Next() {
		var item models.ProductSearchItem
		var sku sql.NullString
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&sku,
			&item.UnitName,
			&item.SellingPrice,
			&item.CurrentStock,
			&item.TaxType,
			&item.TaxRate,
			&item.DefaultPurchasePrice,
			&item.PurchasePriceExclusive,
			&item.PurchasePriceInclusive,
			&item.ProductType,
		); err != nil {
			return nil, fmt.Errorf("scan product search item: %w", err)
		}
		item.SKU = models.StringPtrFromNullString(sku)
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate product search items: %w", err)
	}

	return items, nil
}

func nullIfBlank(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func generateProductSKU(ctx context.Context, tx pgx.Tx, businessID string) (string, error) {
	var activeCount int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM products
		WHERE business_id = $1
		  AND deleted_at IS NULL
	`, businessID).Scan(&activeCount); err != nil {
		return "", fmt.Errorf("count business products: %w", err)
	}

	sequence := activeCount + 1

	for attempts := 0; attempts < 10000; attempts++ {
		candidate := fmt.Sprintf("%04d", sequence)
		var exists bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM products
				WHERE business_id = $1
				  AND LOWER(sku) = LOWER($2)
				  AND deleted_at IS NULL
			)
		`, businessID, candidate).Scan(&exists); err != nil {
			return "", fmt.Errorf("verify generated product sku: %w", err)
		}
		if !exists {
			return candidate, nil
		}
		sequence++
	}

	return "", fmt.Errorf("generate product sku: exhausted sku space")
}
