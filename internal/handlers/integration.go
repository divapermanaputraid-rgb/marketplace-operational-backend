package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/marketplace-ops/backend/internal/marketplace"
	"github.com/marketplace-ops/backend/internal/models"
	"github.com/marketplace-ops/backend/internal/repositories"
	"github.com/marketplace-ops/backend/internal/security"
	"github.com/marketplace-ops/backend/internal/services"
)

type IntegrationHandler struct {
	integrationRepo *repositories.IntegrationRepository
	storeRepo       *repositories.StoreRepository
	orderRepo       *repositories.OrderRepository
	mappingRepo     *repositories.ProductMappingRepository
	syncRepo        *repositories.SyncRepository
	productRepo     *repositories.ProductRepository
	inventoryRepo   *repositories.InventoryRepository
}

func NewIntegrationHandler(
	integrationRepo *repositories.IntegrationRepository,
	storeRepo *repositories.StoreRepository,
	orderRepo *repositories.OrderRepository,
	mappingRepo *repositories.ProductMappingRepository,
	syncRepo *repositories.SyncRepository,
	productRepo *repositories.ProductRepository,
	inventoryRepo *repositories.InventoryRepository,
) *IntegrationHandler {
	return &IntegrationHandler{
		integrationRepo: integrationRepo,
		storeRepo:       storeRepo,
		orderRepo:       orderRepo,
		mappingRepo:     mappingRepo,
		syncRepo:        syncRepo,
		productRepo:     productRepo,
		inventoryRepo:   inventoryRepo,
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

// PullOrders manually pulls orders for a marketplace integration.
// POST /api/stores/:id/integration/orders/pull
func (h *IntegrationHandler) PullOrders(c *gin.Context) {
	startTime := time.Now()
	storeID := c.Param("id")

	var req struct {
		TimeFrom    int64  `json:"time_from"`
		TimeTo      int64  `json:"time_to"`
		OrderStatus string `json:"order_status"` // Optional
		PageSize    int    `json:"page_size"`    // Optional
		Cursor      string `json:"cursor"`       // Optional for pagination
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("INVALID_REQUEST", "Invalid request body"))
		return
	}

	// Validation
	if req.TimeFrom <= 0 || req.TimeTo <= 0 {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("VALIDATION_FAILED", "time_from and time_to are required and must be positive"))
		return
	}

	if req.TimeFrom >= req.TimeTo {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("VALIDATION_FAILED", "time_from must be before time_to"))
		return
	}

	// Limit time window to 15 days for Shopee safety
	maxWindow := int64(15 * 24 * 60 * 60)
	if req.TimeTo-req.TimeFrom > maxWindow {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("VALIDATION_FAILED", "Time window cannot exceed 15 days"))
		return
	}

	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 50 // Default safe batch
	}

	store, err := h.storeRepo.FindByID(storeID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse("NOT_FOUND", "Store not found"))
		return
	}

	adapter, err := marketplace.GetAdapter(store.Marketplace)
	if err != nil {
		c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
			"status":  "unsupported",
			"message": "This marketplace is not supported.",
		}, "Marketplace not supported"))
		return
	}

	// Only Shopee supported for manual pull in this sprint
	if store.Marketplace != "shopee" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("VALIDATION_ERROR", "Order pull is currently only supported for Shopee"))
		return
	}

	credErr := adapter.ValidateCredentials()
	if credErr != nil {
		status := "not_configured"
		msg := "Marketplace API credentials are not configured in backend environment."

		// Create SyncLog for audit
		durationMs := int64(time.Since(startTime).Milliseconds())
		now := time.Now()
		logRec := &models.SyncLog{
			StoreID:       &store.ID,
			Marketplace:   store.Marketplace,
			SyncType:      "orders",
			SyncDirection: "pull",
			Status:        status,
			Message:       &msg,
			StartedAt:     &startTime,
			FinishedAt:    &now,
			DurationMs:    &durationMs,
		}
		_ = h.syncRepo.CreateSyncLog(logRec)

		c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
			"status":               status,
			"message":              msg,
			"records_processed":    0,
			"records_created":      0,
			"records_updated":      0,
			"records_failed":       0,
			"unmapped_items_count": 0,
		}, msg))
		return
	}

	cred, _ := h.integrationRepo.FindCredentialByStoreAndMarketplace(storeID, store.Marketplace)
	if cred == nil {
		status := "not_configured"
		msg := "No credentials configured for this marketplace."

		// Create SyncLog for audit
		durationMs := int64(time.Since(startTime).Milliseconds())
		now := time.Now()
		logRec := &models.SyncLog{
			StoreID:       &store.ID,
			Marketplace:   store.Marketplace,
			SyncType:      "orders",
			SyncDirection: "pull",
			Status:        status,
			Message:       &msg,
			StartedAt:     &startTime,
			FinishedAt:    &now,
			DurationMs:    &durationMs,
		}
		_ = h.syncRepo.CreateSyncLog(logRec)

		c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
			"status":               status,
			"message":              msg,
			"records_processed":    0,
			"records_created":      0,
			"records_updated":      0,
			"records_failed":       0,
			"unmapped_items_count": 0,
		}, msg))
		return
	}

	// Validate token expiration
	if cred.AccessTokenExpiresAt != nil && time.Now().After(*cred.AccessTokenExpiresAt) {
		cred.ConnectionStatus = "expired"
		errStr := "Access token has expired."
		cred.LastError = &errStr
		_ = h.integrationRepo.UpdateCredential(cred)

		status := "expired"
		msg := "Access token expired. Please reconnect."

		// Create SyncLog for audit
		durationMs := int64(time.Since(startTime).Milliseconds())
		now := time.Now()
		logRec := &models.SyncLog{
			StoreID:       &store.ID,
			Marketplace:   store.Marketplace,
			SyncType:      "orders",
			SyncDirection: "pull",
			Status:        status,
			Message:       &msg,
			StartedAt:     &startTime,
			FinishedAt:    &now,
			DurationMs:    &durationMs,
		}
		_ = h.syncRepo.CreateSyncLog(logRec)

		c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
			"status":               status,
			"message":              msg,
			"records_processed":    0,
			"records_created":      0,
			"records_updated":      0,
			"records_failed":       0,
			"unmapped_items_count": 0,
		}, msg))
		return
	}

	if cred.EncryptedAccessToken == nil || *cred.EncryptedAccessToken == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("TOKEN_MISSING", "Access token is missing."))
		return
	}

	accessToken, err := security.DecryptToken(*cred.EncryptedAccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("DECRYPTION_FAILED", "Failed to decrypt token for validation"))
		return
	}

	extStoreID := ""
	if store.ExternalStoreID != nil {
		extStoreID = *store.ExternalStoreID
	}

	// Pull order list from marketplace
	listRes, err := adapter.PullOrders(accessToken, extStoreID, req.TimeFrom, req.TimeTo, req.PageSize, req.Cursor)

	if err != nil {
		if errors.Is(err, marketplace.ErrNotImplemented) {
			c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
				"status":  "not_implemented",
				"message": "Pull orders is not implemented for this marketplace.",
			}, "Pull orders not implemented"))
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("API_ERROR", "Failed to pull orders: "+err.Error()))
		return
	}

	// Attempt to pull details for the batch
	var orderSNs []string
	if listRes != nil {
		for _, o := range listRes.Orders {
			orderSNs = append(orderSNs, o.OrderSN)
		}
	}

	var details []marketplace.ShopeeOrderDetail
	if len(orderSNs) > 0 {
		details, err = adapter.GetOrderDetails(accessToken, extStoreID, orderSNs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse("API_ERROR", "Failed to pull order details: "+err.Error()))
			return
		}
	}

	// Mapper for Shopee orders
	mapper := services.NewShopeeOrderMapper(h.orderRepo, h.mappingRepo)
	var recordsCreated, recordsUpdated, recordsFailed, totalUnmappedCount int
	var pullErrors []string

	for _, detail := range details {
		created, updated, unmappedCount, mapErr := mapper.MapAndPersist(store.ID, detail)
		if mapErr != nil {
			recordsFailed++
			pullErrors = append(pullErrors, fmt.Sprintf("Order %s: %v", detail.OrderSN, mapErr))
			continue
		}
		if created {
			recordsCreated++
		} else if updated {
			recordsUpdated++
		}
		totalUnmappedCount += unmappedCount
	}

	statusMsg := "Orders pulled and mapped successfully."
	finalStatus := "success"
	if recordsFailed > 0 {
		if recordsCreated > 0 || recordsUpdated > 0 {
			finalStatus = "partial"
			statusMsg = "Orders pulled with some failures."
		} else {
			finalStatus = "failed"
			statusMsg = "Failed to process pulled orders."
		}
	}

	// Create SyncLog
	durationMs := int64(time.Since(startTime).Milliseconds())
	now := time.Now()
	msg := statusMsg

	// Prepare raw summary without secrets
	summary := gin.H{
		"time_from":            req.TimeFrom,
		"time_to":              req.TimeTo,
		"order_status_filter":  req.OrderStatus,
		"page_size":            req.PageSize,
		"has_next_page":        false,
		"next_cursor":          "",
		"records_processed":    len(details),
		"records_created":      recordsCreated,
		"records_updated":      recordsUpdated,
		"records_failed":       recordsFailed,
		"unmapped_items_count": totalUnmappedCount,
		"errors":               pullErrors,
	}

	if listRes != nil {
		summary["has_next_page"] = listRes.More
		summary["next_cursor"] = listRes.NextCursor
	}

	summaryJSON, _ := json.Marshal(summary)
	summaryStr := string(summaryJSON)

	var firstErrMsg *string
	if len(pullErrors) > 0 {
		firstErrMsg = &pullErrors[0]
	}

	logRec := &models.SyncLog{
		StoreID:          &store.ID,
		Marketplace:      store.Marketplace,
		SyncType:         "orders",
		SyncDirection:    "pull",
		Status:           finalStatus,
		Message:          &msg,
		RecordsProcessed: len(details),
		RecordsCreated:   recordsCreated,
		RecordsUpdated:   recordsUpdated,
		RecordsFailed:    recordsFailed,
		ErrorMessage:     firstErrMsg,
		StartedAt:        &startTime,
		FinishedAt:       &now,
		DurationMs:       &durationMs,
		RawSummary:       &summaryStr,
	}
	_ = h.syncRepo.CreateSyncLog(logRec)

	response := gin.H{
		"status":               finalStatus,
		"message":              statusMsg,
		"records_processed":    len(details),
		"records_created":      recordsCreated,
		"records_updated":      recordsUpdated,
		"records_failed":       recordsFailed,
		"unmapped_items_count": totalUnmappedCount,
		"sync_log_id":          logRec.ID,
		"errors":               pullErrors,
	}

	if listRes != nil {
		response["has_next_page"] = listRes.More
		response["next_cursor"] = listRes.NextCursor
	}

	c.JSON(http.StatusOK, models.SuccessResponse(response, statusMsg))

}

// PullProducts manually pulls products for a marketplace integration.
// POST /api/stores/:id/integration/products/pull
func (h *IntegrationHandler) PullProducts(c *gin.Context) {
	startTime := time.Now()
	storeID := c.Param("id")

	var req struct {
		Offset     int    `json:"offset"`
		PageSize   int    `json:"page_size"`
		ItemStatus string `json:"item_status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("INVALID_REQUEST", "Invalid request body"))
		return
	}

	store, err := h.storeRepo.FindByID(storeID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse("NOT_FOUND", "Store not found"))
		return
	}

	adapter, err := marketplace.GetAdapter(store.Marketplace)
	if err != nil {
		c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
			"status":  "unsupported",
			"message": "This marketplace is not supported.",
		}, "Marketplace not supported"))
		return
	}

	// Only Shopee supported for products in this sprint
	if store.Marketplace != "shopee" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("VALIDATION_ERROR", "Product pull is currently only supported for Shopee"))
		return
	}

	credErr := adapter.ValidateCredentials()
	if credErr != nil {
		status := "not_configured"
		msg := "Marketplace API credentials are not configured in backend environment."

		// Create SyncLog for audit
		durationMs := int64(time.Since(startTime).Milliseconds())
		now := time.Now()
		logRec := &models.SyncLog{
			StoreID:       &store.ID,
			Marketplace:   store.Marketplace,
			SyncType:      "products",
			SyncDirection: "pull",
			Status:        status,
			Message:       &msg,
			StartedAt:     &startTime,
			FinishedAt:    &now,
			DurationMs:    &durationMs,
		}
		_ = h.syncRepo.CreateSyncLog(logRec)

		c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
			"status":            status,
			"message":           msg,
			"records_processed": 0,
			"products_count":    0,
		}, msg))
		return
	}

	cred, _ := h.integrationRepo.FindCredentialByStoreAndMarketplace(storeID, store.Marketplace)
	if cred == nil {
		status := "not_configured"
		msg := "No credentials configured for this marketplace."

		// Create SyncLog for audit
		durationMs := int64(time.Since(startTime).Milliseconds())
		now := time.Now()
		logRec := &models.SyncLog{
			StoreID:       &store.ID,
			Marketplace:   store.Marketplace,
			SyncType:      "products",
			SyncDirection: "pull",
			Status:        status,
			Message:       &msg,
			StartedAt:     &startTime,
			FinishedAt:    &now,
			DurationMs:    &durationMs,
		}
		_ = h.syncRepo.CreateSyncLog(logRec)

		c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
			"status":            status,
			"message":           msg,
			"records_processed": 0,
			"products_count":    0,
		}, msg))
		return
	}

	// Validate token expiration
	if cred.AccessTokenExpiresAt != nil && time.Now().After(*cred.AccessTokenExpiresAt) {
		cred.ConnectionStatus = "expired"
		errStr := "Access token has expired."
		cred.LastError = &errStr
		_ = h.integrationRepo.UpdateCredential(cred)

		status := "expired"
		msg := "Access token expired. Please reconnect."

		// Create SyncLog for audit
		durationMs := int64(time.Since(startTime).Milliseconds())
		now := time.Now()
		logRec := &models.SyncLog{
			StoreID:       &store.ID,
			Marketplace:   store.Marketplace,
			SyncType:      "products",
			SyncDirection: "pull",
			Status:        status,
			Message:       &msg,
			StartedAt:     &startTime,
			FinishedAt:    &now,
			DurationMs:    &durationMs,
		}
		_ = h.syncRepo.CreateSyncLog(logRec)

		c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
			"status":            status,
			"message":           msg,
			"records_processed": 0,
			"products_count":    0,
		}, msg))
		return
	}

	if cred.EncryptedAccessToken == nil || *cred.EncryptedAccessToken == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("TOKEN_MISSING", "Access token is missing."))
		return
	}

	accessToken, err := security.DecryptToken(*cred.EncryptedAccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("DECRYPTION_FAILED", "Failed to decrypt token"))
		return
	}

	extStoreID := ""
	if store.ExternalStoreID != nil {
		extStoreID = *store.ExternalStoreID
	}

	// 1. Pull product list (IDs)
	listRes, err := adapter.PullProducts(accessToken, extStoreID, req.Offset, req.PageSize, req.ItemStatus)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("API_ERROR", "Failed to pull product list: "+err.Error()))
		return
	}

	// 2. Pull details for the items
	var itemIDs []int64
	for _, item := range listRes.Items {
		itemIDs = append(itemIDs, item.ItemID)
	}

	var products []marketplace.ShopeeProductDetail
	if len(itemIDs) > 0 {
		products, err = adapter.GetProductDetails(accessToken, extStoreID, itemIDs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse("API_ERROR", "Failed to pull product details: "+err.Error()))
			return
		}
	}

	// Sprint 22A: Do not persist to internal database yet.
	// We just return a safe preview of the pulled products.

	statusMsg := "Products pulled successfully (Preview only)."
	finalStatus := "success"

	// Create SyncLog
	durationMs := int64(time.Since(startTime).Milliseconds())
	now := time.Now()
	msg := statusMsg

	// Prepare raw summary without secrets
	summary := gin.H{
		"records_processed": len(products),
		"has_next_page":     listRes.HasNextPage,
		"next_offset":       listRes.NextOffset,
		"product_ids":       itemIDs,
	}
	summaryJSON, _ := json.Marshal(summary)
	summaryStr := string(summaryJSON)

	logRec := &models.SyncLog{
		StoreID:          &store.ID,
		Marketplace:      store.Marketplace,
		SyncType:         "products",
		SyncDirection:    "pull",
		Status:           finalStatus,
		Message:          &msg,
		RecordsProcessed: len(products),
		StartedAt:        &startTime,
		FinishedAt:       &now,
		DurationMs:       &durationMs,
		RawSummary:       &summaryStr,
	}
	_ = h.syncRepo.CreateSyncLog(logRec)

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"status":            finalStatus,
		"message":           statusMsg,
		"records_processed": len(products),
		"products_count":    len(products),
		"has_next_page":     listRes.HasNextPage,
		"next_offset":       listRes.NextOffset,
		"products":          products, // Safe product detail list
		"sync_log_id":       logRec.ID,
	}, statusMsg))
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

	credErr := adapter.ValidateCredentials()
	if credErr != nil {
		status := "missing_credentials"
		message := "Marketplace API credentials are not configured in backend environment."
		if errors.Is(credErr, marketplace.ErrNotImplemented) {
			status = "not_implemented"
			message = "Marketplace integration is not implemented yet."
		}
		c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
			"store_id":          store.ID,
			"marketplace":       store.Marketplace,
			"connection_status": status,
			"message":           message,
			"tested_at":         time.Now(),
		}, message))
		return
	}

	cred, _ := h.integrationRepo.FindCredentialByStoreAndMarketplace(storeID, store.Marketplace)
	if cred == nil {
		c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
			"store_id":          store.ID,
			"marketplace":       store.Marketplace,
			"connection_status": "not_configured",
			"message":           "No credentials configured for this marketplace.",
			"tested_at":         time.Now(),
		}, "No credentials configured"))
		return
	}

	// Validate token expiration
	if cred.AccessTokenExpiresAt != nil && time.Now().After(*cred.AccessTokenExpiresAt) {
		cred.ConnectionStatus = "expired"
		errStr := "Access token has expired."
		cred.LastError = &errStr
		_ = h.integrationRepo.UpdateCredential(cred)

		c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
			"store_id":          store.ID,
			"marketplace":       store.Marketplace,
			"connection_status": cred.ConnectionStatus,
			"message":           "Access token expired.",
			"tested_at":         time.Now(),
			"token_expired":     true,
			"needs_reconnect":   true,
		}, "Access token expired"))
		return
	}

	if cred.EncryptedAccessToken == nil || *cred.EncryptedAccessToken == "" {
		c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
			"store_id":          store.ID,
			"marketplace":       store.Marketplace,
			"connection_status": "failed",
			"message":           "Access token is missing.",
			"tested_at":         time.Now(),
			"needs_reconnect":   true,
		}, "Access token missing"))
		return
	}

	accessToken, err := security.DecryptToken(*cred.EncryptedAccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("DECRYPTION_FAILED", "Failed to decrypt token for validation"))
		return
	}

	// Make the real API call
	extStoreID := ""
	if store.ExternalStoreID != nil {
		extStoreID = *store.ExternalStoreID
	}
	shopInfo, err := adapter.GetShopInfo(accessToken, extStoreID)

	result := gin.H{
		"store_id":    store.ID,
		"marketplace": store.Marketplace,
		"tested_at":   time.Now(),
	}

	if err != nil {
		if errors.Is(err, marketplace.ErrNotImplemented) {
			result["connection_status"] = cred.ConnectionStatus
			result["message"] = "Integration connected, but validation endpoint is not implemented."
			c.JSON(http.StatusOK, models.SuccessResponse(result, "Connected (validation not supported)"))
			return
		}

		errStr := err.Error()
		cred.LastError = &errStr
		cred.ConnectionStatus = "failed"
		_ = h.integrationRepo.UpdateCredential(cred)

		result["connection_status"] = "failed"
		result["message"] = "Shopee API connection failed: " + errStr
		result["last_error"] = errStr

		c.JSON(http.StatusOK, models.SuccessResponse(result, "API Connection Failed"))
		return
	}

	now := time.Now()
	cred.ConnectionStatus = "connected"
	cred.LastConnectedAt = &now
	cred.LastError = nil
	_ = h.integrationRepo.UpdateCredential(cred)

	result["connection_status"] = "connected"
	result["message"] = "Connection to Shopee API successful."
	result["shop_info"] = shopInfo

	c.JSON(http.StatusOK, models.SuccessResponse(result, "Connection successful"))
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

// PreviewMappingCandidates pulls products and identifies mapping status.
// POST /api/stores/:id/integration/products/mapping-candidates
func (h *IntegrationHandler) PreviewMappingCandidates(c *gin.Context) {
	startTime := time.Now()
	storeID := c.Param("id")

	store, err := h.storeRepo.FindByID(storeID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse("NOT_FOUND", "Store not found"))
		return
	}

	adapter, err := marketplace.GetAdapter(store.Marketplace)
	if err != nil || store.Marketplace != "shopee" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("UNSUPPORTED_MARKETPLACE", "Mapping candidates currently only supported for Shopee"))
		return
	}

	credErr := adapter.ValidateCredentials()
	if credErr != nil {
		c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
			"status":            "not_configured",
			"message":           "Marketplace API credentials not configured.",
			"records_processed": 0,
		}, "Credentials not configured"))
		return
	}

	cred, _ := h.integrationRepo.FindCredentialByStoreAndMarketplace(storeID, store.Marketplace)
	if cred == nil || (cred.AccessTokenExpiresAt != nil && time.Now().After(*cred.AccessTokenExpiresAt)) {
		status := "not_configured"
		if cred != nil {
			status = "expired"
		}
		c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
			"status":            status,
			"message":           "Connection expired or not configured.",
			"records_processed": 0,
		}, "Connection issue"))
		return
	}

	accessToken, _ := security.DecryptToken(*cred.EncryptedAccessToken)
	extStoreID := ""
	if store.ExternalStoreID != nil {
		extStoreID = *store.ExternalStoreID
	}

	// 1. Pull product list (IDs)
	listRes, err := adapter.PullProducts(accessToken, extStoreID, 0, 50, "NORMAL")
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("API_ERROR", "Failed to pull product list: "+err.Error()))
		return
	}

	// 2. Pull details
	var itemIDs []int64
	for _, item := range listRes.Items {
		itemIDs = append(itemIDs, item.ItemID)
	}

	var products []marketplace.ShopeeProductDetail
	if len(itemIDs) > 0 {
		products, err = adapter.GetProductDetails(accessToken, extStoreID, itemIDs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse("API_ERROR", "Failed to pull product details: "+err.Error()))
			return
		}
	}

	// 3. Process into candidates
	var candidates []models.ShopeeMappingCandidate
	var mappedCount, unmappedCount int

	for _, p := range products {
		// Handle models (variants)
		if len(p.Models) > 0 {
			for _, m := range p.Models {
				extProdID := fmt.Sprintf("%d", p.ItemID)
				extVarID := fmt.Sprintf("%d", m.ModelID)

				candidate := h.buildCandidate(store, p, &m, extProdID, &extVarID)
				candidates = append(candidates, candidate)

				if candidate.MappingStatus == "mapped" {
					mappedCount++
				} else {
					unmappedCount++
				}
			}
		} else {
			// Single product without variants
			extProdID := fmt.Sprintf("%d", p.ItemID)
			candidate := h.buildCandidate(store, p, nil, extProdID, nil)
			candidates = append(candidates, candidate)

			if candidate.MappingStatus == "mapped" {
				mappedCount++
			} else {
				unmappedCount++
			}
		}
	}

	// Create SyncLog
	durationMs := int64(time.Since(startTime).Milliseconds())
	now := time.Now()
	finalStatus := "success"
	statusMsg := "Mapping candidates retrieved successfully."

	summary := gin.H{
		"records_processed": len(candidates),
		"mapped_count":      mappedCount,
		"unmapped_count":    unmappedCount,
	}
	summaryJSON, _ := json.Marshal(summary)
	summaryStr := string(summaryJSON)

	logRec := &models.SyncLog{
		StoreID:          &store.ID,
		Marketplace:      store.Marketplace,
		SyncType:         "products",
		SyncDirection:    "pull",
		Status:           finalStatus,
		Message:          &statusMsg,
		RecordsProcessed: len(candidates),
		StartedAt:        &startTime,
		FinishedAt:       &now,
		DurationMs:       &durationMs,
		RawSummary:       &summaryStr,
	}
	_ = h.syncRepo.CreateSyncLog(logRec)

	c.JSON(http.StatusOK, models.SuccessResponse(gin.H{
		"status":            finalStatus,
		"message":           statusMsg,
		"records_processed": len(candidates),
		"mapped_count":      mappedCount,
		"unmapped_count":    unmappedCount,
		"candidates":        candidates,
		"sync_log_id":       logRec.ID,
	}, statusMsg))
}

func (h *IntegrationHandler) buildCandidate(store *models.Store, p marketplace.ShopeeProductDetail, m *marketplace.ShopeeProductVariant, extProdID string, extVarID *string) models.ShopeeMappingCandidate {
	candidate := models.ShopeeMappingCandidate{
		ExternalProductID: extProdID,
		ExternalVariantID: extVarID,
		Marketplace:       "shopee",
		StoreID:           store.ID.String(),
		Title:             p.ItemName,
		MappingStatus:     "unmapped",
	}

	if m != nil {
		candidate.VariantName = m.ModelName
		candidate.SKU = m.ModelSKU
		if len(m.PriceInfo) > 0 {
			candidate.Price = m.PriceInfo[0].OriginalPrice
		}
		if len(m.StockInfo) > 0 {
			candidate.Stock = m.StockInfo[0].TotalAvailable
		}
	} else {
		candidate.SKU = p.ItemSKU
		if len(p.PriceInfo) > 0 {
			candidate.Price = p.PriceInfo[0].OriginalPrice
		}
		if len(p.StockInfo) > 0 {
			candidate.Stock = p.StockInfo[0].TotalAvailable
		}
	}

	if len(p.Images.ImageUrlList) > 0 {
		candidate.ImageURL = p.Images.ImageUrlList[0]
	}

	// Check existing mapping
	var vID string
	if extVarID != nil {
		vID = *extVarID
	}
	existing, _ := h.mappingRepo.FindByExternalID(store.ID.String(), extProdID, vID)
	if existing != nil {
		candidate.MappingStatus = "mapped"
		idStr := existing.ID.String()
		candidate.ExistingProductMappingID = &idStr
		prodIDStr := existing.ProductID.String()
		candidate.InternalProductID = &prodIDStr

		if existing.Product != nil {
			candidate.InternalProductName = &existing.Product.Name
		}
		if existing.ProductVariantID != nil {
			vIDStr := existing.ProductVariantID.String()
			candidate.InternalVariantID = &vIDStr
		}
	}

	return candidate
}

// CreateMapping creates a manual mapping from a Shopee candidate.
// POST /api/stores/:id/integration/products/mappings
func (h *IntegrationHandler) CreateMapping(c *gin.Context) {
	storeID := c.Param("id")

	var req struct {
		ExternalProductID string  `json:"external_product_id" binding:"required"`
		ExternalVariantID *string `json:"external_variant_id"`
		InternalProductID string  `json:"internal_product_id" binding:"required"`
		InternalVariantID *string `json:"internal_variant_id"`
		ExternalSKU       *string `json:"external_sku"`
		ExternalName      *string `json:"external_name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("INVALID_REQUEST", err.Error()))
		return
	}

	store, err := h.storeRepo.FindByID(storeID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse("NOT_FOUND", "Store not found"))
		return
	}

	if store.Marketplace != "shopee" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("VALIDATION_ERROR", "Manual mapping currently only supported for Shopee"))
		return
	}

	// Validate internal product
	product, err := h.productRepo.FindByID(req.InternalProductID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("INVALID_PRODUCT", "Internal product not found"))
		return
	}

	// Validate variant if provided
	var internalVariantID *uuid.UUID
	if req.InternalVariantID != nil && *req.InternalVariantID != "" {
		vUUID, err := uuid.Parse(*req.InternalVariantID)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse("INVALID_VARIANT", "Invalid internal variant ID"))
			return
		}

		found := false
		for _, v := range product.Variants {
			if v.ID == vUUID {
				found = true
				break
			}
		}
		if !found {
			c.JSON(http.StatusBadRequest, models.ErrorResponse("INVALID_VARIANT", "Variant not found for this product"))
			return
		}
		internalVariantID = &vUUID
	}

	// Check duplicate mapping
	extVarID := ""
	if req.ExternalVariantID != nil {
		extVarID = *req.ExternalVariantID
	}
	exists, err := h.mappingRepo.CheckDuplicateMapping(storeID, req.ExternalProductID, extVarID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to check existing mapping"))
		return
	}
	if exists {
		c.JSON(http.StatusConflict, models.ErrorResponse("DUPLICATE_MAPPING", "This external item is already mapped for this store"))
		return
	}

	// Create mapping
	mapping := &models.MarketplaceProductMapping{
		ID:                uuid.New(),
		ProductID:         product.ID,
		ProductVariantID:  internalVariantID,
		StoreID:           store.ID,
		Marketplace:       "shopee",
		ExternalProductID: req.ExternalProductID,
		ExternalVariantID: req.ExternalVariantID,
		ExternalSKU:       req.ExternalSKU,
		ListingName:       req.ExternalName,
		ListingStatus:     "active", // Assume active if mapping manually
	}

	if err := h.mappingRepo.Create(mapping); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to create mapping"))
		return
	}

	c.JSON(http.StatusCreated, models.SuccessResponse(mapping, "Mapping created successfully"))
}

// PushStockRequest defines the payload for manual stock push.
type PushStockRequest struct {
	ProductMappingID string `json:"product_mapping_id" binding:"required"`
	Quantity         *int   `json:"quantity"` // If provided, use this quantity. If nil, use available quantity.
	DryRun           bool   `json:"dry_run"`
}

// PushStockResponse defines the response for manual stock push.
type PushStockResponse struct {
	Status            string   `json:"status"`
	Message           string   `json:"message"`
	Marketplace       string   `json:"marketplace"`
	StoreID           string   `json:"store_id"`
	ProductMappingID  string   `json:"product_mapping_id"`
	ExternalProductID string   `json:"external_product_id"`
	ExternalVariantID string   `json:"external_variant_id"`
	PushedQuantity    int      `json:"pushed_quantity"`
	DryRun            bool     `json:"dry_run"`
	SyncLogID         string   `json:"sync_log_id"`
	Errors            []string `json:"errors,omitempty"`
}

// PushStock manually pushes internal stock to the marketplace.
// POST /api/stores/:id/integration/stock/push
func (h *IntegrationHandler) PushStock(c *gin.Context) {
	storeID := c.Param("id")
	var req PushStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("INVALID_REQUEST", err.Error()))
		return
	}

	store, err := h.storeRepo.FindByID(storeID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse("NOT_FOUND", "Store not found"))
		return
	}

	if store.Marketplace != "shopee" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("UNSUPPORTED_MARKETPLACE", "Manual stock push currently only supported for Shopee"))
		return
	}

	// Get mapping
	mapping, err := h.mappingRepo.GetByID(req.ProductMappingID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse("MAPPING_NOT_FOUND", "Product mapping not found"))
		return
	}

	if mapping.StoreID.String() != storeID {
		c.JSON(http.StatusForbidden, models.ErrorResponse("FORBIDDEN", "Mapping does not belong to this store"))
		return
	}

	// Get inventory item
	var inventoryItem *models.InventoryItem
	if mapping.ProductVariantID != nil {
		inventoryItem, err = h.inventoryRepo.GetByProductAndVariant(mapping.ProductID.String(), mapping.ProductVariantID.String(), "Main Warehouse")
	} else {
		inventoryItem, err = h.inventoryRepo.GetByProductAndVariant(mapping.ProductID.String(), "", "Main Warehouse")
	}

	if err != nil || inventoryItem == nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("INVENTORY_NOT_FOUND", "Internal inventory item not found"))
		return
	}

	quantity := inventoryItem.AvailableQuantity
	if req.Quantity != nil {
		quantity = *req.Quantity
	}

	if quantity < 0 {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("INVALID_QUANTITY", "Cannot push negative stock"))
		return
	}

	// Prepare result
	extProdID, _ := strconv.ParseInt(mapping.ExternalProductID, 10, 64)
	extVarID := int64(0)
	if mapping.ExternalVariantID != nil && *mapping.ExternalVariantID != "" {
		extVarID, _ = strconv.ParseInt(*mapping.ExternalVariantID, 10, 64)
	}

	startTime := time.Now()
	syncLog := &models.SyncLog{
		ID:            uuid.New(),
		StoreID:       &store.ID,
		Marketplace:   "shopee",
		SyncType:      "inventory",
		SyncDirection: "push",
		Status:        "started",
		StartedAt:     &startTime,
	}

	response := PushStockResponse{
		Marketplace:       "shopee",
		StoreID:           store.ID.String(),
		ProductMappingID:  mapping.ID.String(),
		ExternalProductID: mapping.ExternalProductID,
		ExternalVariantID: "",
		PushedQuantity:    quantity,
		DryRun:            req.DryRun,
	}
	if mapping.ExternalVariantID != nil {
		response.ExternalVariantID = *mapping.ExternalVariantID
	}

	if req.DryRun {
		now := time.Now()
		syncLog.Status = "success"
		syncLog.FinishedAt = &now
		syncLog.RecordsProcessed = 1
		msg := fmt.Sprintf("Dry run: Would push %d to Shopee item %s", quantity, mapping.ExternalProductID)
		syncLog.Message = &msg
		h.syncRepo.CreateSyncLog(syncLog)

		response.Status = "success"
		response.Message = "Dry run successful"
		response.SyncLogID = syncLog.ID.String()
		c.JSON(http.StatusOK, response)
		return
	}

	// Real push
	adapter, err := marketplace.GetAdapter("shopee")
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to get adapter"))
		return
	}

	creds, err := h.integrationRepo.FindCredentialByStoreID(storeID)
	if err != nil || creds == nil {
		now := time.Now()
		syncLog.Status = "not_configured"
		syncLog.FinishedAt = &now
		syncLog.RecordsFailed = 1
		msg := "Marketplace credentials not configured"
		syncLog.Message = &msg
		h.syncRepo.CreateSyncLog(syncLog)

		response.Status = "failed"
		response.Message = "Marketplace credentials not configured"
		response.SyncLogID = syncLog.ID.String()
		c.JSON(http.StatusPreconditionFailed, response)
		return
	}

	// Decrypt tokens
	if creds.EncryptedAccessToken == nil || *creds.EncryptedAccessToken == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("TOKEN_MISSING", "Access token is missing."))
		return
	}

	accessToken, err := security.DecryptToken(*creds.EncryptedAccessToken)
	if err != nil {
		now := time.Now()
		syncLog.Status = "failed"
		syncLog.FinishedAt = &now
		syncLog.RecordsFailed = 1
		msg := "Failed to decrypt tokens"
		syncLog.Message = &msg
		h.syncRepo.CreateSyncLog(syncLog)

		response.Status = "failed"
		response.Message = "Failed to decrypt credentials"
		response.SyncLogID = syncLog.ID.String()
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	if creds.AccessTokenExpiresAt != nil && time.Now().After(*creds.AccessTokenExpiresAt) {
		now := time.Now()
		syncLog.Status = "expired"
		syncLog.FinishedAt = &now
		syncLog.RecordsFailed = 1
		msg := "Access token expired"
		syncLog.Message = &msg
		h.syncRepo.CreateSyncLog(syncLog)

		response.Status = "failed"
		response.Message = "Access token expired. Please reconnect or refresh."
		response.SyncLogID = syncLog.ID.String()
		c.JSON(http.StatusUnauthorized, response)
		return
	}

	extStoreID := ""
	if store.ExternalStoreID != nil {
		extStoreID = *store.ExternalStoreID
	}

	err = adapter.UpdateStock(accessToken, extStoreID, extProdID, extVarID, quantity)
	now := time.Now()
	syncLog.FinishedAt = &now
	if err != nil {
		syncLog.Status = "failed"
		syncLog.RecordsFailed = 1
		errMsg := err.Error()
		syncLog.ErrorMessage = &errMsg
		msg := fmt.Sprintf("Failed to push stock: %v", err)
		syncLog.Message = &msg
		h.syncRepo.CreateSyncLog(syncLog)

		response.Status = "failed"
		response.Message = "Failed to push stock to Shopee"
		response.SyncLogID = syncLog.ID.String()
		response.Errors = []string{err.Error()}
		c.JSON(http.StatusBadGateway, response)
		return
	}

	syncLog.Status = "success"
	syncLog.RecordsProcessed = 1
	msg := fmt.Sprintf("Successfully pushed %d to Shopee item %s", quantity, mapping.ExternalProductID)
	syncLog.Message = &msg
	h.syncRepo.CreateSyncLog(syncLog)

	response.Status = "success"
	response.Message = "Stock pushed successfully"
	response.SyncLogID = syncLog.ID.String()
	c.JSON(http.StatusOK, response)
}
