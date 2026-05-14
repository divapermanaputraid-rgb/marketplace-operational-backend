package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MarketplaceProductMapping struct {
	ID                uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ProductID         uuid.UUID      `gorm:"type:uuid;not null;index" json:"product_id"`
	ProductVariantID  *uuid.UUID     `gorm:"type:uuid;index" json:"product_variant_id,omitempty"`
	StoreID           uuid.UUID      `gorm:"type:uuid;not null;index" json:"store_id"`
	Marketplace       string         `gorm:"type:varchar(50);not null;index" json:"marketplace"` // shopee, tokopedia_shop, tiktok_shop
	ExternalProductID string         `gorm:"type:varchar(255);not null;index" json:"external_product_id"`
	ExternalVariantID *string        `gorm:"type:varchar(255)" json:"external_variant_id,omitempty"`
	ExternalSKU       *string        `gorm:"type:varchar(255)" json:"external_sku,omitempty"`
	ListingName       *string        `gorm:"type:varchar(255)" json:"listing_name,omitempty"`
	ListingURL        *string        `gorm:"type:text" json:"listing_url,omitempty"`
	ListingStatus     string         `gorm:"type:varchar(50);not null;default:'unknown';index" json:"listing_status"` // draft, active, inactive, deleted, unknown
	Price             *float64       `gorm:"type:numeric(12,2)" json:"price,omitempty"`
	Currency          string         `gorm:"type:varchar(10);not null;default:'IDR'" json:"currency"`
	LastSyncedAt      *time.Time     `gorm:"type:timestamptz" json:"last_synced_at,omitempty"`
	RawPayload        *string        `gorm:"type:jsonb" json:"raw_payload,omitempty"`
	CreatedAt         time.Time      `gorm:"type:timestamptz;not null;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"type:timestamptz;not null;autoUpdateTime" json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"type:timestamptz;index" json:"-"`

	// Relations
	Product        *Product        `gorm:"foreignKey:ProductID" json:"product,omitempty"`
	ProductVariant *ProductVariant `gorm:"foreignKey:ProductVariantID" json:"product_variant,omitempty"`
	Store          *Store          `gorm:"foreignKey:StoreID" json:"store,omitempty"`
}

type ShopeeMappingCandidate struct {
	ExternalProductID        string  `json:"external_product_id"`
	ExternalVariantID        *string `json:"external_variant_id"`
	Marketplace              string  `json:"marketplace"`
	StoreID                  string  `json:"store_id"`
	Title                    string  `json:"title"`
	SKU                      string  `json:"sku"`
	VariantName              string  `json:"variant_name"`
	Price                    float64 `json:"price"`
	Stock                    int     `json:"stock"`
	ImageURL                 string  `json:"image_url"`
	MappingStatus            string  `json:"mapping_status"` // unmapped, mapped, partially_mapped
	ExistingProductMappingID *string `json:"existing_product_mapping_id"`
	InternalProductID        *string `json:"internal_product_id"`
	InternalProductName      *string `json:"internal_product_name"`
	InternalVariantID        *string `json:"internal_variant_id"`
}

func (MarketplaceProductMapping) TableName() string {
	return "marketplace_product_mappings"
}
