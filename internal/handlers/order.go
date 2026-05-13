package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/marketplace-ops/backend/internal/models"
	"github.com/marketplace-ops/backend/internal/repositories"
)

type OrderHandler struct {
	orderRepo *repositories.OrderRepository
	storeRepo *repositories.StoreRepository
}

func NewOrderHandler(orderRepo *repositories.OrderRepository, storeRepo *repositories.StoreRepository) *OrderHandler {
	return &OrderHandler{
		orderRepo: orderRepo,
		storeRepo: storeRepo,
	}
}

type CreateOrderItemRequest struct {
	ProductID         *string `json:"product_id"`
	ProductVariantID  *string `json:"product_variant_id"`
	ProductMappingID  *string `json:"product_mapping_id"`
	SKU               *string `json:"sku"`
	ProductName       string  `json:"product_name" binding:"required"`
	ExternalProductID *string `json:"external_product_id"`
	ExternalVariantID *string `json:"external_variant_id"`
	Quantity          int     `json:"quantity" binding:"required"`
	UnitPrice         float64 `json:"unit_price"`
	Notes             *string `json:"notes"`
}

type CreateOrderRequest struct {
	StoreID         string                   `json:"store_id" binding:"required"`
	Marketplace     string                   `json:"marketplace" binding:"required"`
	ExternalOrderID *string                  `json:"external_order_id"`
	OrderNumber     string                   `json:"order_number" binding:"required"`
	CustomerName    *string                  `json:"customer_name"`
	CustomerPhone   *string                  `json:"customer_phone"`
	CustomerAddress *string                  `json:"customer_address"`
	OrderStatus     *string                  `json:"order_status"`
	PaymentStatus   *string                  `json:"payment_status"`
	ShippingFee     *float64                 `json:"shipping_fee"`
	DiscountAmount  *float64                 `json:"discount_amount"`
	Currency        *string                  `json:"currency"`
	Notes           *string                  `json:"notes"`
	OrderedAt       *time.Time               `json:"ordered_at"`
	Items           []CreateOrderItemRequest `json:"items" binding:"required,min=1"`
}

type UpdateOrderRequest struct {
	OrderStatus   *string    `json:"order_status"`
	PaymentStatus *string    `json:"payment_status"`
	Notes         *string    `json:"notes"`
	PaidAt        *time.Time `json:"paid_at"`
	ShippedAt     *time.Time `json:"shipped_at"`
	CompletedAt   *time.Time `json:"completed_at"`
}

func (h *OrderHandler) ListOrders(c *gin.Context) {
	storeID := c.Query("store_id")
	marketplace := c.Query("marketplace")
	status := c.Query("order_status")
	paymentStatus := c.Query("payment_status")
	search := c.Query("search")
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")

	orders, err := h.orderRepo.ListOrders(storeID, marketplace, status, paymentStatus, search, dateFrom, dateTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to fetch orders"))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(orders, ""))
}

func (h *OrderHandler) GetOrder(c *gin.Context) {
	id := c.Param("id")
	order, err := h.orderRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse("NOT_FOUND", err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(order, ""))
}

func parseUUIDPtr(s *string) *uuid.UUID {
	if s == nil || *s == "" {
		return nil
	}
	id, err := uuid.Parse(*s)
	if err != nil {
		return nil
	}
	return &id
}

func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("VALIDATION_ERROR", err.Error()))
		return
	}

	// Validate Store
	store, err := h.storeRepo.FindByID(req.StoreID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("VALIDATION_ERROR", "Store not found"))
		return
	}

	if req.Marketplace != "manual" && store.Marketplace != req.Marketplace {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("VALIDATION_ERROR", "Marketplace must match store marketplace or be 'manual'"))
		return
	}

	// Check duplicates
	extID := ""
	if req.ExternalOrderID != nil {
		extID = *req.ExternalOrderID
	}
	exists, err := h.orderRepo.CheckDuplicateExternalOrder(req.StoreID, extID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to check duplicate order"))
		return
	}
	if exists {
		c.JSON(http.StatusConflict, models.ErrorResponse("CONFLICT", "Order with this external ID already exists in this store"))
		return
	}

	sID, _ := uuid.Parse(req.StoreID)

	status := "pending"
	if req.OrderStatus != nil && *req.OrderStatus != "" {
		status = *req.OrderStatus
	}

	paymentStatus := "unpaid"
	if req.PaymentStatus != nil && *req.PaymentStatus != "" {
		paymentStatus = *req.PaymentStatus
	}

	currency := "IDR"
	if req.Currency != nil && *req.Currency != "" {
		currency = *req.Currency
	}

	shippingFee := 0.0
	if req.ShippingFee != nil {
		shippingFee = *req.ShippingFee
	}

	discountAmount := 0.0
	if req.DiscountAmount != nil {
		discountAmount = *req.DiscountAmount
	}

	var orderItems []models.OrderItem
	var subtotal float64 = 0

	for _, itemReq := range req.Items {
		if itemReq.Quantity <= 0 {
			c.JSON(http.StatusBadRequest, models.ErrorResponse("VALIDATION_ERROR", "Item quantity must be greater than 0"))
			return
		}

		itemTotal := itemReq.UnitPrice * float64(itemReq.Quantity)
		subtotal += itemTotal

		orderItems = append(orderItems, models.OrderItem{
			ProductID:         parseUUIDPtr(itemReq.ProductID),
			ProductVariantID:  parseUUIDPtr(itemReq.ProductVariantID),
			ProductMappingID:  parseUUIDPtr(itemReq.ProductMappingID),
			SKU:               itemReq.SKU,
			ProductName:       itemReq.ProductName,
			ExternalProductID: itemReq.ExternalProductID,
			ExternalVariantID: itemReq.ExternalVariantID,
			Quantity:          itemReq.Quantity,
			UnitPrice:         itemReq.UnitPrice,
			TotalPrice:        itemTotal,
			Notes:             itemReq.Notes,
		})
	}

	total := subtotal + shippingFee - discountAmount

	orderedAt := time.Now()
	if req.OrderedAt != nil {
		orderedAt = *req.OrderedAt
	}

	order := &models.Order{
		StoreID:         sID,
		Marketplace:     req.Marketplace,
		ExternalOrderID: req.ExternalOrderID,
		OrderNumber:     req.OrderNumber,
		CustomerName:    req.CustomerName,
		CustomerPhone:   req.CustomerPhone,
		CustomerAddress: req.CustomerAddress,
		OrderStatus:     status,
		PaymentStatus:   paymentStatus,
		SubtotalAmount:  subtotal,
		ShippingFee:     shippingFee,
		DiscountAmount:  discountAmount,
		TotalAmount:     total,
		Currency:        currency,
		Notes:           req.Notes,
		OrderedAt:       &orderedAt,
		Items:           orderItems,
	}

	if err := h.orderRepo.Create(order); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to create order"))
		return
	}

	c.JSON(http.StatusCreated, models.SuccessResponse(order, "Order created successfully"))
}

func (h *OrderHandler) UpdateOrder(c *gin.Context) {
	id := c.Param("id")
	var req UpdateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("VALIDATION_ERROR", "Invalid input data"))
		return
	}

	order, err := h.orderRepo.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse("NOT_FOUND", err.Error()))
		return
	}

	if req.OrderStatus != nil {
		order.OrderStatus = *req.OrderStatus
	}
	if req.PaymentStatus != nil {
		order.PaymentStatus = *req.PaymentStatus
	}
	if req.Notes != nil {
		order.Notes = req.Notes
	}
	if req.PaidAt != nil {
		order.PaidAt = req.PaidAt
	}
	if req.ShippedAt != nil {
		order.ShippedAt = req.ShippedAt
	}
	if req.CompletedAt != nil {
		order.CompletedAt = req.CompletedAt
	}

	if err := h.orderRepo.Update(order); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to update order"))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(order, "Order updated successfully"))
}

func (h *OrderHandler) DeleteOrder(c *gin.Context) {
	id := c.Param("id")
	if err := h.orderRepo.SoftDelete(id); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to delete order"))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Order deleted successfully"))
}
