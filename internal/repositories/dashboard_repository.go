package repositories

import (
	"github.com/marketplace-ops/backend/internal/models"
	"gorm.io/gorm"
)

type DashboardRepository struct {
	db *gorm.DB
}

func NewDashboardRepository(db *gorm.DB) *DashboardRepository {
	return &DashboardRepository{db: db}
}

// Store metrics
func (r *DashboardRepository) GetStoreMetrics() (total int64, active int64, byMarketplace map[string]int64, err error) {
	byMarketplace = make(map[string]int64)

	err = r.db.Model(&models.Store{}).Count(&total).Error
	if err != nil {
		return
	}

	err = r.db.Model(&models.Store{}).Where("is_active = ?", true).Count(&active).Error
	if err != nil {
		return
	}

	var results []struct {
		Marketplace string
		Count       int64
	}
	err = r.db.Model(&models.Store{}).Select("marketplace, count(*) as count").Group("marketplace").Find(&results).Error
	if err != nil {
		return
	}

	for _, res := range results {
		byMarketplace[res.Marketplace] = res.Count
	}

	return
}

// Product metrics
func (r *DashboardRepository) GetProductMetrics() (total int64, active int64, draft int64, inactive int64, archived int64, err error) {
	err = r.db.Model(&models.Product{}).Count(&total).Error
	if err != nil {
		return
	}

	var results []struct {
		Status string
		Count  int64
	}
	err = r.db.Model(&models.Product{}).Select("status, count(*) as count").Group("status").Find(&results).Error
	if err != nil {
		return
	}

	for _, res := range results {
		switch res.Status {
		case "active":
			active = res.Count
		case "draft":
			draft = res.Count
		case "inactive":
			inactive = res.Count
		case "archived":
			archived = res.Count
		}
	}

	return
}

// Mapping metrics
func (r *DashboardRepository) GetMappingMetrics() (total int64, mappedProducts int64, unmappedProducts int64, err error) {
	err = r.db.Model(&models.MarketplaceProductMapping{}).Count(&total).Error
	if err != nil {
		return
	}

	err = r.db.Model(&models.MarketplaceProductMapping{}).Distinct("product_id").Count(&mappedProducts).Error
	if err != nil {
		return
	}

	var totalProducts int64
	err = r.db.Model(&models.Product{}).Count(&totalProducts).Error
	if err != nil {
		return
	}

	unmappedProducts = totalProducts - mappedProducts
	if unmappedProducts < 0 {
		unmappedProducts = 0
	}

	return
}

// Inventory metrics
func (r *DashboardRepository) GetInventoryMetrics() (totalItems int64, lowStock int64, totalAvailable int64, totalReserved int64, totalDamaged int64, err error) {
	err = r.db.Model(&models.InventoryItem{}).Count(&totalItems).Error
	if err != nil {
		return
	}

	err = r.db.Model(&models.InventoryItem{}).Where("available_quantity <= safety_stock").Count(&lowStock).Error
	if err != nil {
		return
	}

	var totals struct {
		Available int64
		Reserved  int64
		Damaged   int64
	}

	// Use Raw query for aggregation as GORM Select with SUM into struct can be tricky sometimes
	err = r.db.Raw("SELECT COALESCE(SUM(available_quantity), 0) as available, COALESCE(SUM(reserved_quantity), 0) as reserved, COALESCE(SUM(damaged_quantity), 0) as damaged FROM inventory_items WHERE deleted_at IS NULL").Scan(&totals).Error
	if err != nil {
		return
	}

	totalAvailable = totals.Available
	totalReserved = totals.Reserved
	totalDamaged = totals.Damaged

	return
}

// Order metrics
func (r *DashboardRepository) GetOrderMetrics() (total int64, statusCounts map[string]int64, paymentCounts map[string]int64, salesAmount float64, err error) {
	statusCounts = make(map[string]int64)
	paymentCounts = make(map[string]int64)

	err = r.db.Model(&models.Order{}).Count(&total).Error
	if err != nil {
		return
	}

	var statusRes []struct {
		OrderStatus string
		Count       int64
	}
	err = r.db.Model(&models.Order{}).Select("order_status, count(*) as count").Group("order_status").Find(&statusRes).Error
	if err != nil {
		return
	}
	for _, res := range statusRes {
		statusCounts[res.OrderStatus] = res.Count
	}

	var paymentRes []struct {
		PaymentStatus string
		Count         int64
	}
	err = r.db.Model(&models.Order{}).Select("payment_status, count(*) as count").Group("payment_status").Find(&paymentRes).Error
	if err != nil {
		return
	}
	for _, res := range paymentRes {
		paymentCounts[res.PaymentStatus] = res.Count
	}

	// Calculate total sales amount (only for paid/completed orders usually, but we'll include all paid for simplicity)
	var sum float64
	err = r.db.Model(&models.Order{}).Where("payment_status = ?", "paid").Select("COALESCE(SUM(total_amount), 0)").Scan(&sum).Error
	if err != nil {
		return
	}
	salesAmount = sum

	return
}

// Sync metrics
func (r *DashboardRepository) GetSyncMetrics() (totalJobs int64, notConfigured int64, failed int64, partial int64, success int64, latestLogs []models.SyncLog, err error) {
	err = r.db.Model(&models.SyncJob{}).Count(&totalJobs).Error
	if err != nil {
		return
	}

	err = r.db.Model(&models.SyncLog{}).Where("status = ?", "not_configured").Count(&notConfigured).Error
	if err != nil {
		return
	}

	err = r.db.Model(&models.SyncLog{}).Where("status = ?", "failed").Count(&failed).Error
	if err != nil {
		return
	}

	err = r.db.Model(&models.SyncLog{}).Where("status = ?", "partial").Count(&partial).Error
	if err != nil {
		return
	}

	err = r.db.Model(&models.SyncLog{}).Where("status = ?", "success").Count(&success).Error
	if err != nil {
		return
	}

	err = r.db.Preload("SyncJob").Order("created_at desc").Limit(5).Find(&latestLogs).Error
	return
}

// Report specific queries
func (r *DashboardRepository) GetRecentOrders(limit int) ([]models.Order, error) {
	var orders []models.Order
	err := r.db.Preload("Store").Order("created_at desc").Limit(limit).Find(&orders).Error
	return orders, err
}

func (r *DashboardRepository) GetSalesByMarketplace() (map[string]float64, error) {
	var results []struct {
		Marketplace string
		Total       float64
	}
	err := r.db.Model(&models.Order{}).Where("payment_status = ?", "paid").Select("marketplace, COALESCE(SUM(total_amount), 0) as total").Group("marketplace").Find(&results).Error
	if err != nil {
		return nil, err
	}

	salesMap := make(map[string]float64)
	for _, res := range results {
		salesMap[res.Marketplace] = res.Total
	}
	return salesMap, nil
}

func (r *DashboardRepository) GetLowStockItems(limit int) ([]models.InventoryItem, error) {
	var items []models.InventoryItem
	err := r.db.Preload("Product").Preload("ProductVariant").Where("available_quantity <= safety_stock").Order("available_quantity asc").Limit(limit).Find(&items).Error
	return items, err
}

func (r *DashboardRepository) GetUnmappedProducts(limit int) ([]models.Product, error) {
	var products []models.Product

	// Products that don't have an entry in marketplace_product_mappings
	err := r.db.Where("id NOT IN (SELECT DISTINCT product_id FROM marketplace_product_mappings WHERE deleted_at IS NULL)").Limit(limit).Find(&products).Error

	return products, err
}

func (r *DashboardRepository) GetMappingsByMarketplace() (map[string]int64, error) {
	var results []struct {
		Marketplace string
		Count       int64
	}
	err := r.db.Model(&models.MarketplaceProductMapping{}).Select("marketplace, count(*) as count").Group("marketplace").Find(&results).Error
	if err != nil {
		return nil, err
	}

	countMap := make(map[string]int64)
	for _, res := range results {
		countMap[res.Marketplace] = res.Count
	}
	return countMap, nil
}

func (r *DashboardRepository) GetSyncJobStatusCounts() (map[string]int64, error) {
	var results []struct {
		Status string
		Count  int64
	}
	err := r.db.Model(&models.SyncJob{}).Select("status, count(*) as count").Group("status").Find(&results).Error
	if err != nil {
		return nil, err
	}

	countMap := make(map[string]int64)
	for _, res := range results {
		countMap[res.Status] = res.Count
	}
	return countMap, nil
}

func (r *DashboardRepository) GetSyncLogStatusCounts() (map[string]int64, error) {
	var results []struct {
		Status string
		Count  int64
	}
	err := r.db.Model(&models.SyncLog{}).Select("status, count(*) as count").Group("status").Find(&results).Error
	if err != nil {
		return nil, err
	}

	countMap := make(map[string]int64)
	for _, res := range results {
		countMap[res.Status] = res.Count
	}
	return countMap, nil
}
