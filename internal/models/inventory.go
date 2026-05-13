package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type InventoryItem struct {
	ID                uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ProductID         uuid.UUID      `gorm:"type:uuid;not null;index" json:"product_id"`
	ProductVariantID  *uuid.UUID     `gorm:"type:uuid;index" json:"product_variant_id,omitempty"`
	SKU               string         `gorm:"type:varchar(255);not null;index" json:"sku"`
	LocationName      string         `gorm:"type:varchar(255);not null;default:'Main Warehouse'" json:"location_name"`
	AvailableQuantity int            `gorm:"type:integer;not null;default:0" json:"available_quantity"`
	ReservedQuantity  int            `gorm:"type:integer;not null;default:0" json:"reserved_quantity"`
	DamagedQuantity   int            `gorm:"type:integer;not null;default:0" json:"damaged_quantity"`
	SafetyStock       int            `gorm:"type:integer;not null;default:0" json:"safety_stock"`
	Notes             *string        `gorm:"type:text" json:"notes,omitempty"`
	CreatedAt         time.Time      `gorm:"type:timestamptz;not null;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"type:timestamptz;not null;autoUpdateTime" json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"type:timestamptz;index" json:"-"`

	// Relations
	Product        *Product        `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	ProductVariant *ProductVariant `gorm:"foreignKey:ProductVariantID" json:"product_variant,omitempty"`
}

func (InventoryItem) TableName() string {
	return "inventory_items"
}

type InventoryMovement struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	InventoryItemID  uuid.UUID  `gorm:"type:uuid;not null;index" json:"inventory_item_id"`
	ProductID        uuid.UUID  `gorm:"type:uuid;not null;index" json:"product_id"`
	ProductVariantID *uuid.UUID `gorm:"type:uuid;index" json:"product_variant_id,omitempty"`
	MovementType     string     `gorm:"type:varchar(50);not null;index" json:"movement_type"` // initial, adjustment_in, adjustment_out, reserved, reservation_released, sold, returned, damaged, manual_correction
	QuantityDelta    int        `gorm:"type:integer;not null" json:"quantity_delta"`
	QuantityBefore   int        `gorm:"type:integer;not null" json:"quantity_before"`
	QuantityAfter    int        `gorm:"type:integer;not null" json:"quantity_after"`
	ReferenceType    *string    `gorm:"type:varchar(100)" json:"reference_type,omitempty"`
	ReferenceID      *string    `gorm:"type:varchar(100)" json:"reference_id,omitempty"`
	Notes            *string    `gorm:"type:text" json:"notes,omitempty"`
	CreatedByAdminID *uuid.UUID `gorm:"type:uuid" json:"created_by_admin_id,omitempty"`
	CreatedAt        time.Time  `gorm:"type:timestamptz;not null;autoCreateTime" json:"created_at"`
}

func (InventoryMovement) TableName() string {
	return "inventory_movements"
}
