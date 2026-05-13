package repositories

import (
	"github.com/marketplace-ops/backend/internal/models"
	"gorm.io/gorm"
)

// AdminRepository handles database operations for Admin model.
type AdminRepository struct {
	db *gorm.DB
}

// NewAdminRepository creates a new AdminRepository.
func NewAdminRepository(db *gorm.DB) *AdminRepository {
	return &AdminRepository{db: db}
}

// FindByEmail finds an admin by email address.
func (r *AdminRepository) FindByEmail(email string) (*models.Admin, error) {
	var admin models.Admin
	result := r.db.Where("email = ?", email).First(&admin)
	if result.Error != nil {
		return nil, result.Error
	}
	return &admin, nil
}

// FindByID finds an admin by UUID.
func (r *AdminRepository) FindByID(id string) (*models.Admin, error) {
	var admin models.Admin
	result := r.db.Where("id = ?", id).First(&admin)
	if result.Error != nil {
		return nil, result.Error
	}
	return &admin, nil
}

// Create inserts a new admin record.
func (r *AdminRepository) Create(admin *models.Admin) error {
	return r.db.Create(admin).Error
}

// Update saves changes to an existing admin record.
func (r *AdminRepository) Update(admin *models.Admin) error {
	return r.db.Save(admin).Error
}
