package main

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/marketplace-ops/backend/internal/config"
	"github.com/marketplace-ops/backend/internal/database"
	"github.com/marketplace-ops/backend/internal/handlers"
	"github.com/marketplace-ops/backend/internal/middleware"
	"github.com/marketplace-ops/backend/internal/repositories"
	"github.com/marketplace-ops/backend/internal/services"
	"github.com/marketplace-ops/backend/internal/workers"
)

func main() {
	cfg := config.Load()

	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	db := database.Connect(cfg)

	if cfg.IsDevelopment() {
		if err := database.AutoMigrate(db); err != nil {
			log.Fatalf("Auto-migration failed: %v", err)
		}
	}

	// Repositories
	adminRepo := repositories.NewAdminRepository(db)
	storeRepo := repositories.NewStoreRepository(db)
	productRepo := repositories.NewProductRepository(db)
	productMappingRepo := repositories.NewProductMappingRepository(db)
	inventoryRepo := repositories.NewInventoryRepository(db)
	orderRepo := repositories.NewOrderRepository(db)
	syncRepo := repositories.NewSyncRepository(db)
	dashboardRepo := repositories.NewDashboardRepository(db)
	integrationRepo := repositories.NewIntegrationRepository(db)

	database.SeedAdmin(db, cfg)

	jwtService := services.NewJWTService(cfg.JWTSecret)
	reservationService := services.NewInventoryReservationService(inventoryRepo, orderRepo)
	syncExecutionService := services.NewSyncExecutionService(syncRepo, storeRepo, integrationRepo, orderRepo, productMappingRepo, productRepo, inventoryRepo)

	// Workers
	syncWorker := workers.NewSyncWorker(syncRepo, syncExecutionService, cfg.SyncWorkerInterval, cfg.SyncWorkerEnabled)
	go syncWorker.Start(context.Background())

	// Handlers
	healthHandler := handlers.NewHealthHandler()
	authHandler := handlers.NewAuthHandler(adminRepo, jwtService)
	storeHandler := handlers.NewStoreHandler(storeRepo)
	productHandler := handlers.NewProductHandler(productRepo)
	productMappingHandler := handlers.NewProductMappingHandler(productMappingRepo, productRepo, storeRepo)
	inventoryHandler := handlers.NewInventoryHandler(inventoryRepo, productRepo, productMappingRepo)
	orderHandler := handlers.NewOrderHandler(orderRepo, storeRepo, reservationService)
	syncHandler := handlers.NewSyncHandler(syncRepo, storeRepo, syncExecutionService)
	dashboardHandler := handlers.NewDashboardHandler(dashboardRepo)
	integrationHandler := handlers.NewIntegrationHandler(integrationRepo, storeRepo, orderRepo, productMappingRepo, syncRepo, productRepo, inventoryRepo)

	router := gin.Default()
	router.Use(middleware.SetupCORS(cfg))

	api := router.Group("/api")

	api.GET("/health", healthHandler.Check)

	// OAuth callback route (semi-public, validates state internally)
	api.GET("/integrations/:marketplace/callback", integrationHandler.OAuthCallback)

	auth := api.Group("/auth")
	{
		auth.POST("/login", authHandler.Login)
		authProtected := auth.Group("")
		authProtected.Use(middleware.AuthRequired(jwtService))
		{
			authProtected.GET("/me", authHandler.Me)
			authProtected.POST("/logout", authHandler.Logout)
			authProtected.PATCH("/change-password", authHandler.ChangePassword)
		}
	}

	protected := api.Group("")
	protected.Use(middleware.AuthRequired(jwtService))
	{
		stores := protected.Group("/stores")
		{
			stores.GET("", storeHandler.ListStores)
			stores.POST("", storeHandler.CreateStore)
			stores.GET("/:id", storeHandler.GetStore)
			stores.PATCH("/:id", storeHandler.UpdateStore)
			stores.DELETE("/:id", storeHandler.DeleteStore)
		}

		products := protected.Group("/products")
		{
			products.GET("", productHandler.ListProducts)
			products.POST("", productHandler.CreateProduct)
			products.GET("/:id", productHandler.GetProduct)
			products.PATCH("/:id", productHandler.UpdateProduct)
			products.DELETE("/:id", productHandler.DeleteProduct)
		}

		mappings := protected.Group("/product-mappings")
		{
			mappings.GET("", productMappingHandler.ListMappings)
			mappings.POST("", productMappingHandler.CreateMapping)
			mappings.GET("/:id", productMappingHandler.GetMapping)
			mappings.PATCH("/:id", productMappingHandler.UpdateMapping)
			mappings.DELETE("/:id", productMappingHandler.DeleteMapping)
			mappings.GET("/product/:productId", productMappingHandler.ListMappingsByProduct)
			mappings.GET("/store/:storeId", productMappingHandler.ListMappingsByStore)
		}

		inventory := protected.Group("/inventory")
		{
			inventory.GET("", inventoryHandler.ListInventory)
			inventory.POST("", inventoryHandler.CreateInventoryItem)
			inventory.GET("/movements/all", inventoryHandler.ListAllMovements)
			inventory.GET("/:id", inventoryHandler.GetInventoryItem)
			inventory.PATCH("/:id", inventoryHandler.UpdateInventoryMetadata)
			inventory.DELETE("/:id", inventoryHandler.DeleteInventoryItem)
			inventory.POST("/:id/adjust", inventoryHandler.AdjustStock)
			inventory.GET("/:id/movements", inventoryHandler.ListMovements)
		}

		orders := protected.Group("/orders")
		{
			orders.GET("", orderHandler.ListOrders)
			orders.POST("", orderHandler.CreateOrder)
			orders.GET("/:id", orderHandler.GetOrder)
			orders.PATCH("/:id", orderHandler.UpdateOrder)
			orders.DELETE("/:id", orderHandler.DeleteOrder)
			orders.POST("/:id/reserve-stock", orderHandler.ReserveStock)
			orders.POST("/:id/release-reservation", orderHandler.ReleaseReservation)
		}

		sync := protected.Group("/sync")
		{
			sync.GET("/jobs", syncHandler.ListJobs)
			sync.POST("/jobs", syncHandler.CreateJob)
			sync.GET("/logs", syncHandler.ListLogs)
			sync.GET("/jobs/:id", syncHandler.GetJob)
			sync.PATCH("/jobs/:id", syncHandler.UpdateJob)
			sync.DELETE("/jobs/:id", syncHandler.DeleteJob)
			sync.POST("/jobs/:id/run", syncHandler.RunJob)
			sync.GET("/jobs/:id/logs", syncHandler.ListJobLogs)
		}

		dashboard := protected.Group("/dashboard")
		{
			dashboard.GET("/summary", dashboardHandler.GetSummary)
			dashboard.GET("/shopee-operations", dashboardHandler.GetShopeeOperations)
		}

		reports := protected.Group("/reports")
		{
			reports.GET("/orders", dashboardHandler.GetOrdersReport)
			reports.GET("/inventory", dashboardHandler.GetInventoryReport)
			reports.GET("/products", dashboardHandler.GetProductsReport)
			reports.GET("/sync", dashboardHandler.GetSyncReport)
			reports.GET("/shopee/reconciliation", dashboardHandler.GetShopeeReconciliation)
		}

		integrations := protected.Group("/integrations")
		{
			integrations.GET("", integrationHandler.ListIntegrations)
			integrations.GET("/marketplaces", integrationHandler.ListSupportedMarketplaces)
		}

		// Store-scoped integration endpoints
		stores.GET("/:id/integration", integrationHandler.GetStoreIntegration)
		stores.POST("/:id/integration/initiate", integrationHandler.InitiateIntegration)
		stores.POST("/:id/integration/disconnect", integrationHandler.DisconnectIntegration)
		stores.POST("/:id/integration/test", integrationHandler.TestConnection)
		stores.POST("/:id/integration/orders/pull", integrationHandler.PullOrders)
		stores.POST("/:id/integration/products/pull", integrationHandler.PullProducts)
		stores.POST("/:id/integration/products/mapping-candidates", integrationHandler.PreviewMappingCandidates)
		stores.POST("/:id/integration/products/mappings", integrationHandler.CreateMapping)
		stores.POST("/:id/integration/stock/push", integrationHandler.PushStock)

		orders.POST("/:id/confirm-sale", orderHandler.ConfirmSale)
	}

	log.Printf("Starting server on port %s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
