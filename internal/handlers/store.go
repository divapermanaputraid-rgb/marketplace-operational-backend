package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/marketplace-ops/backend/internal/models"
	"github.com/marketplace-ops/backend/internal/repositories"
)

type StoreHandler struct {
	storeRepo *repositories.StoreRepository
}

func NewStoreHandler(storeRepo *repositories.StoreRepository) *StoreHandler {
	return &StoreHandler{storeRepo: storeRepo}
}

type CreateStoreRequest struct {
	Marketplace     string  `json:"marketplace" binding:"required,oneof=shopee tokopedia_shop tiktok_shop"`
	StoreName       string  `json:"store_name" binding:"required"`
	StoreURL        *string `json:"store_url"`
	ExternalStoreID *string `json:"external_store_id"`
	Notes           *string `json:"notes"`
}

type UpdateStoreRequest struct {
	StoreName       *string `json:"store_name"`
	StoreURL        *string `json:"store_url"`
	ExternalStoreID *string `json:"external_store_id"`
	Notes           *string `json:"notes"`
	IsActive        *bool   `json:"is_active"`
}

func (h *StoreHandler) ListStores(c *gin.Context) {
	marketplace := c.Query("marketplace")
	status := c.Query("connection_status")
	search := c.Query("search")

	stores, err := h.storeRepo.FindAll(marketplace, status, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to fetch stores"))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(stores, ""))
}

func (h *StoreHandler) GetStore(c *gin.Context) {
	id := c.Param("id")
	store, err := h.storeRepo.FindByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse("NOT_FOUND", err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(store, ""))
}

func (h *StoreHandler) CreateStore(c *gin.Context) {
	var req CreateStoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("VALIDATION_ERROR", "Invalid input data: marketplace must be shopee, tokopedia_shop, or tiktok_shop and store_name is required."))
		return
	}

	exists, err := h.storeRepo.CheckDuplicateStore(req.Marketplace, req.StoreName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to check duplicates"))
		return
	}
	if exists {
		c.JSON(http.StatusConflict, models.ErrorResponse("CONFLICT", "Store with this name already exists in this marketplace"))
		return
	}

	store := &models.Store{
		Marketplace:      req.Marketplace,
		StoreName:        req.StoreName,
		StoreURL:         req.StoreURL,
		ExternalStoreID:  req.ExternalStoreID,
		Notes:            req.Notes,
		ConnectionStatus: "not_connected",
		IsActive:         true,
	}

	if err := h.storeRepo.Create(store); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to create store"))
		return
	}

	c.JSON(http.StatusCreated, models.SuccessResponse(store, "Store created successfully"))
}

func (h *StoreHandler) UpdateStore(c *gin.Context) {
	id := c.Param("id")
	var req UpdateStoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("VALIDATION_ERROR", "Invalid input data"))
		return
	}

	store, err := h.storeRepo.FindByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse("NOT_FOUND", err.Error()))
		return
	}

	if req.StoreName != nil {
		store.StoreName = *req.StoreName
	}
	if req.StoreURL != nil {
		store.StoreURL = req.StoreURL
	}
	if req.ExternalStoreID != nil {
		store.ExternalStoreID = req.ExternalStoreID
	}
	if req.Notes != nil {
		store.Notes = req.Notes
	}
	if req.IsActive != nil {
		store.IsActive = *req.IsActive
	}

	if err := h.storeRepo.Update(store); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to update store"))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(store, "Store updated successfully"))
}

func (h *StoreHandler) DeleteStore(c *gin.Context) {
	id := c.Param("id")
	if err := h.storeRepo.SoftDelete(id); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to delete store"))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Store deleted successfully"))
}
