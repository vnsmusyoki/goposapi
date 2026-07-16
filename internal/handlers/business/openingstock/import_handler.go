package openingstock

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
	"pos/internal/auth"
	repoopeningstock "pos/internal/repository/business/openingstock"
)

type openingStockImportBatchResponse struct {
	Batch   repoopeningstock.OpeningStockImportBatch      `json:"batch"`
	Rows    []repoopeningstock.OpeningStockImportBatchRow `json:"rows"`
	Message string                                        `json:"message"`
}

type parsedOpeningStockImportRow struct {
	rowNumber int
	data      map[string]string
}

type updateOpeningStockImportRowPayload struct {
	SKU               *string `json:"sku"`
	Location          *string `json:"location"`
	Quantity          *string `json:"quantity"`
	UnitCostBeforeTax *string `json:"unitCostBeforeTax"`
	LotNumber         *string `json:"lotNumber"`
	ExpiryDate        *string `json:"expiryDate"`
}

func DownloadOpeningStockImportTemplateRequestHandler(authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("download opening stock import template: auth lookup failed err=%v", err)
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
			return
		}
		if !hasBusinessRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{"message": "Business access is required"})
			return
		}

		filename := "opening_stock_import_template_" + time.Now().Format("2006-01-02") + ".csv"
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", "attachment; filename="+filename)
		c.Header("Cache-Control", "no-store")
		c.Status(http.StatusOK)

		writer := csv.NewWriter(c.Writer)
		defer writer.Flush()

		headers := []string{
			"SKU",
			"Location",
			"Quantity",
			"Unit Cost (Before Tax)",
			"Lot Number",
			"Expiry Date",
		}
		sample := []string{
			"SKU-001",
			"",
			"10",
			"100.00",
			"LOT-001",
			"07/16/2026",
		}
		_ = writer.Write(headers)
		_ = writer.Write(sample)
	}
}

func PreviewOpeningStockImportRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("preview opening stock import: auth lookup failed err=%v", err)
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

		file, header, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"file": "Upload a CSV file."}))
			return
		}
		defer file.Close()

		rows, err := parseOpeningStockImportCSV(file)
		if err != nil {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"file": err.Error()}))
			return
		}

		stagedRows := make([]repoopeningstock.OpeningStockImportBatchRow, 0, len(rows))
		for _, row := range rows {
			stagedRows = append(stagedRows, repoopeningstock.OpeningStockImportBatchRow{
				RowNumber: row.rowNumber,
				RowData:   row.data,
			})
		}

		batch, err := repoopeningstock.CreateOpeningStockImportPreviewRepository(pool, businessID, header.Filename, user.ID, stagedRows)
		if err != nil {
			log.Printf("preview opening stock import: repository failed business_id=%s err=%v", businessID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to store opening stock import preview"})
			return
		}

		loadedBatch, loadedRows, err := repoopeningstock.ListOpeningStockImportBatchRowsRepository(pool, businessID, batch.ID)
		if err != nil {
			log.Printf("preview opening stock import: reload batch failed business_id=%s batch_id=%s err=%v", businessID, batch.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load opening stock import preview"})
			return
		}

		c.JSON(http.StatusOK, openingStockImportBatchResponse{
			Batch:   *loadedBatch,
			Rows:    loadedRows,
			Message: "Opening stock import preview loaded successfully",
		})
	}
}

func ListOpeningStockImportBatchRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
			return
		}
		if !hasBusinessRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{"message": "Business access is required"})
			return
		}

		businessID := strings.TrimSpace(user.ActiveBusinessID)
		batchID := strings.TrimSpace(c.Param("batchId"))
		if businessID == "" || batchID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Opening stock import batch is required."}))
			return
		}

		batch, rows, err := repoopeningstock.ListOpeningStockImportBatchRowsRepository(pool, businessID, batchID)
		if err != nil {
			if errors.Is(err, repoopeningstock.ErrOpeningStockImportNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"message": "Opening stock import batch not found."})
				return
			}
			log.Printf("list opening stock import batch: repository failed business_id=%s batch_id=%s err=%v", businessID, batchID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load opening stock import batch"})
			return
		}

		c.JSON(http.StatusOK, openingStockImportBatchResponse{
			Batch:   *batch,
			Rows:    rows,
			Message: "Opening stock import batch loaded successfully",
		})
	}
}

func LatestOpeningStockImportBatchRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
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
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Opening stock import batch is required."}))
			return
		}

		batch, rows, err := repoopeningstock.GetLatestOpeningStockImportBatchRepository(pool, businessID)
		if err != nil {
			if errors.Is(err, repoopeningstock.ErrOpeningStockImportNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"message": "No staged opening stock import batch found."})
				return
			}
			log.Printf("latest opening stock import batch: repository failed business_id=%s err=%v", businessID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load latest opening stock import batch"})
			return
		}

		c.JSON(http.StatusOK, openingStockImportBatchResponse{
			Batch:   *batch,
			Rows:    rows,
			Message: "Latest opening stock import batch loaded successfully",
		})
	}
}

func UpdateOpeningStockImportRowRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
			return
		}
		if !hasBusinessRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{"message": "Business access is required"})
			return
		}

		businessID := strings.TrimSpace(user.ActiveBusinessID)
		batchID := strings.TrimSpace(c.Param("batchId"))
		rowID := strings.TrimSpace(c.Param("rowId"))
		if businessID == "" || batchID == "" || rowID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Opening stock import row is required."}))
			return
		}

		row, err := repoopeningstock.GetOpeningStockImportBatchRowRepository(pool, businessID, batchID, rowID)
		if err != nil {
			if errors.Is(err, repoopeningstock.ErrOpeningStockImportNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"message": "Opening stock import row not found."})
				return
			}
			log.Printf("update opening stock import row: load failed business_id=%s batch_id=%s row_id=%s err=%v", businessID, batchID, rowID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load opening stock import row"})
			return
		}
		if row.Status == "processed" || row.Status == "imported" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "This opening stock row has already been processed and cannot be edited."}))
			return
		}

		var payload updateOpeningStockImportRowPayload
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body must be valid JSON."}))
			return
		}

		nextData := map[string]string{
			"sku":                  stringOrEmpty(payload.SKU),
			"location":             stringOrEmpty(payload.Location),
			"quantity":             stringOrEmpty(payload.Quantity),
			"unit_cost_before_tax": stringOrEmpty(payload.UnitCostBeforeTax),
			"lot_number":           stringOrEmpty(payload.LotNumber),
			"expiry_date":          stringOrEmpty(payload.ExpiryDate),
		}

		updatedRow, validationErrors, err := repoopeningstock.UpdateOpeningStockImportBatchRowDataRepository(pool, businessID, batchID, rowID, nextData, user.ID)
		if err != nil {
			if errors.Is(err, repoopeningstock.ErrOpeningStockImportNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"message": "Opening stock import row not found."})
				return
			}
			log.Printf("update opening stock import row: repository failed business_id=%s batch_id=%s row_id=%s err=%v", businessID, batchID, rowID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update opening stock import row"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":          "Opening stock import row updated successfully",
			"row":              updatedRow,
			"validationErrors": validationErrors,
		})
	}
}

func ImportOpeningStockImportRowRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("import opening stock row: auth lookup failed err=%v", err)
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
			return
		}
		if !hasBusinessRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{"message": "Business access is required"})
			return
		}

		businessID := strings.TrimSpace(user.ActiveBusinessID)
		batchID := strings.TrimSpace(c.Param("batchId"))
		rowID := strings.TrimSpace(c.Param("rowId"))
		if businessID == "" || batchID == "" || rowID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Opening stock import row is required."}))
			return
		}

		row, err := repoopeningstock.GetOpeningStockImportBatchRowRepository(pool, businessID, batchID, rowID)
		if err != nil {
			if errors.Is(err, repoopeningstock.ErrOpeningStockImportNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"message": "Opening stock import row not found."})
				return
			}
			log.Printf("import opening stock row: load failed business_id=%s batch_id=%s row_id=%s err=%v", businessID, batchID, rowID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load opening stock import row"})
			return
		}

		if row.Status == "processed" || row.Status == "imported" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "This opening stock row has already been processed."}))
			return
		}

		rowInput, validationErrors, err := repoopeningstock.BuildOpeningStockImportInput(pool, row.RowData, businessID, user.ID)
		if err != nil {
			log.Printf("import opening stock row: build failed business_id=%s batch_id=%s row_id=%s err=%v", businessID, batchID, rowID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to prepare opening stock import row"})
			return
		}

		if len(validationErrors) > 0 {
			_ = repoopeningstock.UpdateOpeningStockImportBatchRowStatusRepository(pool, businessID, batchID, rowID, "invalid", "", validationErrors)
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": strings.Join(validationErrors, " ")}))
			return
		}

		rowInput.SourceID = row.ID
		rowInput.SourceType = "opening_stock_import"
		rowInput.CreatedBy = user.ID

		inventoryBatchID, err := repoopeningstock.ImportOpeningStockRepository(pool, rowInput)
		if err != nil {
			log.Printf("import opening stock row: create failed business_id=%s batch_id=%s row_id=%s err=%v", businessID, batchID, rowID, err)
			_ = repoopeningstock.UpdateOpeningStockImportBatchRowStatusRepository(pool, businessID, batchID, rowID, "error", "", []string{err.Error()})
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": err.Error()}))
			return
		}

		_ = repoopeningstock.UpdateOpeningStockImportBatchRowStatusRepository(pool, businessID, batchID, rowID, "processed", inventoryBatchID, nil)
		c.JSON(http.StatusOK, gin.H{
			"message": "Opening stock processed successfully",
		})
	}
}

func parseOpeningStockImportCSV(file io.Reader) ([]parsedOpeningStockImportRow, error) {
	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true

	headers, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read CSV header: %w", err)
	}

	normalizedHeaders := make([]string, 0, len(headers))
	for _, header := range headers {
		normalizedHeaders = append(normalizedHeaders, normalizeImportHeader(header))
	}

	rows := make([]parsedOpeningStockImportRow, 0)
	lineNumber := 1
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		lineNumber++
		if err != nil {
			return nil, fmt.Errorf("read CSV row %d: %w", lineNumber, err)
		}

		rowData := map[string]string{}
		for idx, value := range record {
			if idx < len(normalizedHeaders) {
				rowData[normalizedHeaders[idx]] = strings.TrimSpace(value)
			}
		}

		rows = append(rows, parsedOpeningStockImportRow{
			rowNumber: lineNumber,
			data:      rowData,
		})
	}

	return rows, nil
}

func normalizeImportHeader(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(value, "_")
	return strings.Trim(value, "_")
}

func validationFailed(errorsMap map[string]string) gin.H {
	if len(errorsMap) == 0 {
		errorsMap = map[string]string{"form": "Validation failed."}
	}
	return gin.H{
		"message": "Validation failed",
		"errors":  errorsMap,
	}
}

func hasBusinessRole(roles []auth.RoleResponse) bool {
	for _, role := range roles {
		if strings.EqualFold(role.Name, "business") || strings.EqualFold(role.Code, "business") {
			return true
		}
	}
	return false
}

func stringOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
