package product

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
	"pos/internal/auth"
	repoproduct "pos/internal/repository/business/product"
)

type productImportTemplateResponse struct {
	Message string `json:"message"`
}

type productImportBatchResponse struct {
	Batch   repoproduct.ProductImportBatch      `json:"batch"`
	Rows    []repoproduct.ProductImportBatchRow `json:"rows"`
	Message string                              `json:"message"`
}

func DownloadProductImportTemplateRequestHandler(authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("download product import template: auth lookup failed err=%v", err)
			http.SetCookie(c.Writer, authService.ClearSessionCookie())
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired. Please log in again."})
			return
		}
		if !hasBusinessRole(user.Roles) {
			c.JSON(http.StatusForbidden, gin.H{"message": "Business access is required"})
			return
		}

		filename := "product_import_template_" + time.Now().Format("2006-01-02") + ".csv"
		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", "attachment; filename="+filename)
		c.Header("Cache-Control", "no-store")
		c.Status(http.StatusOK)

		writer := csv.NewWriter(c.Writer)
		defer writer.Flush()

		headers := []string{
			"Product Name",
			"SKU",
			"Barcode",
			"Product Type",
			"Unit",
			"Brand",
			"Category Code",
			"Sub Category Code",
			"Location Code",
			"Manage Stock",
			"Alert Quantity",
			"Is For Selling",
			"Tax Type",
			"Tax Rate",
			"Default Purchase Price",
			"Default Selling Price",
			"Description",
		}
		sample := []string{
			"Sample Product",
			"SAMPLE-001",
			"",
			"single",
			"Piece",
			"Sample Brand",
			"CAT-001",
			"SUB-001",
			"LOC-001",
			"true",
			"2",
			"true",
			"exclusive",
			"16",
			"100",
			"150",
			"Example description",
		}
		_ = writer.Write(headers)
		_ = writer.Write(sample)
	}
}

func PreviewProductImportRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("preview product import: auth lookup failed err=%v", err)
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

		rows, err := parseProductImportCSV(file)
		if err != nil {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"file": err.Error()}))
			return
		}

		stagedRows := make([]repoproduct.ProductImportBatchRow, 0, len(rows))
		for _, row := range rows {
			stagedRow := repoproduct.ProductImportBatchRow{
				RowNumber:        row.rowNumber,
				RowData:          row.data,
				ValidationErrors: row.errors,
				Status:           row.status,
			}
			stagedRows = append(stagedRows, stagedRow)
		}

		batch, err := repoproduct.CreateProductImportPreviewRepository(pool, businessID, header.Filename, user.ID, stagedRows)
		if err != nil {
			log.Printf("preview product import: repository failed business_id=%s err=%v", businessID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to store product import preview"})
			return
		}

		loadedBatch, loadedRows, err := repoproduct.ListProductImportBatchRowsRepository(pool, businessID, batch.ID)
		if err != nil {
			log.Printf("preview product import: reload batch failed business_id=%s batch_id=%s err=%v", businessID, batch.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load product import preview"})
			return
		}

		c.JSON(http.StatusOK, productImportBatchResponse{
			Batch:   *loadedBatch,
			Rows:    loadedRows,
			Message: "Product import preview loaded successfully",
		})
	}
}

func ListProductImportBatchRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
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
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Product import batch is required."}))
			return
		}

		batch, rows, err := repoproduct.ListProductImportBatchRowsRepository(pool, businessID, batchID)
		if err != nil {
			if errors.Is(err, repoproduct.ErrProductNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"message": "Product import batch not found."})
				return
			}
			log.Printf("list product import batch: repository failed business_id=%s batch_id=%s err=%v", businessID, batchID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load product import batch"})
			return
		}

		c.JSON(http.StatusOK, productImportBatchResponse{
			Batch:   *batch,
			Rows:    rows,
			Message: "Product import batch loaded successfully",
		})
	}
}

func LatestProductImportBatchRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
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
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Product import batch is required."}))
			return
		}

		batch, rows, err := repoproduct.GetLatestProductImportBatchRepository(pool, businessID)
		if err != nil {
			if errors.Is(err, repoproduct.ErrProductNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"message": "No staged product import batch found."})
				return
			}
			log.Printf("latest product import batch: repository failed business_id=%s err=%v", businessID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load latest product import batch"})
			return
		}

		c.JSON(http.StatusOK, productImportBatchResponse{
			Batch:   *batch,
			Rows:    rows,
			Message: "Latest product import batch loaded successfully",
		})
	}
}

func ImportProductImportRowRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
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
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Product import row is required."}))
			return
		}

		row, err := repoproduct.GetProductImportBatchRowRepository(pool, businessID, batchID, rowID)
		if err != nil {
			if errors.Is(err, repoproduct.ErrProductNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"message": "Product import row not found."})
				return
			}
			log.Printf("import product row: load failed business_id=%s batch_id=%s row_id=%s err=%v", businessID, batchID, rowID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load product import row"})
			return
		}

		rowInput, validationErrors, err := repoproduct.BuildProductImportProductInput(pool, row.RowData, businessID, user.ID)
		if err != nil {
			log.Printf("import product row: build failed business_id=%s batch_id=%s row_id=%s err=%v", businessID, batchID, rowID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to prepare product import row"})
			return
		}

		if len(validationErrors) > 0 {
			_ = repoproduct.UpdateProductImportBatchRowStatusRepository(pool, businessID, batchID, rowID, "invalid", "", validationErrors)
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": strings.Join(validationErrors, " ")}))
			return
		}

		product, err := repoproduct.CreateProductRepository(pool, rowInput)
		if err != nil {
			log.Printf("import product row: create failed business_id=%s batch_id=%s row_id=%s err=%v", businessID, batchID, rowID, err)
			_ = repoproduct.UpdateProductImportBatchRowStatusRepository(pool, businessID, batchID, rowID, "error", "", []string{err.Error()})
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": err.Error()}))
			return
		}

		_ = repoproduct.UpdateProductImportBatchRowStatusRepository(pool, businessID, batchID, rowID, "imported", product.ID, nil)
		c.JSON(http.StatusOK, gin.H{
			"message": "Product imported successfully",
			"product": product,
		})
	}
}

type updateProductImportRowPayload struct {
	ProductName          *string `json:"productName"`
	SKU                  *string `json:"sku"`
	Barcode              *string `json:"barcode"`
	ProductType          *string `json:"productType"`
	Unit                 *string `json:"unit"`
	Brand                *string `json:"brand"`
	CategoryCode         *string `json:"categoryCode"`
	SubCategoryCode      *string `json:"subCategoryCode"`
	LocationCode         *string `json:"locationCode"`
	ManageStock          *bool   `json:"manageStock"`
	AlertQuantity        *string `json:"alertQuantity"`
	IsForSelling         *bool   `json:"isForSelling"`
	TaxType              *string `json:"taxType"`
	TaxRate              *string `json:"taxRate"`
	DefaultPurchasePrice *string `json:"defaultPurchasePrice"`
	DefaultSellingPrice  *string `json:"defaultSellingPrice"`
	Description          *string `json:"description"`
}

func UpdateProductImportRowRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
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
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Product import row is required."}))
			return
		}

		var payload updateProductImportRowPayload
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body must be valid JSON."}))
			return
		}

		nextData := map[string]string{
			"name":                   stringOrEmpty(payload.ProductName),
			"sku":                    stringOrEmpty(payload.SKU),
			"barcode":                stringOrEmpty(payload.Barcode),
			"product_type":           stringOrEmpty(payload.ProductType),
			"unit":                   stringOrEmpty(payload.Unit),
			"brand":                  stringOrEmpty(payload.Brand),
			"category_code":          stringOrEmpty(payload.CategoryCode),
			"sub_category_code":      stringOrEmpty(payload.SubCategoryCode),
			"location_code":          stringOrEmpty(payload.LocationCode),
			"manage_stock":           boolString(payload.ManageStock),
			"alert_quantity":         stringOrEmpty(payload.AlertQuantity),
			"is_for_selling":         boolString(payload.IsForSelling),
			"tax_type":               stringOrEmpty(payload.TaxType),
			"tax_rate":               stringOrEmpty(payload.TaxRate),
			"default_purchase_price": stringOrEmpty(payload.DefaultPurchasePrice),
			"default_selling_price":  stringOrEmpty(payload.DefaultSellingPrice),
			"description":            stringOrEmpty(payload.Description),
		}

		updatedRow, validationErrors, err := repoproduct.UpdateProductImportBatchRowDataRepository(pool, businessID, batchID, rowID, nextData, user.ID)
		if err != nil {
			log.Printf("update product import row: repository failed business_id=%s batch_id=%s row_id=%s err=%v", businessID, batchID, rowID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update product import row"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":          "Product import row updated successfully",
			"row":              updatedRow,
			"validationErrors": validationErrors,
		})
	}
}

type parsedProductImportRow struct {
	rowNumber int
	data      map[string]string
	errors    []string
	status    string
}

func parseProductImportCSV(file io.Reader) ([]parsedProductImportRow, error) {
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

	rows := make([]parsedProductImportRow, 0)
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

		rowData["product_type"] = strings.ToLower(firstNonEmpty(rowData["product_type"], "single"))
		rowData["tax_type"] = strings.ToLower(firstNonEmpty(rowData["tax_type"], "exclusive"))

		errors := validateProductImportRowData(rowData)
		status := "valid"
		if len(errors) > 0 {
			status = "invalid"
		}

		rows = append(rows, parsedProductImportRow{
			rowNumber: lineNumber,
			data:      rowData,
			errors:    errors,
			status:    status,
		})
	}

	return rows, nil
}

func normalizeImportHeader(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "_")
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, "/", "_")
	return strings.Trim(value, "_")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func validateProductImportRowData(rowData map[string]string) []string {
	errs := make([]string, 0)
	if strings.TrimSpace(rowData["name"]) == "" {
		errs = append(errs, "Product name is required.")
	}
	if strings.TrimSpace(rowData["unit"]) == "" {
		errs = append(errs, "Unit is required.")
	}
	if strings.TrimSpace(rowData["category_code"]) == "" {
		errs = append(errs, "Category code is required.")
	}
	if strings.TrimSpace(rowData["location_code"]) == "" {
		errs = append(errs, "Location code is required.")
	}
	if tp := strings.ToLower(strings.TrimSpace(rowData["product_type"])); tp != "" && tp != "single" {
		errs = append(errs, "Only single products are supported for import.")
	}
	return errs
}

func stringOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func boolString(value *bool) string {
	if value == nil {
		return ""
	}
	if *value {
		return "true"
	}
	return "false"
}
