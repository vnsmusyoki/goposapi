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

func GetBusinessSettingsRepository(pool *pgxpool.Pool, businessID string) (*models.BusinessSettings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	if businessID == "" {
		return nil, ErrBusinessNotResolved
	}

	var settings models.BusinessSettings
	var defaultProfit sql.NullFloat64
	var transactionEditDays sql.NullInt32
	var currencyPrecision sql.NullInt32
	var quantityPrecision sql.NullInt32

	err := pool.QueryRow(ctx, `
		SELECT
			id::text,
			COALESCE(name, ''),
			COALESCE(start_date::text, ''),
			default_profit_percentage,
			COALESCE(currency, ''),
			COALESCE(currency_symbol_placement, ''),
			COALESCE(timezone, ''),
			COALESCE(logo_url, ''),
			COALESCE(financial_year_start_month, ''),
			COALESCE(stock_accounting_method, ''),
			COALESCE(preserve_sale_order_requests, FALSE),
			transaction_edit_days,
			COALESCE(date_format, ''),
			COALESCE(time_format, ''),
			currency_precision,
			quantity_precision
		FROM businesses
		WHERE id = $1
		LIMIT 1
	`, businessID).Scan(
		&settings.ID,
		&settings.Name,
		&settings.StartDate,
		&defaultProfit,
		&settings.Currency,
		&settings.CurrencySymbolPlacement,
		&settings.Timezone,
		&settings.LogoURL,
		&settings.FinancialYearStartMonth,
		&settings.StockAccountingMethod,
		&settings.PreserveSaleOrderRequests,
		&transactionEditDays,
		&settings.DateFormat,
		&settings.TimeFormat,
		&currencyPrecision,
		&quantityPrecision,
	)
	if err != nil {
		return nil, fmt.Errorf("load business settings: %w", err)
	}

	if defaultProfit.Valid {
		value := defaultProfit.Float64
		settings.DefaultProfitPercentage = &value
	}
	if transactionEditDays.Valid {
		value := int(transactionEditDays.Int32)
		settings.TransactionEditDays = &value
	}
	if currencyPrecision.Valid {
		value := int(currencyPrecision.Int32)
		settings.CurrencyPrecision = &value
	}
	if quantityPrecision.Valid {
		value := int(quantityPrecision.Int32)
		settings.QuantityPrecision = &value
	}

	return &settings, nil
}

func UpdateBusinessSettingsRepository(pool *pgxpool.Pool, req UpdateBusinessSettingsInput) (*models.BusinessSettings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.Name = strings.TrimSpace(req.Name)
	req.StartDate = strings.TrimSpace(req.StartDate)
	req.Currency = strings.TrimSpace(req.Currency)
	req.CurrencySymbolPlacement = strings.TrimSpace(req.CurrencySymbolPlacement)
	req.Timezone = strings.TrimSpace(req.Timezone)
	req.LogoURL = strings.TrimSpace(req.LogoURL)
	req.FinancialYearStartMonth = strings.TrimSpace(req.FinancialYearStartMonth)
	req.StockAccountingMethod = strings.TrimSpace(req.StockAccountingMethod)
	req.DateFormat = strings.TrimSpace(req.DateFormat)
	req.TimeFormat = strings.TrimSpace(req.TimeFormat)

	if req.LogoURL != "" {
		var err error
		req.LogoURL, err = normalizeBusinessSettingsLogoDataURL(req.LogoURL)
		if err != nil {
			return nil, err
		}
	}

	if req.BusinessID == "" || req.Name == "" {
		return nil, ErrInvalidBusinessSettingsInput
	}

	if _, err := time.Parse("2006-01-02", req.StartDate); err != nil {
		return nil, ErrInvalidBusinessSettingsInput
	}

	_, err := pool.Exec(ctx, `
		UPDATE businesses
		SET name = $2,
			start_date = $3,
			default_profit_percentage = $4,
			currency = $5,
			currency_symbol_placement = $6,
			timezone = $7,
			logo_url = $8,
			financial_year_start_month = $9,
			stock_accounting_method = $10,
			preserve_sale_order_requests = $11,
			transaction_edit_days = $12,
			date_format = $13,
			time_format = $14,
			currency_precision = $15,
			quantity_precision = $16
		WHERE id = $1
	`, req.BusinessID, req.Name, req.StartDate, req.DefaultProfitPercentage, req.Currency, req.CurrencySymbolPlacement, req.Timezone, nullIfBlank(req.LogoURL), req.FinancialYearStartMonth, req.StockAccountingMethod, req.PreserveSaleOrderRequests, req.TransactionEditDays, req.DateFormat, req.TimeFormat, req.CurrencyPrecision, req.QuantityPrecision)
	if err != nil {
		return nil, fmt.Errorf("update business settings: %w", err)
	}

	return GetBusinessSettingsRepository(pool, req.BusinessID)
}

func nullIfBlank(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	return strings.TrimSpace(value)
}
