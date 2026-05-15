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

// Shopee Operations specific methods
func (r *DashboardRepository) GetShopeeStoreMetrics() (total int64, connected int64, expired int64, err error) {
	err = r.db.Model(&models.Store{}).Where("marketplace = ?", "shopee").Count(&total).Error
	if err != nil {
		return
	}

	err = r.db.Model(&models.Store{}).Where("marketplace = ? AND is_active = ?", "shopee", true).Count(&connected).Error
	if err != nil {
		return
	}

	err = r.db.Model(&models.MarketplaceCredential{}).Where("marketplace = ? AND access_token_expires_at < NOW()", "shopee").Count(&expired).Error
	return
}

func (r *DashboardRepository) GetShopeeSyncMetrics() (failed int64, partial int64, lastOrderSync *models.SyncLog, lastProductSync *models.SyncLog, lastStockPush *models.SyncLog, err error) {
	err = r.db.Model(&models.SyncLog{}).Where("marketplace = ? AND status = ? AND created_at > NOW() - INTERVAL '24 hours'", "shopee", "failed").Count(&failed).Error
	if err != nil {
		return
	}

	err = r.db.Model(&models.SyncLog{}).Where("marketplace = ? AND status = ? AND created_at > NOW() - INTERVAL '24 hours'", "shopee", "partial").Count(&partial).Error
	if err != nil {
		return
	}

	var logs []models.SyncLog
	err = r.db.Where("marketplace = ? AND status = ? AND sync_type = ?", "shopee", "success", "orders").Order("created_at desc").Limit(1).Find(&logs).Error
	if len(logs) > 0 {
		lastOrderSync = &logs[0]
	}

	var productLogs []models.SyncLog
	err = r.db.Where("marketplace = ? AND status = ? AND sync_type = ?", "shopee", "success", "products").Order("created_at desc").Limit(1).Find(&productLogs).Error
	if len(productLogs) > 0 {
		lastProductSync = &productLogs[0]
	}

	var stockLogs []models.SyncLog
	err = r.db.Where("marketplace = ? AND status = ? AND sync_type = ?", "shopee", "success", "stock").Order("created_at desc").Limit(1).Find(&stockLogs).Error
	if len(stockLogs) > 0 {
		lastStockPush = &stockLogs[0]
	}

	return
}

func (r *DashboardRepository) GetShopeeMappingMetrics() (mapped int64, unmapped int64, err error) {
	err = r.db.Model(&models.MarketplaceProductMapping{}).Where("marketplace = ?", "shopee").Count(&mapped).Error
	if err != nil {
		return
	}

	err = r.db.Model(&models.ShopeeMappingCandidate{}).Count(&unmapped).Error
	return
}

func (r *DashboardRepository) GetShopeeReconciliationData() ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	// We join mappings with inventory items to compare stock
	// This only works if we have some form of "last known marketplace stock"
	// For Shopee, we can try to find the candidate if it exists, or just show internal stock for now.
	// Actually, let's look for ShopeeMappingCandidate which holds the snapshot.

	type ReconRow struct {
		ID                      string
		StoreID                 string
		ExternalProductID       string
		ExternalVariantID       string
		InternalProductName     string
		InternalAvailableQty    int
		LastKnownMarketplaceQty int
		MarketplaceName         string
	}

	var rows []ReconRow
	query := `
		SELECT 
			m.id, 
			m.store_id, 
			m.external_product_id, 
			m.external_variant_id, 
			p.name as internal_product_name,
			COALESCE(i.available_quantity, 0) as internal_available_qty,
			COALESCE(c.available_quantity, 0) as last_known_marketplace_qty,
			m.marketplace as marketplace_name
		FROM marketplace_product_mappings m
		JOIN products p ON m.product_id = p.id
		LEFT JOIN inventory_items i ON (m.product_id = i.product_id AND (m.product_variant_id IS NULL OR m.product_variant_id = i.product_variant_id))
		LEFT JOIN shopee_mapping_candidates c ON (m.external_product_id = CAST(c.item_id AS TEXT) AND (m.external_variant_id IS NULL OR m.external_variant_id = CAST(c.model_id AS TEXT)))
		WHERE m.marketplace = 'shopee' AND m.deleted_at IS NULL
	`

	err := r.db.Raw(query).Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		diff := row.InternalAvailableQty - row.LastKnownMarketplaceQty
		severity := "ok"
		if diff != 0 {
			severity = "warning"
			if row.InternalAvailableQty == 0 && row.LastKnownMarketplaceQty > 0 {
				severity = "critical" // Overselling risk
			} else if row.InternalAvailableQty > 0 && row.LastKnownMarketplaceQty == 0 {
				severity = "info" // Missing sales opportunity
			}
		}

		results = append(results, map[string]interface{}{
			"product_mapping_id":           row.ID,
			"store_id":                     row.StoreID,
			"external_product_id":          row.ExternalProductID,
			"external_variant_id":          row.ExternalVariantID,
			"internal_product_name":        row.InternalProductName,
			"internal_available_quantity":  row.InternalAvailableQty,
			"last_known_marketplace_stock": row.LastKnownMarketplaceQty,
			"difference":                   diff,
			"severity":                     severity,
			"recommendation":               "review_mapping",
		})
	}

	return results, nil
}
