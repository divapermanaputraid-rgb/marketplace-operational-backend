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
		for _, item := range order.Items {
			result.RecordsProcessed++

			// 1. Only process mapped items
			if item.ProductID == nil {
				result.RecordsSkipped++
				continue
			}

			// 2. Find inventory item
			vID := ""
			if item.ProductVariantID != nil {
				vID = item.ProductVariantID.String()
			}

			// Default to Main Warehouse for now
			invItem, err := s.inventoryRepo.GetByProductAndVariant(item.ProductID.String(), vID, "Main Warehouse")
			if err != nil {
				return fmt.Errorf("failed to find inventory for SKU %s: %w", item.ProductName, err)
			}
			if invItem == nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Inventory item not found for product %s", item.ProductName))
				result.Status = "error"
				return errors.New("missing inventory item for mapped product")
			}

			// 3. Idempotency check: Already reserved?
			existing, err := s.inventoryRepo.GetMovementByReference(invItem.ID.String(), "order", order.ID.String(), "reserve")
			if err != nil {
				return err
			}
			if existing != nil {
				result.RecordsSkipped++
				continue
			}

			// 4. Check stock
			if invItem.AvailableQuantity < item.Quantity {
				result.Status = "insufficient_stock"
				return fmt.Errorf("insufficient stock for %s: have %d, need %d", invItem.SKU, invItem.AvailableQuantity, item.Quantity)
			}

			// 5. Update quantities
			quantityBefore := invItem.AvailableQuantity
			invItem.AvailableQuantity -= item.Quantity
			invItem.ReservedQuantity += item.Quantity

			if err := tx.Save(invItem).Error; err != nil {
				return err
			}

			// 6. Create movement
			refType := "order"
			refID := order.ID.String()
			notes := fmt.Sprintf("Reserved for Order %s", order.OrderNumber)

			movement := &models.InventoryMovement{
				InventoryItemID:  invItem.ID,
				ProductID:        invItem.ProductID,
				ProductVariantID: invItem.ProductVariantID,
				MovementType:     "reserve",
				QuantityDelta:    item.Quantity,
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
		// 1. Find all reserve movements for this order
		movements, err := s.inventoryRepo.ListMovementsByReference("order", order.ID.String())
		if err != nil {
			return err
		}

		for _, m := range movements {
			if m.MovementType != "reserve" {
				continue
			}
			result.RecordsProcessed++

			// 2. Check if already released
			// We check if there's a release_reservation for this item and order
			released, err := s.inventoryRepo.GetMovementByReference(m.InventoryItemID.String(), "order", order.ID.String(), "release_reservation")
			if err != nil {
				return err
			}
			if released != nil {
				result.RecordsSkipped++
				continue
			}

			// 3. Load current inventory item
			var invItem models.InventoryItem
			if err := tx.Where("id = ?", m.InventoryItemID).First(&invItem).Error; err != nil {
				return err
			}

			// 4. Update quantities
			quantityBefore := invItem.AvailableQuantity
			invItem.AvailableQuantity += m.QuantityDelta
			invItem.ReservedQuantity -= m.QuantityDelta

			if err := tx.Save(&invItem).Error; err != nil {
				return err
			}

			// 5. Create movement
			refType := "order"
			refID := order.ID.String()
			notes := fmt.Sprintf("Released reservation for Order %s", order.OrderNumber)

			releaseMovement := &models.InventoryMovement{
				InventoryItemID:  invItem.ID,
				ProductID:        invItem.ProductID,
				ProductVariantID: invItem.ProductVariantID,
				MovementType:     "release_reservation",
				QuantityDelta:    m.QuantityDelta,
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
		for _, item := range order.Items {
			result.RecordsProcessed++

			// 1. Only process mapped items
			if item.ProductID == nil {
				result.RecordsSkipped++
				continue
			}

			// 2. Find inventory item
			vID := ""
			if item.ProductVariantID != nil {
				vID = item.ProductVariantID.String()
			}

			invItem, err := s.inventoryRepo.GetByProductAndVariant(item.ProductID.String(), vID, "Main Warehouse")
			if err != nil {
				return fmt.Errorf("failed to find inventory for SKU %s: %w", item.ProductName, err)
			}
			if invItem == nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Inventory item not found for product %s", item.ProductName))
				result.Status = "error"
				return errors.New("missing inventory item for mapped product")
			}

			// 3. Idempotency check: Already confirmed/sold?
			confirmed, err := s.inventoryRepo.GetMovementByReference(invItem.ID.String(), "order", order.ID.String(), "confirm_sale")
			if err != nil {
				return err
			}
			if confirmed != nil {
				result.RecordsSkipped++
				continue
			}

			// 4. Check if reservation exists
			reserved, err := s.inventoryRepo.GetMovementByReference(invItem.ID.String(), "order", order.ID.String(), "reserve")
			if err != nil {
				return err
			}
			if reserved == nil {
				result.Errors = append(result.Errors, fmt.Sprintf("No active reservation found for %s", item.ProductName))
				result.Status = "partially_mapped" // Or a specialized status
				result.RecordsSkipped++
				continue
			}

			// 5. Load current inventory item (re-load to ensure fresh data in TX)
			// Actually invItem was already loaded, but we might want to ensure we're inside the TX lock if using SELECT FOR UPDATE
			// For now, simple update is fine as we are in a transaction.

			// Check if we have enough in ReservedQuantity just in case
			if invItem.ReservedQuantity < item.Quantity {
				// This shouldn't happen if reservation was recorded, but safety first
				result.Errors = append(result.Errors, fmt.Sprintf("Inconsistent reservation for %s: reserved %d, need %d", item.ProductName, invItem.ReservedQuantity, item.Quantity))
				result.Status = "error"
				return fmt.Errorf("inconsistent reservation for %s", invItem.SKU)
			}

			// 6. Update quantities
			// Sale from Reserved: ReservedQuantity decreases, AvailableQuantity remains the same.
			quantityBefore := invItem.AvailableQuantity
			invItem.ReservedQuantity -= item.Quantity

			if err := tx.Save(invItem).Error; err != nil {
				return err
			}

			// 7. Create movement
			refType := "order"
			refID := order.ID.String()
			notes := fmt.Sprintf("Sale confirmed for Order %s", order.OrderNumber)

			movement := &models.InventoryMovement{
				InventoryItemID:  invItem.ID,
				ProductID:        invItem.ProductID,
				ProductVariantID: invItem.ProductVariantID,
				MovementType:     "confirm_sale",
				QuantityDelta:    item.Quantity,
				QuantityBefore:   quantityBefore,
				QuantityAfter:    invItem.AvailableQuantity,
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
