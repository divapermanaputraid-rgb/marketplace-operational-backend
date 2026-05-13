package repositories

import (
	"errors"
	"time"

	"github.com/marketplace-ops/backend/internal/models"
	"gorm.io/gorm"
)

type IntegrationRepository struct {
	db *gorm.DB
}

func NewIntegrationRepository(db *gorm.DB) *IntegrationRepository {
	return &IntegrationRepository{db: db}
}

// --- MarketplaceCredential operations ---

// FindAllCredentials returns all active (non-deleted) marketplace credentials.
func (r *IntegrationRepository) FindAllCredentials() ([]models.MarketplaceCredential, error) {
	var creds []models.MarketplaceCredential
	err := r.db.Preload("Store").Order("created_at desc").Find(&creds).Error
	return creds, err
}

// FindCredentialByStoreID returns the credential for a specific store.
func (r *IntegrationRepository) FindCredentialByStoreID(storeID string) (*models.MarketplaceCredential, error) {
	var cred models.MarketplaceCredential
	err := r.db.Where("store_id = ?", storeID).First(&cred).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // No credential found, not an error
		}
		return nil, err
	}
	return &cred, nil
}

// FindCredentialByStoreAndMarketplace returns the credential for a specific store and marketplace.
func (r *IntegrationRepository) FindCredentialByStoreAndMarketplace(storeID, marketplace string) (*models.MarketplaceCredential, error) {
	var cred models.MarketplaceCredential
	err := r.db.Where("store_id = ? AND marketplace = ?", storeID, marketplace).First(&cred).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &cred, nil
}

// CreateCredential creates a new marketplace credential record.
func (r *IntegrationRepository) CreateCredential(cred *models.MarketplaceCredential) error {
	return r.db.Create(cred).Error
}

// UpdateCredential updates an existing marketplace credential record.
func (r *IntegrationRepository) UpdateCredential(cred *models.MarketplaceCredential) error {
	return r.db.Save(cred).Error
}

// DisconnectCredential marks a credential as disconnected and clears encrypted tokens.
func (r *IntegrationRepository) DisconnectCredential(cred *models.MarketplaceCredential) error {
	cred.ConnectionStatus = "disconnected"
	cred.EncryptedAccessToken = nil
	cred.EncryptedRefreshToken = nil
	cred.AccessTokenExpiresAt = nil
	cred.RefreshTokenExpiresAt = nil
	cred.LastError = nil
	return r.db.Save(cred).Error
}

// --- MarketplaceOAuthState operations ---

// CreateOAuthState creates a new OAuth state record.
func (r *IntegrationRepository) CreateOAuthState(state *models.MarketplaceOAuthState) error {
	return r.db.Create(state).Error
}

// FindOAuthStateByState looks up an OAuth state by its state string.
func (r *IntegrationRepository) FindOAuthStateByState(state string) (*models.MarketplaceOAuthState, error) {
	var oauthState models.MarketplaceOAuthState
	err := r.db.Where("state = ?", state).First(&oauthState).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &oauthState, nil
}

// MarkOAuthStateUsed marks an OAuth state as used.
func (r *IntegrationRepository) MarkOAuthStateUsed(state *models.MarketplaceOAuthState) error {
	now := time.Now()
	state.Status = "used"
	state.UsedAt = &now
	return r.db.Save(state).Error
}

// MarkOAuthStateFailed marks an OAuth state as failed.
func (r *IntegrationRepository) MarkOAuthStateFailed(state *models.MarketplaceOAuthState) error {
	state.Status = "failed"
	return r.db.Save(state).Error
}
