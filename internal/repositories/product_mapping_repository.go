package repositories

import (
	"errors"
	"strings"

	"github.com/marketplace-ops/backend/internal/models"
	"gorm.io/gorm"
)

type ProductMappingRepository struct {
	db *gorm.DB
}

func NewProductMappingRepository(db *gorm.DB) *ProductMappingRepository {
	return &ProductMappingRepository{db: db}
}

func (r *ProductMappingRepository) ListMappings(productID, storeID, marketplace, listingStatus, search string) ([]models.MarketplaceProductMapping, error) {
	var mappings []models.MarketplaceProductMapping
	query := r.db.Model(&models.MarketplaceProductMapping{}).
		Preload("Product").
		Preload("ProductVariant").
		Preload("Store")

	if productID != "" {
		query = query.Where("product_id = ?", productID)
	}

	if storeID != "" {
		query = query.Where("store_id = ?", storeID)
	}

	if marketplace != "" {
		query = query.Where("marketplace = ?", marketplace)
	}

	if listingStatus != "" {
		query = query.Where("listing_status = ?", listingStatus)
	}

	if search != "" {
		searchTerm := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(listing_name) LIKE ? OR LOWER(external_product_id) LIKE ? OR LOWER(external_sku) LIKE ?", searchTerm, searchTerm, searchTerm)
	}

	err := query.Order("created_at desc").Find(&mappings).Error
	return mappings, err
}

func (r *ProductMappingRepository) GetByID(id string) (*models.MarketplaceProductMapping, error) {
	var mapping models.MarketplaceProductMapping
	err := r.db.Preload("Product").Preload("ProductVariant").Preload("Store").Where("id = ?", id).First(&mapping).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("product mapping not found")
		}
		return nil, err
	}
	return &mapping, nil
}

func (r *ProductMappingRepository) ListByProduct(productID string) ([]models.MarketplaceProductMapping, error) {
	var mappings []models.MarketplaceProductMapping
	err := r.db.Preload("Store").Where("product_id = ?", productID).Order("created_at desc").Find(&mappings).Error
	return mappings, err
}

func (r *ProductMappingRepository) ListByStore(storeID string) ([]models.MarketplaceProductMapping, error) {
	var mappings []models.MarketplaceProductMapping
	err := r.db.Preload("Product").Preload("ProductVariant").Where("store_id = ?", storeID).Order("created_at desc").Find(&mappings).Error
	return mappings, err
}

func (r *ProductMappingRepository) Create(mapping *models.MarketplaceProductMapping) error {
	return r.db.Create(mapping).Error
}

func (r *ProductMappingRepository) Update(mapping *models.MarketplaceProductMapping) error {
	return r.db.Save(mapping).Error
}

func (r *ProductMappingRepository) Delete(id string) error {
	return r.db.Delete(&models.MarketplaceProductMapping{}, "id = ?", id).Error
}

func (r *ProductMappingRepository) CheckDuplicateMapping(storeID, externalProductID, externalVariantID string) (bool, error) {
	var count int64
	err := r.db.Model(&models.MarketplaceProductMapping{}).
		Where("store_id = ?", storeID).
		Where("external_product_id = ?", externalProductID).
		Where("COALESCE(external_variant_id, '') = ?", externalVariantID).
		Count(&count).Error
	return count > 0, err
}

func (r *ProductMappingRepository) FindByExternalID(storeID, externalProductID, externalVariantID string) (*models.MarketplaceProductMapping, error) {
	var mapping models.MarketplaceProductMapping
	err := r.db.Where("store_id = ?", storeID).
		Where("external_product_id = ?", externalProductID).
		Where("COALESCE(external_variant_id, '') = ?", externalVariantID).
		First(&mapping).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // Return nil, nil when not found
		}
		return nil, err
	}
	return &mapping, nil
}
