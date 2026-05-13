package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/marketplace-ops/backend/internal/config"
	"github.com/marketplace-ops/backend/internal/database"
	"github.com/marketplace-ops/backend/internal/handlers"
	"github.com/marketplace-ops/backend/internal/middleware"
	"github.com/marketplace-ops/backend/internal/repositories"
	"github.com/marketplace-ops/backend/internal/services"
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

	database.SeedAdmin(db, cfg)

	jwtService := services.NewJWTService(cfg.JWTSecret)

	// Handlers
	healthHandler := handlers.NewHealthHandler()
	authHandler := handlers.NewAuthHandler(adminRepo, jwtService)
	storeHandler := handlers.NewStoreHandler(storeRepo)
	productHandler := handlers.NewProductHandler(productRepo)
	productMappingHandler := handlers.NewProductMappingHandler(productMappingRepo, productRepo, storeRepo)
	inventoryHandler := handlers.NewInventoryHandler(inventoryRepo, productRepo, productMappingRepo)
	orderHandler := handlers.NewOrderHandler(orderRepo, storeRepo)
	syncHandler := handlers.NewSyncHandler(syncRepo, storeRepo)
	dashboardHandler := handlers.NewDashboardHandler(dashboardRepo)

	router := gin.Default()
	router.Use(middleware.SetupCORS(cfg))

	api := router.Group("/api")

	api.GET("/health", healthHandler.Check)

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
			dashboard.GET("/reports/orders", dashboardHandler.GetOrdersReport)
			dashboard.GET("/reports/inventory", dashboardHandler.GetInventoryReport)
			dashboard.GET("/reports/products", dashboardHandler.GetProductsReport)
			dashboard.GET("/reports/sync", dashboardHandler.GetSyncReport)
		}
	}

	log.Printf("Starting server on port %s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
