package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/marketplace-ops/backend/internal/marketplace"
	"github.com/marketplace-ops/backend/internal/models"
	"github.com/marketplace-ops/backend/internal/repositories"
	"github.com/marketplace-ops/backend/internal/security"
)

type IntegrationHandler struct {
	integrationRepo *repositories.IntegrationRepository
	storeRepo       *repositories.StoreRepository
}

func NewIntegrationHandler(integrationRepo *repositories.IntegrationRepository, storeRepo *repositories.StoreRepository) *IntegrationHandler {
	return &IntegrationHandler{
		integrationRepo: integrationRepo,
		storeRepo:       storeRepo,
	}
}

// ListIntegrations returns all marketplace credentials (safe response, no tokens).
// GET /api/integrations
func (h *IntegrationHandler) ListIntegrations(c *gin.Context) {
	creds, err := h.integrationRepo.FindAllCredentials()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to fetch integrations"))
		return
	}

	// Convert to safe responses
	responses := make([]models.CredentialResponse, len(creds))
	for i, cred := range creds {
		responses[i] = cred.ToResponse()
	}

	c.JSON(http.StatusOK, models.SuccessResponse(responses, ""))
}

// GetStoreIntegration returns the integration status for a specific store.
// GET /api/stores/:id/integration
func (h *IntegrationHandler) GetStoreIntegration(c *gin.Context) {
	storeID := c.Param("id")

	store, err := h.storeRepo.FindByID(storeID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse("NOT_FOUND", "Store not found"))
		return
	}

	cred, err := h.integrationRepo.FindCredentialByStoreAndMarketplace(storeID, store.Marketplace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to fetch integration"))
		return
	}

	if cred == nil {
		c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
			"store_id":          store.ID,
			"marketplace":       store.Marketplace,
			"connection_status": "not_configured",
			"has_credential":    false,
		}, ""))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(cred.ToResponse(), ""))
}

// InitiateIntegration starts the OAuth flow for a store's marketplace.
// POST /api/stores/:id/integration/initiate
func (h *IntegrationHandler) InitiateIntegration(c *gin.Context) {
	storeID := c.Param("id")

	store, err := h.storeRepo.FindByID(storeID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse("NOT_FOUND", "Store not found"))
		return
	}

	// Get adapter for this marketplace
	adapter, err := marketplace.GetAdapter(store.Marketplace)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("UNSUPPORTED_MARKETPLACE", err.Error()))
		return
	}

	// Generate cryptographically random state
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to generate OAuth state"))
		return
	}
	state := hex.EncodeToString(stateBytes)

	// Build the authorization URL
	authURL, err := adapter.BuildAuthorizationURL(storeID, state)
	if err != nil {
		// If credentials are missing, return a clear status
		if errors.Is(err, marketplace.ErrMissingCredentials) {
			c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
				"store_id":             store.ID,
				"marketplace":          store.Marketplace,
				"status":               "missing_credentials",
				"message":              "Marketplace API credentials are not configured. Please set up API keys in backend environment first.",
				"authorization_url":    nil,
				"requires_credentials": true,
			}, "Marketplace credentials not configured"))
			return
		}
		if errors.Is(err, marketplace.ErrNotImplemented) {
			c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
				"store_id":    store.ID,
				"marketplace": store.Marketplace,
				"status":      "not_implemented",
				"message":     "OAuth flow for this marketplace is not implemented yet.",
			}, "Integration not implemented"))
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to build authorization URL"))
		return
	}

	// Save OAuth state for callback validation
	oauthState := &models.MarketplaceOAuthState{
		State:       state,
		StoreID:     store.ID,
		Marketplace: store.Marketplace,
		Status:      "pending",
		ExpiresAt:   time.Now().Add(10 * time.Minute),
	}

	if err := h.integrationRepo.CreateOAuthState(oauthState); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to save OAuth state"))
		return
	}

	// Ensure credential record exists
	cred, err := h.integrationRepo.FindCredentialByStoreAndMarketplace(storeID, store.Marketplace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to check credentials"))
		return
	}
	if cred == nil {
		cred = &models.MarketplaceCredential{
			StoreID:          store.ID,
			Marketplace:      store.Marketplace,
			ConnectionStatus: "pending_authorization",
		}
		if err := h.integrationRepo.CreateCredential(cred); err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to create credential record"))
			return
		}
	} else {
		cred.ConnectionStatus = "pending_authorization"
		if err := h.integrationRepo.UpdateCredential(cred); err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to update credential"))
			return
		}
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"store_id":          store.ID,
		"marketplace":       store.Marketplace,
		"authorization_url": authURL,
		"state":             state,
		"expires_at":        oauthState.ExpiresAt,
	}, "Authorization URL generated"))
}

// OAuthCallback handles the OAuth callback from a marketplace.
// GET /api/integrations/:marketplace/callback
func (h *IntegrationHandler) OAuthCallback(c *gin.Context) {
	mktplace := c.Param("marketplace")
	state := c.Query("state")
	code := c.Query("code")

	// Validate marketplace
	adapter, err := marketplace.GetAdapter(mktplace)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("UNSUPPORTED_MARKETPLACE", "Unsupported marketplace: "+mktplace))
		return
	}

	// Validate state parameter
	if state == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("VALIDATION_ERROR", "Missing state parameter"))
		return
	}

	// Look up the OAuth state
	oauthState, err := h.integrationRepo.FindOAuthStateByState(state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to validate OAuth state"))
		return
	}

	if oauthState == nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("INVALID_STATE", "OAuth state not found or already used"))
		return
	}

	// Check if state is expired
	if time.Now().After(oauthState.ExpiresAt) {
		oauthState.Status = "expired"
		_ = h.integrationRepo.MarkOAuthStateFailed(oauthState)
		c.JSON(http.StatusBadRequest, models.ErrorResponse("STATE_EXPIRED", "OAuth state has expired. Please initiate the flow again."))
		return
	}

	// Check if state was already used
	if oauthState.Status != "pending" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("STATE_USED", "OAuth state has already been used"))
		return
	}

	// Validate code parameter
	if code == "" {
		_ = h.integrationRepo.MarkOAuthStateFailed(oauthState)
		c.JSON(http.StatusBadRequest, models.ErrorResponse("VALIDATION_ERROR", "Missing authorization code"))
		return
	}

	// Mark state as used
	_ = h.integrationRepo.MarkOAuthStateUsed(oauthState)

	shopID := c.Query("shop_id")
	tokenResult, err := adapter.ExchangeAuthCode(code, shopID)
	if err != nil {
		if errors.Is(err, marketplace.ErrNotImplemented) {
			c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
				"marketplace":  mktplace,
				"store_id":     oauthState.StoreID,
				"state_valid":  true,
				"code_present": code != "",
				"status":       "callback_received",
				"message":      "OAuth callback received and validated. Token exchange is not implemented yet.",
			}, "OAuth callback received but token exchange is not implemented yet"))
			return
		}

		cred, _ := h.integrationRepo.FindCredentialByStoreAndMarketplace(oauthState.StoreID.String(), mktplace)
		if cred != nil {
			errStr := err.Error()
			cred.LastError = &errStr
			cred.ConnectionStatus = "failed"
			_ = h.integrationRepo.UpdateCredential(cred)
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse("EXCHANGE_FAILED", "Failed to exchange authorization code: "+err.Error()))
		return
	}

	cred, _ := h.integrationRepo.FindCredentialByStoreAndMarketplace(oauthState.StoreID.String(), mktplace)
	if cred == nil {
		cred = &models.MarketplaceCredential{
			StoreID:     oauthState.StoreID,
			Marketplace: mktplace,
		}
	}

	if tokenResult != nil {
		encAccess, errA := security.EncryptToken(tokenResult.AccessToken)
		encRefresh, errR := security.EncryptToken(tokenResult.RefreshToken)

		if errA == nil && errR == nil {
			cred.EncryptedAccessToken = &encAccess
			cred.EncryptedRefreshToken = &encRefresh
			cred.AccessTokenExpiresAt = &tokenResult.AccessTokenExpiresAt
			cred.RefreshTokenExpiresAt = &tokenResult.RefreshTokenExpiresAt

			now := time.Now()
			cred.LastConnectedAt = &now
			cred.ConnectionStatus = "connected"
			cred.LastError = nil

			if cred.ID == uuid.Nil {
				_ = h.integrationRepo.CreateCredential(cred)
			} else {
				_ = h.integrationRepo.UpdateCredential(cred)
			}

			c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
				"marketplace": mktplace,
				"store_id":    oauthState.StoreID,
				"status":      "connected",
				"message":     "OAuth integration completed successfully.",
			}, "Integration successful"))
			return
		} else {
			errStr := "Failed to encrypt tokens"
			cred.LastError = &errStr
			cred.ConnectionStatus = "failed"
			if cred.ID == uuid.Nil {
				_ = h.integrationRepo.CreateCredential(cred)
			} else {
				_ = h.integrationRepo.UpdateCredential(cred)
			}
			c.JSON(http.StatusInternalServerError, models.ErrorResponse("ENCRYPTION_FAILED", errStr))
			return
		}
	}

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"marketplace":  mktplace,
		"store_id":     oauthState.StoreID,
		"state_valid":  true,
		"code_present": code != "",
		"status":       "callback_received",
		"message":      "OAuth callback received and validated.",
	}, "OAuth callback received"))
}

// DisconnectIntegration disconnects a store's marketplace integration.
// POST /api/stores/:id/integration/disconnect
func (h *IntegrationHandler) DisconnectIntegration(c *gin.Context) {
	storeID := c.Param("id")

	store, err := h.storeRepo.FindByID(storeID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse("NOT_FOUND", "Store not found"))
		return
	}

	cred, err := h.integrationRepo.FindCredentialByStoreAndMarketplace(storeID, store.Marketplace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to fetch credential"))
		return
	}

	if cred == nil {
		c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
			"store_id":          store.ID,
			"marketplace":       store.Marketplace,
			"connection_status": "not_configured",
			"message":           "No integration configured for this store.",
		}, "No integration to disconnect"))
		return
	}

	// Clear tokens and mark as disconnected
	if err := h.integrationRepo.DisconnectCredential(cred); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to disconnect integration"))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(cred.ToResponse(), "Integration disconnected successfully"))
}

// TestConnection tests the connection status for a store's marketplace integration.
// POST /api/stores/:id/integration/test
func (h *IntegrationHandler) TestConnection(c *gin.Context) {
	storeID := c.Param("id")

	store, err := h.storeRepo.FindByID(storeID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse("NOT_FOUND", "Store not found"))
		return
	}

	// Check if adapter exists
	adapter, err := marketplace.GetAdapter(store.Marketplace)
	if err != nil {
		c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
			"store_id":    store.ID,
			"marketplace": store.Marketplace,
			"status":      "unsupported",
			"message":     "This marketplace is not supported.",
		}, "Marketplace not supported"))
		return
	}

	// Check credential validity
	credErr := adapter.ValidateCredentials()

	cred, _ := h.integrationRepo.FindCredentialByStoreAndMarketplace(storeID, store.Marketplace)

	status := "not_configured"
	message := "No credentials configured for this marketplace."

	if cred != nil {
		status = cred.ConnectionStatus
	}

	if credErr != nil {
		if errors.Is(credErr, marketplace.ErrMissingCredentials) {
			status = "missing_credentials"
			message = "Marketplace API credentials are not configured in backend environment."
		} else if errors.Is(credErr, marketplace.ErrNotImplemented) {
			status = "not_implemented"
			message = "Marketplace integration is not implemented yet."
		}
	}

	result := gin.H{
		"store_id":          store.ID,
		"marketplace":       store.Marketplace,
		"connection_status": status,
		"message":           message,
		"tested_at":         time.Now(),
	}

	// Add credential info if exists
	if cred != nil {
		result["has_credential"] = true
		result["has_access_token"] = cred.EncryptedAccessToken != nil && *cred.EncryptedAccessToken != ""
		result["has_refresh_token"] = cred.EncryptedRefreshToken != nil && *cred.EncryptedRefreshToken != ""
	} else {
		result["has_credential"] = false
	}

	c.JSON(http.StatusOK, models.SuccessResponse(result, message))
}

// ListSupportedMarketplaces returns the list of supported marketplaces and their integration status.
// GET /api/integrations/marketplaces
func (h *IntegrationHandler) ListSupportedMarketplaces(c *gin.Context) {
	marketplaces := marketplace.SupportedMarketplaces()

	// For each marketplace, check if adapter validates
	for i, mp := range marketplaces {
		mpID, ok := mp["id"].(string)
		if !ok {
			continue
		}
		adapter, err := marketplace.GetAdapter(mpID)
		if err != nil {
			continue
		}
		credErr := adapter.ValidateCredentials()
		if credErr == nil {
			marketplaces[i]["credentials_configured"] = true
		} else {
			marketplaces[i]["credentials_configured"] = false
		}
	}

	c.JSON(http.StatusOK, models.SuccessResponse(marketplaces, ""))
}
