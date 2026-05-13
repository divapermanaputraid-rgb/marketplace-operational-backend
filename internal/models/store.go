package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Store struct {
	ID               uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Marketplace      string         `gorm:"type:varchar(50);not null;index" json:"marketplace"` // shopee, tokopedia_shop, tiktok_shop
	StoreName        string         `gorm:"type:varchar(255);not null" json:"store_name"`
	StoreURL         *string        `gorm:"type:text" json:"store_url,omitempty"`
	ExternalStoreID  *string        `gorm:"type:varchar(255);index" json:"external_store_id,omitempty"`
	ConnectionStatus string         `gorm:"type:varchar(50);not null;default:'not_connected'" json:"connection_status"` // not_connected, connected, token_expired, sync_error, disabled
	IsActive         bool           `gorm:"type:boolean;not null;default:true" json:"is_active"`
	Notes            *string        `gorm:"type:text" json:"notes,omitempty"`
	LastSyncAt       *time.Time     `gorm:"type:timestamptz" json:"last_sync_at,omitempty"`
	CreatedAt        time.Time      `gorm:"type:timestamptz;not null;autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time      `gorm:"type:timestamptz;not null;autoUpdateTime" json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"type:timestamptz;index" json:"-"`
}

func (Store) TableName() string {
	return "stores"
}

[diff_block_start]
@@ -28,21 +28,1 @@
-
-type MarketplaceConnection struct {
-	ID                    uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
-	StoreID               uuid.UUID  `gorm:"type:uuid;not null;index;unique" json:"store_id"`
-	Marketplace           string     `gorm:"type:varchar(50);not null" json:"marketplace"`
-	ConnectionStatus      string     `gorm:"type:varchar(50);not null;default:'not_connected'" json:"connection_status"`
-	AccessTokenEncrypted  *string    `gorm:"type:text" json:"-"` // DO NOT EXPOSE
-	RefreshTokenEncrypted *string    `gorm:"type:text" json:"-"` // DO NOT EXPOSE
-	TokenExpiresAt        *time.Time `gorm:"type:timestamptz" json:"token_expires_at,omitempty"`
-	Scopes                *string    `gorm:"type:text" json:"scopes,omitempty"`
-	ConnectedAt           *time.Time `gorm:"type:timestamptz" json:"connected_at,omitempty"`
-	DisconnectedAt        *time.Time `gorm:"type:timestamptz" json:"disconnected_at,omitempty"`
-	LastError             *string    `gorm:"type:text" json:"last_error,omitempty"`
-	CreatedAt             time.Time  `gorm:"type:timestamptz;not null;autoCreateTime" json:"created_at"`
-	UpdatedAt             time.Time  `gorm:"type:timestamptz;not null;autoUpdateTime" json:"updated_at"`
-}
-
-func (MarketplaceConnection) TableName() string {
-	return "marketplace_connections"
-}
[diff_block_end]
