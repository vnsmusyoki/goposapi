package sales

import (
	"fmt"
	"strings"
)

func buildSalesOrderActivityNote(action, referenceNumber, actorName, previousStatus, nextStatus string, reserveOrderItems bool, saleCreated bool) string {
	referenceNumber = strings.TrimSpace(referenceNumber)
	actorName = strings.TrimSpace(actorName)
	previousStatus = strings.TrimSpace(previousStatus)
	nextStatus = strings.TrimSpace(nextStatus)

	switch strings.ToLower(strings.TrimSpace(action)) {
	case "created":
		if reserveOrderItems {
			return fmt.Sprintf("%s created by %s and stock was reserved for the order.", referenceNumber, actorName)
		}
		return fmt.Sprintf("%s created by %s.", referenceNumber, actorName)
	case "status_changed":
		if previousStatus == "" || nextStatus == "" {
			return fmt.Sprintf("%s status updated by %s.", referenceNumber, actorName)
		}
		return fmt.Sprintf("%s status changed from %s to %s by %s.", referenceNumber, humanizeSalesOrderState(previousStatus), humanizeSalesOrderState(nextStatus), actorName)
	case "updated":
		return fmt.Sprintf("%s updated by %s.", referenceNumber, actorName)
	case "finalized":
		if saleCreated {
			return fmt.Sprintf("%s finalized by %s and converted into a sale.", referenceNumber, actorName)
		}
		return fmt.Sprintf("%s finalized by %s.", referenceNumber, actorName)
	case "deleted":
		return fmt.Sprintf("%s deleted by %s.", referenceNumber, actorName)
	default:
		return fmt.Sprintf("%s activity recorded by %s.", referenceNumber, actorName)
	}
}

func humanizeSalesOrderState(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}

	switch strings.ToLower(value) {
	case "draft":
		return "Draft"
	case "pending_approval":
		return "Pending Approval"
	case "approved":
		return "Approved"
	case "processing":
		return "Processing"
	case "ready_for_shipment":
		return "Ready for Processing"
	case "completed":
		return "Completed"
	default:
		value = strings.ReplaceAll(value, "_", " ")
		parts := strings.Fields(value)
		for i, part := range parts {
			if len(part) == 0 {
				continue
			}
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
		return strings.Join(parts, " ")
	}
}
