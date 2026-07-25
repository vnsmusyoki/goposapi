package sales

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type CreatePosSalePaymentInput struct {
	PaymentMethodCode string
	Amount            float64
	ReferenceNumber   string
	Phone             string
	Notes             string
}

type CreatePosSaleInput struct {
	BusinessID    string
	LocationID    string
	CustomerID    string
	CustomerName  string
	CustomerPhone string
	CustomerEmail string
	SaleDate      string
	Notes         string
	CreatedBy     string
	Items         []CreateSaleItemInput
	Payments      []CreatePosSalePaymentInput
	Subtotal      float64
	TotalDiscount float64
	TotalTax      float64
	GrandTotal    float64
	ItemsCount    int
	TotalQuantity float64
}

type PosSalePayment struct {
	ID                string  `json:"id"`
	PaymentMethodID   string  `json:"paymentMethodId"`
	PaymentMethodCode string  `json:"paymentMethodCode"`
	PaymentMethodName string  `json:"paymentMethodName"`
	Amount            float64 `json:"amount"`
	ReferenceNumber   string  `json:"referenceNumber"`
	Phone             string  `json:"phone"`
	IsCredit          bool    `json:"isCredit"`
	Status            string  `json:"status"`
}

type PosSaleResult struct {
	Sale     Sale             `json:"sale"`
	Payments []PosSalePayment `json:"payments"`
}

type paymentMethodSnapshot struct {
	ID            string
	Code          string
	Name          string
	IsCredit      bool
	RequiresPhone bool
}

func CreatePosSaleRepository(pool *pgxpool.Pool, req CreatePosSaleInput) (*PosSaleResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.LocationID = strings.TrimSpace(req.LocationID)
	req.CustomerID = strings.TrimSpace(req.CustomerID)
	req.CustomerName = strings.TrimSpace(req.CustomerName)
	req.CustomerPhone = strings.TrimSpace(req.CustomerPhone)
	req.CustomerEmail = strings.ToLower(strings.TrimSpace(req.CustomerEmail))
	req.SaleDate = strings.TrimSpace(req.SaleDate)
	req.Notes = strings.TrimSpace(req.Notes)
	req.CreatedBy = strings.TrimSpace(req.CreatedBy)
	if req.BusinessID == "" || req.LocationID == "" || req.CreatedBy == "" || len(req.Items) == 0 || len(req.Payments) == 0 || req.GrandTotal <= 0 {
		return nil, ErrInvalidSaleInput
	}
	if req.SaleDate == "" {
		req.SaleDate = time.Now().Format(time.RFC3339)
	}
	if req.ItemsCount <= 0 {
		req.ItemsCount = len(req.Items)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin POS sale tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	activeRegisterID, err := getActiveRegisterIDForSaleTx(ctx, tx, req.BusinessID, req.LocationID, req.CreatedBy)
	if err != nil {
		return nil, err
	}

	methods, err := loadPaymentMethodsForSaleTx(ctx, tx, req.BusinessID)
	if err != nil {
		return nil, err
	}
	if len(methods) == 0 {
		return nil, ErrSalePaymentMethodNotEnabled
	}

	var paidAmount float64
	var creditAmount float64
	var cashPaidAmount float64
	paymentRows := make([]PosSalePayment, 0, len(req.Payments))
	for _, payment := range req.Payments {
		code := normalizeSalePaymentCode(payment.PaymentMethodCode)
		method, ok := methods[code]
		if !ok {
			return nil, ErrSalePaymentMethodNotEnabled
		}
		if payment.Amount <= 0 {
			return nil, ErrInvalidSalePayment
		}
		if method.RequiresPhone && strings.TrimSpace(payment.Phone) == "" {
			return nil, ErrInvalidSalePayment
		}
		if method.IsCredit {
			creditAmount += payment.Amount
		} else {
			paidAmount += payment.Amount
			if method.Code == "cash" {
				cashPaidAmount += payment.Amount
			}
		}
		paymentRows = append(paymentRows, PosSalePayment{
			PaymentMethodID:   method.ID,
			PaymentMethodCode: method.Code,
			PaymentMethodName: method.Name,
			Amount:            payment.Amount,
			ReferenceNumber:   strings.TrimSpace(payment.ReferenceNumber),
			Phone:             strings.TrimSpace(payment.Phone),
			IsCredit:          method.IsCredit,
			Status:            "completed",
		})
	}

	if math.Abs((paidAmount+creditAmount)-req.GrandTotal) > 0.01 {
		return nil, ErrInvalidSalePayment
	}

	if creditAmount > 0 {
		if req.CustomerID == "" {
			return nil, ErrSaleCreditCustomerRequired
		}
		if err := validateAndApplyCustomerCreditTx(ctx, tx, req.BusinessID, req.CustomerID, creditAmount); err != nil {
			return nil, err
		}
	}

	referenceNumber, err := nextSaleReferenceNumberTx(ctx, tx, req.BusinessID)
	if err != nil {
		return nil, err
	}

	balanceDue := math.Max(0, req.GrandTotal-paidAmount)
	paymentStatus := "paid"
	if balanceDue > 0 && paidAmount > 0 {
		paymentStatus = "partially_paid"
	} else if balanceDue > 0 {
		paymentStatus = "unpaid"
	}

	var saleID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO sales (
			business_id,
			location_id,
			customer_id,
			reference_number,
			sale_date,
			customer_name,
			customer_phone,
			customer_email,
			status,
			subtotal,
			total_discount,
			total_tax,
			grand_total,
			paid_amount,
			credit_amount,
			balance_due,
			payment_status,
			items_count,
			total_quantity,
			notes,
			created_by
		)
		VALUES (
			$1::uuid,
			$2::uuid,
			NULLIF($3, '')::uuid,
			$4,
			$5::timestamptz,
			$6,
			$7,
			$8,
			'completed',
			$9,
			$10,
			$11,
			$12,
			$13,
			$14,
			$15,
			$16,
			$17,
			$18,
			$19,
			NULLIF($20, '')::uuid
		)
		RETURNING id::text
	`, req.BusinessID, req.LocationID, req.CustomerID, referenceNumber, req.SaleDate, req.CustomerName, req.CustomerPhone, req.CustomerEmail, req.Subtotal, req.TotalDiscount, req.TotalTax, req.GrandTotal, paidAmount, creditAmount, balanceDue, paymentStatus, req.ItemsCount, req.TotalQuantity, req.Notes, req.CreatedBy).Scan(&saleID); err != nil {
		return nil, fmt.Errorf("insert POS sale: %w", err)
	}

	for idx, item := range req.Items {
		productName, sku, unitName, err := loadSaleProductSnapshotTx(ctx, tx, req.BusinessID, item.ProductID)
		if err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO sale_items (
				sale_id,
				business_id,
				product_id,
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
				sort_order
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
				$9,
				$10,
				$11,
				$12,
				$13,
				$14,
				$15,
				$16
			)
		`, saleID, req.BusinessID, item.ProductID, productName, sku, unitName, item.Quantity, item.UnitCost, item.DiscountPercentage, item.DiscountAmount, item.TaxRate, item.TaxAmount, item.UnitPrice, item.LineTotal, item.BatchTrackingEnabled, idx); err != nil {
			return nil, fmt.Errorf("insert POS sale item: %w", err)
		}
	}

	for index, payment := range paymentRows {
		if err := tx.QueryRow(ctx, `
			INSERT INTO sale_payments (
				business_id,
				sale_id,
				payment_method_id,
				payment_method_code,
				payment_method_name,
				amount,
				reference_number,
				phone,
				is_credit,
				status,
				notes,
				created_by
			)
			VALUES (
				$1::uuid,
				$2::uuid,
				NULLIF($3, '')::uuid,
				$4,
				$5,
				$6,
				$7,
				$8,
				$9,
				'completed',
				$10,
				NULLIF($11, '')::uuid
			)
			RETURNING id::text
		`, req.BusinessID, saleID, payment.PaymentMethodID, payment.PaymentMethodCode, payment.PaymentMethodName, payment.Amount, payment.ReferenceNumber, payment.Phone, payment.IsCredit, "", req.CreatedBy).Scan(&paymentRows[index].ID); err != nil {
			return nil, fmt.Errorf("insert POS sale payment: %w", err)
		}
		if !payment.IsCredit {
			if _, err := tx.Exec(ctx, `
				INSERT INTO cash_register_transactions (
					business_id,
					cash_register_id,
					sale_id,
					transaction_type,
					payment_method,
					amount,
					reference_number,
					notes,
					created_by
				)
				VALUES ($1::uuid, $2::uuid, $3::uuid, 'sale_payment', $4, $5, $6, $7, NULLIF($8, '')::uuid)
			`, req.BusinessID, activeRegisterID, saleID, payment.PaymentMethodCode, payment.Amount, payment.ReferenceNumber, "POS sale payment", req.CreatedBy); err != nil {
				return nil, fmt.Errorf("insert POS sale register transaction: %w", err)
			}
		}
	}

	if paidAmount > 0 {
		if _, err := tx.Exec(ctx, `
			UPDATE cash_registers
			SET cash_sales_amount = cash_sales_amount + $1,
			    expected_closing_cash_amount = expected_closing_cash_amount + $2
			WHERE id = $3::uuid
			  AND business_id = $4::uuid
		`, paidAmount, cashPaidAmount, activeRegisterID, req.BusinessID); err != nil {
			return nil, fmt.Errorf("update cash register sale totals: %w", err)
		}
	}

	sale, err := GetSaleRepositoryTx(ctx, tx, req.BusinessID, saleID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit POS sale tx: %w", err)
	}

	return &PosSaleResult{Sale: *sale, Payments: paymentRows}, nil
}

func loadPaymentMethodsForSaleTx(ctx context.Context, tx saleInventoryTx, businessID string) (map[string]paymentMethodSnapshot, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text, code, name, is_credit, requires_phone
		FROM payment_methods
		WHERE business_id = $1::uuid
		  AND is_enabled = TRUE
	`, businessID)
	if err != nil {
		return nil, fmt.Errorf("load sale payment methods: %w", err)
	}
	defer rows.Close()

	methods := make(map[string]paymentMethodSnapshot)
	for rows.Next() {
		var method paymentMethodSnapshot
		if err := rows.Scan(&method.ID, &method.Code, &method.Name, &method.IsCredit, &method.RequiresPhone); err != nil {
			return nil, fmt.Errorf("scan sale payment method: %w", err)
		}
		method.Code = normalizeSalePaymentCode(method.Code)
		methods[method.Code] = method
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sale payment methods: %w", err)
	}
	return methods, nil
}

func getActiveRegisterIDForSaleTx(ctx context.Context, tx saleInventoryTx, businessID, locationID, userID string) (string, error) {
	var registerID string
	if err := tx.QueryRow(ctx, `
		SELECT id::text
		FROM cash_registers
		WHERE business_id = $1::uuid
		  AND business_location_id = $2::uuid
		  AND opened_by = $3::uuid
		  AND status = 'open'
		ORDER BY opened_at DESC
		LIMIT 1
	`, businessID, locationID, userID).Scan(&registerID); err != nil {
		if err == pgx.ErrNoRows {
			return "", ErrActiveCashRegisterRequired
		}
		return "", fmt.Errorf("load active register for POS sale: %w", err)
	}
	return registerID, nil
}

func validateAndApplyCustomerCreditTx(ctx context.Context, tx saleInventoryTx, businessID, customerID string, creditAmount float64) error {
	var creditLimit float64
	var totalSaleDue float64
	if err := tx.QueryRow(ctx, `
		SELECT credit_limit, total_sale_due
		FROM customers
		WHERE business_id = $1::uuid
		  AND id = $2::uuid
		  AND deleted = FALSE
		  AND is_active = TRUE
		FOR UPDATE
	`, businessID, customerID).Scan(&creditLimit, &totalSaleDue); err != nil {
		if err == pgx.ErrNoRows {
			return ErrSaleCreditCustomerRequired
		}
		return fmt.Errorf("load credit customer: %w", err)
	}
	if totalSaleDue+creditAmount > creditLimit+0.01 {
		return ErrSaleCreditLimitExceeded
	}
	if _, err := tx.Exec(ctx, `
		UPDATE customers
		SET total_sale_due = total_sale_due + $1
		WHERE business_id = $2::uuid
		  AND id = $3::uuid
	`, creditAmount, businessID, customerID); err != nil {
		return fmt.Errorf("update customer credit due: %w", err)
	}
	return nil
}

func nextSaleReferenceNumberTx(ctx context.Context, tx saleInventoryTx, businessID string) (string, error) {
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1 || ':sales'))`, businessID); err != nil {
		return "", fmt.Errorf("lock sale sequence: %w", err)
	}

	var count int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*) + 1
		FROM sales
		WHERE business_id = $1::uuid
	`, businessID).Scan(&count); err != nil {
		return "", fmt.Errorf("generate sale reference number: %w", err)
	}
	return fmt.Sprintf("SALE-%05d", count), nil
}

func normalizeSalePaymentCode(value string) string {
	normalized := strings.NewReplacer("-", "", "_", "", " ", "").Replace(strings.ToLower(strings.TrimSpace(value)))
	switch normalized {
	case "mobile", "mpesastk", "stkpush":
		return "mpesa"
	case "giftcard", "voucher":
		return "gift"
	case "customercredit":
		return "credit"
	default:
		return normalized
	}
}
