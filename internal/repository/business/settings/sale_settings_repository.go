package settings

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"pos/internal/models"
)

func GetBusinessSaleSettingsRepository(pool *pgxpool.Pool, businessID string) (*models.BusinessSaleSettings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	if businessID == "" {
		return nil, ErrBusinessNotResolved
	}

	var settings models.BusinessSaleSettings

	err := pool.QueryRow(ctx, `
		SELECT
			id::text,
			COALESCE(default_sale_discount, 0)::float8,
			COALESCE(default_sale_tax, 0)::float8,
			COALESCE(sale_item_addition_method, ''),
			COALESCE(enable_sale_order, FALSE),
			COALESCE(is_pay_term_required, FALSE),
			COALESCE(sale_price_is_minimum_selling_price, FALSE),
			COALESCE(enable_sale_commission_agent, FALSE),
			COALESCE(commission_calculation_type, ''),
			COALESCE(is_commission_agent_required, FALSE)
		FROM businesses
		WHERE id = $1
		LIMIT 1
	`, businessID).Scan(
		&settings.ID,
		&settings.DefaultSaleDiscount,
		&settings.DefaultSaleTax,
		&settings.SaleItemAdditionMethod,
		&settings.EnableSaleOrder,
		&settings.IsPayTermRequired,
		&settings.SalePriceIsMinimumSellingPrice,
		&settings.EnableSaleCommissionAgent,
		&settings.CommissionCalculationType,
		&settings.IsCommissionAgentRequired,
	)
	if err != nil {
		return nil, fmt.Errorf("load business sale settings: %w", err)
	}

	return &settings, nil
}

func UpdateBusinessSaleSettingsRepository(pool *pgxpool.Pool, req UpdateBusinessSaleSettingsInput) (*models.BusinessSaleSettings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.SaleItemAdditionMethod = strings.TrimSpace(req.SaleItemAdditionMethod)
	req.CommissionCalculationType = strings.TrimSpace(req.CommissionCalculationType)

	if req.BusinessID == "" || req.SaleItemAdditionMethod == "" || req.CommissionCalculationType == "" {
		return nil, ErrInvalidBusinessSettingsInput
	}

	_, err := pool.Exec(ctx, `
		UPDATE businesses
		SET default_sale_discount = $2,
			default_sale_tax = $3,
			sale_item_addition_method = $4,
			enable_sale_order = $5,
			is_pay_term_required = $6,
			sale_price_is_minimum_selling_price = $7,
			enable_sale_commission_agent = $8,
			commission_calculation_type = $9,
			is_commission_agent_required = $10
		WHERE id = $1
	`, req.BusinessID, req.DefaultSaleDiscount, req.DefaultSaleTax, req.SaleItemAdditionMethod, req.EnableSaleOrder, req.IsPayTermRequired, req.SalePriceIsMinimumSellingPrice, req.EnableSaleCommissionAgent, req.CommissionCalculationType, req.IsCommissionAgentRequired)
	if err != nil {
		return nil, fmt.Errorf("update business sale settings: %w", err)
	}

	return GetBusinessSaleSettingsRepository(pool, req.BusinessID)
}
