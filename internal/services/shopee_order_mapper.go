package services

import (
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/marketplace-ops/backend/internal/marketplace"
	"github.com/marketplace-ops/backend/internal/models"
	"github.com/marketplace-ops/backend/internal/repositories"
)

type ShopeeOrderMapper struct {
	orderRepo   *repositories.OrderRepository
	mappingRepo *repositories.ProductMappingRepository
}

func NewShopeeOrderMapper(orderRepo *repositories.OrderRepository, mappingRepo *repositories.ProductMappingRepository) *ShopeeOrderMapper {
	return &ShopeeOrderMapper{
		orderRepo:   orderRepo,
		mappingRepo: mappingRepo,
	}
}

// MapShopeeStatusToInternal converts Shopee order status to internal status
func MapShopeeStatusToInternal(status string) string {
	switch status {
	case "UNPAID":
		return "pending"
	case "READY_TO_SHIP":
		return "ready_to_process"
	case "PROCESSED":
		return "packed"
	case "SHIPPED":
		return "shipped"
	case "COMPLETED":
		return "completed"
	case "IN_CANCEL":
		return "pending" // Cancellation requested, but still active until fully cancelled
	case "CANCELLED":
		return "cancelled"
	case "TO_RETURN":
		return "returned"
	case "RETURNED":
		return "returned"
	default:
		return "pending" // Fallback to pending for safety, original kept in raw_payload
	}
}

// MapShopeePaymentToInternal infers payment status
func MapShopeePaymentToInternal(payTime int64) string {
	if payTime > 0 {
		return "paid"
	}
	return "unpaid"
}

// MapAndPersist maps a single Shopee order to internal models and persists it.
// Returns (created bool, updated bool, unmappedCount int, error).
func (s *ShopeeOrderMapper) MapAndPersist(storeID uuid.UUID, detail marketplace.ShopeeOrderDetail) (bool, bool, int, error) {
	// 1. Map order fields
	var orderedAt, paidAt *time.Time
	if detail.CreateTime > 0 {
		t := time.Unix(detail.CreateTime, 0)
		orderedAt = &t
	}
	if detail.PayTime > 0 {
		t := time.Unix(detail.PayTime, 0)
		paidAt = &t
	}

	orderStatus := MapShopeeStatusToInternal(detail.OrderStatus)
	paymentStatus := MapShopeePaymentToInternal(detail.PayTime)
	extOrderID := detail.OrderSN

	order := &models.Order{
		StoreID:         storeID,
		Marketplace:     "shopee",
		ExternalOrderID: &extOrderID,
		OrderNumber:     detail.OrderSN,
		OrderStatus:     orderStatus,
		PaymentStatus:   paymentStatus,
		SubtotalAmount:  0, // Will be computed or kept 0 if not provided
		ShippingFee:     0,
		DiscountAmount:  0,
		TotalAmount:     detail.TotalAmount,
		Currency:        detail.Currency,
		OrderedAt:       orderedAt,
		PaidAt:          paidAt,
	}

	rawPayload := detail.RawPayload
	if rawPayload != "" {
		order.RawPayload = &rawPayload
	}

	// 2. Map order items
	var internalItems []models.OrderItem
	var unmappedCount int
	for _, item := range detail.ItemList {
		extProductID := strconv.FormatInt(item.ItemID, 10)
		extVariantID := strconv.FormatInt(item.ModelID, 10)

		if extVariantID == "0" {
			extVariantID = "" // Some APIs use 0 for no variant
		}

		orderItem := models.OrderItem{
			ProductName:       item.ItemName,
			ExternalProductID: &extProductID,
			Quantity:          item.ModelQty,
			UnitPrice:         item.ModelPrice,
			TotalPrice:        item.ModelPrice * float64(item.ModelQty),
		}

		if extVariantID != "" {
			orderItem.ExternalVariantID = &extVariantID
		}
		if item.ItemSKU != "" {
			orderItem.SKU = &item.ItemSKU
		} else if item.ModelSKU != "" {
			orderItem.SKU = &item.ModelSKU
		}

		// Try to match mapping
		mapping, _ := s.mappingRepo.FindByExternalID(storeID.String(), extProductID, extVariantID)
		if mapping != nil {
			orderItem.ProductMappingID = &mapping.ID
			orderItem.ProductID = &mapping.ProductID
			orderItem.ProductVariantID = mapping.ProductVariantID
		} else {
			notes := "Unmapped marketplace item"
			orderItem.Notes = &notes
			unmappedCount++
		}

		internalItems = append(internalItems, orderItem)
	}

	order.Items = internalItems

	// Compute Subtotal as fallback if TotalAmount exists but subtotal doesn't
	var subtotal float64
	for _, it := range internalItems {
		subtotal += it.TotalPrice
	}
	order.SubtotalAmount = subtotal

	// 3. Persist
	created, err := s.orderRepo.UpsertOrderWithItems(order)
	if err != nil {
		return false, false, 0, err
	}

	return created, !created, unmappedCount, nil
}
