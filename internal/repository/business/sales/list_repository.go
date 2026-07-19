package sales

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

func ListSalesOrdersRepository(pool *pgxpool.Pool, businessID string, filters SalesOrderFilters) ([]SalesOrderListItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	if businessID == "" {
		return nil, ErrBusinessNotResolved
	}

	query, args := salesOrderListQuery(businessID, filters)
	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list sales orders: %w", err)
	}
	defer rows.Close()

	orders := make([]SalesOrderListItem, 0)
	for rows.Next() {
		var (
			order      SalesOrderListItem
			customerID string
			paidAmount float64
		)
		if err := rows.Scan(
			&order.ID,
			&order.BusinessID,
			&order.LocationID,
			&customerID,
			&order.LocationName,
			&order.ReferenceNumber,
			&order.SaleDate,
			&order.CustomerName,
			&order.CustomerPhone,
			&order.Status,
			&order.ShippingStatus,
			&order.ItemsCount,
			&order.GrandTotal,
			&paidAmount,
			&order.BalanceDue,
			&order.SaleID,
			&order.ConvertedAt,
			&order.CreatedAt,
			&order.UpdatedAt,
		); err != nil {
			return nil, err
		}
		order.CustomerID = customerID
		order.PaidAmount = paidAmount
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sales orders: %w", err)
	}

	return orders, nil
}

func GetSalesOrderByIDRepository(pool *pgxpool.Pool, businessID, salesOrderID string) (*Sale, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	salesOrderID = strings.TrimSpace(salesOrderID)
	if businessID == "" || salesOrderID == "" {
		return nil, ErrBusinessNotResolved
	}

	row := pool.QueryRow(ctx, salesOrderSelectQuery()+`
		WHERE so.business_id = $1::uuid
		  AND so.id = $2::uuid
		  AND so.deleted_at IS NULL
		LIMIT 1
	`, businessID, salesOrderID)

	order, err := scanSalesOrder(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrSaleNotFound
		}
		return nil, err
	}

	return &order, nil
}

func salesOrderSelectQuery() string {
	return `
		SELECT
			so.id::text,
			so.business_id::text,
			so.location_id::text,
			COALESCE(so.customer_id::text, '') AS customer_id,
			COALESCE(loc.location_name, '') AS location_name,
			so.reference_number,
			so.sale_date::text,
			COALESCE(
				NULLIF(
					TRIM(CONCAT_WS(' ', c.first_name, c.middle_name, c.last_name)),
					''
				),
				NULLIF(c.company_name, ''),
				COALESCE(so.customer_name, '')
			) AS customer_name,
			COALESCE(NULLIF(c.phone, ''), so.customer_phone, '') AS customer_phone,
			so.status,
			CASE
				WHEN so.sale_id IS NOT NULL OR so.status = 'completed' THEN 'completed'
				WHEN so.status = 'ready_for_shipment' THEN 'ready_for_shipment'
				WHEN so.status = 'processing' THEN 'processing'
				WHEN so.status = 'approved' THEN 'pending'
				WHEN so.status = 'pending_approval' THEN 'pending'
				ELSE 'pending'
			END AS shipping_status,
			so.items_count,
			so.grand_total,
			COALESCE(so.paid_amount, 0)::numeric,
			GREATEST(so.grand_total - COALESCE(so.paid_amount, 0), 0)::numeric AS balance_due,
			COALESCE(so.sale_id::text, '') AS sale_id,
			COALESCE(so.converted_at::text, '') AS converted_at,
			so.created_at::text,
			so.updated_at::text
		FROM sales_orders so
		LEFT JOIN business_locations loc ON loc.id = so.location_id
		LEFT JOIN customers c ON c.id = so.customer_id`
}

func salesOrderListQuery(businessID string, filters SalesOrderFilters) (string, []any) {
	query := salesOrderSelectQuery()
	conditions := []string{
		"so.business_id = $1",
		"so.deleted_at IS NULL",
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
		addCondition("so.location_id::text = ?", value)
	}
	if value := strings.TrimSpace(filters.CustomerID); value != "" {
		addCondition("so.customer_id::text = ?", value)
	}
	if value := strings.TrimSpace(filters.Status); value != "" {
		addCondition("so.status = ?", strings.ToLower(value))
	}
	if value := strings.TrimSpace(filters.ShippingStatus); value != "" {
		addCondition(`CASE
			WHEN so.sale_id IS NOT NULL OR so.status = 'completed' THEN 'completed'
			WHEN so.status = 'ready_for_shipment' THEN 'ready_for_shipment'
			WHEN so.status = 'processing' THEN 'processing'
			WHEN so.status = 'approved' THEN 'pending'
			WHEN so.status = 'pending_approval' THEN 'pending'
			ELSE 'pending'
		END = ?`, strings.ToLower(value))
	}
	if value := strings.TrimSpace(filters.DateFrom); value != "" {
		addCondition("so.sale_date::date >= ?::date", value)
	}
	if value := strings.TrimSpace(filters.DateTo); value != "" {
		addCondition("so.sale_date::date <= ?::date", value)
	}
	if value := strings.TrimSpace(filters.SearchQuery); value != "" {
		search := "%" + strings.ToLower(value) + "%"
		addCondition(`(
			LOWER(so.reference_number) LIKE ?
			OR LOWER(COALESCE(so.customer_name, '')) LIKE ?
			OR LOWER(COALESCE(so.customer_phone, '')) LIKE ?
			OR LOWER(COALESCE(loc.location_name, '')) LIKE ?
		)`, search, search, search, search)
	}

	if len(conditions) > 0 {
		query += "\n\t\tWHERE " + strings.Join(conditions, "\n\t\t  AND ")
	}
	query += "\n\t\tORDER BY so.created_at DESC"
	return query, args
}

func scanSalesOrder(scanner interface {
	Scan(dest ...any) error
}) (Sale, error) {
	var (
		order        Sale
		customerID   string
		locationName string
		paidAmount   float64
		balanceDue   float64
	)

	if err := scanner.Scan(
		&order.ID,
		&order.BusinessID,
		&order.LocationID,
		&customerID,
		&locationName,
		&order.ReferenceNumber,
		&order.SaleDate,
		&order.CustomerName,
		&order.CustomerPhone,
		&order.Status,
		&order.ShippingStatus,
		&order.ItemsCount,
		&order.GrandTotal,
		&paidAmount,
		&balanceDue,
		&order.SaleID,
		&order.ConvertedAt,
		&order.CreatedAt,
		&order.UpdatedAt,
	); err != nil {
		return Sale{}, err
	}

	order.CustomerID = customerID
	order.PaidAmount = paidAmount
	order.BalanceDue = balanceDue
	_ = locationName
	return order, nil
}
