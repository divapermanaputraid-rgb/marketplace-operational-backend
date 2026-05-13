package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/marketplace-ops/backend/internal/models"
	"github.com/marketplace-ops/backend/internal/repositories"
)

type ProductHandler struct {
	productRepo *repositories.ProductRepository
}

func NewProductHandler(productRepo *repositories.ProductRepository) *ProductHandler {
	return &ProductHandler{productRepo: productRepo}
}

type CreateProductRequest struct {
	SKU          string   `json:"sku" binding:"required"`
	Name         string   `json:"name" binding:"required"`
	Description  *string  `json:"description"`
	Brand        *string  `json:"brand"`
	Category     *string  `json:"category"`
	Status       *string  `json:"status"` // draft, active, inactive, archived
	CostPrice    *float64 `json:"cost_price"`
	SellingPrice *float64 `json:"selling_price"`
	Currency     *string  `json:"currency"`
	WeightGrams  *int     `json:"weight_grams"`
	LengthCm     *float64 `json:"length_cm"`
	WidthCm      *float64 `json:"width_cm"`
	HeightCm     *float64 `json:"height_cm"`

	PrimaryImageURL *string `json:"primary_image_url"`
}

type UpdateProductRequest struct {
	Name         *string  `json:"name"`
	Description  *string  `json:"description"`
	Brand        *string  `json:"brand"`
	Category     *string  `json:"category"`
	Status       *string  `json:"status"`
	CostPrice    *float64 `json:"cost_price"`
	SellingPrice *float64 `json:"selling_price"`
	WeightGrams  *int     `json:"weight_grams"`
	LengthCm     *float64 `json:"length_cm"`
	WidthCm      *float64 `json:"width_cm"`
	HeightCm     *float64 `json:"height_cm"`

	PrimaryImageURL *string `json:"primary_image_url"`
}

func (h *ProductHandler) ListProducts(c *gin.Context) {
	status := c.Query("status")
	category := c.Query("category")
	search := c.Query("search")
	sku := c.Query("sku")

	products, err := h.productRepo.FindAll(status, category, search, sku)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to fetch products"))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(products, ""))
}

func (h *ProductHandler) GetProduct(c *gin.Context) {
	id := c.Param("id")
	product, err := h.productRepo.FindByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse("NOT_FOUND", err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(product, ""))
}

func generateSlug(name string) string {
	return strings.ReplaceAll(strings.ToLower(name), " ", "-")
}

func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("VALIDATION_ERROR", err.Error()))
		return
	}

	exists, err := h.productRepo.CheckDuplicateSKU(req.SKU)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to check SKU duplicate"))
		return
	}
	if exists {
		c.JSON(http.StatusConflict, models.ErrorResponse("CONFLICT", "Product with this SKU already exists"))
		return
	}

	status := "draft"
	if req.Status != nil && *req.Status != "" {
		status = *req.Status
	}

	currency := "IDR"
	if req.Currency != nil && *req.Currency != "" {
		currency = *req.Currency
	}

	slugStr := generateSlug(req.Name)

	product := &models.Product{
		SKU:          req.SKU,
		Name:         req.Name,
		Slug:         &slugStr,
		Description:  req.Description,
		Brand:        req.Brand,
		Category:     req.Category,
		Status:       status,
		CostPrice:    req.CostPrice,
		SellingPrice: req.SellingPrice,
		Currency:     currency,
		WeightGrams:  req.WeightGrams,
		LengthCm:     req.LengthCm,
		WidthCm:      req.WidthCm,
		HeightCm:     req.HeightCm,
	}

	if req.PrimaryImageURL != nil && *req.PrimaryImageURL != "" {
		product.Images = []models.ProductImage{
			{
				ImageURL:  req.PrimaryImageURL,
				IsPrimary: true,
			},
		}
	}

	if err := h.productRepo.Create(product); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to create product"))
		return
	}

	c.JSON(http.StatusCreated, models.SuccessResponse(product, "Product created successfully"))
}

func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	id := c.Param("id")
	var req UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("VALIDATION_ERROR", "Invalid input data"))
		return
	}

	product, err := h.productRepo.FindByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse("NOT_FOUND", err.Error()))
		return
	}

	if req.Name != nil {
		product.Name = *req.Name
	}
	if req.Description != nil {
		product.Description = req.Description
	}
	if req.Brand != nil {
		product.Brand = req.Brand
	}
	if req.Category != nil {
		product.Category = req.Category
	}
	if req.Status != nil {
		product.Status = *req.Status
	}
	if req.CostPrice != nil {
		product.CostPrice = req.CostPrice
	}
	if req.SellingPrice != nil {
		product.SellingPrice = req.SellingPrice
	}
	if req.WeightGrams != nil {
		product.WeightGrams = req.WeightGrams
	}
	if req.LengthCm != nil {
		product.LengthCm = req.LengthCm
	}
	if req.WidthCm != nil {
		product.WidthCm = req.WidthCm
	}
	if req.HeightCm != nil {
		product.HeightCm = req.HeightCm
	}

	if req.PrimaryImageURL != nil {
		if *req.PrimaryImageURL == "" {
			product.Images = []models.ProductImage{}
		} else {
			found := false
			for i, img := range product.Images {
				if img.IsPrimary {
					product.Images[i].ImageURL = req.PrimaryImageURL
					found = true
					break
				}
			}
			if !found {
				product.Images = append(product.Images, models.ProductImage{
					ImageURL:  req.PrimaryImageURL,
					IsPrimary: true,
				})
			}
		}
	}

	if err := h.productRepo.Update(product); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to update product"))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(product, "Product updated successfully"))
}

func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	id := c.Param("id")
	if err := h.productRepo.SoftDelete(id); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to delete product"))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Product deleted successfully"))
}
