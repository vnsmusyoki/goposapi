package purchaseorder

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
	"pos/internal/auth"
	repopurchaseorder "pos/internal/repository/business/purchaseorder"
)

type purchaseOrderExportColumn struct {
	Key   string
	Label string
	Value func(order repopurchaseorder.PurchaseOrder) string
}

var purchaseOrderExportColumnOrder = []string{
	"orderDate",
	"referenceNumber",
	"location",
	"supplier",
	"status",
	"items",
	"deliveryStatus",
	"paymentStatus",
	"addedBy",
	"totalAmount",
}

var purchaseOrderExportColumns = map[string]purchaseOrderExportColumn{
	"orderDate": {
		Key:   "orderDate",
		Label: "Date",
		Value: func(order repopurchaseorder.PurchaseOrder) string { return order.OrderDate },
	},
	"referenceNumber": {
		Key:   "referenceNumber",
		Label: "Reference No.",
		Value: func(order repopurchaseorder.PurchaseOrder) string { return order.ReferenceNumber },
	},
	"location": {
		Key:   "location",
		Label: "Location",
		Value: func(order repopurchaseorder.PurchaseOrder) string { return order.LocationName },
	},
	"supplier": {
		Key:   "supplier",
		Label: "Supplier",
		Value: func(order repopurchaseorder.PurchaseOrder) string { return order.SupplierName },
	},
	"status": {
		Key:   "status",
		Label: "Order Status",
		Value: func(order repopurchaseorder.PurchaseOrder) string { return purchaseOrderStatusLabel(order.Status) },
	},
	"items": {
		Key:   "items",
		Label: "Items",
		Value: func(order repopurchaseorder.PurchaseOrder) string { return strconv.Itoa(order.ItemsCount) },
	},
	"deliveryStatus": {
		Key:   "deliveryStatus",
		Label: "Delivery Status",
		Value: func(order repopurchaseorder.PurchaseOrder) string {
			return purchaseOrderDeliveryStatusLabel(order.DeliveryStatus)
		},
	},
	"paymentStatus": {
		Key:   "paymentStatus",
		Label: "Payment Status",
		Value: func(order repopurchaseorder.PurchaseOrder) string {
			return purchaseOrderPaymentStatusLabel(order.PaymentStatus)
		},
	},
	"addedBy": {
		Key:   "addedBy",
		Label: "Added By",
		Value: func(order repopurchaseorder.PurchaseOrder) string {
			if order.CreatedBy != nil && strings.TrimSpace(order.CreatedBy.Name) != "" {
				return order.CreatedBy.Name
			}
			return "System"
		},
	},
	"totalAmount": {
		Key:   "totalAmount",
		Label: "Total Amount",
		Value: func(order repopurchaseorder.PurchaseOrder) string { return fmt.Sprintf("%.2f", order.GrandTotal) },
	},
}

func ExportPurchaseOrdersCSVRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		orders, err := loadFilteredPurchaseOrders(c, pool, authService)
		if err != nil {
			handlePurchaseOrderExportError(c, err, "Failed to export purchase orders")
			return
		}

		columns := resolvePurchaseOrderExportColumns(c.Query("columns"))
		filename := "purchase_orders_" + time.Now().Format("2006-01-02") + ".csv"
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", "attachment; filename="+filename)
		c.Header("Cache-Control", "no-store")
		c.Status(http.StatusOK)

		writer := csv.NewWriter(c.Writer)
		defer writer.Flush()

		headers := make([]string, 0, len(columns))
		for _, column := range columns {
			headers = append(headers, column.Label)
		}

		if err := writer.Write(headers); err != nil {
			log.Printf("export purchase orders csv: write header failed err=%v", err)
			return
		}

		for _, order := range orders {
			row := make([]string, 0, len(columns))
			for _, column := range columns {
				row = append(row, column.Value(order))
			}
			if err := writer.Write(row); err != nil {
				log.Printf("export purchase orders csv: write row failed order_id=%s err=%v", order.ID, err)
				return
			}
		}
	}
}

func ExportPurchaseOrdersPDFRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		orders, err := loadFilteredPurchaseOrders(c, pool, authService)
		if err != nil {
			handlePurchaseOrderExportError(c, err, "Failed to export purchase orders")
			return
		}

		columns := resolvePurchaseOrderExportColumns(c.Query("columns"))
		headers := make([]string, 0, len(columns))
		for _, column := range columns {
			headers = append(headers, column.Label)
		}

		rows := make([][]string, 0, len(orders))
		for _, order := range orders {
			row := make([]string, 0, len(columns))
			for _, column := range columns {
				row = append(row, column.Value(order))
			}
			rows = append(rows, row)
		}

		lines := buildPurchaseOrderTableLines(
			"Purchase Orders Export",
			headers,
			rows,
			orders,
		)

		pdfBytes, err := buildLandscapePdf("Purchase Orders Export", lines)
		if err != nil {
			log.Printf("export purchase orders pdf: build pdf failed err=%v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to export purchase orders"})
			return
		}

		filename := "purchase_orders_" + time.Now().Format("2006-01-02") + ".pdf"
		c.Header("Content-Type", "application/pdf")
		c.Header("Content-Disposition", "attachment; filename="+filename)
		c.Header("Cache-Control", "no-store")
		c.Data(http.StatusOK, "application/pdf", pdfBytes)
	}
}

func ExportPurchaseOrderPDFRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("export purchase order pdf handler: auth lookup failed err=%v", err)
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
			return
		}

		if !hasBusinessRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{"message": "Business access is required"})
			return
		}

		businessID := strings.TrimSpace(user.ActiveBusinessID)
		orderID := strings.TrimSpace(c.Param("id"))
		if businessID == "" || orderID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"business_id": "Active business could not be resolved.",
			}))
			return
		}

		order, err := repopurchaseorder.GetPurchaseOrderByIDRepository(pool, businessID, orderID)
		if err != nil {
			switch err {
			case repopurchaseorder.ErrBusinessNotResolved:
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
					"business_id": "Active business could not be resolved.",
				}))
			case repopurchaseorder.ErrPurchaseOrderNotFound:
				c.JSON(http.StatusNotFound, gin.H{"message": "Purchase order not found"})
			default:
				log.Printf("export purchase order pdf handler: repository failed business_id=%s order_id=%s err=%v", businessID, orderID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to export purchase order"})
			}
			return
		}

		lines := []string{
			"Purchase Order Summary",
			"",
			"Reference No.: " + order.ReferenceNumber,
			"Date: " + order.OrderDate,
			"Location: " + order.LocationName,
			"Supplier: " + order.SupplierName,
			"Status: " + purchaseOrderStatusLabel(order.Status),
			"Delivery Status: " + purchaseOrderDeliveryStatusLabel(order.DeliveryStatus),
			"Payment Status: " + purchaseOrderPaymentStatusLabel(order.PaymentStatus),
			"Items: " + strconv.Itoa(order.ItemsCount),
			"Total Amount: " + fmt.Sprintf("%.2f", order.GrandTotal),
			"Added By: " + purchaseOrderCreatedByName(*order),
			"Created At: " + order.CreatedAt,
			"Updated At: " + order.UpdatedAt,
		}

		pdfBytes, err := buildLandscapePdf("Purchase Order "+order.ReferenceNumber, lines)
		if err != nil {
			log.Printf("export purchase order pdf: build pdf failed order_id=%s err=%v", orderID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to export purchase order"})
			return
		}

		filename := sanitizeFilename("purchase_order_" + order.ReferenceNumber + ".pdf")
		c.Header("Content-Type", "application/pdf")
		c.Header("Content-Disposition", "attachment; filename="+filename)
		c.Header("Cache-Control", "no-store")
		c.Data(http.StatusOK, "application/pdf", pdfBytes)
	}
}

func SendPurchaseOrderNotificationRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("send purchase order notification handler: auth lookup failed err=%v", err)
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
			return
		}

		if !hasBusinessRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{"message": "Business access is required"})
			return
		}

		businessID := strings.TrimSpace(user.ActiveBusinessID)
		orderID := strings.TrimSpace(c.Param("id"))
		if businessID == "" || orderID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
				"business_id": "Active business could not be resolved.",
			}))
			return
		}

		order, err := repopurchaseorder.GetPurchaseOrderByIDRepository(pool, businessID, orderID)
		if err != nil {
			switch err {
			case repopurchaseorder.ErrBusinessNotResolved:
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
					"business_id": "Active business could not be resolved.",
				}))
			case repopurchaseorder.ErrPurchaseOrderNotFound:
				c.JSON(http.StatusNotFound, gin.H{"message": "Purchase order not found"})
			default:
				log.Printf("send purchase order notification handler: repository failed business_id=%s order_id=%s err=%v", businessID, orderID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to send notification"})
			}
			return
		}

		log.Printf("purchase order notification queued business_id=%s order_id=%s reference=%s", businessID, order.ID, order.ReferenceNumber)
		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("Notification queued for %s", order.ReferenceNumber),
		})
	}
}

func loadFilteredPurchaseOrders(c *gin.Context, pool *pgxpool.Pool, authService *auth.Service) ([]repopurchaseorder.PurchaseOrder, error) {
	user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
	if err != nil {
		http.SetCookie(c.Writer, authService.ClearSessionCookie())
		return nil, errPurchaseOrderSessionExpired
	}

	if !hasBusinessRole(user.Roles) {
		return nil, fmt.Errorf("business access is required")
	}

	businessID := strings.TrimSpace(user.ActiveBusinessID)
	if businessID == "" {
		return nil, repopurchaseorder.ErrBusinessNotResolved
	}

	orders, err := repopurchaseorder.ListPurchaseOrdersWithFiltersRepository(pool, businessID, repopurchaseorder.ListPurchaseOrdersFilters{
		LocationID:     c.Query("locationId"),
		SupplierID:     c.Query("supplierId"),
		Status:         c.Query("status"),
		DeliveryStatus: c.Query("deliveryStatus"),
		PaymentStatus:  c.Query("paymentStatus"),
		SearchQuery:    firstNonEmpty(c.Query("searchQuery"), c.Query("search")),
		DateFrom:       firstNonEmpty(c.Query("from"), c.Query("dateFrom")),
		DateTo:         firstNonEmpty(c.Query("to"), c.Query("dateTo")),
	})
	if err != nil {
		return nil, err
	}

	return orders, nil
}

func handlePurchaseOrderExportError(c *gin.Context, err error, message string) {
	if err == nil {
		return
	}

	if errors.Is(err, errPurchaseOrderSessionExpired) {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
		return
	}

	if strings.Contains(err.Error(), "business access is required") {
		c.JSON(http.StatusForbidden, gin.H{"message": "Business access is required"})
		return
	}

	switch err {
	case repopurchaseorder.ErrBusinessNotResolved:
		c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
			"business_id": "Active business could not be resolved.",
		}))
	default:
		log.Printf("purchase order export handler failed err=%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": message})
	}
}

func resolvePurchaseOrderExportColumns(raw string) []purchaseOrderExportColumn {
	keys := make([]string, 0)
	for _, part := range strings.Split(raw, ",") {
		key := strings.TrimSpace(part)
		if key != "" {
			keys = append(keys, key)
		}
	}
	if len(keys) == 0 {
		keys = append(keys, purchaseOrderExportColumnOrder...)
	}

	columns := make([]purchaseOrderExportColumn, 0, len(keys))
	for _, key := range keys {
		column, ok := purchaseOrderExportColumns[key]
		if !ok {
			continue
		}
		columns = append(columns, column)
	}
	if len(columns) == 0 {
		for _, key := range purchaseOrderExportColumnOrder {
			if column, ok := purchaseOrderExportColumns[key]; ok {
				columns = append(columns, column)
			}
		}
	}
	return columns
}

func buildPurchaseOrderTableLines(title string, headers []string, rows [][]string, orders []repopurchaseorder.PurchaseOrder) []string {
	lines := []string{
		title,
		"",
		fmt.Sprintf("Generated: %s", time.Now().Format(time.RFC3339)),
		fmt.Sprintf("Orders: %d", len(orders)),
		"",
	}

	if len(headers) == 0 {
		return lines
	}

	widths := deriveColumnWidths(headers, rows, 150)
	lines = append(lines, formatTableRow(headers, widths))
	lines = append(lines, strings.Repeat("-", maxRuneCount(formatTableRow(headers, widths))))
	for _, row := range rows {
		lines = append(lines, formatTableRow(row, widths))
	}

	return lines
}

func deriveColumnWidths(headers []string, rows [][]string, maxWidth int) []int {
	widths := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = max(8, runeCount(header)+1)
	}

	for _, row := range rows {
		for i, value := range row {
			if i >= len(widths) {
				break
			}
			if width := runeCount(value) + 1; width > widths[i] {
				widths[i] = width
			}
		}
	}

	for i := range widths {
		if widths[i] > 20 {
			widths[i] = 20
		}
	}

	for totalWidth(widths) > maxWidth {
		index := widestColumn(widths)
		if index < 0 || widths[index] <= 8 {
			break
		}
		widths[index]--
	}

	return widths
}

func widestColumn(widths []int) int {
	index := -1
	maxWidthValue := 0
	for i, width := range widths {
		if width > maxWidthValue {
			index = i
			maxWidthValue = width
		}
	}
	return index
}

func totalWidth(widths []int) int {
	if len(widths) == 0 {
		return 0
	}

	total := 0
	for _, width := range widths {
		total += width
	}
	total += (len(widths) - 1) * 3
	return total
}

func formatTableRow(values []string, widths []int) string {
	parts := make([]string, 0, len(widths))
	for i, width := range widths {
		value := ""
		if i < len(values) {
			value = values[i]
		}
		parts = append(parts, padRight(truncate(value, width), width))
	}
	return strings.TrimRight(strings.Join(parts, " | "), " ")
}

func buildLandscapePdf(title string, lines []string) ([]byte, error) {
	pages := chunkLines(lines, 42)
	if len(pages) == 0 {
		pages = [][]string{{title}}
	}

	var objects [][]byte
	fontObjectID := 1
	objects = append(objects, []byte("<< /Type /Font /Subtype /Type1 /BaseFont /Courier >>"))

	contentObjectIDs := make([]int, 0, len(pages))
	pageObjectIDs := make([]int, 0, len(pages))

	for _, pageLines := range pages {
		var content bytes.Buffer
		content.WriteString("BT\n/F1 9 Tf\n12 TL\n40 540 Td\n")
		for idx, line := range pageLines {
			if idx == 0 {
				content.WriteString("(")
				content.WriteString(escapePDFText(line))
				content.WriteString(") Tj\n")
				continue
			}
			content.WriteString("T*\n(")
			content.WriteString(escapePDFText(line))
			content.WriteString(") Tj\n")
		}
		content.WriteString("ET\n")

		contentObjectIDs = append(contentObjectIDs, len(objects)+1)
		objects = append(objects, []byte(fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", content.Len(), content.String())))

		pageObjectIDs = append(pageObjectIDs, len(objects)+1)
		objects = append(objects, []byte(fmt.Sprintf("<< /Type /Page /Parent 0 0 R /MediaBox [0 0 842 595] /Resources << /Font << /F1 %d 0 R >> >> /Contents %d 0 R >>", fontObjectID, contentObjectIDs[len(contentObjectIDs)-1])))
	}

	pagesObjectID := len(objects) + 1
	kids := make([]string, 0, len(pageObjectIDs))
	for _, pageID := range pageObjectIDs {
		kids = append(kids, fmt.Sprintf("%d 0 R", pageID))
	}
	objects = append(objects, []byte(fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), len(pageObjectIDs))))

	catalogObjectID := len(objects) + 1
	objects = append(objects, []byte(fmt.Sprintf("<< /Type /Catalog /Pages %d 0 R >>", pagesObjectID)))

	// Update page parent references now that the /Pages object id is known.
	for i := range pageObjectIDs {
		pageIndex := pageObjectIDs[i] - 1
		objects[pageIndex] = []byte(fmt.Sprintf("<< /Type /Page /Parent %d 0 R /MediaBox [0 0 842 595] /Resources << /Font << /F1 %d 0 R >> >> /Contents %d 0 R >>", pagesObjectID, fontObjectID, contentObjectIDs[i]))
	}

	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n")

	offsets := make([]int, 0, len(objects)+1)
	offsets = append(offsets, 0)

	for i, object := range objects {
		offsets = append(offsets, out.Len())
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", i+1, object)
	}

	xrefOffset := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n", len(objects)+1)
	out.WriteString("0000000000 65535 f \n")
	for i := 1; i < len(offsets); i++ {
		fmt.Fprintf(&out, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root %d 0 R >>\nstartxref\n%d\n%%%%EOF", len(objects)+1, catalogObjectID, xrefOffset)

	return out.Bytes(), nil
}

func chunkLines(lines []string, size int) [][]string {
	if size <= 0 {
		size = 40
	}

	pages := make([][]string, 0)
	for len(lines) > 0 {
		if len(lines) <= size {
			pages = append(pages, lines)
			break
		}
		pages = append(pages, lines[:size])
		lines = lines[size:]
	}
	return pages
}

func escapePDFText(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "(", `\(`)
	value = strings.ReplaceAll(value, ")", `\)`)
	return value
}

func truncate(value string, width int) string {
	if width <= 0 {
		return ""
	}

	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	if width <= 3 {
		return string(runes[:width])
	}
	return string(runes[:width-3]) + "..."
}

func padRight(value string, width int) string {
	runes := []rune(value)
	if len(runes) >= width {
		return value
	}
	return value + strings.Repeat(" ", width-len(runes))
}

func runeCount(value string) int {
	return len([]rune(value))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func maxRuneCount(value string) int {
	return len([]rune(value))
}

func purchaseOrderStatusLabel(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "draft":
		return "Draft"
	case "pending":
		return "Pending"
	case "approved":
		return "Approved"
	case "ordered":
		return "Ordered"
	case "received":
		return "Received"
	case "partially_received":
		return "Partially Received"
	case "cancelled":
		return "Cancelled"
	case "completed":
		return "Completed"
	default:
		return strings.TrimSpace(status)
	}
}

func purchaseOrderDeliveryStatusLabel(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending_delivery":
		return "Pending Delivery"
	case "in_transit":
		return "In Transit"
	case "delivered":
		return "Delivered"
	default:
		return strings.TrimSpace(status)
	}
}

func purchaseOrderPaymentStatusLabel(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "unpaid":
		return "Unpaid"
	case "partially_paid":
		return "Partially Paid"
	case "paid":
		return "Paid"
	default:
		return strings.TrimSpace(status)
	}
}

func purchaseOrderCreatedByName(order repopurchaseorder.PurchaseOrder) string {
	if order.CreatedBy != nil && strings.TrimSpace(order.CreatedBy.Name) != "" {
		return order.CreatedBy.Name
	}
	return "System"
}

func sanitizeFilename(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, " ", "_")
	value = strings.ReplaceAll(value, "/", "_")
	value = strings.ReplaceAll(value, "\\", "_")
	value = strings.Trim(value, "._")
	if value == "" {
		return "purchase_order.pdf"
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

var errPurchaseOrderSessionExpired = errors.New("purchase order session expired")
