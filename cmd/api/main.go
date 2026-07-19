package main

import (
	"log"
	"net/http"
	"pos/internal/auth"
	"pos/internal/config"
	"pos/internal/database"
	adminhandler "pos/internal/handlers/admin"
	brandhandler "pos/internal/handlers/business/brand"
	categoryhandler "pos/internal/handlers/business/category"
	customerhandler "pos/internal/handlers/business/customer"
	locationhandler "pos/internal/handlers/business/location"
	openingstockhandler "pos/internal/handlers/business/openingstock"
	producthandler "pos/internal/handlers/business/product"
	purchaseorderhandler "pos/internal/handlers/business/purchaseorder"
	saleshandler "pos/internal/handlers/business/sales"
	settingshandler "pos/internal/handlers/business/settings"
	subcategoryhandler "pos/internal/handlers/business/subcategory"
	supplierhandler "pos/internal/handlers/business/supplier"
	unithandler "pos/internal/handlers/business/unit"
	warrantyhandler "pos/internal/handlers/business/warranty"
	"pos/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
)

func main() {
	var conf *config.Config
	var err error
	conf, err = config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	var pool *pgxpool.Pool
	pool, err = database.Connect(conf.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	defer pool.Close()

	router := gin.Default()
	router.SetTrustedProxies(nil)
	router.Use(middleware.CORS(conf.FrontendOrigin))
	router.Use(middleware.RequestLogger())

	api := router.Group("/api")
	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	authService := auth.NewService(pool, conf.CookieSecure)
	authService.RegisterRoutes(api.Group("/auth"))
	api.POST("/businesses", adminhandler.CreateBusinessRequestHandler(pool))
	api.GET("/admin/businesses", adminhandler.ListBusinessesRequestHandler(pool, authService))
	api.POST("/admin/businesses/:id/sync-modules", adminhandler.SyncBusinessModulesRequestHandler(pool, authService))
	api.POST("/packages", adminhandler.CreatePackageRequestHandler(pool))
	api.GET("/admin/roles", adminhandler.ListRolesRequestHandler(pool, authService))
	api.GET("/admin/modules", adminhandler.ListModulesRequestHandler(pool, authService))
	api.POST("/admin/modules", adminhandler.CreateModuleRequestHandler(pool))
	api.PATCH("/admin/modules/reorder", adminhandler.ReorderModulesRequestHandler(pool))
	api.POST("/admin/modules/submodules", adminhandler.CreateSubmoduleRequestHandler(pool))
	api.PATCH("/admin/modules/submodules/:id", adminhandler.UpdateSubmoduleRequestHandler(pool))
	api.PATCH("/admin/modules/submodules/reorder", adminhandler.ReorderSubmodulesRequestHandler(pool))
	api.POST("/categories", categoryhandler.CreateCategoryRequestHandler(pool, authService))
	api.GET("/categories", categoryhandler.ListCategoriesRequestHandler(pool, authService))
	api.GET("/categories/export", categoryhandler.ExportCategoriesRequestHandler(pool, authService))
	api.DELETE("/categories/:id", categoryhandler.DeleteCategoryRequestHandler(pool, authService))
	api.POST("/products", producthandler.CreateProductRequestHandler(pool, authService))
	api.GET("/products/import/template.csv", producthandler.DownloadProductImportTemplateRequestHandler(authService))
	api.POST("/products/import/preview", producthandler.PreviewProductImportRequestHandler(pool, authService))
	api.GET("/products/import/batches/latest", producthandler.LatestProductImportBatchRequestHandler(pool, authService))
	api.GET("/products/import/batches/:batchId", producthandler.ListProductImportBatchRequestHandler(pool, authService))
	api.PUT("/products/import/batches/:batchId/rows/:rowId", producthandler.UpdateProductImportRowRequestHandler(pool, authService))
	api.POST("/products/import/batches/:batchId/rows/:rowId/import", producthandler.ImportProductImportRowRequestHandler(pool, authService))
	api.GET("/products/opening-stock/import/template.csv", openingstockhandler.DownloadOpeningStockImportTemplateRequestHandler(authService))
	api.POST("/products/opening-stock/import/preview", openingstockhandler.PreviewOpeningStockImportRequestHandler(pool, authService))
	api.GET("/products/opening-stock/import/batches/latest", openingstockhandler.LatestOpeningStockImportBatchRequestHandler(pool, authService))
	api.GET("/products/opening-stock/import/batches/:batchId", openingstockhandler.ListOpeningStockImportBatchRequestHandler(pool, authService))
	api.PUT("/products/opening-stock/import/batches/:batchId/rows/:rowId", openingstockhandler.UpdateOpeningStockImportRowRequestHandler(pool, authService))
	api.POST("/products/opening-stock/import/batches/:batchId/rows/:rowId/import", openingstockhandler.ImportOpeningStockImportRowRequestHandler(pool, authService))
	api.GET("/products", producthandler.ListProductsRequestHandler(pool, authService))
	api.GET("/products/search", producthandler.SearchProductsRequestHandler(pool, authService))
	api.GET("/products/:id", producthandler.GetProductRequestHandler(pool, authService))
	api.GET("/products/:id/price-history", producthandler.ListProductPriceHistoryRequestHandler(pool, authService))
	api.PUT("/products/:id", producthandler.UpdateProductRequestHandler(pool, authService))
	api.GET("/brands", brandhandler.ListBrandsRequestHandler(pool, authService))
	api.POST("/brands", brandhandler.CreateBrandRequestHandler(pool, authService))
	api.PUT("/brands/:id", brandhandler.UpdateBrandRequestHandler(pool, authService))
	api.DELETE("/brands/:id", brandhandler.DeleteBrandRequestHandler(pool, authService))
	api.GET("/warranties", warrantyhandler.ListWarrantiesRequestHandler(pool, authService))
	api.POST("/warranties", warrantyhandler.CreateWarrantyRequestHandler(pool, authService))
	api.PUT("/warranties/:id", warrantyhandler.UpdateWarrantyRequestHandler(pool, authService))
	api.DELETE("/warranties/:id", warrantyhandler.DeleteWarrantyRequestHandler(pool, authService))
	api.GET("/sub-categories", subcategoryhandler.ListSubCategoriesRequestHandler(pool, authService))
	api.POST("/sub-categories", subcategoryhandler.CreateSubCategoryRequestHandler(pool, authService))
	api.PUT("/sub-categories/:id", subcategoryhandler.UpdateSubCategoryRequestHandler(pool, authService))
	api.DELETE("/sub-categories/:id", subcategoryhandler.DeleteSubCategoryRequestHandler(pool, authService))
	api.GET("/business/locations", locationhandler.GetBusinessLocationsRequestHandler(pool, authService))
	api.POST("/business/locations", locationhandler.CreateBusinessLocationRequestHandler(pool, authService))
	api.DELETE("/business/locations/:id", locationhandler.DeleteBusinessLocationRequestHandler(pool, authService))
	api.GET("/business/customers", customerhandler.ListBusinessCustomersRequestHandler(pool, authService))
	api.POST("/business/customers", customerhandler.CreateBusinessCustomerRequestHandler(pool, authService))
	api.PUT("/business/customers/:id", customerhandler.UpdateBusinessCustomerRequestHandler(pool, authService))
	api.DELETE("/business/customers/:id", customerhandler.DeleteBusinessCustomerRequestHandler(pool, authService))
	api.GET("/business/units", unithandler.GetBusinessUnitsRequestHandler(pool, authService))
	api.POST("/business/units", unithandler.CreateBusinessUnitRequestHandler(pool, authService))
	api.PUT("/business/units/:id", unithandler.UpdateBusinessUnitRequestHandler(pool, authService))
	api.DELETE("/business/units/:id", unithandler.DeleteBusinessUnitRequestHandler(pool, authService))
	api.GET("/business/suppliers", supplierhandler.ListBusinessSuppliersRequestHandler(pool, authService))
	api.POST("/business/suppliers", supplierhandler.CreateBusinessSupplierRequestHandler(pool, authService))
	api.GET("/business/suppliers/:id/stock-report", supplierhandler.GetBusinessSupplierStockReportRequestHandler(pool, authService))
	api.GET("/purchases/orders", purchaseorderhandler.ListPurchaseOrdersRequestHandler(pool, authService))
	api.GET("/purchases/order-statuses", purchaseorderhandler.ListPurchaseOrderStatusesRequestHandler(pool, authService))
	api.GET("/purchases/orders/:id", purchaseorderhandler.GetPurchaseOrderRequestHandler(pool, authService))
	api.GET("/purchases/returns/search", purchaseorderhandler.SearchPurchaseReturnableStockRequestHandler(pool, authService))
	api.GET("/purchases/returns", purchaseorderhandler.ListPurchaseReturnsRequestHandler(pool, authService))
	api.POST("/purchases/returns", purchaseorderhandler.CreatePurchaseReturnRequestHandler(pool, authService))
	api.GET("/purchases/returns/:id", purchaseorderhandler.GetPurchaseReturnRequestHandler(pool, authService))
	api.PUT("/purchases/returns/:id", purchaseorderhandler.UpdatePurchaseReturnRequestHandler(pool, authService))
	api.DELETE("/purchases/returns/:id", purchaseorderhandler.DeletePurchaseReturnRequestHandler(pool, authService))
	api.GET("/purchases/returns/export/csv", purchaseorderhandler.ExportPurchaseReturnsCSVRequestHandler(pool, authService))
	api.GET("/purchases/returns/export/pdf", purchaseorderhandler.ExportPurchaseReturnsPDFRequestHandler(pool, authService))
	api.GET("/purchases/returns/:id/export/pdf", purchaseorderhandler.ExportPurchaseReturnPDFRequestHandler(pool, authService))
	api.GET("/purchases/orders/export/csv", purchaseorderhandler.ExportPurchaseOrdersCSVRequestHandler(pool, authService))
	api.GET("/purchases/orders/export/pdf", purchaseorderhandler.ExportPurchaseOrdersPDFRequestHandler(pool, authService))
	api.GET("/purchases/orders/:id/export/pdf", purchaseorderhandler.ExportPurchaseOrderPDFRequestHandler(pool, authService))
	api.POST("/purchases/orders/:id/notify", purchaseorderhandler.SendPurchaseOrderNotificationRequestHandler(pool, authService))
	api.POST("/purchases/orders", purchaseorderhandler.CreatePurchaseOrderRequestHandler(pool, authService))
	api.PUT("/purchases/orders/:id", purchaseorderhandler.UpdatePurchaseOrderRequestHandler(pool, authService))
	api.DELETE("/purchases/orders/:id", purchaseorderhandler.DeletePurchaseOrderRequestHandler(pool, authService))
	api.POST("/sales/orders", saleshandler.CreateSaleOrderRequestHandler(pool, authService))
	api.GET("/sales/order-statuses", saleshandler.ListSalesOrderStatusesRequestHandler(pool, authService))
	api.GET("/sales", saleshandler.ListSalesRequestHandler(pool, authService))
	api.GET("/sales/orders", saleshandler.ListSalesOrdersRequestHandler(pool, authService))
	api.GET("/sales/orders/:id", saleshandler.GetSalesOrderRequestHandler(pool, authService))
	api.PATCH("/sales/orders/:id", saleshandler.UpdateSaleOrderRequestHandler(pool, authService))
	api.PATCH("/sales/orders/:id/status", saleshandler.UpdateSalesOrderStatusRequestHandler(pool, authService))
	api.DELETE("/sales/orders/:id", saleshandler.DeleteSalesOrderRequestHandler(pool, authService))
	api.GET("/business/settings", settingshandler.GetBusinessSettingsRequestHandler(pool, authService))
	api.PUT("/business/settings", settingshandler.UpdateBusinessSettingsRequestHandler(pool, authService))
	api.GET("/business/settings/invoice", settingshandler.GetBusinessInvoiceSettingsRequestHandler(pool, authService))
	api.POST("/business/settings/invoice", settingshandler.CreateBusinessInvoiceSettingsRequestHandler(pool, authService))
	api.GET("/business/settings/invoice/:id", settingshandler.GetBusinessInvoiceSettingRequestHandler(pool, authService))
	api.PUT("/business/settings/invoice/:id", settingshandler.UpdateBusinessInvoiceSettingsRequestHandler(pool, authService))
	api.GET("/business/settings/product", settingshandler.GetBusinessProductSettingsRequestHandler(pool, authService))
	api.PUT("/business/settings/product", settingshandler.UpdateBusinessProductSettingsRequestHandler(pool, authService))
	api.GET("/business/settings/contact", settingshandler.GetBusinessContactSettingsRequestHandler(pool, authService))
	api.PUT("/business/settings/contact", settingshandler.UpdateBusinessContactSettingsRequestHandler(pool, authService))
	api.GET("/business/settings/sale", settingshandler.GetBusinessSaleSettingsRequestHandler(pool, authService))
	api.PUT("/business/settings/sale", settingshandler.UpdateBusinessSaleSettingsRequestHandler(pool, authService))
	api.GET("/business/settings/pos", settingshandler.GetBusinessPosSettingsRequestHandler(pool, authService))
	api.PUT("/business/settings/pos", settingshandler.UpdateBusinessPosSettingsRequestHandler(pool, authService))
	api.GET("/business/settings/purchases", settingshandler.GetBusinessPurchasesSettingsRequestHandler(pool, authService))
	api.PUT("/business/settings/purchases", settingshandler.UpdateBusinessPurchasesSettingsRequestHandler(pool, authService))

	router.Run(":" + conf.Port)
}
