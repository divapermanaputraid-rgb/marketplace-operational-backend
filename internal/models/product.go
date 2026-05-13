package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Product struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SKU          string         `gorm:"type:varchar(255);not null;uniqueIndex" json:"sku"`
	Name         string         `gorm:"type:varchar(255);not null;index" json:"name"`
	Slug         *string        `gorm:"type:varchar(255);uniqueIndex" json:"slug,omitempty"`
	Description  *string        `gorm:"type:text" json:"description,omitempty"`
	Brand        *string        `gorm:"type:varchar(255)" json:"brand,omitempty"`
	Category     *string        `gorm:"type:varchar(255)" json:"category,omitempty"`
	Status       string         `gorm:"type:varchar(50);not null;default:'draft';index" json:"status"` // draft, active, inactive, archived
	CostPrice    *float64       `gorm:"type:numeric(12,2)" json:"cost_price,omitempty"`
	SellingPrice *float64       `gorm:"type:numeric(12,2)" json:"selling_price,omitempty"`
	Currency     string         `gorm:"type:varchar(10);not null;default:'IDR'" json:"currency"`
	WeightGrams  *int           `gorm:"type:integer" json:"weight_grams,omitempty"`
	LengthCm     *float64       `gorm:"type:numeric(10,2)" json:"length_cm,omitempty"`
	WidthCm      *float64       `gorm:"type:numeric(10,2)" json:"width_cm,omitempty"`
	HeightCm     *float64       `gorm:"type:numeric(10,2)" json:"height_cm,omitempty"`
	CreatedAt    time.Time      `gorm:"type:timestamptz;not null;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"type:timestamptz;not null;autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"type:timestamptz;index" json:"-"`

	Images   []ProductImage   `gorm:"foreignKey:ProductID" json:"images,omitempty"`
	Variants []ProductVariant `gorm:"foreignKey:ProductID" json:"variants,omitempty"`
}

func (Product) TableName() string {
	return "products"
}

type ProductImage struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ProductID   uuid.UUID `gorm:"type:uuid;not null;index" json:"product_id"`
	ImageURL    *string   `gorm:"type:text" json:"image_url,omitempty"`
	StoragePath *string   `gorm:"type:text" json:"storage_path,omitempty"`
	AltText     *string   `gorm:"type:varchar(255)" json:"alt_text,omitempty"`
	SortOrder   int       `gorm:"type:integer;not null;default:0" json:"sort_order"`
	IsPrimary   bool      `gorm:"type:boolean;not null;default:false" json:"is_primary"`
	CreatedAt   time.Time `gorm:"type:timestamptz;not null;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"type:timestamptz;not null;autoUpdateTime" json:"updated_at"`
}

func (ProductImage) TableName() string {
	return "product_images"
}

type ProductVariant struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ProductID    uuid.UUID      `gorm:"type:uuid;not null;index" json:"product_id"`
	SKU          string         `gorm:"type:varchar(255);not null;uniqueIndex" json:"sku"`
	Name         string         `gorm:"type:varchar(255);not null" json:"name"`
	Option1Name  *string        `gorm:"type:varchar(100)" json:"option_1_name,omitempty"`
	Option1Value *string        `gorm:"type:varchar(255)" json:"option_1_value,omitempty"`
	Option2Name  *string        `gorm:"type:varchar(100)" json:"option_2_name,omitempty"`
	Option2Value *string        `gorm:"type:varchar(255)" json:"option_2_value,omitempty"`
	CostPrice    *float64       `gorm:"type:numeric(12,2)" json:"cost_price,omitempty"`
	SellingPrice *float64       `gorm:"type:numeric(12,2)" json:"selling_price,omitempty"`
	IsActive     bool           `gorm:"type:boolean;not null;default:true" json:"is_active"`
	CreatedAt    time.Time      `gorm:"type:timestamptz;not null;autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"type:timestamptz;not null;autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"type:timestamptz;index" json:"-"`
}

func (ProductVariant) TableName() string {
	return "product_variants"
}
