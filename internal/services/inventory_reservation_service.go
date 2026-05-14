package services

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/marketplace-ops/backend/internal/models"
	"github.com/marketplace-ops/backend/internal/repositories"
	"gorm.io/gorm"
)

type InventoryReservationService struct {
	inventoryRepo *repositories.InventoryRepository
	orderRepo     *repositories.OrderRepository
}

func NewInventoryReservationService(
	inventoryRepo *repositories.InventoryRepository,
	orderRepo *repositories.OrderRepository,
) *InventoryReservationService {
	return &InventoryReservationService{
		inventoryRepo: inventoryRepo,
		orderRepo:     orderRepo,
	}
}

type ReservationResult struct {
	Status           string   `json:"status"` // success, insufficient_stock, partially_mapped, error
	Message          string   `json:"message"`
	RecordsProcessed int      `json:"records_processed"`
	RecordsReserved  int      `json:"records_reserved"`
	RecordsReleased  int      `json:"records_released"`
	RecordsConfirmed int      `json:"records_confirmed"`
	RecordsSkipped   int      `json:"records_skipped"` // Unmapped or already processed
	Errors           []string `json:"errors,omitempty"`
}

func (s *InventoryReservationService) ReserveStockForOrder(orderID uuid.UUID) (*ReservationResult, error) {
	order, err := s.orderRepo.GetByID(orderID.String())
	if err != nil {
		return nil, err
	}

	result := &ReservationResult{
		Status:  "success",
		Message: "Stock reservation processed",
	}

	err = s.inventoryRepo.WithTransaction(func(tx *gorm.DB) error {
		// Aggregate quantities by inventory item to handle same-SKU line items safely
		type invKey struct {
			ProductID string
			VariantID string
		}
		itemTotals := make(map[invKey]int)
		for _, item := range order.Items {
			result.RecordsProcessed++
			if item.ProductID == nil {
				result.RecordsSkipped++
				continue
			}
			vID := ""
			if item.ProductVariantID != nil {
				vID = item.ProductVariantID.String()
			}
			key := invKey{ProductID: item.ProductID.String(), VariantID: vID}
			itemTotals[key] += item.Quantity
		}

		for key, totalQty := range itemTotals {
			// Find inventory item
			invItem, err := s.inventoryRepo.GetByProductAndVariant(key.ProductID, key.VariantID, "Main Warehouse")
			if err != nil {
				return fmt.Errorf("failed to find inventory for product %s: %w", key.ProductID, err)
			}
			if invItem == nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Inventory item not found for product %s", key.ProductID))
				result.Status = "error"
				return errors.New("missing inventory item for mapped product")
			}

			// Idempotency check: Already reserved?
			existing, err := s.inventoryRepo.GetMovementByReference(invItem.ID.String(), "order", order.ID.String(), "reserve")
			if err != nil {
				return err
			}
			if existing != nil {
				result.RecordsSkipped++
				continue
			}

			// Check stock
			if invItem.AvailableQuantity < totalQty {
				result.Status = "insufficient_stock"
				return fmt.Errorf("insufficient stock for %s: have %d, need %d", invItem.SKU, invItem.AvailableQuantity, totalQty)
			}

			// Update quantities
			quantityBefore := invItem.AvailableQuantity
			invItem.AvailableQuantity -= totalQty
			invItem.ReservedQuantity += totalQty

			if err := tx.Save(invItem).Error; err != nil {
				return err
			}

			// Create movement
			refType := "order"
			refID := order.ID.String()
			notes := fmt.Sprintf("Reserved for Order %s", order.OrderNumber)

			movement := &models.InventoryMovement{
				InventoryItemID:  invItem.ID,
				ProductID:        invItem.ProductID,
				ProductVariantID: invItem.ProductVariantID,
				MovementType:     "reserve",
				QuantityDelta:    totalQty,
				QuantityBefore:   quantityBefore,
				QuantityAfter:    invItem.AvailableQuantity,
				ReferenceType:    &refType,
				ReferenceID:      &refID,
				Notes:            &notes,
			}

			if err := tx.Create(movement).Error; err != nil {
				return err
			}

			result.RecordsReserved++
		}
		return nil
	})

	if err != nil {
		if result.Status == "success" {
			result.Status = "error"
		}
		result.Message = err.Error()
		return result, nil // Return result with error status, not a go error to avoid 500
	}

	return result, nil
}

func (s *InventoryReservationService) ReleaseReservationForOrder(orderID uuid.UUID) (*ReservationResult, error) {
	order, err := s.orderRepo.GetByID(orderID.String())
	if err != nil {
		return nil, err
	}

	result := &ReservationResult{
		Status:  "success",
		Message: "Reservation release processed",
	}

	err = s.inventoryRepo.WithTransaction(func(tx *gorm.DB) error {
		// 1. Find all movements for this order
		movements, err := s.inventoryRepo.ListMovementsByReference("order", order.ID.String())
		if err != nil {
			return err
		}

		// 2. Aggregate reserve quantities by inventory item
		type releaseInfo struct {
			InventoryItemID uuid.UUID
			TotalDelta      int
			ProductID       uuid.UUID
			VariantID       *uuid.UUID
		}
		aggregateMap := make(map[uuid.UUID]*releaseInfo)

		for _, m := range movements {
			if m.MovementType != "reserve" {
				continue
			}

			info, ok := aggregateMap[m.InventoryItemID]
			if !ok {
				info = &releaseInfo{
					InventoryItemID: m.InventoryItemID,
					TotalDelta:      0,
					ProductID:       m.ProductID,
					VariantID:       m.ProductVariantID,
				}
				aggregateMap[m.InventoryItemID] = info
			}
			info.TotalDelta += m.QuantityDelta
		}

		for itemID, info := range aggregateMap {
			result.RecordsProcessed++

			// 3. Idempotency check: Already released?
			released, err := s.inventoryRepo.GetMovementByReference(itemID.String(), "order", order.ID.String(), "release_reservation")
			if err != nil {
				return err
			}
			if released != nil {
				result.RecordsSkipped++
				continue
			}

			// 4. Load current inventory item
			var invItem models.InventoryItem
			if err := tx.Where("id = ?", itemID).First(&invItem).Error; err != nil {
				return err
			}

			// 5. Update quantities
			quantityBefore := invItem.AvailableQuantity
			invItem.AvailableQuantity += info.TotalDelta
			invItem.ReservedQuantity -= info.TotalDelta

			if err := tx.Save(&invItem).Error; err != nil {
				return err
			}

			// 6. Create movement
			refType := "order"
			refID := order.ID.String()
			notes := fmt.Sprintf("Released reservation for Order %s", order.OrderNumber)

			releaseMovement := &models.InventoryMovement{
				InventoryItemID:  invItem.ID,
				ProductID:        invItem.ProductID,
				ProductVariantID: invItem.ProductVariantID,
				MovementType:     "release_reservation",
				QuantityDelta:    info.TotalDelta,
				QuantityBefore:   quantityBefore,
				QuantityAfter:    invItem.AvailableQuantity,
				ReferenceType:    &refType,
				ReferenceID:      &refID,
				Notes:            &notes,
			}

			if err := tx.Create(releaseMovement).Error; err != nil {
				return err
			}

			result.RecordsReleased++
		}
		return nil
	})

	if err != nil {
		result.Status = "error"
		result.Message = err.Error()
		return result, nil
	}

	return result, nil
}

func (s *InventoryReservationService) ConfirmSaleForOrder(orderID uuid.UUID) (*ReservationResult, error) {
	order, err := s.orderRepo.GetByID(orderID.String())
	if err != nil {
		return nil, err
	}

	result := &ReservationResult{
		Status:  "success",
		Message: "Order sale confirmation processed",
	}

	err = s.inventoryRepo.WithTransaction(func(tx *gorm.DB) error {
		// Aggregate quantities by inventory item to handle same-SKU line items safely
		type invKey struct {
			ProductID string
			VariantID string
		}
		itemTotals := make(map[invKey]int)
		for _, item := range order.Items {
			result.RecordsProcessed++
			if item.ProductID == nil {
				result.RecordsSkipped++
				continue
			}
			vID := ""
			if item.ProductVariantID != nil {
				vID = item.ProductVariantID.String()
			}
			key := invKey{ProductID: item.ProductID.String(), VariantID: vID}
			itemTotals[key] += item.Quantity
		}

		for key, totalQty := range itemTotals {
			// Find inventory item
			invItem, err := s.inventoryRepo.GetByProductAndVariant(key.ProductID, key.VariantID, "Main Warehouse")
			if err != nil {
				return fmt.Errorf("failed to find inventory for product %s: %w", key.ProductID, err)
			}
			if invItem == nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Inventory item not found for product %s", key.ProductID))
				result.Status = "error"
				return errors.New("missing inventory item for mapped product")
			}

			// Idempotency check: Already confirmed/sold?
			confirmed, err := s.inventoryRepo.GetMovementByReference(invItem.ID.String(), "order", order.ID.String(), "confirm_sale")
			if err != nil {
				return err
			}
			if confirmed != nil {
				result.RecordsSkipped++
				continue
			}

			// Check if reservation exists
			reserved, err := s.inventoryRepo.GetMovementByReference(invItem.ID.String(), "order", order.ID.String(), "reserve")
			if err != nil {
				return err
			}
			if reserved == nil {
				result.Errors = append(result.Errors, fmt.Sprintf("No active reservation found for product %s", key.ProductID))
				result.Status = "partially_mapped"
				result.RecordsSkipped++
				continue
			}

			// Check if we have enough in ReservedQuantity
			if invItem.ReservedQuantity < totalQty {
				result.Errors = append(result.Errors, fmt.Sprintf("Inconsistent reservation for product %s: reserved %d, need %d", key.ProductID, invItem.ReservedQuantity, totalQty))
				result.Status = "error"
				return fmt.Errorf("inconsistent reservation for %s", invItem.SKU)
			}

			// Update quantities
			// QuantityBefore/After for confirm_sale should track ReservedQuantity
			quantityBefore := invItem.ReservedQuantity
			invItem.ReservedQuantity -= totalQty

			if err := tx.Save(invItem).Error; err != nil {
				return err
			}

			// Create movement
			refType := "order"
			refID := order.ID.String()
			notes := fmt.Sprintf("Sale confirmed for Order %s", order.OrderNumber)

			movement := &models.InventoryMovement{
				InventoryItemID:  invItem.ID,
				ProductID:        invItem.ProductID,
				ProductVariantID: invItem.ProductVariantID,
				MovementType:     "confirm_sale",
				QuantityDelta:    totalQty,
				QuantityBefore:   quantityBefore,
				QuantityAfter:    invItem.ReservedQuantity,
				ReferenceType:    &refType,
				ReferenceID:      &refID,
				Notes:            &notes,
			}

			if err := tx.Create(movement).Error; err != nil {
				return err
			}

			result.RecordsConfirmed++
		}
		return nil
	})

	if err != nil {
		if result.Status == "success" {
			result.Status = "error"
		}
		result.Message = err.Error()
		return result, nil
	}

	return result, nil
}
