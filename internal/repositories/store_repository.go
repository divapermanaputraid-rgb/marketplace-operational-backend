package repositories

import (
	"errors"
	"strings"

	"github.com/marketplace-ops/backend/internal/models"
	"gorm.io/gorm"
)

type StoreRepository struct {
	db *gorm.DB
}

func NewStoreRepository(db *gorm.DB) *StoreRepository {
	return &StoreRepository{db: db}
}

func (r *StoreRepository) FindAll(marketplace string, status string, search string) ([]models.Store, error) {
	var stores []models.Store
	query := r.db.Model(&models.Store{}).Where("is_active = ?", true)

	if marketplace != "" {
		query = query.Where("marketplace = ?", marketplace)
	}

	if status != "" {
		query = query.Where("connection_status = ?", status)
	}

	if search != "" {
		searchTerm := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(store_name) LIKE ?", searchTerm)
	}

	err := query.Order("created_at desc").Find(&stores).Error
	return stores, err
}

func (r *StoreRepository) FindByID(id string) (*models.Store, error) {
	var store models.Store
	err := r.db.Where("id = ? AND is_active = ?", id, true).First(&store).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("store not found")
		}
		return nil, err
	}
	return &store, nil
}

func (r *StoreRepository) Create(store *models.Store) error {
	return r.db.Create(store).Error
}

func (r *StoreRepository) Update(store *models.Store) error {
	return r.db.Save(store).Error
}

func (r *StoreRepository) SoftDelete(id string) error {
	return r.db.Model(&models.Store{}).Where("id = ?", id).Update("is_active", false).Error
}

func (r *StoreRepository) CheckDuplicateStore(marketplace string, storeName string) (bool, error) {
	var count int64
	err := r.db.Model(&models.Store{}).
		Where("marketplace = ? AND LOWER(store_name) = ? AND is_active = ?", marketplace, strings.ToLower(storeName), true).
		Count(&count).Error
	return count > 0, err
}
