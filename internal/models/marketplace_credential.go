package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MarketplaceCredential stores OAuth credentials for marketplace integrations.
// Tokens are encrypted at rest using AES-256-GCM.
// SECURITY: encrypted_access_token and encrypted_refresh_token must NEVER
// be returned in API responses.
type MarketplaceCredential struct {
	ID                    uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	StoreID               uuid.UUID      `gorm:"type:uuid;not null;index:idx_cred_store_marketplace,unique,where:deleted_at IS NULL" json:"store_id"`
	Marketplace           string         `gorm:"type:varchar(50);not null;index:idx_cred_store_marketplace,unique,where:deleted_at IS NULL;index:idx_cred_marketplace" json:"marketplace"` // shopee, tiktok_shop, tokopedia_shop
	ConnectionStatus      string         `gorm:"type:varchar(50);not null;default:'not_configured';index:idx_cred_connection_status" json:"connection_status"`                             // not_configured, pending_authorization, connected, expired, failed, disconnected
	AppID                 *string        `gorm:"type:varchar(255)" json:"app_id,omitempty"`
	EncryptedAccessToken  *string        `gorm:"type:text" json:"-"` // NEVER expose in JSON
	EncryptedRefreshToken *string        `gorm:"type:text" json:"-"` // NEVER expose in JSON
	AccessTokenExpiresAt  *time.Time     `gorm:"type:timestamptz" json:"access_token_expires_at,omitempty"`
	RefreshTokenExpiresAt *time.Time     `gorm:"type:timestamptz" json:"refresh_token_expires_at,omitempty"`
	Scopes                *string        `gorm:"type:text" json:"scopes,omitempty"`
	LastConnectedAt       *time.Time     `gorm:"type:timestamptz" json:"last_connected_at,omitempty"`
	LastRefreshedAt       *time.Time     `gorm:"type:timestamptz" json:"last_refreshed_at,omitempty"`
	LastError             *string        `gorm:"type:text" json:"last_error,omitempty"`
	CreatedAt             time.Time      `gorm:"type:timestamptz;not null;autoCreateTime" json:"created_at"`
	UpdatedAt             time.Time      `gorm:"type:timestamptz;not null;autoUpdateTime" json:"updated_at"`
	DeletedAt             gorm.DeletedAt `gorm:"type:timestamptz;index:idx_cred_deleted_at" json:"-"`

	Store *Store `gorm:"foreignKey:StoreID" json:"store,omitempty"`
}

func (MarketplaceCredential) TableName() string {
	return "marketplace_credentials"
}

// CredentialResponse is the safe API representation that excludes encrypted tokens.
type CredentialResponse struct {
	ID                    uuid.UUID  `json:"id"`
	StoreID               uuid.UUID  `json:"store_id"`
	Marketplace           string     `json:"marketplace"`
	ConnectionStatus      string     `json:"connection_status"`
	AppID                 *string    `json:"app_id,omitempty"`
	HasAccessToken        bool       `json:"has_access_token"`
	HasRefreshToken       bool       `json:"has_refresh_token"`
	AccessTokenExpiresAt  *time.Time `json:"access_token_expires_at,omitempty"`
	RefreshTokenExpiresAt *time.Time `json:"refresh_token_expires_at,omitempty"`
	Scopes                *string    `json:"scopes,omitempty"`
	LastConnectedAt       *time.Time `json:"last_connected_at,omitempty"`
	LastRefreshedAt       *time.Time `json:"last_refreshed_at,omitempty"`
	LastError             *string    `json:"last_error,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

// ToResponse converts the credential to a safe API response without any token data.
func (c *MarketplaceCredential) ToResponse() CredentialResponse {
	return CredentialResponse{
		ID:                    c.ID,
		StoreID:               c.StoreID,
		Marketplace:           c.Marketplace,
		ConnectionStatus:      c.ConnectionStatus,
		AppID:                 c.AppID,
		HasAccessToken:        c.EncryptedAccessToken != nil && *c.EncryptedAccessToken != "",
		HasRefreshToken:       c.EncryptedRefreshToken != nil && *c.EncryptedRefreshToken != "",
		AccessTokenExpiresAt:  c.AccessTokenExpiresAt,
		RefreshTokenExpiresAt: c.RefreshTokenExpiresAt,
		Scopes:                c.Scopes,
		LastConnectedAt:       c.LastConnectedAt,
		LastRefreshedAt:       c.LastRefreshedAt,
		LastError:             c.LastError,
		CreatedAt:             c.CreatedAt,
		UpdatedAt:             c.UpdatedAt,
	}
}

// MarketplaceOAuthState tracks OAuth authorization flow state to prevent CSRF
// and ensure valid callback handling.
type MarketplaceOAuthState struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	State       string     `gorm:"type:varchar(255);uniqueIndex:idx_oauth_state;not null" json:"state"`
	StoreID     uuid.UUID  `gorm:"type:uuid;not null;index:idx_oauth_state_store" json:"store_id"`
	Marketplace string     `gorm:"type:varchar(50);not null" json:"marketplace"`
	RedirectURI *string    `gorm:"type:text" json:"redirect_uri,omitempty"`
	Status      string     `gorm:"type:varchar(50);not null;default:'pending'" json:"status"` // pending, used, expired, failed
	ExpiresAt   time.Time  `gorm:"type:timestamptz;not null;index:idx_oauth_state_expires" json:"expires_at"`
	UsedAt      *time.Time `gorm:"type:timestamptz" json:"used_at,omitempty"`
	CreatedAt   time.Time  `gorm:"type:timestamptz;not null;autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"type:timestamptz;not null;autoUpdateTime" json:"updated_at"`

	Store *Store `gorm:"foreignKey:StoreID" json:"store,omitempty"`
}

func (MarketplaceOAuthState) TableName() string {
	return "marketplace_oauth_states"
}
