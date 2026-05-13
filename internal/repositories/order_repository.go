package repositories

import (
	"errors"
	"strings"

	"github.com/marketplace-ops/backend/internal/models"
	"gorm.io/gorm"
)

type OrderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) ListOrders(storeID, marketplace, status, paymentStatus, search, dateFrom, dateTo string) ([]models.Order, error) {
	var orders []models.Order
	query := r.db.Model(&models.Order{}).Preload("Store")

	if storeID != "" {
		query = query.Where("store_id = ?", storeID)
	}

	if marketplace != "" {
		query = query.Where("marketplace = ?", marketplace)
	}

	if status != "" {
		query = query.Where("order_status = ?", status)
	}

	if paymentStatus != "" {
		query = query.Where("payment_status = ?", paymentStatus)
	}

	if dateFrom != "" {
		query = query.Where("ordered_at >= ?", dateFrom)
	}

	if dateTo != "" {
		query = query.Where("ordered_at <= ?", dateTo)
	}

	if search != "" {
		searchTerm := "%" + strings.ToLower(search) + "%"
		query = query.Where("LOWER(order_number) LIKE ? OR LOWER(customer_name) LIKE ? OR LOWER(external_order_id) LIKE ?", searchTerm, searchTerm, searchTerm)
	}

	err := query.Order("created_at desc").Find(&orders).Error
	return orders, err
}

func (r *OrderRepository) GetByID(id string) (*models.Order, error) {
	var order models.Order
	err := r.db.Preload("Store").Preload("Items").Where("id = ?", id).First(&order).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("order not found")
		}
		return nil, err
	}
	return &order, nil
}

func (r *OrderRepository) GetByNumber(orderNumber string) (*models.Order, error) {
	var order models.Order
	err := r.db.Preload("Store").Preload("Items").Where("order_number = ?", orderNumber).First(&order).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("order not found")
		}
		return nil, err
	}
	return &order, nil
}

func (r *OrderRepository) Create(order *models.Order) error {
	return r.db.Create(order).Error
}

func (r *OrderRepository) Update(order *models.Order) error {
	return r.db.Save(order).Error
}

func (r *OrderRepository) SoftDelete(id string) error {
	return r.db.Delete(&models.Order{}, "id = ?", id).Error
}

func (r *OrderRepository) CheckDuplicateExternalOrder(storeID, externalOrderID string) (bool, error) {
	if externalOrderID == "" {
		return false, nil
	}
	var count int64
	err := r.db.Model(&models.Order{}).
		Where("store_id = ? AND external_order_id = ?", storeID, externalOrderID).
		Count(&count).Error
	return count > 0, err
}

// UpsertOrderWithItems safely creates or updates an order and its items in a transaction.
// Returns (created bool, err error).
func (r *OrderRepository) UpsertOrderWithItems(order *models.Order) (bool, error) {
	created := false
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var existing models.Order
		err := tx.Where("store_id = ? AND external_order_id = ?", order.StoreID, order.ExternalOrderID).First(&existing).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create new order with items
			created = true
			return tx.Create(order).Error
		}

		// Update existing order (safe fields only)
		existing.OrderStatus = order.OrderStatus
		existing.PaymentStatus = order.PaymentStatus
		existing.SubtotalAmount = order.SubtotalAmount
		existing.ShippingFee = order.ShippingFee
		existing.DiscountAmount = order.DiscountAmount
		existing.TotalAmount = order.TotalAmount
		existing.Currency = order.Currency
		existing.OrderedAt = order.OrderedAt
		existing.PaidAt = order.PaidAt
		existing.ShippedAt = order.ShippedAt
		existing.CompletedAt = order.CompletedAt
		existing.RawPayload = order.RawPayload

		if err := tx.Save(&existing).Error; err != nil {
			return err
		}

		// Replace items by deleting old ones and creating new ones
		if err := tx.Where("order_id = ?", existing.ID).Delete(&models.OrderItem{}).Error; err != nil {
			return err
		}

		for i := range order.Items {
			order.Items[i].OrderID = existing.ID
		}

		if len(order.Items) > 0 {
			if err := tx.Create(&order.Items).Error; err != nil {
				return err
			}
		}

		// Set order ID back to the object so caller knows it
		order.ID = existing.ID
		return nil
	})

	return created, err
}
