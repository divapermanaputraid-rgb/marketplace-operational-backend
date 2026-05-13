package repositories

import (
	"errors"
	"strings"

	"github.com/marketplace-ops/backend/internal/models"
	"gorm.io/gorm"
)

type InventoryRepository struct {
	db *gorm.DB
}

func NewInventoryRepository(db *gorm.DB) *InventoryRepository {
	return &InventoryRepository{db: db}
}

func (r *InventoryRepository) ListItems(productID, variantID, search, location, lowStock string) ([]models.InventoryItem, error) {
	var items []models.InventoryItem
	query := r.db.Model(&models.InventoryItem{}).
		Preload("Product").
		Preload("ProductVariant")

	if productID != "" {
		query = query.Where("product_id = ?", productID)
	}

	if variantID != "" {
		query = query.Where("product_variant_id = ?", variantID)
	}

	if location != "" {
		query = query.Where("location_name = ?", location)
	}

	if lowStock == "true" {
		query = query.Where("available_quantity <= safety_stock")
	}

	if search != "" {
		searchTerm := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(sku) LIKE ?", searchTerm)
	}

	err := query.Order("created_at desc").Find(&items).Error
	return items, err
}

func (r *InventoryRepository) GetByID(id string) (*models.InventoryItem, error) {
	var item models.InventoryItem
	err := r.db.Preload("Product").Preload("ProductVariant").Where("id = ?", id).First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("inventory item not found")
		}
		return nil, err
	}
	return &item, nil
}

func (r *InventoryRepository) GetByProductAndVariant(productID, variantID string, location string) (*models.InventoryItem, error) {
	var item models.InventoryItem
	query := r.db.Where("product_id = ? AND location_name = ?", productID, location)

	if variantID != "" {
		query = query.Where("product_variant_id = ?", variantID)
	} else {
		query = query.Where("product_variant_id IS NULL")
	}

	err := query.First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Return nil, nil if not found to allow creation
		}
		return nil, err
	}
	return &item, nil
}

func (r *InventoryRepository) Create(item *models.InventoryItem) error {
	return r.db.Create(item).Error
}

func (r *InventoryRepository) Update(item *models.InventoryItem) error {
	return r.db.Save(item).Error
}

func (r *InventoryRepository) SoftDelete(id string) error {
	return r.db.Delete(&models.InventoryItem{}, "id = ?", id).Error
}

func (r *InventoryRepository) CreateMovement(tx *gorm.DB, movement *models.InventoryMovement) error {
	if tx != nil {
		return tx.Create(movement).Error
	}
	return r.db.Create(movement).Error
}

func (r *InventoryRepository) ListMovements(itemID string, movementType string) ([]models.InventoryMovement, error) {
	var movements []models.InventoryMovement
	query := r.db.Model(&models.InventoryMovement{})

	if itemID != "" {
		query = query.Where("inventory_item_id = ?", itemID)
	}

	if movementType != "" {
		query = query.Where("movement_type = ?", movementType)
	}

	err := query.Order("created_at desc").Find(&movements).Error
	return movements, err
}

func (r *InventoryRepository) GetMovementByReference(itemID string, refType string, refID string, movementType string) (*models.InventoryMovement, error) {
	var movement models.InventoryMovement
	err := r.db.Where("inventory_item_id = ? AND reference_type = ? AND reference_id = ? AND movement_type = ?", itemID, refType, refID, movementType).First(&movement).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &movement, nil
}

func (r *InventoryRepository) ListMovementsByReference(refType string, refID string) ([]models.InventoryMovement, error) {
	var movements []models.InventoryMovement
	err := r.db.Where("reference_type = ? AND reference_id = ?", refType, refID).Find(&movements).Error
	return movements, err
}

func (r *InventoryRepository) WithTransaction(fn func(tx *gorm.DB) error) error {
	return r.db.Transaction(fn)
}
