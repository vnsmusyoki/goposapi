package sales

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

func ListSalesRepository(pool *pgxpool.Pool, businessID string, filters SalesOrderFilters) ([]SalesOrderListItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	if businessID == "" {
		return nil, ErrBusinessNotResolved
	}

	query, args := salesListQuery(businessID, filters)
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list sales: %w", err)
	}
	defer rows.Close()

	sales := make([]SalesOrderListItem, 0)
	for rows.Next() {
		var (
			sale       SalesOrderListItem
			customerID string
			paidAmount float64
		)
		if err := rows.Scan(
			&sale.ID,
			&sale.BusinessID,
			&sale.LocationID,
			&customerID,
			&sale.LocationName,
			&sale.ReferenceNumber,
			&sale.SaleDate,
			&sale.CustomerName,
			&sale.CustomerPhone,
			&sale.Status,
			&sale.ShippingStatus,
			&sale.ItemsCount,
			&sale.GrandTotal,
			&paidAmount,
			&sale.BalanceDue,
			&sale.SaleID,
			&sale.ConvertedAt,
			&sale.CreatedAt,
			&sale.UpdatedAt,
		); err != nil {
			return nil, err
		}
		sale.CustomerID = customerID
		sale.PaidAmount = paidAmount
		sales = append(sales, sale)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sales: %w", err)
	}

	return sales, nil
}

func salesListQuery(businessID string, filters SalesOrderFilters) (string, []any) {
	query := `
		SELECT
			s.id::text,
			s.business_id::text,
			s.location_id::text,
			'' AS customer_id,
			COALESCE(loc.location_name, '') AS location_name,
			s.reference_number,
			s.sale_date::text,
			COALESCE(s.customer_name, '') AS customer_name,
			COALESCE(s.customer_phone, '') AS customer_phone,
			s.status,
			'completed' AS shipping_status,
			s.items_count,
			s.grand_total,
			0::numeric AS paid_amount,
			s.grand_total::numeric AS balance_due,
			s.id::text AS sale_id,
			s.created_at::text AS converted_at,
			s.created_at::text,
			s.updated_at::text
		FROM sales s
		LEFT JOIN business_locations loc ON loc.id = s.location_id
	`

	conditions := []string{
		"s.business_id = $1",
		"s.deleted_at IS NULL",
	}
	args := []any{businessID}

	addCondition := func(condition string, values ...any) {
		updated := condition
		for _, value := range values {
			args = append(args, value)
			placeholder := fmt.Sprintf("$%d", len(args))
			updated = strings.Replace(updated, "?", placeholder, 1)
		}
		conditions = append(conditions, updated)
	}

	if value := strings.TrimSpace(filters.LocationID); value != "" {
		addCondition("s.location_id::text = ?", value)
	}
	if value := strings.TrimSpace(filters.Status); value != "" && !strings.EqualFold(value, "completed") {
		addCondition("1 = 0")
	}
	if value := strings.TrimSpace(filters.ShippingStatus); value != "" && !strings.EqualFold(value, "completed") {
		addCondition("1 = 0")
	}
	if value := strings.TrimSpace(filters.DateFrom); value != "" {
		addCondition("s.sale_date::date >= ?::date", value)
	}
	if value := strings.TrimSpace(filters.DateTo); value != "" {
		addCondition("s.sale_date::date <= ?::date", value)
	}
	if value := strings.TrimSpace(filters.SearchQuery); value != "" {
		search := "%" + strings.ToLower(value) + "%"
		addCondition(`(
			LOWER(s.reference_number) LIKE ?
			OR LOWER(COALESCE(s.customer_name, '')) LIKE ?
			OR LOWER(COALESCE(s.customer_phone, '')) LIKE ?
			OR LOWER(COALESCE(loc.location_name, '')) LIKE ?
		)`, search, search, search, search)
	}

	if len(conditions) > 0 {
		query += "\n\tWHERE " + strings.Join(conditions, "\n\t  AND ")
	}
	query += "\n\tORDER BY s.created_at DESC"
	return query, args
}

func GetSaleRepository(pool *pgxpool.Pool, businessID, saleID string) (*Sale, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	saleID = strings.TrimSpace(saleID)
	if businessID == "" || saleID == "" {
		return nil, ErrBusinessNotResolved
	}

	row := pool.QueryRow(ctx, `
		SELECT
			id::text,
			business_id::text,
			location_id::text,
			'' AS customer_id,
			reference_number,
			sale_date::text,
			COALESCE(customer_name, ''),
			COALESCE(customer_phone, ''),
			COALESCE(customer_email, ''),
			status,
			subtotal,
			total_discount,
			total_tax,
			grand_total,
			items_count,
			total_quantity,
			COALESCE(notes, ''),
			'' AS stock_accounting_method,
			FALSE AS reserve_order_items,
			'completed' AS shipping_status,
			0::numeric AS paid_amount,
			grand_total::numeric AS balance_due,
			id::text AS sale_id,
			created_at::text AS converted_at,
			COALESCE(created_by::text, ''),
			created_at::text,
			updated_at::text
		FROM sales
		WHERE business_id = $1::uuid
		  AND id = $2::uuid
		  AND deleted_at IS NULL
		LIMIT 1
	`, businessID, saleID)

	var sale Sale
	var customerID string
	if err := row.Scan(
		&sale.ID,
		&sale.BusinessID,
		&sale.LocationID,
		&customerID,
		&sale.ReferenceNumber,
		&sale.SaleDate,
		&sale.CustomerName,
		&sale.CustomerPhone,
		&sale.CustomerEmail,
		&sale.Status,
		&sale.Subtotal,
		&sale.TotalDiscount,
		&sale.TotalTax,
		&sale.GrandTotal,
		&sale.ItemsCount,
		&sale.TotalQuantity,
		&sale.Notes,
		&sale.StockAccountingMethod,
		&sale.ReserveOrderItems,
		&sale.ShippingStatus,
		&sale.PaidAmount,
		&sale.BalanceDue,
		&sale.SaleID,
		&sale.ConvertedAt,
		&sale.CreatedBy,
		&sale.CreatedAt,
		&sale.UpdatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrSaleNotFound
		}
		return nil, fmt.Errorf("load sale: %w", err)
	}

	_ = customerID
	return &sale, nil
}
