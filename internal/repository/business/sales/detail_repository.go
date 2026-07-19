package sales

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

func GetSalesOrderDetailRepository(pool *pgxpool.Pool, businessID, salesOrderID string) (*SalesOrderDetail, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	businessID = strings.TrimSpace(businessID)
	salesOrderID = strings.TrimSpace(salesOrderID)
	if businessID == "" || salesOrderID == "" {
		return nil, ErrBusinessNotResolved
	}

	order, err := GetSaleByIDRepositoryTx(ctx, pool, businessID, salesOrderID)
	if err != nil {
		return nil, err
	}

	customer, err := getSalesOrderCustomerDetails(ctx, pool, businessID, order.CustomerID)
	if err != nil {
		return nil, err
	}
	if customer.Name == "" || customer.Name == "Customer" {
		customer.Name = firstNonEmptySales(order.CustomerName, customer.Name)
	}
	if customer.Phone == "" {
		customer.Phone = order.CustomerPhone
	}
	if customer.Email == "" {
		customer.Email = order.CustomerEmail
	}

	business, err := getSalesOrderBusinessDetails(ctx, pool, businessID)
	if err != nil {
		return nil, err
	}

	location, err := getSalesOrderLocationDetails(ctx, pool, businessID, order.LocationID)
	if err != nil {
		return nil, err
	}
	business.BranchName = firstNonEmptySales(business.BranchName, location.Name)
	business.BranchAddress = firstNonEmptySales(business.BranchAddress, location.Address)

	items, err := listSalesOrderItemsByOrderID(ctx, pool, businessID, salesOrderID)
	if err != nil {
		return nil, err
	}

	activities, err := listSalesOrderLogsByOrderID(ctx, pool, businessID, salesOrderID)
	if err != nil {
		return nil, err
	}

	return &SalesOrderDetail{
		SaleOrder:  *order,
		Customer:   customer,
		Business:   business,
		Location:   location,
		Items:      items,
		Activities: activities,
	}, nil
}

func getSalesOrderCustomerDetails(ctx context.Context, querier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, businessID, customerID string) (SalesOrderCustomerDetails, error) {
	if strings.TrimSpace(customerID) == "" {
		return SalesOrderCustomerDetails{}, nil
	}

	var (
		name    sql.NullString
		email   sql.NullString
		phone   sql.NullString
		address sql.NullString
	)

	if err := querier.QueryRow(ctx, `
		SELECT
			COALESCE(
				NULLIF(TRIM(CONCAT_WS(' ', c.first_name, c.middle_name, c.last_name)), ''),
				NULLIF(c.company_name, ''),
				''
			) AS name,
			COALESCE(c.email, ''),
			COALESCE(c.phone, ''),
			COALESCE(c.address, '')
		FROM customers c
		WHERE c.business_id = $1::uuid
		  AND c.id = $2::uuid
		  AND c.deleted_at IS NULL
		LIMIT 1
	`, businessID, customerID).Scan(&name, &email, &phone, &address); err != nil {
		if err == pgx.ErrNoRows {
			return SalesOrderCustomerDetails{}, ErrSaleNotFound
		}
		return SalesOrderCustomerDetails{}, fmt.Errorf("load sales order customer details: %w", err)
	}

	return SalesOrderCustomerDetails{
		Name:    firstNonEmptySales(name.String, "Customer"),
		Email:   email.String,
		Phone:   phone.String,
		Address: address.String,
	}, nil
}

func getSalesOrderBusinessDetails(ctx context.Context, querier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, businessID string) (SalesOrderBusinessDetails, error) {
	var name, email, phone sql.NullString

	if err := querier.QueryRow(ctx, `
		SELECT
			COALESCE(name, ''),
			COALESCE(business_email, ''),
			COALESCE(business_phone, '')
		FROM businesses
		WHERE id = $1::uuid
		LIMIT 1
	`, businessID).Scan(&name, &email, &phone); err != nil {
		if err == pgx.ErrNoRows {
			return SalesOrderBusinessDetails{}, ErrSaleNotFound
		}
		return SalesOrderBusinessDetails{}, fmt.Errorf("load sales order business details: %w", err)
	}

	return SalesOrderBusinessDetails{
		Name:  name.String,
		Email: email.String,
		Phone: phone.String,
	}, nil
}

func getSalesOrderLocationDetails(ctx context.Context, querier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, businessID, locationID string) (SalesOrderLocationDetails, error) {
	var (
		name    sql.NullString
		email   sql.NullString
		phone   sql.NullString
		address sql.NullString
	)

	if err := querier.QueryRow(ctx, `
		SELECT
			COALESCE(location_name, ''),
			COALESCE(email, ''),
			COALESCE(mobile, ''),
			COALESCE(exact_address, '')
		FROM business_locations
		WHERE business_id = $1::uuid
		  AND id = $2::uuid
		LIMIT 1
	`, businessID, locationID).Scan(&name, &email, &phone, &address); err != nil {
		if err == pgx.ErrNoRows {
			return SalesOrderLocationDetails{}, ErrSaleNotFound
		}
		return SalesOrderLocationDetails{}, fmt.Errorf("load sales order location details: %w", err)
	}

	return SalesOrderLocationDetails{
		Name:    name.String,
		Email:   email.String,
		Phone:   phone.String,
		Address: address.String,
	}, nil
}

func listSalesOrderItemsByOrderID(ctx context.Context, querier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, businessID, salesOrderID string) ([]SaleItem, error) {
	rows, err := querier.Query(ctx, `
		SELECT
			id::text,
			sales_order_id::text,
			business_id::text,
			product_id::text,
			product_name,
			sku,
			unit,
			quantity,
			unit_cost,
			discount_percentage,
			discount_amount,
			tax_rate,
			tax_amount,
			unit_price,
			line_total,
			batch_tracking_enabled,
			sort_order,
			created_at::text,
			updated_at::text
		FROM sales_order_items
		WHERE business_id = $1::uuid
		  AND sales_order_id = $2::uuid
		  AND deleted_at IS NULL
		ORDER BY sort_order ASC, created_at ASC, id ASC
	`, businessID, salesOrderID)
	if err != nil {
		return nil, fmt.Errorf("load sales order items: %w", err)
	}
	defer rows.Close()

	items := make([]SaleItem, 0)
	for rows.Next() {
		var item SaleItem
		if err := rows.Scan(
			&item.ID,
			&item.SaleID,
			&item.BusinessID,
			&item.ProductID,
			&item.ProductName,
			&item.SKU,
			&item.Unit,
			&item.Quantity,
			&item.UnitCost,
			&item.DiscountPercentage,
			&item.DiscountAmount,
			&item.TaxRate,
			&item.TaxAmount,
			&item.UnitPrice,
			&item.LineTotal,
			&item.BatchTrackingEnabled,
			&item.SortOrder,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan sales order item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sales order items: %w", err)
	}

	return items, nil
}

func listSalesOrderLogsByOrderID(ctx context.Context, querier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, businessID, salesOrderID string) ([]SalesOrderLog, error) {
	rows, err := querier.Query(ctx, `
		SELECT
			sol.id::text,
			sol.business_id::text,
			sol.sales_order_id::text,
			sol.action,
			COALESCE(u.id::text, ''),
			COALESCE(u.full_name, 'System'),
			COALESCE(sol.note, ''),
			sol.action_date::text
		FROM sales_order_logs sol
		LEFT JOIN users u ON u.id = sol.actioned_by
		WHERE sol.business_id = $1::uuid
		  AND sol.sales_order_id = $2::uuid
		ORDER BY sol.action_date ASC, sol.created_at ASC, sol.id ASC
	`, businessID, salesOrderID)
	if err != nil {
		return nil, fmt.Errorf("load sales order logs: %w", err)
	}
	defer rows.Close()

	activities := make([]SalesOrderLog, 0)
	for rows.Next() {
		var (
			entry         SalesOrderLog
			actionedByID  sql.NullString
			actionedByName sql.NullString
		)
		if err := rows.Scan(
			&entry.ID,
			&entry.BusinessID,
			&entry.SalesOrderID,
			&entry.Action,
			&actionedByID,
			&actionedByName,
			&entry.Note,
			&entry.ActionDate,
		); err != nil {
			return nil, fmt.Errorf("scan sales order log: %w", err)
		}
		if actionedByID.Valid || actionedByName.Valid {
			entry.ActionedBy = &SalesOrderActor{
				ID:   actionedByID.String,
				Name: actionedByName.String,
			}
		}
		activities = append(activities, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sales order logs: %w", err)
	}

	return activities, nil
}

func firstNonEmptySales(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
