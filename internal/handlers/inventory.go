package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/marketplace-ops/backend/internal/models"
	"github.com/marketplace-ops/backend/internal/repositories"
	"gorm.io/gorm"
)

type InventoryHandler struct {
	inventoryRepo *repositories.InventoryRepository
	productRepo   *repositories.ProductRepository
	mappingRepo   *repositories.ProductMappingRepository
}

func NewInventoryHandler(
	inventoryRepo *repositories.InventoryRepository,
	productRepo *repositories.ProductRepository,
	mappingRepo *repositories.ProductMappingRepository,
) *InventoryHandler {
	return &InventoryHandler{
		inventoryRepo: inventoryRepo,
		productRepo:   productRepo,
		mappingRepo:   mappingRepo,
	}
}

type InventoryItemResponse struct {
	models.InventoryItem
	MappedListingCount int      `json:"mapped_listing_count"`
	MarketplacesMapped []string `json:"marketplaces_mapped"`
	SyncStatusLabel    string   `json:"sync_status_label"`
	IsLowStock         bool     `json:"is_low_stock"`
}

type CreateInventoryItemRequest struct {
	ProductID        string  `json:"product_id" binding:"required"`
	ProductVariantID *string `json:"product_variant_id"`
	SKU              string  `json:"sku" binding:"required"`
	LocationName     *string `json:"location_name"`
	InitialQuantity  int     `json:"initial_quantity"`
	SafetyStock      int     `json:"safety_stock"`
	Notes            *string `json:"notes"`
}

type AdjustStockRequest struct {
	MovementType  string  `json:"movement_type" binding:"required"` // adjustment_in, adjustment_out, damaged, etc.
	QuantityDelta int     `json:"quantity_delta" binding:"required"`
	Notes         *string `json:"notes"`
	ReferenceType *string `json:"reference_type"`
	ReferenceID   *string `json:"reference_id"`
}

func (h *InventoryHandler) ListInventory(c *gin.Context) {
	productID := c.Query("product_id")
	variantID := c.Query("product_variant_id")
	search := c.Query("search")
	location := c.Query("location_name")
	lowStock := c.Query("low_stock")

	items, err := h.inventoryRepo.ListItems(productID, variantID, search, location, lowStock)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to fetch inventory items"))
		return
	}

	response := make([]InventoryItemResponse, len(items))
	for i, item := range items {
		// Get mapped listings
		var mappings []models.MarketplaceProductMapping
		if item.ProductVariantID != nil {
			// This is not quite right in current mappingRepo, it lists by productID.
			// I'll just use a filter if possible or just count all mappings for the product for now
			// actually I should filter by variant too if applicable.
			// Let's just use the productID mappings as a fallback or if variant is null.
			mappings, _ = h.mappingRepo.ListMappings(item.ProductID.String(), "", "", "", "")
			// Filter by variant in memory for now if needed, or update repository later.
			// For MVP, just count product mappings.
		} else {
			mappings, _ = h.mappingRepo.ListMappings(item.ProductID.String(), "", "", "", "")
		}

		marketplaces := make(map[string]bool)
		mappedCount := 0
		for _, m := range mappings {
			if (item.ProductVariantID != nil && m.ProductVariantID != nil && *m.ProductVariantID == *item.ProductVariantID) ||
				(item.ProductVariantID == nil && m.ProductVariantID == nil) {
				marketplaces[m.Marketplace] = true
				mappedCount++
			}
		}

		marketplaceList := make([]string, 0, len(marketplaces))
		for m := range marketplaces {
			marketplaceList = append(marketplaceList, m)
		}

		response[i] = InventoryItemResponse{
			InventoryItem:      item,
			MappedListingCount: mappedCount,
			MarketplacesMapped: marketplaceList,
			SyncStatusLabel:    "Ready for future sync",
			IsLowStock:         item.AvailableQuantity <= item.SafetyStock,
		}
	}

	c.JSON(http.StatusOK, models.SuccessResponse(response, ""))
}

func (h *InventoryHandler) GetInventoryItem(c *gin.Context) {
	id := c.Param("id")
	item, err := h.inventoryRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse("NOT_FOUND", err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(item, ""))
}

func (h *InventoryHandler) CreateInventoryItem(c *gin.Context) {
	var req CreateInventoryItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("VALIDATION_ERROR", err.Error()))
		return
	}

	// Validate product
	_, err := h.productRepo.FindByID(req.ProductID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse("NOT_FOUND", "Product not found"))
		return
	}

	location := "Main Warehouse"
	if req.LocationName != nil && *req.LocationName != "" {
		location = *req.LocationName
	}

	vID := ""
	if req.ProductVariantID != nil {
		vID = *req.ProductVariantID
	}

	// Check duplicate
	existing, err := h.inventoryRepo.GetByProductAndVariant(req.ProductID, vID, location)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to check existing inventory"))
		return
	}
	if existing != nil {
		c.JSON(http.StatusConflict, models.ErrorResponse("CONFLICT", "Inventory item already exists for this product/variant and location"))
		return
	}

	pID, _ := uuid.Parse(req.ProductID)
	var pvID *uuid.UUID
	if vID != "" {
		parsed, _ := uuid.Parse(vID)
		pvID = &parsed
	}

	item := &models.InventoryItem{
		ProductID:         pID,
		ProductVariantID:  pvID,
		SKU:               req.SKU,
		LocationName:      location,
		AvailableQuantity: req.InitialQuantity,
		SafetyStock:       req.SafetyStock,
		Notes:             req.Notes,
	}

	err = h.inventoryRepo.WithTransaction(func(tx *gorm.DB) error {
		if err := tx.Create(item).Error; err != nil {
			return err
		}

		if req.InitialQuantity > 0 {
			movement := &models.InventoryMovement{
				InventoryItemID:  item.ID,
				ProductID:        item.ProductID,
				ProductVariantID: item.ProductVariantID,
				MovementType:     "initial",
				QuantityDelta:    req.InitialQuantity,
				QuantityBefore:   0,
				QuantityAfter:    req.InitialQuantity,
				Notes:            req.Notes,
			}
			if err := tx.Create(movement).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to create inventory item"))
		return
	}

	c.JSON(http.StatusCreated, models.SuccessResponse(item, "Inventory item created successfully"))
}

func (h *InventoryHandler) AdjustStock(c *gin.Context) {
	id := c.Param("id")
	var req AdjustStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("VALIDATION_ERROR", err.Error()))
		return
	}

	err := h.inventoryRepo.WithTransaction(func(tx *gorm.DB) error {
		var item models.InventoryItem
		if err := tx.Where("id = ?", id).First(&item).Error; err != nil {
			return err
		}

		quantityBefore := item.AvailableQuantity
		quantityAfter := quantityBefore + req.QuantityDelta

		if quantityAfter < 0 {
			return errors.New("insufficient stock: available quantity cannot be negative")
		}

		item.AvailableQuantity = quantityAfter
		if err := tx.Save(&item).Error; err != nil {
			return err
		}

		movement := &models.InventoryMovement{
			InventoryItemID:  item.ID,
			ProductID:        item.ProductID,
			ProductVariantID: item.ProductVariantID,
			MovementType:     req.MovementType,
			QuantityDelta:    req.QuantityDelta,
			QuantityBefore:   quantityBefore,
			QuantityAfter:    quantityAfter,
			ReferenceType:    req.ReferenceType,
			ReferenceID:      req.ReferenceID,
			Notes:            req.Notes,
		}

		return tx.Create(movement).Error
	})

	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("STOCK_ERROR", err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Stock adjusted successfully"))
}

func (h *InventoryHandler) ListMovements(c *gin.Context) {
	id := c.Param("id")
	movements, err := h.inventoryRepo.ListMovements(id, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to fetch movements"))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(movements, ""))
}

func (h *InventoryHandler) ListAllMovements(c *gin.Context) {
	movementType := c.Query("movement_type")
	movements, err := h.inventoryRepo.ListMovements("", movementType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to fetch movements"))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(movements, ""))
}

func (h *InventoryHandler) UpdateInventoryMetadata(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		SafetyStock *int    `json:"safety_stock"`
		Notes       *string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("VALIDATION_ERROR", "Invalid input"))
		return
	}

	item, err := h.inventoryRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse("NOT_FOUND", err.Error()))
		return
	}

	if req.SafetyStock != nil {
		item.SafetyStock = *req.SafetyStock
	}
	if req.Notes != nil {
		item.Notes = req.Notes
	}

	if err := h.inventoryRepo.Update(item); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to update metadata"))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(item, "Inventory metadata updated"))
}

func (h *InventoryHandler) DeleteInventoryItem(c *gin.Context) {
	id := c.Param("id")
	if err := h.inventoryRepo.SoftDelete(id); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to delete item"))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Inventory item deleted"))
}
