package settings

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"pos/internal/models"
)

func GetBusinessPurchasesSettingsRepository(pool *pgxpool.Pool, businessID string) (*models.BusinessPurchasesSettings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	if businessID == "" {
		return nil, ErrBusinessNotResolved
	}

	var settings models.BusinessPurchasesSettings

	err := pool.QueryRow(ctx, `
		SELECT
			id::text,
			COALESCE(enable_editing_product_price_from_purchase_screen, FALSE),
			COALESCE(enable_purchase_status, FALSE),
			COALESCE(enable_lot_number, FALSE),
			COALESCE(enable_purchase_order, FALSE)
		FROM businesses
		WHERE id = $1
		LIMIT 1
	`, businessID).Scan(
		&settings.ID,
		&settings.EnableEditingProductPriceFromPurchaseScreen,
		&settings.EnablePurchaseStatus,
		&settings.EnableLotNumber,
		&settings.EnablePurchaseOrder,
	)
	if err != nil {
		return nil, fmt.Errorf("load business purchases settings: %w", err)
	}

	return &settings, nil
}

func UpdateBusinessPurchasesSettingsRepository(pool *pgxpool.Pool, req UpdateBusinessPurchasesSettingsInput) (*models.BusinessPurchasesSettings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req.BusinessID = strings.TrimSpace(req.BusinessID)
	if req.BusinessID == "" {
		return nil, ErrInvalidBusinessSettingsInput
	}

	_, err := pool.Exec(ctx, `
		UPDATE businesses
		SET enable_editing_product_price_from_purchase_screen = $2,
			enable_purchase_status = $3,
			enable_lot_number = $4,
			enable_purchase_order = $5
		WHERE id = $1
	`, req.BusinessID, req.EnableEditingProductPriceFromPurchaseScreen, req.EnablePurchaseStatus, req.EnableLotNumber, req.EnablePurchaseOrder)
	if err != nil {
		return nil, fmt.Errorf("update business purchases settings: %w", err)
	}

	return GetBusinessPurchasesSettingsRepository(pool, req.BusinessID)
}
