package database

import (
	"fmt"
	"log"

	"github.com/marketplace-ops/backend/internal/config"
	"github.com/marketplace-ops/backend/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connect establishes a connection to Supabase PostgreSQL using GORM.
// Uses PreferSimpleProtocol to work with Supabase transaction pooler (port 6543).
func Connect(cfg *config.Config) *gorm.DB {
	// Choose GORM log level based on environment
	logLevel := logger.Warn
	if cfg.IsDevelopment() {
		logLevel = logger.Info
	}

	// PreferSimpleProtocol disables prepared statements which are not
	// supported by Supabase's transaction pooler (PgBouncer on port 6543).
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  cfg.DatabaseURL,
		PreferSimpleProtocol: true,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Get underlying sql.DB to configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get underlying database connection: %v", err)
	}

	// Conservative pool settings for Supabase Free tier
	sqlDB.SetMaxOpenConns(5)
	sqlDB.SetMaxIdleConns(2)

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	fmt.Println("✅ Database connected successfully")
	return db
}

// AutoMigrate runs GORM AutoMigrate for development convenience.
// Production should use SQL migration files.
func AutoMigrate(db *gorm.DB) error {
	fmt.Println("🔄 Running development AutoMigrate...")

	err := db.AutoMigrate(
		&models.Admin{},
		&models.Store{},
		&models.MarketplaceConnection{},
		&models.Product{},
		&models.ProductImage{},
		&models.ProductVariant{},
		&models.MarketplaceProductMapping{},
		&models.InventoryItem{},
		&models.InventoryMovement{},
		&models.Order{},
		&models.OrderItem{},
		&models.SyncJob{},
		&models.SyncLog{},
	)
	if err != nil {
		return fmt.Errorf("auto-migration failed: %w", err)
	}

	fmt.Println("✅ AutoMigrate completed successfully")
	return nil
}
