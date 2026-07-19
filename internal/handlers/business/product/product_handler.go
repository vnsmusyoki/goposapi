package product

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
	"pos/internal/auth"
	"pos/internal/models"
	repoproduct "pos/internal/repository/business/product"
)

type createProductPayload struct {
	Name                    *string                         `json:"name"`
	SKU                     *string                         `json:"sku"`
	Barcode                 *string                         `json:"barcode"`
	ProductType             *string                         `json:"product_type"`
	UnitID                  *string                         `json:"unit_id"`
	SubUnitIDs              []string                        `json:"sub_unit_ids"`
	BrandID                 *string                         `json:"brand_id"`
	CategoryID              *string                         `json:"category_id"`
	SubCategoryID           *string                         `json:"sub_category_id"`
	LocationIDs             []string                        `json:"location_ids"`
	AllLocations            *bool                           `json:"all_locations"`
	ManageStock             *bool                           `json:"manage_stock"`
	AlertQuantity           *int                            `json:"alert_quantity"`
	IsForSelling            *bool                           `json:"is_for_selling"`
	TaxType                 *string                         `json:"tax_type"`
	TaxRate                 *float64                        `json:"tax_rate"`
	DefaultPurchasePrice    *float64                        `json:"default_purchase_price"`
	PurchasePriceExclusive  *float64                        `json:"purchase_price_exclusive"`
	PurchasePriceInclusive  *float64                        `json:"purchase_price_inclusive"`
	ProfitMargin            *float64                        `json:"profit_margin"`
	DefaultSellingPrice     *float64                        `json:"default_selling_price"`
	Description             *string                         `json:"description"`
	HasWarranty             *bool                           `json:"has_warranty"`
	WarrantyDuration        *string                         `json:"warranty_duration"`
	WarrantyPeriod          *string                         `json:"warranty_period"`
	WarrantyCoverage        *string                         `json:"warranty_coverage"`
	BrochureName            *string                         `json:"brochure_name"`
	BrochureURL             *string                         `json:"brochure_url"`
	CurrencyCode            *string                         `json:"currency_code"`
	CurrencySymbolPlacement *string                         `json:"currency_symbol_placement"`
	CurrencyPrecision       *int                            `json:"currency_precision"`
	Images                  []createProductImagePayload     `json:"images"`
	ComboItems              []createProductComboItemPayload `json:"combo_items"`
	Variants                []createProductVariantPayload   `json:"variants"`
	ProductPrices           []createProductPricePayload     `json:"product_prices"`
}

type createProductImagePayload struct {
	Name      *string `json:"name"`
	URL       *string `json:"url"`
	IsPrimary *bool   `json:"is_primary"`
}

type createProductComboItemPayload struct {
	ProductID   *string  `json:"product_id"`
	ProductName *string  `json:"product_name"`
	SKU         *string  `json:"sku"`
	Unit        *string  `json:"unit"`
	Quantity    *float64 `json:"quantity"`
	PriceEach   *float64 `json:"price_each"`
	Subtotal    *float64 `json:"subtotal"`
}

type createProductVariantPayload struct {
	Name               *string  `json:"name"`
	SKU                *string  `json:"sku"`
	Barcode            *string  `json:"barcode"`
	Cost               *float64 `json:"cost"`
	Selling            *float64 `json:"selling"`
	Stock              *float64 `json:"stock"`
	ShowOptionalFields *bool    `json:"show_optional_fields"`
	Weight             *string  `json:"weight"`
	Length             *string  `json:"length"`
	Width              *string  `json:"width"`
	Height             *string  `json:"height"`
	ImageName          *string  `json:"image_name"`
	ImageURL           *string  `json:"image_url"`
	ReorderLevel       *int     `json:"reorder_level"`
	ExpiryDate         *string  `json:"expiry_date"`
	SupplierCode       *string  `json:"supplier_code"`
}

type createProductPricePayload struct {
	PriceType     *string  `json:"price_type"`
	MinQuantity   *float64 `json:"min_quantity"`
	Price         *float64 `json:"price"`
	LocationID    *string  `json:"location_id"`
	CustomerGroup *string  `json:"customer_group"`
	StartsAt      *string  `json:"starts_at"`
	EndsAt        *string  `json:"ends_at"`
	Active        *bool    `json:"active"`
	Priority      *int     `json:"priority"`
}

type createProductResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	SKU         *string `json:"sku"`
	ProductType string  `json:"productType"`
	Message     string  `json:"message"`
}

type productSearchResponse struct {
	ID                     string  `json:"id"`
	Name                   string  `json:"name"`
	SKU                    *string `json:"sku"`
	UnitName               string  `json:"unitName"`
	SellingPrice           float64 `json:"sellingPrice"`
	CurrentStock           int     `json:"currentStock"`
	TaxType                string  `json:"taxType"`
	TaxRate                float64 `json:"taxRate"`
	DefaultPurchasePrice   float64 `json:"defaultPurchasePrice"`
	PurchasePriceExclusive float64 `json:"purchasePriceExclusive"`
	PurchasePriceInclusive float64 `json:"purchasePriceInclusive"`
	ProductType            string  `json:"productType"`
}

type productListItemResponse struct {
	ID                    string                 `json:"id"`
	Name                  string                 `json:"name"`
	SKU                   *string                `json:"sku"`
	ImageURL              string                 `json:"imageUrl"`
	Barcode               string                 `json:"barcode"`
	ProductType           string                 `json:"productType"`
	UnitID                string                 `json:"unitId"`
	UnitName              string                 `json:"unitName"`
	BrandID               string                 `json:"brandId"`
	BrandName             string                 `json:"brandName"`
	CategoryID            string                 `json:"categoryId"`
	CategoryName          string                 `json:"categoryName"`
	SubCategoryID         string                 `json:"subCategoryId"`
	SubCategoryName       string                 `json:"subCategoryName"`
	LocationIDs           []string               `json:"locationIds"`
	LocationNames         []string               `json:"locationNames"`
	ManageStock           bool                   `json:"manageStock"`
	AlertQuantity         int                    `json:"alertQuantity"`
	IsForSelling          bool                   `json:"isForSelling"`
	TaxType               string                 `json:"taxType"`
	TaxRate               float64                `json:"taxRate"`
	DefaultPurchasePrice  float64                `json:"defaultPurchasePrice"`
	ProfitAmount          float64                `json:"profitAmount"`
	DefaultSellingPrice   float64                `json:"defaultSellingPrice"`
	ProfitMargin          float64                `json:"profitMargin"`
	CurrentStock          int                    `json:"currentStock"`
	CurrentStockValue     float64                `json:"currentStockValue"`
	TotalUnitsSold        int                    `json:"totalUnitsSold"`
	TotalUnitsTransferred int                    `json:"totalUnitsTransferred"`
	TotalUnitsAdjusted    int                    `json:"totalUnitsAdjusted"`
	CreatedAt             string                 `json:"createdAt"`
	UpdatedAt             string                 `json:"updatedAt"`
	Status                string                 `json:"status"`
	ProductPrices         []productPriceResponse `json:"productPrices"`
}

type listProductsResponse struct {
	Products []productListItemResponse `json:"products"`
	Message  string                    `json:"message"`
}

type productDetailResponse struct {
	productListItemResponse
	UnitID                  string                             `json:"unitId"`
	SubUnitIDs              []string                           `json:"subUnitIds"`
	BrandID                 string                             `json:"brandId"`
	CategoryID              string                             `json:"categoryId"`
	SubCategoryID           string                             `json:"subCategoryId"`
	LocationIDs             []string                           `json:"locationIds"`
	AllLocations            bool                               `json:"allLocations"`
	ManageStock             bool                               `json:"manageStock"`
	AlertQuantity           int                                `json:"alertQuantity"`
	IsForSelling            bool                               `json:"isForSelling"`
	TaxType                 string                             `json:"taxType"`
	TaxRate                 float64                            `json:"taxRate"`
	DefaultPurchasePrice    float64                            `json:"defaultPurchasePrice"`
	PurchasePriceExclusive  float64                            `json:"purchasePriceExclusive"`
	ProfitAmount            float64                            `json:"profitAmount"`
	PurchasePriceInclusive  float64                            `json:"purchasePriceInclusive"`
	ProfitMargin            float64                            `json:"profitMargin"`
	DefaultSellingPrice     float64                            `json:"defaultSellingPrice"`
	Description             string                             `json:"description"`
	HasWarranty             bool                               `json:"hasWarranty"`
	WarrantyDuration        string                             `json:"warrantyDuration"`
	WarrantyPeriod          string                             `json:"warrantyPeriod"`
	WarrantyCoverage        string                             `json:"warrantyCoverage"`
	BrochureName            string                             `json:"brochureName"`
	BrochureURL             string                             `json:"brochureUrl"`
	CurrencyCode            string                             `json:"currencyCode"`
	CurrencySymbolPlacement string                             `json:"currencySymbolPlacement"`
	CurrencyPrecision       int                                `json:"currencyPrecision"`
	Images                  []repoproduct.ProductImageItem     `json:"images"`
	ComboItems              []repoproduct.ProductComboItemItem `json:"comboItems"`
	Variants                []repoproduct.ProductVariantItem   `json:"variants"`
	ProductPrices           []productPriceResponse             `json:"productPrices"`
}

type productPriceResponse struct {
	ID            string  `json:"id"`
	PriceType     string  `json:"priceType"`
	MinQuantity   float64 `json:"minQuantity"`
	Price         float64 `json:"price"`
	LocationID    string  `json:"locationId"`
	CustomerGroup string  `json:"customerGroup"`
	StartsAt      string  `json:"startsAt"`
	EndsAt        string  `json:"endsAt"`
	Active        bool    `json:"active"`
	Priority      int     `json:"priority"`
}

type updateProductResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	SKU         *string `json:"sku"`
	ProductType string  `json:"productType"`
	Message     string  `json:"message"`
}

type productPriceHistoryResponse struct {
	ID           string  `json:"id"`
	BuyingPrice  float64 `json:"buyingPrice"`
	SellingPrice float64 `json:"sellingPrice"`
	ChangedBy    string  `json:"changedBy"`
	Reason       *string `json:"reason"`
	CreatedAt    string  `json:"createdAt"`
}

func mapProductDetailResponse(detail *repoproduct.ProductDetail) productDetailResponse {
	if detail == nil {
		return productDetailResponse{}
	}

	response := productDetailResponse{
		productListItemResponse: productListItemResponse{
			ID:                    detail.ID,
			Name:                  detail.Name,
			SKU:                   detail.SKU,
			ImageURL:              detail.ImageURL,
			Barcode:               detail.Barcode,
			ProductType:           detail.ProductType,
			UnitID:                detail.UnitID,
			UnitName:              detail.UnitName,
			BrandID:               detail.BrandID,
			BrandName:             detail.BrandName,
			CategoryID:            detail.CategoryID,
			CategoryName:          detail.CategoryName,
			SubCategoryID:         detail.SubCategoryID,
			SubCategoryName:       detail.SubCategoryName,
			LocationIDs:           detail.LocationIDs,
			LocationNames:         detail.LocationNames,
			ManageStock:           detail.ManageStock,
			AlertQuantity:         detail.AlertQuantity,
			IsForSelling:          detail.IsForSelling,
			TaxType:               detail.TaxType,
			TaxRate:               detail.TaxRate,
			DefaultPurchasePrice:  detail.DefaultPurchasePrice,
			ProfitAmount:          detail.ProfitAmount,
			DefaultSellingPrice:   detail.DefaultSellingPrice,
			ProfitMargin:          detail.ProfitMargin,
			CurrentStock:          detail.CurrentStock,
			CurrentStockValue:     detail.CurrentStockValue,
			TotalUnitsSold:        detail.TotalUnitsSold,
			TotalUnitsTransferred: detail.TotalUnitsTransferred,
			TotalUnitsAdjusted:    detail.TotalUnitsAdjusted,
			CreatedAt:             detail.CreatedAt,
			UpdatedAt:             detail.UpdatedAt,
			Status:                detail.Status,
			ProductPrices:         mapRepositoryProductPrices(detail.ProductPrices),
		},
		UnitID:                  detail.UnitID,
		SubUnitIDs:              detail.SubUnitIDs,
		BrandID:                 detail.BrandID,
		CategoryID:              detail.CategoryID,
		SubCategoryID:           detail.SubCategoryID,
		LocationIDs:             detail.LocationIDs,
		AllLocations:            detail.AllLocations,
		ManageStock:             detail.ManageStock,
		AlertQuantity:           detail.AlertQuantity,
		IsForSelling:            detail.IsForSelling,
		TaxType:                 detail.TaxType,
		TaxRate:                 detail.TaxRate,
		DefaultPurchasePrice:    detail.DefaultPurchasePrice,
		PurchasePriceExclusive:  detail.PurchasePriceExclusive,
		ProfitAmount:            detail.ProfitAmount,
		PurchasePriceInclusive:  detail.PurchasePriceInclusive,
		ProfitMargin:            detail.ProfitMargin,
		DefaultSellingPrice:     detail.DefaultSellingPrice,
		Description:             detail.Description,
		HasWarranty:             detail.HasWarranty,
		WarrantyDuration:        detail.WarrantyDuration,
		WarrantyPeriod:          detail.WarrantyPeriod,
		WarrantyCoverage:        detail.WarrantyCoverage,
		BrochureName:            detail.BrochureName,
		BrochureURL:             detail.BrochureURL,
		CurrencyCode:            detail.CurrencyCode,
		CurrencySymbolPlacement: detail.CurrencySymbolPlacement,
		CurrencyPrecision:       detail.CurrencyPrecision,
		Images:                  detail.Images,
		ComboItems:              detail.ComboItems,
		Variants:                detail.Variants,
		ProductPrices:           mapRepositoryProductPrices(detail.ProductPrices),
	}

	if response.CurrencyCode == "" {
		response.CurrencyCode = "USD"
	}
	if response.CurrencySymbolPlacement == "" {
		response.CurrencySymbolPlacement = "before"
	}
	if response.CurrencyPrecision < 0 {
		response.CurrencyPrecision = 2
	}
	return response
}

func mapRepositoryProductPrices(items []repoproduct.ProductPriceItem) []productPriceResponse {
	response := make([]productPriceResponse, 0, len(items))
	for _, item := range items {
		response = append(response, productPriceResponse{
			ID:            item.ID,
			PriceType:     item.PriceType,
			MinQuantity:   item.MinQuantity,
			Price:         item.Price,
			LocationID:    item.LocationID,
			CustomerGroup: item.CustomerGroup,
			StartsAt:      item.StartsAt,
			EndsAt:        item.EndsAt,
			Active:        item.Active,
			Priority:      item.Priority,
		})
	}
	return response
}

func mapModelProductPrices(items []models.ProductPriceItem) []productPriceResponse {
	response := make([]productPriceResponse, 0, len(items))
	for _, item := range items {
		response = append(response, productPriceResponse{
			ID:            item.ID,
			PriceType:     item.PriceType,
			MinQuantity:   item.MinQuantity,
			Price:         item.Price,
			LocationID:    item.LocationID,
			CustomerGroup: item.CustomerGroup,
			StartsAt:      item.StartsAt,
			EndsAt:        item.EndsAt,
			Active:        item.Active,
			Priority:      item.Priority,
		})
	}
	return response
}

func CreateProductRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("create product handler: auth lookup failed err=%v", err)
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

		body, err := c.GetRawData()
		if err != nil && !errors.Is(err, io.EOF) {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Unable to read request body."}))
			return
		}
		if len(strings.TrimSpace(string(body))) == 0 {
			c.JSON(http.StatusBadRequest, validationFailed(productFieldErrors(nil)))
			return
		}

		var payload createProductPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			log.Printf("create product handler: invalid json err=%v body=%s", err, string(body))
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body must be valid JSON."}))
			return
		}

		if errs := productFieldErrors(&payload); len(errs) > 0 {
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

		req := repoproduct.CreateProductInput{
			BusinessID:    businessID,
			Name:          derefString(payload.Name),
			SKU:           derefString(payload.SKU),
			Barcode:       derefString(payload.Barcode),
			ProductType:   strings.ToLower(derefString(payload.ProductType)),
			UnitID:        derefString(payload.UnitID),
			SubUnitIDs:    cleanStrings(payload.SubUnitIDs),
			BrandID:       derefString(payload.BrandID),
			CategoryID:    derefString(payload.CategoryID),
			SubCategoryID: derefString(payload.SubCategoryID),
			LocationIDs:   cleanStrings(payload.LocationIDs),
			AllLocations:  boolValue(payload.AllLocations, false),
			ManageStock:   boolValue(payload.ManageStock, false),
			AlertQuantity: payload.AlertQuantity,
			IsForSelling:  boolValue(payload.IsForSelling, true),
			TaxType: func() string {
				value := strings.ToLower(derefString(payload.TaxType))
				if value == "" {
					return "exclusive"
				}
				return value
			}(),
			TaxRate:                 floatValue(payload.TaxRate, 0),
			DefaultPurchasePrice:    payload.DefaultPurchasePrice,
			PurchasePriceExclusive:  payload.PurchasePriceExclusive,
			PurchasePriceInclusive:  payload.PurchasePriceInclusive,
			ProfitMargin:            payload.ProfitMargin,
			DefaultSellingPrice:     payload.DefaultSellingPrice,
			Description:             derefString(payload.Description),
			HasWarranty:             boolValue(payload.HasWarranty, false),
			WarrantyDuration:        derefString(payload.WarrantyDuration),
			WarrantyPeriod:          strings.ToLower(derefString(payload.WarrantyPeriod)),
			WarrantyCoverage:        derefString(payload.WarrantyCoverage),
			BrochureName:            derefString(payload.BrochureName),
			BrochureURL:             derefString(payload.BrochureURL),
			CurrencyCode:            strings.ToUpper(strings.TrimSpace(derefString(payload.CurrencyCode))),
			CurrencySymbolPlacement: strings.ToLower(derefString(payload.CurrencySymbolPlacement)),
			CurrencyPrecision:       intValue(payload.CurrencyPrecision, 2),
			CreatedBy:               user.ID,
		}

		for _, image := range payload.Images {
			if image.URL == nil || strings.TrimSpace(*image.URL) == "" {
				continue
			}
			req.Images = append(req.Images, repoproduct.CreateProductImageInput{
				Name:      derefString(image.Name),
				URL:       derefString(image.URL),
				IsPrimary: boolValue(image.IsPrimary, false),
			})
		}

		for _, item := range payload.ComboItems {
			req.ComboItems = append(req.ComboItems, repoproduct.CreateProductComboItemInput{
				ProductID:   derefString(item.ProductID),
				ProductName: derefString(item.ProductName),
				SKU:         derefString(item.SKU),
				Unit:        derefString(item.Unit),
				Quantity:    floatValue(item.Quantity, 0),
				PriceEach:   floatValue(item.PriceEach, 0),
				Subtotal:    floatValue(item.Subtotal, 0),
			})
		}

		for _, variant := range payload.Variants {
			req.Variants = append(req.Variants, repoproduct.CreateProductVariantInput{
				Name:               derefString(variant.Name),
				SKU:                derefString(variant.SKU),
				Barcode:            derefString(variant.Barcode),
				Cost:               floatValue(variant.Cost, 0),
				Selling:            floatValue(variant.Selling, 0),
				Stock:              floatValue(variant.Stock, 0),
				ShowOptionalFields: boolValue(variant.ShowOptionalFields, false),
				Weight:             derefString(variant.Weight),
				Length:             derefString(variant.Length),
				Width:              derefString(variant.Width),
				Height:             derefString(variant.Height),
				ImageName:          derefString(variant.ImageName),
				ImageURL:           derefString(variant.ImageURL),
				ReorderLevel:       variant.ReorderLevel,
				ExpiryDate:         derefString(variant.ExpiryDate),
				SupplierCode:       derefString(variant.SupplierCode),
			})
		}

		for _, price := range payload.ProductPrices {
			req.ProductPrices = append(req.ProductPrices, repoproduct.CreateProductPriceInput{
				PriceType:     strings.ToLower(derefString(price.PriceType)),
				MinQuantity:   floatValue(price.MinQuantity, 1),
				Price:         floatValue(price.Price, 0),
				LocationID:    derefString(price.LocationID),
				CustomerGroup: derefString(price.CustomerGroup),
				StartsAt:      derefString(price.StartsAt),
				EndsAt:        derefString(price.EndsAt),
				Active:        boolValue(price.Active, true),
				Priority:      intValue(price.Priority, 100),
			})
		}

		product, err := repoproduct.CreateProductRepository(pool, req)
		if err != nil {
			switch {
			case errors.Is(err, repoproduct.ErrProductAlreadyExists):
				c.JSON(http.StatusConflict, gin.H{
					"message": "Product SKU already exists.",
					"errors": gin.H{
						"sku": "Product SKU already exists.",
					},
				})
			case errors.Is(err, repoproduct.ErrInvalidComboProduct):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
					"comboItems": "Combo items must reference existing single products.",
				}))
			case errors.Is(err, repoproduct.ErrInvalidProductInput), errors.Is(err, repoproduct.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
					"form": err.Error(),
				}))
			default:
				log.Printf("create product handler: repository failed business_id=%s sku=%q err=%v", businessID, req.SKU, err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"message": "Failed to create product",
				})
			}
			return
		}

		c.JSON(http.StatusCreated, createProductResponse{
			ID:          product.ID,
			Name:        product.Name,
			SKU:         product.SKU,
			ProductType: product.ProductType,
			Message:     "Product created successfully",
		})
	}
}

func SearchProductsRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("search products handler: auth lookup failed err=%v", err)
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
			c.JSON(http.StatusBadRequest, gin.H{"message": "Active business is required."})
			return
		}

		query := strings.TrimSpace(c.Query("query"))
		productType := strings.TrimSpace(c.Query("product_type"))
		if len(query) < 3 {
			c.JSON(http.StatusOK, []productSearchResponse{})
			return
		}

		items, err := repoproduct.SearchProductsRepository(pool, businessID, query, productType)
		if err != nil {
			switch {
			case errors.Is(err, repoproduct.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, gin.H{"message": "Active business is required."})
			default:
				log.Printf("search products handler: repository failed business_id=%s query=%q err=%v", businessID, query, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to search products"})
			}
			return
		}

		results := make([]productSearchResponse, 0, len(items))
		for _, item := range items {
			results = append(results, productSearchResponse{
				ID:                     item.ID,
				Name:                   item.Name,
				SKU:                    item.SKU,
				UnitName:               item.UnitName,
				SellingPrice:           item.SellingPrice,
				CurrentStock:           item.CurrentStock,
				TaxType:                item.TaxType,
				TaxRate:                item.TaxRate,
				DefaultPurchasePrice:   item.DefaultPurchasePrice,
				PurchasePriceExclusive: item.PurchasePriceExclusive,
				PurchasePriceInclusive: item.PurchasePriceInclusive,
				ProductType:            item.ProductType,
			})
		}

		c.JSON(http.StatusOK, results)
	}
}

func ListProductsRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("list products handler: auth lookup failed err=%v", err)
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
			c.JSON(http.StatusBadRequest, gin.H{"message": "Active business is required."})
			return
		}

		items, err := repoproduct.ListProductsRepository(pool, businessID, repoproduct.ListProductsFilters{
			Search:            c.Query("search"),
			ProductType:       c.Query("product_type"),
			CategoryID:        c.Query("category_id"),
			BrandID:           c.Query("brand_id"),
			UnitID:            c.Query("unit_id"),
			LocationID:        c.Query("location_id"),
			TaxType:           c.Query("tax_type"),
			ShowNotForSelling: strings.EqualFold(c.Query("show_not_for_selling"), "true"),
		})
		if err != nil {
			switch {
			case errors.Is(err, repoproduct.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, gin.H{"message": "Active business is required."})
			default:
				log.Printf("list products handler: repository failed business_id=%s err=%v", businessID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load products"})
			}
			return
		}

		response := make([]productListItemResponse, 0, len(items))
		for _, item := range items {
			response = append(response, productListItemResponse{
				ID:                    item.ID,
				Name:                  item.Name,
				SKU:                   item.SKU,
				ImageURL:              item.ImageURL,
				Barcode:               item.Barcode,
				ProductType:           item.ProductType,
				UnitID:                item.UnitID,
				UnitName:              item.UnitName,
				BrandID:               item.BrandID,
				BrandName:             item.BrandName,
				CategoryID:            item.CategoryID,
				CategoryName:          item.CategoryName,
				SubCategoryID:         item.SubCategoryID,
				SubCategoryName:       item.SubCategoryName,
				LocationIDs:           item.LocationIDs,
				LocationNames:         item.LocationNames,
				ManageStock:           item.ManageStock,
				AlertQuantity:         item.AlertQuantity,
				IsForSelling:          item.IsForSelling,
				TaxType:               item.TaxType,
				TaxRate:               item.TaxRate,
				DefaultPurchasePrice:  item.DefaultPurchasePrice,
				ProfitAmount:          item.ProfitAmount,
				DefaultSellingPrice:   item.DefaultSellingPrice,
				ProfitMargin:          item.ProfitMargin,
				CurrentStock:          item.CurrentStock,
				CurrentStockValue:     item.CurrentStockValue,
				TotalUnitsSold:        item.TotalUnitsSold,
				TotalUnitsTransferred: item.TotalUnitsTransferred,
				TotalUnitsAdjusted:    item.TotalUnitsAdjusted,
				CreatedAt:             item.CreatedAt,
				UpdatedAt:             item.UpdatedAt,
				Status:                item.Status,
				ProductPrices:         mapModelProductPrices(item.ProductPrices),
			})
		}

		c.JSON(http.StatusOK, listProductsResponse{
			Products: response,
			Message:  "Products loaded successfully",
		})
	}
}

func GetProductRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("get product handler: auth lookup failed err=%v", err)
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
			c.JSON(http.StatusBadRequest, gin.H{"message": "Active business is required."})
			return
		}

		productID := strings.TrimSpace(c.Param("id"))
		if productID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Product ID is required."})
			return
		}

		detail, err := repoproduct.GetProductByIDRepository(pool, businessID, productID)
		if err != nil {
			switch {
			case errors.Is(err, repoproduct.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, gin.H{"message": "Active business is required."})
			case errors.Is(err, repoproduct.ErrProductNotFound):
				c.JSON(http.StatusNotFound, gin.H{"message": "Product not found."})
			default:
				log.Printf("get product handler: repository failed business_id=%s product_id=%s err=%v", businessID, productID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load product"})
			}
			return
		}

		c.JSON(http.StatusOK, mapProductDetailResponse(detail))
	}
}

func UpdateProductRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("update product handler: auth lookup failed err=%v", err)
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

		productID := strings.TrimSpace(c.Param("id"))
		if productID == "" {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"product_id": "Product ID is required."}))
			return
		}

		body, err := c.GetRawData()
		if err != nil && !errors.Is(err, io.EOF) {
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Unable to read request body."}))
			return
		}
		if len(strings.TrimSpace(string(body))) == 0 {
			c.JSON(http.StatusBadRequest, validationFailed(productFieldErrors(nil)))
			return
		}

		var payload createProductPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			log.Printf("update product handler: invalid json err=%v body=%s", err, string(body))
			c.JSON(http.StatusBadRequest, validationFailed(map[string]string{"form": "Request body must be valid JSON."}))
			return
		}

		if errs := productFieldErrors(&payload); len(errs) > 0 {
			c.JSON(http.StatusBadRequest, validationFailed(errs))
			return
		}

		req := repoproduct.CreateProductInput{
			BusinessID:    businessID,
			Name:          derefString(payload.Name),
			SKU:           derefString(payload.SKU),
			Barcode:       derefString(payload.Barcode),
			ProductType:   strings.ToLower(derefString(payload.ProductType)),
			UnitID:        derefString(payload.UnitID),
			SubUnitIDs:    cleanStrings(payload.SubUnitIDs),
			BrandID:       derefString(payload.BrandID),
			CategoryID:    derefString(payload.CategoryID),
			SubCategoryID: derefString(payload.SubCategoryID),
			LocationIDs:   cleanStrings(payload.LocationIDs),
			AllLocations:  boolValue(payload.AllLocations, false),
			ManageStock:   boolValue(payload.ManageStock, false),
			AlertQuantity: payload.AlertQuantity,
			IsForSelling:  boolValue(payload.IsForSelling, true),
			TaxType: func() string {
				value := strings.ToLower(derefString(payload.TaxType))
				if value == "" {
					return "exclusive"
				}
				return value
			}(),
			TaxRate:                 floatValue(payload.TaxRate, 0),
			DefaultPurchasePrice:    payload.DefaultPurchasePrice,
			PurchasePriceExclusive:  payload.PurchasePriceExclusive,
			PurchasePriceInclusive:  payload.PurchasePriceInclusive,
			ProfitMargin:            payload.ProfitMargin,
			DefaultSellingPrice:     payload.DefaultSellingPrice,
			Description:             derefString(payload.Description),
			HasWarranty:             boolValue(payload.HasWarranty, false),
			WarrantyDuration:        derefString(payload.WarrantyDuration),
			WarrantyPeriod:          strings.ToLower(derefString(payload.WarrantyPeriod)),
			WarrantyCoverage:        derefString(payload.WarrantyCoverage),
			BrochureName:            derefString(payload.BrochureName),
			BrochureURL:             derefString(payload.BrochureURL),
			CurrencyCode:            strings.ToUpper(strings.TrimSpace(derefString(payload.CurrencyCode))),
			CurrencySymbolPlacement: strings.ToLower(derefString(payload.CurrencySymbolPlacement)),
			CurrencyPrecision:       intValue(payload.CurrencyPrecision, 2),
			CreatedBy:               user.ID,
		}

		for _, image := range payload.Images {
			if image.URL == nil || strings.TrimSpace(*image.URL) == "" {
				continue
			}
			req.Images = append(req.Images, repoproduct.CreateProductImageInput{
				Name:      derefString(image.Name),
				URL:       derefString(image.URL),
				IsPrimary: boolValue(image.IsPrimary, false),
			})
		}

		for _, item := range payload.ComboItems {
			req.ComboItems = append(req.ComboItems, repoproduct.CreateProductComboItemInput{
				ProductID:   derefString(item.ProductID),
				ProductName: derefString(item.ProductName),
				SKU:         derefString(item.SKU),
				Unit:        derefString(item.Unit),
				Quantity:    floatValue(item.Quantity, 0),
				PriceEach:   floatValue(item.PriceEach, 0),
				Subtotal:    floatValue(item.Subtotal, 0),
			})
		}

		for _, variant := range payload.Variants {
			req.Variants = append(req.Variants, repoproduct.CreateProductVariantInput{
				Name:               derefString(variant.Name),
				SKU:                derefString(variant.SKU),
				Barcode:            derefString(variant.Barcode),
				Cost:               floatValue(variant.Cost, 0),
				Selling:            floatValue(variant.Selling, 0),
				Stock:              floatValue(variant.Stock, 0),
				ShowOptionalFields: boolValue(variant.ShowOptionalFields, false),
				Weight:             derefString(variant.Weight),
				Length:             derefString(variant.Length),
				Width:              derefString(variant.Width),
				Height:             derefString(variant.Height),
				ImageName:          derefString(variant.ImageName),
				ImageURL:           derefString(variant.ImageURL),
				ReorderLevel:       variant.ReorderLevel,
				ExpiryDate:         derefString(variant.ExpiryDate),
				SupplierCode:       derefString(variant.SupplierCode),
			})
		}

		for _, price := range payload.ProductPrices {
			req.ProductPrices = append(req.ProductPrices, repoproduct.CreateProductPriceInput{
				PriceType:     strings.ToLower(derefString(price.PriceType)),
				MinQuantity:   floatValue(price.MinQuantity, 1),
				Price:         floatValue(price.Price, 0),
				LocationID:    derefString(price.LocationID),
				CustomerGroup: derefString(price.CustomerGroup),
				StartsAt:      derefString(price.StartsAt),
				EndsAt:        derefString(price.EndsAt),
				Active:        boolValue(price.Active, true),
				Priority:      intValue(price.Priority, 100),
			})
		}

		product, err := repoproduct.UpdateProductRepository(pool, productID, req, user.ID)
		if err != nil {
			switch {
			case errors.Is(err, repoproduct.ErrProductNotFound):
				c.JSON(http.StatusNotFound, gin.H{"message": "Product not found."})
			case errors.Is(err, repoproduct.ErrProductAlreadyExists):
				c.JSON(http.StatusConflict, gin.H{
					"message": "Product SKU already exists.",
					"errors": gin.H{
						"sku": "Product SKU already exists.",
					},
				})
			case errors.Is(err, repoproduct.ErrInvalidComboProduct):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
					"comboItems": "Combo items must reference existing single products.",
				}))
			case errors.Is(err, repoproduct.ErrInvalidProductInput), errors.Is(err, repoproduct.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, validationFailed(map[string]string{
					"form": err.Error(),
				}))
			default:
				log.Printf("update product handler: repository failed business_id=%s product_id=%s err=%v", businessID, productID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to update product"})
			}
			return
		}

		c.JSON(http.StatusOK, updateProductResponse{
			ID:          product.ID,
			Name:        product.Name,
			SKU:         product.SKU,
			ProductType: product.ProductType,
			Message:     "Product updated successfully",
		})
	}
}

func ListProductPriceHistoryRequestHandler(pool *pgxpool.Pool, authService *auth.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, _, err := authService.CurrentUserFromRequest(c.Request.Context(), c.Request)
		if err != nil {
			log.Printf("list product price history handler: auth lookup failed err=%v", err)
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
			c.JSON(http.StatusBadRequest, gin.H{"message": "Active business is required."})
			return
		}

		productID := strings.TrimSpace(c.Param("id"))
		if productID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"message": "Product ID is required."})
			return
		}

		items, err := repoproduct.ListProductPriceHistoryRepository(pool, businessID, productID)
		if err != nil {
			switch {
			case errors.Is(err, repoproduct.ErrBusinessNotResolved):
				c.JSON(http.StatusBadRequest, gin.H{"message": "Active business is required."})
			default:
				log.Printf("list product price history handler: repository failed business_id=%s product_id=%s err=%v", businessID, productID, err)
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Failed to load price history"})
			}
			return
		}

		response := make([]productPriceHistoryResponse, 0, len(items))
		for _, item := range items {
			var reason *string
			if item.Reason.Valid {
				value := item.Reason.String
				reason = &value
			}
			response = append(response, productPriceHistoryResponse{
				ID:           item.ID,
				BuyingPrice:  item.BuyingPrice,
				SellingPrice: item.SellingPrice,
				ChangedBy:    item.ChangedByName,
				Reason:       reason,
				CreatedAt:    item.CreatedAt,
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"items":   response,
			"message": "Price history loaded successfully",
		})
	}
}

func productFieldErrors(payload *createProductPayload) map[string]string {
	errs := map[string]string{}
	if payload == nil {
		errs["form"] = "Validation failed."
		return errs
	}

	if payload.Name == nil || strings.TrimSpace(*payload.Name) == "" {
		errs["name"] = "Product name is required."
	}
	if payload.ProductType == nil || strings.TrimSpace(*payload.ProductType) == "" {
		errs["productType"] = "Select a product type."
	}
	if payload.UnitID == nil || strings.TrimSpace(*payload.UnitID) == "" {
		errs["unitId"] = "Select a base unit."
	}
	if payload.CategoryID == nil || strings.TrimSpace(*payload.CategoryID) == "" {
		errs["categoryId"] = "Select a category."
	}
	if payload.AllLocations == nil || !*payload.AllLocations {
		if len(payload.LocationIDs) == 0 {
			errs["locationIds"] = "Select at least one business location."
		}
	}
	if payload.ManageStock != nil && *payload.ManageStock && payload.AlertQuantity != nil && *payload.AlertQuantity < 2 {
		errs["alertQuantity"] = "Alert quantity must be at least 2."
	}
	if payload.ProductType != nil {
		switch strings.ToLower(strings.TrimSpace(*payload.ProductType)) {
		case "single":
			if payload.DefaultPurchasePrice == nil {
				errs["defaultPurchasePrice"] = "Default purchase price is required."
			}
			if payload.DefaultSellingPrice == nil {
				errs["defaultSellingPrice"] = "Default selling price is required."
			}
		case "combo":
			if len(payload.ComboItems) == 0 {
				errs["comboItems"] = "Add at least one combo item."
			}
		case "variable":
			if len(payload.Variants) == 0 {
				errs["variants"] = "Add at least one variant."
			}
		default:
			errs["productType"] = "Select a valid product type."
		}
	}

	for i, item := range payload.ComboItems {
		if item.ProductID == nil || strings.TrimSpace(*item.ProductID) == "" {
			errs[fmt.Sprintf("comboItems.%d.productId", i)] = "Select a product."
		}
		if item.Quantity == nil || *item.Quantity < 1 {
			errs[fmt.Sprintf("comboItems.%d.quantity", i)] = "Quantity must be at least 1."
		}
		if item.PriceEach == nil || *item.PriceEach < 0 {
			errs[fmt.Sprintf("comboItems.%d.priceEach", i)] = "Price each cannot be negative."
		}
		if item.Subtotal == nil || *item.Subtotal < 0 {
			errs[fmt.Sprintf("comboItems.%d.subtotal", i)] = "Subtotal cannot be negative."
		}
	}

	for i, variant := range payload.Variants {
		if variant.Name == nil || strings.TrimSpace(*variant.Name) == "" {
			errs[fmt.Sprintf("variants.%d.name", i)] = "Variant name is required."
		}
		if variant.SKU == nil || strings.TrimSpace(*variant.SKU) == "" {
			errs[fmt.Sprintf("variants.%d.sku", i)] = "Variant SKU is required."
		}
		if variant.Cost == nil || *variant.Cost < 1 {
			errs[fmt.Sprintf("variants.%d.cost", i)] = "Cost must be at least 1."
		}
		if variant.Selling == nil || *variant.Selling < 1 {
			errs[fmt.Sprintf("variants.%d.selling", i)] = "Selling price must be at least 1."
		}
		if variant.Cost != nil && variant.Selling != nil && *variant.Cost > *variant.Selling {
			errs[fmt.Sprintf("variants.%d.cost", i)] = "Cost cannot be higher than selling price."
			errs[fmt.Sprintf("variants.%d.selling", i)] = "Selling price must be the same as cost or higher."
		}
	}

	return errs
}

func validationFailed(errorsMap map[string]string) gin.H {
	if len(errorsMap) == 0 {
		errorsMap = map[string]string{"form": "Validation failed."}
	}
	return gin.H{
		"message": "Validation failed.",
		"errors":  errorsMap,
	}
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func boolValue(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func floatValue(value *float64, fallback float64) float64 {
	if value == nil {
		return fallback
	}
	return *value
}

func intValue(value *int, fallback int) int {
	if value == nil {
		return fallback
	}
	return *value
}

func cleanStrings(values []string) []string {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return cleaned
}

func hasBusinessRole(roles []auth.RoleResponse) bool {
	for _, role := range roles {
		if strings.EqualFold(role.Name, "business") {
			return true
		}
	}
	return false
}
