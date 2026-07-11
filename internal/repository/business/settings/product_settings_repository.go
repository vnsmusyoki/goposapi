package settings

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"pos/internal/models"
)

func GetBusinessProductSettingsRepository(pool *pgxpool.Pool, businessID string) (*models.BusinessProductSettings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	if businessID == "" {
		return nil, ErrBusinessNotResolved
	}

	var settings models.BusinessProductSettings
	var stopSellingDays sql.NullInt32

	err := pool.QueryRow(ctx, `
		SELECT
			id::text,
			COALESCE(sku_prefix, ''),
			COALESCE(enable_product_expiry, FALSE),
			COALESCE(expiry_tracking_method, ''),
			COALESCE(expiry_selling_behavior, ''),
			stop_selling_days_before,
			COALESCE(enable_brands, FALSE),
			COALESCE(enable_categories, FALSE),
			COALESCE(enable_sub_categories, FALSE),
			COALESCE(enable_price_tax_info, FALSE),
			COALESCE(default_unit, ''),
			COALESCE(enable_sub_units, FALSE),
			COALESCE(enable_secondary_unit, FALSE),
			COALESCE(enable_racks, FALSE),
			COALESCE(enable_row, FALSE),
			COALESCE(enable_position, FALSE),
			COALESCE(enable_warranty, FALSE)
		FROM businesses
		WHERE id = $1
		LIMIT 1
	`, businessID).Scan(
		&settings.ID,
		&settings.SKUPrefix,
		&settings.EnableProductExpiry,
		&settings.ExpiryTrackingMethod,
		&settings.ExpirySellingBehavior,
		&stopSellingDays,
		&settings.EnableBrands,
		&settings.EnableCategories,
		&settings.EnableSubCategories,
		&settings.EnablePriceTaxInfo,
		&settings.DefaultUnit,
		&settings.EnableSubUnits,
		&settings.EnableSecondaryUnit,
		&settings.EnableRacks,
		&settings.EnableRow,
		&settings.EnablePosition,
		&settings.EnableWarranty,
	)
	if err != nil {
		return nil, fmt.Errorf("load business product settings: %w", err)
	}

	if stopSellingDays.Valid {
		value := int(stopSellingDays.Int32)
		settings.StopSellingDaysBefore = &value
	}

	return &settings, nil
}

func UpdateBusinessProductSettingsRepository(pool *pgxpool.Pool, req UpdateBusinessProductSettingsInput) (*models.BusinessProductSettings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.SKUPrefix = strings.TrimSpace(req.SKUPrefix)
	req.ExpiryTrackingMethod = strings.TrimSpace(req.ExpiryTrackingMethod)
	req.ExpirySellingBehavior = strings.TrimSpace(req.ExpirySellingBehavior)
	req.DefaultUnit = strings.TrimSpace(req.DefaultUnit)

	if req.BusinessID == "" || req.DefaultUnit == "" {
		return nil, ErrInvalidBusinessSettingsInput
	}

	_, err := pool.Exec(ctx, `
		UPDATE businesses
		SET sku_prefix = $2,
			enable_product_expiry = $3,
			expiry_tracking_method = $4,
			expiry_selling_behavior = $5,
			stop_selling_days_before = $6,
			enable_brands = $7,
			enable_categories = $8,
			enable_sub_categories = $9,
			enable_price_tax_info = $10,
			default_unit = $11,
			enable_sub_units = $12,
			enable_secondary_unit = $13,
			enable_racks = $14,
			enable_row = $15,
			enable_position = $16,
			enable_warranty = $17
		WHERE id = $1
	`, req.BusinessID, nullIfBlankValue(req.SKUPrefix), req.EnableProductExpiry, req.ExpiryTrackingMethod, req.ExpirySellingBehavior, req.StopSellingDaysBefore, req.EnableBrands, req.EnableCategories, req.EnableSubCategories, req.EnablePriceTaxInfo, req.DefaultUnit, req.EnableSubUnits, req.EnableSecondaryUnit, req.EnableRacks, req.EnableRow, req.EnablePosition, req.EnableWarranty)
	if err != nil {
		return nil, fmt.Errorf("update business product settings: %w", err)
	}

	return GetBusinessProductSettingsRepository(pool, req.BusinessID)
}

func nullIfBlankValue(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	return strings.TrimSpace(value)
}
