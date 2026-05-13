package repositories

import (
	"errors"
	"strings"

	"github.com/marketplace-ops/backend/internal/models"
	"gorm.io/gorm"
)

type ProductRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

func (r *ProductRepository) FindAll(status string, category string, search string, sku string) ([]models.Product, error) {
	var products []models.Product
	query := r.db.Model(&models.Product{}).Preload("Images").Preload("Variants")

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if category != "" {
		query = query.Where("category = ?", category)
	}

	if search != "" {
		searchTerm := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(name) LIKE ?", searchTerm)
	}

	if sku != "" {
		query = query.Where("sku = ?", sku)
	}

	err := query.Order("created_at desc").Find(&products).Error
	return products, err
}

func (r *ProductRepository) FindByID(id string) (*models.Product, error) {
	var product models.Product
	err := r.db.Preload("Images").Preload("Variants").Where("id = ?", id).First(&product).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("product not found")
		}
		return nil, err
	}
	return &product, nil
}

func (r *ProductRepository) Create(product *models.Product) error {
	return r.db.Create(product).Error
}

func (r *ProductRepository) Update(product *models.Product) error {
	return r.db.Session(&gorm.Session{FullSaveAssociations: true}).Save(product).Error
}

func (r *ProductRepository) SoftDelete(id string) error {
	return r.db.Delete(&models.Product{}, "id = ?", id).Error // GORM uses deleted_at for soft delete
}

func (r *ProductRepository) CheckDuplicateSKU(sku string) (bool, error) {
	var count int64
	err := r.db.Model(&models.Product{}).
		Where("sku = ?", sku).
		Count(&count).Error
	return count > 0, err
}
