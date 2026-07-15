package purchaseorder

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

type createPurchaseReturnPayload struct {
	LocationID *string                    `json:"locationId"`
	SupplierID *string                    `json:"supplierId"`
	Reason     *string                    `json:"reason"`
	Note       *string                    `json:"note"`
	Items      []createPurchaseReturnLine `json:"items"`
}

type createPurchaseReturnLine struct {
	ProductID *string  `json:"productId"`
	BatchID   *string  `json:"batchId"`
	Quantity  *float64 `json:"quantity"`
	UnitPrice *float64 `json:"unitPrice"`
}

type purchaseReturnResponse struct {
	PurchaseReturn repopurchaseorder.PurchaseReturn `json:"purchaseReturn"`
	Message        string                           `json:"message,omitempty"`
}

type purchaseReturnsResponse struct {
	PurchaseOrderReturns []repopurchaseorder.PurchaseReturn `json:"purchaseOrderReturns"`
	Returns              []repopurchaseorder.PurchaseReturn `json:"returns"`
	Data                 []repopurchaseorder.PurchaseReturn `json:"data"`
	Message              string                             `json:"message"`
}

type purchaseReturnDetailResponse struct {
	PurchaseReturn repopurchaseorder.PurchaseReturn       `json:"purchaseReturn"`
	Items          []repopurchaseorder.PurchaseReturnItem `json:"items"`
	Activities     []repopurchaseorder.PurchaseReturnLog  `json:"activities"`
	Message        string                                 `json:"message,omitempty"`
}

type purchaseReturnExportColumn struct {
	Key   string
	Label string
	Value func(entry repopurchaseorder.PurchaseReturn) string
}

var purchaseReturnExportColumnOrder = []string{
	"returnDate",
	"referenceNumber",
	"parentPurchase",
	"location",
	"supplier",
	"status",
	"paymentStatus",
	"grandTotal",
	"paymentDue",
}

var purchaseReturnExportColumns = map[string]purchaseReturnExportColumn{
	"returnDate": {
		Key:   "returnDate",
		Label: "Date",
		Value: func(entry repopurchaseorder.PurchaseReturn) string { return entry.ReturnDate },
	},
	"referenceNumber": {
		Key:   "referenceNumber",
		Label: "Reference No.",
		Value: func(entry repopurchaseorder.PurchaseReturn) string { return entry.ReferenceNumber },
	},
	"parentPurchase": {
		Key:   "parentPurchase",
		Label: "Parent Purchase",
		Value: func(entry repopurchaseorder.PurchaseReturn) string { return entry.ParentPurchaseReference },
	},
	"location": {
		Key:   "location",
		Label: "Location",
		Value: func(entry repopurchaseorder.PurchaseReturn) string { return entry.LocationName },
	},
	"supplier": {
		Key:   "supplier",
		Label: "Supplier",
		Value: func(entry repopurchaseorder.PurchaseReturn) string { return entry.SupplierName },
	},
	"status": {
		Key:   "status",
		Label: "Return Status",
		Value: func(entry repopurchaseorder.PurchaseReturn) string { return purchaseReturnStatusLabel(entry.Status) },
	},
	"paymentStatus": {
		Key:   "paymentStatus",
		Label: "Payment Status",
		Value: func(entry repopurchaseorder.PurchaseReturn) string {
			return purchaseReturnPaymentStatusLabel(entry.PaymentStatus)
		},
	},
	"grandTotal": {
		Key:   "grandTotal",
		Label: "Grand Total",
		Value: func(entry repopurchaseorder.PurchaseReturn) string { return fmt.Sprintf("%.2f", entry.GrandTotal) },
	},
	"paymentDue": {
		Key:   "paymentDue",
		Label: "Payment Due",
		Value: func(entry repopurchaseorder.PurchaseReturn) string { return fmt.Sprintf("%.2f", entry.PaymentDue) },
	},
}

func CreatePurchaseReturnRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("create purchase return handler: auth lookup failed err=%v", err)
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
			return
		}

		if !hasBusinessRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{"message": "Business access is required"})
			return
		}

		businessID := strings.TrimSpace(user.ActiveBusinessID)
		if businessID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"business_id": "Active business could not be resolved."}))
			return
		}

		var payload createPurchaseReturnPayload
		if err := c.ShouldBindJSON(&payload); err != nil {
			log.Printf("create purchase return handler: invalid json err=%v", err)
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body must be valid JSON."}))
			return
		}

		errs := purchaseReturnPayloadErrors(&payload)
		if len(errs) > 0 {
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

		items := make([]repopurchaseorder.CreatePurchaseReturnItemInput, 0, len(payload.Items))
		for _, item := range payload.Items {
			items = append(items, repopurchaseorder.CreatePurchaseReturnItemInput{
				ProductID: strings.TrimSpace(*item.ProductID),
				BatchKey:  strings.TrimSpace(*item.BatchID),
				Quantity:  *item.Quantity,
				UnitPrice: *item.UnitPrice,
			})
		}

		createdReturn, err := repopurchaseorder.CreatePurchaseReturnRepository(pool, repopurchaseorder.CreatePurchaseReturnInput{
			BusinessID:    businessID,
			LocationID:    strings.TrimSpace(*payload.LocationID),
			SupplierID:    normalizeOptionalString(payload.SupplierID),
			ReturnReason:  normalizeOptionalString(payload.Reason),
			Notes:         normalizeOptionalString(payload.Note),
			Status:        "returned",
			PaymentStatus: "unpaid",
			CreatedBy:     user.ID,
			Items:         items,
		})
		if err != nil {
			log.Printf("create purchase return handler: repository failed business_id=%s err=%v", businessID, err)
			switch {
			case errors.Is(err, repopurchaseorder.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"business_id": "Active business could not be resolved."}))
			case errors.Is(err, repopurchaseorder.ErrInvalidPurchaseReturnInput):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Please provide valid return details."}))
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			}
			return
		}

		c.JSON(http.StatusCreated, purchaseReturnResponse{
			PurchaseReturn: *createdReturn,
			Message:        "Purchase return created successfully",
		})
	}
}

func purchaseReturnPayloadErrors(payload *createPurchaseReturnPayload) map[string]string {
	errs := map[string]string{}
	if payload == nil {
		errs["form"] = "Request body must be valid JSON."
		return errs
	}

	if payload.LocationID == nil || strings.TrimSpace(*payload.LocationID) == "" {
		errs["locationId"] = "Business location is required."
	}
	if len(payload.Items) == 0 {
		errs["items"] = "Add at least one product to return."
	}
	for idx, item := range payload.Items {
		if item.ProductID == nil || strings.TrimSpace(*item.ProductID) == "" {
			errs[fmt.Sprintf("items[%d].productId", idx)] = "Product is required."
		}
		if item.BatchID == nil || strings.TrimSpace(*item.BatchID) == "" {
			errs[fmt.Sprintf("items[%d].batchId", idx)] = "Stock group is required."
		}
		if item.Quantity == nil || *item.Quantity <= 0 {
			errs[fmt.Sprintf("items[%d].quantity", idx)] = "Quantity must be greater than zero."
		}
		if item.UnitPrice == nil || *item.UnitPrice < 0 {
			errs[fmt.Sprintf("items[%d].unitPrice", idx)] = "Unit price must be zero or greater."
		}
	}

	return errs
}

func UpdatePurchaseReturnRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := c.GetRawData()
		if err != nil && !errors.Is(err, io.EOF) {
			log.Printf("update purchase return handler: read body failed err=%v", err)
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Unable to read request body."}))
			return
		}

		if len(strings.TrimSpace(string(body))) == 0 {
			c.JSON(http.StatusBadRequest, validationFailed(purchaseReturnPayloadErrors(nil)))
			return
		}

		var payload createPurchaseReturnPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			log.Printf("update purchase return handler: invalid json err=%v body=%s", err, string(body))
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body must be valid JSON."}))
			return
		}

		if errs := purchaseReturnPayloadErrors(&payload); len(errs) > 0 {
			log.Printf("update purchase return handler: validation failed errors=%v", errs)
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("update purchase return handler: auth lookup failed err=%v", err)
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
			return
		}

		if !hasBusinessRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{"message": "Business access is required"})
			return
		}

		businessID := strings.TrimSpace(user.ActiveBusinessID)
		returnID := strings.TrimSpace(c.Param("id"))
		if businessID == "" || returnID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Purchase return id is required."}))
			return
		}

		_, err = repopurchaseorder.GetPurchaseReturnByIDRepository(pool, businessID, returnID)
		if err != nil {
			switch err {
			case repopurchaseorder.ErrBusinessNotResolved:
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"business_id": "Active business could not be resolved."}))
			case repopurchaseorder.ErrPurchaseReturnNotFound:
				c.JSON(http.StatusNotFound, gin.H{"message": "Purchase return not found"})
			default:
				log.Printf("update purchase return handler: lookup failed business_id=%s return_id=%s err=%v", businessID, returnID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load purchase return"})
			}
			return
		}

		items := make([]repopurchaseorder.CreatePurchaseReturnItemInput, 0, len(payload.Items))
		for _, item := range payload.Items {
			items = append(items, repopurchaseorder.CreatePurchaseReturnItemInput{
				ProductID: strings.TrimSpace(*item.ProductID),
				BatchKey:  strings.TrimSpace(*item.BatchID),
				Quantity:  *item.Quantity,
				UnitPrice: *item.UnitPrice,
			})
		}

		updatedReturn, err := repopurchaseorder.UpdatePurchaseReturnRepository(pool, repopurchaseorder.UpdatePurchaseReturnInput{
			BusinessID:       businessID,
			PurchaseReturnID: returnID,
			LocationID:       strings.TrimSpace(*payload.LocationID),
			SupplierID:       normalizeOptionalString(payload.SupplierID),
			ReturnReason:     normalizeOptionalString(payload.Reason),
			Notes:            normalizeOptionalString(payload.Note),
			UpdatedBy:        user.ID,
			Items:            items,
		})
		if err != nil {
			log.Printf("update purchase return handler: repository failed business_id=%s return_id=%s err=%v", businessID, returnID, err)
			switch {
			case errors.Is(err, repopurchaseorder.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"business_id": "Active business could not be resolved."}))
			case errors.Is(err, repopurchaseorder.ErrInvalidPurchaseReturnInput):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Please provide valid return details."}))
			case errors.Is(err, repopurchaseorder.ErrPurchaseReturnNotFound):
				c.JSON(http.StatusNotFound, gin.H{"message": "Purchase return not found"})
			default:
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update purchase return"})
			}
			return
		}

		c.JSON(http.StatusOK, purchaseReturnResponse{
			PurchaseReturn: *updatedReturn,
			Message:        "Purchase return updated successfully",
		})
	}
}

func ListPurchaseReturnsRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("list purchase returns handler: auth lookup failed err=%v", err)
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
			return
		}

		if !hasBusinessRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{"message": "Business access is required"})
			return
		}

		businessID := strings.TrimSpace(user.ActiveBusinessID)
		if businessID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"business_id": "Active business could not be resolved."}))
			return
		}

		returns, err := repopurchaseorder.ListPurchaseReturnsRepository(pool, businessID, purchaseReturnFiltersFromContext(c))
		if err != nil {
			switch err {
			case repopurchaseorder.ErrBusinessNotResolved:
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"business_id": "Active business could not be resolved."}))
			default:
				log.Printf("list purchase returns handler: repository failed business_id=%s err=%v", businessID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load purchase returns"})
			}
			return
		}

		c.JSON(http.StatusOK, purchaseReturnsResponse{
			PurchaseOrderReturns: returns,
			Returns:              returns,
			Data:                 returns,
			Message:              "Purchase returns loaded successfully",
		})
	}
}

func GetPurchaseReturnRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("get purchase return handler: auth lookup failed err=%v", err)
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
			return
		}

		if !hasBusinessRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{"message": "Business access is required"})
			return
		}

		businessID := strings.TrimSpace(user.ActiveBusinessID)
		returnID := strings.TrimSpace(c.Param("id"))
		if businessID == "" || returnID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Purchase return id is required."}))
			return
		}

		entry, err := repopurchaseorder.GetPurchaseReturnByIDRepository(pool, businessID, returnID)
		if err != nil {
			switch err {
			case repopurchaseorder.ErrBusinessNotResolved:
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"business_id": "Active business could not be resolved."}))
			case repopurchaseorder.ErrPurchaseReturnNotFound:
				c.JSON(http.StatusNotFound, gin.H{"message": "Purchase return not found"})
			default:
				log.Printf("get purchase return handler: repository failed business_id=%s return_id=%s err=%v", businessID, returnID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load purchase return"})
			}
			return
		}

		c.JSON(http.StatusOK, purchaseReturnDetailResponse{
			PurchaseReturn: *entry,
			Items:          entry.Items,
			Activities:     entry.Activities,
			Message:        "Purchase return loaded successfully",
		})
	}
}

func DeletePurchaseReturnRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("delete purchase return handler: auth lookup failed err=%v", err)
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
			return
		}

		if !hasBusinessRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{"message": "Business access is required"})
			return
		}

		businessID := strings.TrimSpace(user.ActiveBusinessID)
		returnID := strings.TrimSpace(c.Param("id"))
		if businessID == "" || returnID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Purchase return id is required."}))
			return
		}

		if err := repopurchaseorder.DeletePurchaseReturnRepository(pool, businessID, returnID, user.ID); err != nil {
			switch err {
			case repopurchaseorder.ErrBusinessNotResolved:
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"business_id": "Active business could not be resolved."}))
			case repopurchaseorder.ErrPurchaseReturnNotFound:
				c.JSON(http.StatusNotFound, gin.H{"message": "Purchase return not found"})
			default:
				log.Printf("delete purchase return handler: repository failed business_id=%s return_id=%s err=%v", businessID, returnID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to delete purchase return"})
			}
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Purchase return deleted successfully"})
	}
}

func ExportPurchaseReturnsCSVRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		returns, err := loadFilteredPurchaseReturns(c, pool, authService)
		if err != nil {
			handlePurchaseReturnExportError(c, err, "Failed to export purchase returns")
			return
		}

		columns := resolvePurchaseReturnExportColumns(c.Query("columns"))
		filename := "purchase_returns_" + time.Now().Format("2006-01-02") + ".csv"
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
			log.Printf("export purchase returns csv: write header failed err=%v", err)
			return
		}

		for _, entry := range returns {
			row := make([]string, 0, len(columns))
			for _, column := range columns {
				row = append(row, column.Value(entry))
			}
			if err := writer.Write(row); err != nil {
				log.Printf("export purchase returns csv: write row failed return_id=%s err=%v", entry.ID, err)
				return
			}
		}
	}
}

func ExportPurchaseReturnsPDFRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		returns, err := loadFilteredPurchaseReturns(c, pool, authService)
		if err != nil {
			handlePurchaseReturnExportError(c, err, "Failed to export purchase returns")
			return
		}

		columns := resolvePurchaseReturnExportColumns(c.Query("columns"))
		headers := make([]string, 0, len(columns))
		for _, column := range columns {
			headers = append(headers, column.Label)
		}

		rows := make([][]string, 0, len(returns))
		for _, entry := range returns {
			row := make([]string, 0, len(columns))
			for _, column := range columns {
				row = append(row, column.Value(entry))
			}
			rows = append(rows, row)
		}

		lines := buildPurchaseReturnTableLines("Purchase Returns Export", headers, rows, len(returns))
		pdfBytes, err := buildLandscapePdf("Purchase Returns Export", lines)
		if err != nil {
			log.Printf("export purchase returns pdf: build pdf failed err=%v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to export purchase returns"})
			return
		}

		filename := "purchase_returns_" + time.Now().Format("2006-01-02") + ".pdf"
		c.Header("Content-Type", "application/pdf")
		c.Header("Content-Disposition", "attachment; filename="+filename)
		c.Header("Cache-Control", "no-store")
		c.Data(http.StatusOK, "application/pdf", pdfBytes)
	}
}

func ExportPurchaseReturnPDFRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		entry, items, err := loadPurchaseReturnForPDF(c, pool, authService)
		if err != nil {
			handlePurchaseReturnExportError(c, err, "Failed to export purchase return")
			return
		}

		lines := []string{
			"Purchase Return",
			"",
			fmt.Sprintf("Reference No: %s", entry.ReferenceNumber),
			fmt.Sprintf("Parent Purchase: %s", entry.ParentPurchaseReference),
			fmt.Sprintf("Location: %s", entry.LocationName),
			fmt.Sprintf("Supplier: %s", entry.SupplierName),
			fmt.Sprintf("Date: %s", entry.ReturnDate),
			"",
			"Items:",
			"",
		}

		headers := []string{"Product", "Lot", "Qty", "Unit Price", "Line Total"}
		rows := make([][]string, 0, len(items))
		for _, item := range items {
			rows = append(rows, []string{
				item.ProductName,
				item.LotNumber,
				strconv.FormatFloat(item.Quantity, 'f', 2, 64),
				strconv.FormatFloat(item.UnitPrice, 'f', 2, 64),
				strconv.FormatFloat(item.LineTotal, 'f', 2, 64),
			})
		}
		lines = append(lines, formatReturnTableLines(headers, rows)...)
		lines = append(lines, "", fmt.Sprintf("Grand Total: %.2f", entry.GrandTotal), fmt.Sprintf("Payment Due: %.2f", entry.PaymentDue))

		pdfBytes, err := buildLandscapePdf("Purchase Return "+entry.ReferenceNumber, lines)
		if err != nil {
			log.Printf("export purchase return pdf: build pdf failed return_id=%s err=%v", entry.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to export purchase return"})
			return
		}

		filename := sanitizeFilename("purchase_return_" + entry.ReferenceNumber + ".pdf")
		c.Header("Content-Type", "application/pdf")
		c.Header("Content-Disposition", "attachment; filename="+filename)
		c.Header("Cache-Control", "no-store")
		c.Data(http.StatusOK, "application/pdf", pdfBytes)
	}
}

func loadFilteredPurchaseReturns(c *gin.Context, pool *pgxpool.Pool, authService *auth.Service) ([]repopurchaseorder.PurchaseReturn, error) {
	user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
	if err != nil {
		http.SetCookie(c.Writer, authService.ClearSessionCookie())
		return nil, err
	}

	if !hasBusinessRole(user.Roles) {
		return nil, fmt.Errorf("business access is required")
	}

	businessID := strings.TrimSpace(user.ActiveBusinessID)
	if businessID == "" {
		return nil, repopurchaseorder.ErrBusinessNotResolved
	}

	return repopurchaseorder.ListPurchaseReturnsRepository(pool, businessID, purchaseReturnFiltersFromContext(c))
}

func loadPurchaseReturnForPDF(c *gin.Context, pool *pgxpool.Pool, authService *auth.Service) (*repopurchaseorder.PurchaseReturn, []repopurchaseorder.PurchaseReturnItem, error) {
	user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
	if err != nil {
		http.SetCookie(c.Writer, authService.ClearSessionCookie())
		return nil, nil, err
	}

	if !hasBusinessRole(user.Roles) {
		return nil, nil, fmt.Errorf("business access is required")
	}

	businessID := strings.TrimSpace(user.ActiveBusinessID)
	returnID := strings.TrimSpace(c.Param("id"))
	if businessID == "" || returnID == "" {
		return nil, nil, repopurchaseorder.ErrBusinessNotResolved
	}

	entry, err := repopurchaseorder.GetPurchaseReturnByIDRepository(pool, businessID, returnID)
	if err != nil {
		return nil, nil, err
	}
	return entry, entry.Items, nil
}

func purchaseReturnFiltersFromContext(c *gin.Context) repopurchaseorder.ListPurchaseReturnsFilters {
	return repopurchaseorder.ListPurchaseReturnsFilters{
		LocationID:    strings.TrimSpace(c.Query("locationId")),
		SupplierID:    strings.TrimSpace(c.Query("supplierId")),
		Status:        strings.TrimSpace(c.Query("status")),
		PaymentStatus: strings.TrimSpace(c.Query("paymentStatus")),
		SearchQuery:   strings.TrimSpace(c.Query("searchQuery")),
		DateFrom:      strings.TrimSpace(c.Query("from")),
		DateTo:        strings.TrimSpace(c.Query("to")),
	}
}

func resolvePurchaseReturnExportColumns(raw string) []purchaseReturnExportColumn {
	if strings.TrimSpace(raw) == "" {
		columns := make([]purchaseReturnExportColumn, 0, len(purchaseReturnExportColumnOrder))
		for _, key := range purchaseReturnExportColumnOrder {
			if column, ok := purchaseReturnExportColumns[key]; ok {
				columns = append(columns, column)
			}
		}
		return columns
	}

	selected := make(map[string]struct{})
	for _, part := range strings.Split(raw, ",") {
		key := strings.TrimSpace(part)
		if key == "" {
			continue
		}
		selected[key] = struct{}{}
	}

	columns := make([]purchaseReturnExportColumn, 0)
	for _, key := range purchaseReturnExportColumnOrder {
		if _, ok := selected[key]; !ok {
			continue
		}
		if column, ok := purchaseReturnExportColumns[key]; ok {
			columns = append(columns, column)
		}
	}
	if len(columns) == 0 {
		for _, key := range purchaseReturnExportColumnOrder {
			if column, ok := purchaseReturnExportColumns[key]; ok {
				columns = append(columns, column)
			}
		}
	}
	return columns
}

func handlePurchaseReturnExportError(c *gin.Context, err error, message string) {
	switch {
	case errors.Is(err, repopurchaseorder.ErrBusinessNotResolved):
		c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"business_id": "Active business could not be resolved."}))
	default:
		log.Printf("purchase return export handler: err=%v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": message})
	}
}

func buildPurchaseReturnTableLines(title string, headers []string, rows [][]string, count int) []string {
	lines := []string{
		title,
		"",
		fmt.Sprintf("Generated: %s", time.Now().Format(time.RFC3339)),
		fmt.Sprintf("Returns: %d", count),
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

func formatReturnTableLines(headers []string, rows [][]string) []string {
	if len(headers) == 0 {
		return nil
	}
	widths := deriveColumnWidths(headers, rows, 140)
	lines := []string{formatTableRow(headers, widths), strings.Repeat("-", maxRuneCount(formatTableRow(headers, widths)))}
	for _, row := range rows {
		lines = append(lines, formatTableRow(row, widths))
	}
	return lines
}

func purchaseReturnPaymentStatusLabel(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "unpaid":
		return "Unpaid"
	case "partially_paid":
		return "Partially Paid"
	case "paid":
		return "Paid"
	case "pending_refund":
		return "Pending Refund"
	case "credit_note":
		return "Credit Note"
	case "refunded":
		return "Refunded"
	default:
		return strings.TrimSpace(status)
	}
}

func purchaseReturnStatusLabel(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "returned":
		return "Returned"
	case "partially_returned":
		return "Partially Returned"
	case "refunded":
		return "Refunded"
	case "rejected":
		return "Rejected"
	case "exchange":
		return "Exchange"
	case "draft":
		return "Draft"
	case "pending":
		return "Pending"
	case "approved":
		return "Approved"
	case "cancelled":
		return "Cancelled"
	default:
		return strings.TrimSpace(status)
	}
}

func normalizeOptionalString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
