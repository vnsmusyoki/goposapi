package settings

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"pos/internal/models"
)

func GetBusinessPosSettingsRepository(pool *pgxpool.Pool, businessID string) (*models.BusinessPosSettings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	if businessID == "" {
		return nil, ErrBusinessNotResolved
	}

	var settings models.BusinessPosSettings

	err := pool.QueryRow(ctx, `
		SELECT
			id::text,
			COALESCE(disable_multiple_pay, FALSE),
			COALESCE(disable_draft, FALSE),
			COALESCE(disable_express_checkout, FALSE),
			COALESCE(disable_discount, FALSE),
			COALESCE(disable_order_tax, FALSE),
			COALESCE(disable_credit_sale_button, FALSE),
			COALESCE(disable_suspend_sale, FALSE),
			COALESCE(subtotal_editable, FALSE),
			COALESCE(hide_product_suggestion, FALSE),
			COALESCE(show_pricing_on_product_suggestion_tooltip, FALSE),
			COALESCE(hide_recent_transactions, FALSE),
			COALESCE(enable_transaction_date_on_pos_screen, FALSE),
			COALESCE(enable_weighing_scale, FALSE),
			COALESCE(enable_service_staff_in_product_line, FALSE),
			COALESCE(is_service_staff_required, FALSE),
			COALESCE(invoice_scheme, ''),
			COALESCE(invoice_layout, ''),
			COALESCE(print_invoice_on_suspend, FALSE)
		FROM businesses
		WHERE id = $1
		LIMIT 1
	`, businessID).Scan(
		&settings.ID,
		&settings.DisableMultiplePay,
		&settings.DisableDraft,
		&settings.DisableExpressCheckout,
		&settings.DisableDiscount,
		&settings.DisableOrderTax,
		&settings.DisableCreditSaleButton,
		&settings.DisableSuspendSale,
		&settings.SubtotalEditable,
		&settings.HideProductSuggestion,
		&settings.ShowPricingOnProductSuggestionTooltip,
		&settings.HideRecentTransactions,
		&settings.EnableTransactionDateOnPosScreen,
		&settings.EnableWeighingScale,
		&settings.EnableServiceStaffInProductLine,
		&settings.IsServiceStaffRequired,
		&settings.InvoiceScheme,
		&settings.InvoiceLayout,
		&settings.PrintInvoiceOnSuspend,
	)
	if err != nil {
		return nil, fmt.Errorf("load business pos settings: %w", err)
	}

	return &settings, nil
}

func UpdateBusinessPosSettingsRepository(pool *pgxpool.Pool, req UpdateBusinessPosSettingsInput) (*models.BusinessPosSettings, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.InvoiceScheme = strings.TrimSpace(req.InvoiceScheme)
	req.InvoiceLayout = strings.TrimSpace(req.InvoiceLayout)

	if req.BusinessID == "" || req.InvoiceScheme == "" || req.InvoiceLayout == "" {
		return nil, ErrInvalidBusinessSettingsInput
	}

	_, err := pool.Exec(ctx, `
		UPDATE businesses
		SET disable_multiple_pay = $2,
			disable_draft = $3,
			disable_express_checkout = $4,
			disable_discount = $5,
			disable_order_tax = $6,
			disable_credit_sale_button = $7,
			disable_suspend_sale = $8,
			subtotal_editable = $9,
			hide_product_suggestion = $10,
			show_pricing_on_product_suggestion_tooltip = $11,
			hide_recent_transactions = $12,
			enable_transaction_date_on_pos_screen = $13,
			enable_weighing_scale = $14,
			enable_service_staff_in_product_line = $15,
			is_service_staff_required = $16,
			invoice_scheme = $17,
			invoice_layout = $18,
			print_invoice_on_suspend = $19
		WHERE id = $1
	`, req.BusinessID, req.DisableMultiplePay, req.DisableDraft, req.DisableExpressCheckout, req.DisableDiscount, req.DisableOrderTax, req.DisableCreditSaleButton, req.DisableSuspendSale, req.SubtotalEditable, req.HideProductSuggestion, req.ShowPricingOnProductSuggestionTooltip, req.HideRecentTransactions, req.EnableTransactionDateOnPosScreen, req.EnableWeighingScale, req.EnableServiceStaffInProductLine, req.IsServiceStaffRequired, req.InvoiceScheme, req.InvoiceLayout, req.PrintInvoiceOnSuspend)
	if err != nil {
		return nil, fmt.Errorf("update business pos settings: %w", err)
	}

	return GetBusinessPosSettingsRepository(pool, req.BusinessID)
}
