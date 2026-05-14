package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/marketplace-ops/backend/internal/models"
	"github.com/marketplace-ops/backend/internal/repositories"
)

type DashboardHandler struct {
	repo *repositories.DashboardRepository
}

func NewDashboardHandler(repo *repositories.DashboardRepository) *DashboardHandler {
	return &DashboardHandler{repo: repo}
}

func (h *DashboardHandler) GetSummary(c *gin.Context) {
	// Store Metrics
	sTotal, sActive, sByMarketplace, err := h.repo.GetStoreMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to fetch store metrics"))
		return
	}

	// Product Metrics
	pTotal, pActive, pDraft, pInactive, pArchived, err := h.repo.GetProductMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to fetch product metrics"))
		return
	}

	// Mapping Metrics
	mTotal, mMapped, mUnmapped, err := h.repo.GetMappingMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to fetch mapping metrics"))
		return
	}

	var mCoverage float64
	if pTotal > 0 {
		mCoverage = float64(mMapped) / float64(pTotal) * 100
	}

	// Inventory Metrics
	iTotalItems, iLowStock, iTotalAvailable, iTotalReserved, iTotalDamaged, err := h.repo.GetInventoryMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to fetch inventory metrics"))
		return
	}

	// Order Metrics
	oTotal, oStatusCounts, oPaymentCounts, oSalesAmount, err := h.repo.GetOrderMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to fetch order metrics"))
		return
	}

	// Sync Metrics
	syncTotalJobs, syncNotConfigured, syncFailed, syncPartial, syncSuccess, syncLatestLogs, err := h.repo.GetSyncMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to fetch sync metrics"))
		return
	}

	// Assemble Response
	response := gin.H{
		"stores": gin.H{
			"total":          sTotal,
			"active":         sActive,
			"by_marketplace": sByMarketplace,
		},
		"products": gin.H{
			"total":    pTotal,
			"active":   pActive,
			"draft":    pDraft,
			"inactive": pInactive,
			"archived": pArchived,
		},
		"product_mappings": gin.H{
			"total":             mTotal,
			"mapped_products":   mMapped,
			"unmapped_products": mUnmapped,
			"coverage_percent":  mCoverage,
		},
		"inventory": gin.H{
			"total_items":              iTotalItems,
			"low_stock_count":          iLowStock,
			"total_available_quantity": iTotalAvailable,
			"total_reserved_quantity":  iTotalReserved,
			"total_damaged_quantity":   iTotalDamaged,
		},
		"orders": gin.H{
			"total":              oTotal,
			"pending":            oStatusCounts["pending"],
			"ready_to_process":   oStatusCounts["ready_to_process"],
			"packed":             oStatusCounts["packed"],
			"shipped":            oStatusCounts["shipped"],
			"completed":          oStatusCounts["completed"],
			"cancelled":          oStatusCounts["cancelled"],
			"returned":           oStatusCounts["returned"],
			"failed":             oStatusCounts["failed"],
			"payment_counts":     oPaymentCounts,
			"total_sales_amount": oSalesAmount,
		},
		"sync": gin.H{
			"total_jobs":     syncTotalJobs,
			"not_configured": syncNotConfigured,
			"failed":         syncFailed,
			"partial":        syncPartial,
			"success":        syncSuccess,
			"latest_logs":    syncLatestLogs,
		},
	}

	c.JSON(http.StatusOK, models.SuccessResponse(response, ""))
}

func (h *DashboardHandler) GetOrdersReport(c *gin.Context) {
	total, statusCounts, paymentCounts, salesAmount, err := h.repo.GetOrderMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to fetch order report metrics"))
		return
	}

	salesByMarketplace, err := h.repo.GetSalesByMarketplace()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to fetch sales by marketplace"))
		return
	}

	recentOrders, err := h.repo.GetRecentOrders(10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to fetch recent orders"))
		return
	}

	res := gin.H{
		"order_status_counts":   statusCounts,
		"payment_status_counts": paymentCounts,
		"total_orders":          total,
		"total_sales_amount":    salesAmount,
		"sales_by_marketplace":  salesByMarketplace,
		"recent_orders":         recentOrders,
	}
	c.JSON(http.StatusOK, models.SuccessResponse(res, ""))
}

func (h *DashboardHandler) GetInventoryReport(c *gin.Context) {
	_, _, totalAvailable, totalReserved, totalDamaged, err := h.repo.GetInventoryMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to fetch inventory report metrics"))
		return
	}

	lowStockItems, err := h.repo.GetLowStockItems(20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to fetch low stock items"))
		return
	}

	res := gin.H{
		"low_stock_items":          lowStockItems,
		"total_available_quantity": totalAvailable,
		"total_reserved_quantity":  totalReserved,
		"total_damaged_quantity":   totalDamaged,
	}
	c.JSON(http.StatusOK, models.SuccessResponse(res, ""))
}

func (h *DashboardHandler) GetProductsReport(c *gin.Context) {
	pTotal, pActive, pDraft, pInactive, pArchived, err := h.repo.GetProductMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to fetch product metrics"))
		return
	}

	mTotal, mMapped, mUnmapped, err := h.repo.GetMappingMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to fetch mapping metrics"))
		return
	}

	var mCoverage float64
	if pTotal > 0 {
		mCoverage = float64(mMapped) / float64(pTotal) * 100
	}

	unmappedProducts, err := h.repo.GetUnmappedProducts(20)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to fetch unmapped products"))
		return
	}

	mappingsByMarketplace, err := h.repo.GetMappingsByMarketplace()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to fetch mappings by marketplace"))
		return
	}

	res := gin.H{
		"product_status_counts": gin.H{
			"active":   pActive,
			"draft":    pDraft,
			"inactive": pInactive,
			"archived": pArchived,
		},
		"mapping_coverage":        mCoverage,
		"mapped_products_count":   mMapped,
		"unmapped_products_count": mUnmapped,
		"unmapped_products":       unmappedProducts,
		"mappings_by_marketplace": mappingsByMarketplace,
		"total_mappings":          mTotal,
	}
	c.JSON(http.StatusOK, models.SuccessResponse(res, ""))
}

func (h *DashboardHandler) GetSyncReport(c *gin.Context) {
	jobStatusCounts, err := h.repo.GetSyncJobStatusCounts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to fetch job status counts"))
		return
	}

	logStatusCounts, err := h.repo.GetSyncLogStatusCounts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to fetch log status counts"))
		return
	}

	_, _, _, _, _, latestLogs, err := h.repo.GetSyncMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to fetch sync metrics"))
		return
	}

	res := gin.H{
		"sync_job_status_counts": jobStatusCounts,
		"sync_log_status_counts": logStatusCounts,
		"latest_logs":            latestLogs,
	}
	c.JSON(http.StatusOK, models.SuccessResponse(res, ""))
}
