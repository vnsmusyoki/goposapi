package purchaseorder

import (
	"context"
	"fmt"
	"net/mail"
	"strings"

	"github.com/jackc/pgconn"
)

type CreatePurchaseOrderNotificationInput struct {
	BusinessID              string
	PurchaseOrderID         string
	PurchaseOrderStatusCode string
	Channels                []string
	Receivers               []string
	EmailSubject            string
	EmailCc                 []string
	EmailBcc                []string
	Message                 string
	Note                    string
	CreatedBy               string
}

func CreatePurchaseOrderNotificationTx(ctx context.Context, tx interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}, req CreatePurchaseOrderNotificationInput) error {
	return insertPurchaseOrderNotification(ctx, tx, req)
}

func insertPurchaseOrderNotification(ctx context.Context, querier interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}, req CreatePurchaseOrderNotificationInput) error {
	req.BusinessID = strings.TrimSpace(req.BusinessID)
	req.PurchaseOrderID = strings.TrimSpace(req.PurchaseOrderID)
	req.PurchaseOrderStatusCode = strings.TrimSpace(req.PurchaseOrderStatusCode)
	req.EmailSubject = strings.TrimSpace(req.EmailSubject)
	req.Message = strings.TrimSpace(req.Message)
	req.Note = strings.TrimSpace(req.Note)
	req.CreatedBy = strings.TrimSpace(req.CreatedBy)
	req.Channels = normalizeApprovalReminderChannels(req.Channels)
	req.Receivers = normalizeNotificationReceivers(req.Channels, req.Receivers)
	req.EmailCc = normalizeEmailAddresses(req.EmailCc)
	req.EmailBcc = normalizeEmailAddresses(req.EmailBcc)

	if req.BusinessID == "" || req.PurchaseOrderID == "" || req.PurchaseOrderStatusCode == "" {
		return ErrBusinessNotResolved
	}

	if _, err := querier.Exec(ctx, `
		INSERT INTO notifications (
			business_id,
			purchase_order_id,
			purchase_order_status_code,
			channels,
			receivers,
			email_subject,
			email_cc,
			email_bcc,
			message,
			note,
			created_by,
			created_at,
			updated_at
		)
		VALUES (
			$1::uuid,
			$2::uuid,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9,
			$10,
			NULLIF($11, '')::uuid,
			CURRENT_TIMESTAMP,
			CURRENT_TIMESTAMP
		)
	`, req.BusinessID, req.PurchaseOrderID, req.PurchaseOrderStatusCode, req.Channels, req.Receivers, req.EmailSubject, req.EmailCc, req.EmailBcc, req.Message, req.Note, req.CreatedBy); err != nil {
		return fmt.Errorf("insert purchase order notification: %w", err)
	}

	return nil
}

func normalizePhoneNumbers(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	normalized := make([]string, 0, len(values))

	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if !strings.HasPrefix(value, "0") || len(value) != 10 {
			continue
		}
		valid := true
		for _, ch := range value {
			if ch < '0' || ch > '9' {
				valid = false
				break
			}
		}
		if !valid {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}

	return normalized
}

func normalizeApprovalReminderChannels(channels []string) []string {
	seen := make(map[string]struct{}, len(channels))
	normalized := make([]string, 0, len(channels))

	for _, channel := range channels {
		channel = strings.ToLower(strings.TrimSpace(channel))
		if channel == "" {
			continue
		}
		switch channel {
		case "notification", "email", "sms", "whatsapp":
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

func normalizeEmailAddresses(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	normalized := make([]string, 0, len(values))

	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, err := mail.ParseAddress(value); err != nil {
			continue
		}
		key := strings.ToLower(value)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, value)
	}

	return normalized
}

func normalizeNotificationReceivers(channels []string, values []string) []string {
	for _, channel := range channels {
		if strings.EqualFold(channel, "email") {
			return normalizeEmailAddresses(values)
		}
	}
	return normalizePhoneNumbers(values)
}
