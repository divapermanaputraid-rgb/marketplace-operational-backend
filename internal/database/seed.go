package database

import (
	"fmt"
	"log"

	"github.com/marketplace-ops/backend/internal/config"
	"github.com/marketplace-ops/backend/internal/models"
	"gorm.io/gorm"
)

// SeedAdmin creates the default admin account if it does not already exist.
// Uses bcrypt to hash the password. Never stores plain text.
func SeedAdmin(db *gorm.DB, cfg *config.Config) {
	if cfg.AdminSeedEmail == "" || cfg.AdminSeedPassword == "" {
		log.Println("⚠️  Admin seed skipped: ADMIN_SEED_EMAIL or ADMIN_SEED_PASSWORD not set")
		return
	}

	// Check if admin already exists
	var existing models.Admin
	result := db.Where("email = ?", cfg.AdminSeedEmail).First(&existing)
	if result.Error == nil {
		fmt.Printf("ℹ️  Admin already exists: %s\n", cfg.AdminSeedEmail)
		return
	}

	// Create new admin
	admin := models.Admin{
		Name:   cfg.AdminSeedName,
		Email:  cfg.AdminSeedEmail,
		Status: "active",
	}

	if err := admin.SetPassword(cfg.AdminSeedPassword); err != nil {
		log.Fatalf("Failed to hash admin password: %v", err)
	}

	if err := db.Create(&admin).Error; err != nil {
		log.Fatalf("Failed to seed admin: %v", err)
	}

	fmt.Printf("✅ Admin seeded: %s (%s)\n", admin.Email, admin.Name)
}
