package purchaseorder

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4/pgxpool"
)

type CreatePurchaseOrderApprovalInput struct {
	BusinessID       string
	PurchaseOrderID  string
	ApprovalStatus   string
	ReminderChannels []string
	ReminderMessage  string
	Note             string
	RequestedBy      string
	ActionedBy       string
	ReminderSentAt   *time.Time
	ActionedAt       *time.Time
}

func CreatePurchaseOrderApprovalRepository(pool *pgxpool.Pool, req CreatePurchaseOrderApprovalInput) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return insertPurchaseOrderApproval(ctx, pool, req)
}

func CreatePurchaseOrderApprovalTx(ctx context.Context, tx interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}, req CreatePurchaseOrderApprovalInput) error {
	return insertPurchaseOrderApproval(ctx, tx, req)
}

func insertPurchaseOrderApproval(ctx context.Context, querier interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}, req CreatePurchaseOrderApprovalInput) error {
	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.PurchaseOrderID = strings.TrimSpace(req.PurchaseOrderID)
	req.ApprovalStatus = strings.ToLower(strings.TrimSpace(req.ApprovalStatus))
	req.ReminderMessage = strings.TrimSpace(req.ReminderMessage)
	req.Note = strings.TrimSpace(req.Note)
	req.RequestedBy = strings.TrimSpace(req.RequestedBy)
	req.ActionedBy = strings.TrimSpace(req.ActionedBy)
	req.ReminderChannels = normalizeReminderChannels(req.ReminderChannels)

	if req.BusinessID == "" || req.PurchaseOrderID == "" || req.ApprovalStatus == "" {
		return ErrBusinessNotResolved
	}

	if _, err := querier.Exec(ctx, `
		INSERT INTO purchase_order_approvals (
			business_id,
			purchase_order_id,
			approval_status,
			reminder_channels,
			reminder_message,
			note,
			requested_by,
			actioned_by,
			requested_at,
			actioned_at,
			reminder_sent_at
		)
		VALUES (
			$1::uuid,
			$2::uuid,
			$3,
			$4,
			$5,
			$6,
			NULLIF($7, '')::uuid,
			NULLIF($8, '')::uuid,
			CURRENT_TIMESTAMP,
			$9,
			$10
		)
	`, req.BusinessID, req.PurchaseOrderID, req.ApprovalStatus, req.ReminderChannels, req.ReminderMessage, req.Note, req.RequestedBy, req.ActionedBy, req.ActionedAt, req.ReminderSentAt); err != nil {
		return fmt.Errorf("insert purchase order approval: %w", err)
	}

	return nil
}

func normalizeReminderChannels(channels []string) []string {
	seen := make(map[string]struct{}, len(channels))
	normalized := make([]string, 0, len(channels))

	for _, channel := range channels {
		channel = strings.ToLower(strings.TrimSpace(channel))
		if channel == "" {
			continue
		}
		switch channel {
		case "notification", "sms", "whatsapp":
		default:
			continue
		}
		if _, exists := seen[channel]; exists {
			continue
		}
		seen[channel] = struct{}{}
		normalized = append(normalized, channel)
	}

	if len(normalized) == 0 {
		return []string{"notification"}
	}

	return normalized
}
