package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Order struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	StoreID         uuid.UUID      `gorm:"type:uuid;not null;index" json:"store_id"`
	Marketplace     string         `gorm:"type:varchar(50);not null;index" json:"marketplace"` // shopee, tokopedia_shop, tiktok_shop, manual
	ExternalOrderID *string        `gorm:"type:varchar(255);index" json:"external_order_id,omitempty"`
	OrderNumber     string         `gorm:"type:varchar(255);not null;index" json:"order_number"`
	CustomerName    *string        `gorm:"type:varchar(255)" json:"customer_name,omitempty"`
	CustomerPhone   *string        `gorm:"type:varchar(50)" json:"customer_phone,omitempty"`
	CustomerAddress *string        `gorm:"type:text" json:"customer_address,omitempty"`
	OrderStatus     string         `gorm:"type:varchar(50);not null;default:'pending';index" json:"order_status"`  // pending, ready_to_process, packed, shipped, completed, cancelled, returned, failed
	PaymentStatus   string         `gorm:"type:varchar(50);not null;default:'unpaid';index" json:"payment_status"` // unpaid, paid, cod, refunded, unknown
	SubtotalAmount  float64        `gorm:"type:numeric(12,2);not null;default:0" json:"subtotal_amount"`
	ShippingFee     float64        `gorm:"type:numeric(12,2);not null;default:0" json:"shipping_fee"`
	DiscountAmount  float64        `gorm:"type:numeric(12,2);not null;default:0" json:"discount_amount"`
	TotalAmount     float64        `gorm:"type:numeric(12,2);not null;default:0" json:"total_amount"`
	Currency        string         `gorm:"type:varchar(10);not null;default:'IDR'" json:"currency"`
	Notes           *string        `gorm:"type:text" json:"notes,omitempty"`
	OrderedAt       *time.Time     `gorm:"type:timestamptz;index" json:"ordered_at,omitempty"`
	PaidAt          *time.Time     `gorm:"type:timestamptz" json:"paid_at,omitempty"`
	ShippedAt       *time.Time     `gorm:"type:timestamptz" json:"shipped_at,omitempty"`
	CompletedAt     *time.Time     `gorm:"type:timestamptz" json:"completed_at,omitempty"`
	RawPayload      *string        `gorm:"type:jsonb" json:"raw_payload,omitempty"`
	CreatedAt       time.Time      `gorm:"type:timestamptz;not null;autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time      `gorm:"type:timestamptz;not null;autoUpdateTime" json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"type:timestamptz;index" json:"-"`

	// Relations
	Store *Store      `gorm:"foreignKey:StoreID" json:"store,omitempty"`
	Items []OrderItem `gorm:"foreignKey:OrderID" json:"items,omitempty"`
}

func (Order) TableName() string {
	return "orders"
}

type OrderItem struct {
	ID                uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrderID           uuid.UUID  `gorm:"type:uuid;not null;index" json:"order_id"`
	ProductID         *uuid.UUID `gorm:"type:uuid;index" json:"product_id,omitempty"`
	ProductVariantID  *uuid.UUID `gorm:"type:uuid;index" json:"product_variant_id,omitempty"`
	ProductMappingID  *uuid.UUID `gorm:"type:uuid;index" json:"product_mapping_id,omitempty"`
	SKU               *string    `gorm:"type:varchar(255);index" json:"sku,omitempty"`
	ProductName       string     `gorm:"type:varchar(255);not null" json:"product_name"`
	ExternalProductID *string    `gorm:"type:varchar(255)" json:"external_product_id,omitempty"`
	ExternalVariantID *string    `gorm:"type:varchar(255)" json:"external_variant_id,omitempty"`
	Quantity          int        `gorm:"type:integer;not null" json:"quantity"`
	UnitPrice         float64    `gorm:"type:numeric(12,2);not null;default:0" json:"unit_price"`
	TotalPrice        float64    `gorm:"type:numeric(12,2);not null;default:0" json:"total_price"`
	Notes             *string    `gorm:"type:text" json:"notes,omitempty"`
	CreatedAt         time.Time  `gorm:"type:timestamptz;not null;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"type:timestamptz;not null;autoUpdateTime" json:"updated_at"`

	// Relations
	Product        *Product                   `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	ProductVariant *ProductVariant            `gorm:"foreignKey:ProductVariantID" json:"product_variant,omitempty"`
	ProductMapping *MarketplaceProductMapping `gorm:"foreignKey:ProductMappingID" json:"product_mapping,omitempty"`
}

func (OrderItem) TableName() string {
	return "order_items"
}
