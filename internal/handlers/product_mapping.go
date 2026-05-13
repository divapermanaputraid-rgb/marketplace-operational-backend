package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/marketplace-ops/backend/internal/models"
	"github.com/marketplace-ops/backend/internal/repositories"
)

type ProductMappingHandler struct {
	mappingRepo *repositories.ProductMappingRepository
	productRepo *repositories.ProductRepository
	storeRepo   *repositories.StoreRepository
}

func NewProductMappingHandler(
	mappingRepo *repositories.ProductMappingRepository,
	productRepo *repositories.ProductRepository,
	storeRepo *repositories.StoreRepository,
) *ProductMappingHandler {
	return &ProductMappingHandler{
		mappingRepo: mappingRepo,
		productRepo: productRepo,
		storeRepo:   storeRepo,
	}
}

type CreateProductMappingRequest struct {
	ProductID         string   `json:"product_id" binding:"required"`
	ProductVariantID  *string  `json:"product_variant_id"`
	StoreID           string   `json:"store_id" binding:"required"`
	Marketplace       string   `json:"marketplace" binding:"required"`
	ExternalProductID string   `json:"external_product_id" binding:"required"`
	ExternalVariantID *string  `json:"external_variant_id"`
	ExternalSKU       *string  `json:"external_sku"`
	ListingName       *string  `json:"listing_name"`
	ListingURL        *string  `json:"listing_url"`
	ListingStatus     *string  `json:"listing_status"`
	Price             *float64 `json:"price"`
	Currency          *string  `json:"currency"`
}

type UpdateProductMappingRequest struct {
	ListingStatus *string  `json:"listing_status"`
	Price         *float64 `json:"price"`
	ListingName   *string  `json:"listing_name"`
	ListingURL    *string  `json:"listing_url"`
	ExternalSKU   *string  `json:"external_sku"`
}

func (h *ProductMappingHandler) ListMappings(c *gin.Context) {
	productID := c.Query("product_id")
	storeID := c.Query("store_id")
	marketplace := c.Query("marketplace")
	listingStatus := c.Query("listing_status")
	search := c.Query("search")

	mappings, err := h.mappingRepo.ListMappings(productID, storeID, marketplace, listingStatus, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to fetch product mappings"))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(mappings, ""))
}

func (h *ProductMappingHandler) GetMapping(c *gin.Context) {
	id := c.Param("id")
	mapping, err := h.mappingRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse("NOT_FOUND", err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(mapping, ""))
}

func (h *ProductMappingHandler) CreateMapping(c *gin.Context) {
	var req CreateProductMappingRequest
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

	// Validate store
	store, err := h.storeRepo.FindByID(req.StoreID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse("NOT_FOUND", "Store not found"))
		return
	}

	// Validate marketplace match
	if store.Marketplace != req.Marketplace {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("VALIDATION_ERROR", "Store marketplace does not match mapping marketplace"))
		return
	}

	// Check duplicate mapping
	extVariantID := ""
	if req.ExternalVariantID != nil {
		extVariantID = *req.ExternalVariantID
	}
	exists, err := h.mappingRepo.CheckDuplicateMapping(req.StoreID, req.ExternalProductID, extVariantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to check duplicate mapping"))
		return
	}
	if exists {
		c.JSON(http.StatusConflict, models.ErrorResponse("CONFLICT", "Mapping already exists for this store and external IDs"))
		return
	}

	pID, _ := uuid.Parse(req.ProductID)
	sID, _ := uuid.Parse(req.StoreID)

	var pvID *uuid.UUID
	if req.ProductVariantID != nil && *req.ProductVariantID != "" {
		parsed, _ := uuid.Parse(*req.ProductVariantID)
		pvID = &parsed
	}

	status := "unknown"
	if req.ListingStatus != nil && *req.ListingStatus != "" {
		status = *req.ListingStatus
	}

	currency := "IDR"
	if req.Currency != nil && *req.Currency != "" {
		currency = *req.Currency
	}

	mapping := &models.MarketplaceProductMapping{
		ProductID:         pID,
		ProductVariantID:  pvID,
		StoreID:           sID,
		Marketplace:       req.Marketplace,
		ExternalProductID: req.ExternalProductID,
		ExternalVariantID: req.ExternalVariantID,
		ExternalSKU:       req.ExternalSKU,
		ListingName:       req.ListingName,
		ListingURL:        req.ListingURL,
		ListingStatus:     status,
		Price:             req.Price,
		Currency:          currency,
	}

	if err := h.mappingRepo.Create(mapping); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to create product mapping"))
		return
	}

	// Reload with relations for the response
	fullMapping, _ := h.mappingRepo.GetByID(mapping.ID.String())

	c.JSON(http.StatusCreated, models.SuccessResponse(fullMapping, "Product mapping created successfully"))
}

func (h *ProductMappingHandler) UpdateMapping(c *gin.Context) {
	id := c.Param("id")
	var req UpdateProductMappingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("VALIDATION_ERROR", "Invalid input data"))
		return
	}

	mapping, err := h.mappingRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse("NOT_FOUND", err.Error()))
		return
	}

	if req.ListingStatus != nil {
		mapping.ListingStatus = *req.ListingStatus
	}
	if req.Price != nil {
		mapping.Price = req.Price
	}
	if req.ListingName != nil {
		mapping.ListingName = req.ListingName
	}
	if req.ListingURL != nil {
		mapping.ListingURL = req.ListingURL
	}
	if req.ExternalSKU != nil {
		mapping.ExternalSKU = req.ExternalSKU
	}

	if err := h.mappingRepo.Update(mapping); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to update product mapping"))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(mapping, "Product mapping updated successfully"))
}

func (h *ProductMappingHandler) DeleteMapping(c *gin.Context) {
	id := c.Param("id")
	if err := h.mappingRepo.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to delete product mapping"))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Product mapping deleted successfully"))
}

func (h *ProductMappingHandler) ListMappingsByProduct(c *gin.Context) {
	productID := c.Param("id")
	mappings, err := h.mappingRepo.ListByProduct(productID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to fetch mappings for product"))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(mappings, ""))
}

func (h *ProductMappingHandler) ListMappingsByStore(c *gin.Context) {
	storeID := c.Param("id")
	mappings, err := h.mappingRepo.ListByStore(storeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to fetch mappings for store"))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(mappings, ""))
}
