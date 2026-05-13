package marketplace

// ShopeeAdapter is the placeholder adapter for Shopee Open Platform integration.
// Real API calls will be implemented in a future sprint after credentials are obtained.
type ShopeeAdapter struct{}

func (a *ShopeeAdapter) MarketplaceName() string {
	return "shopee"
}

// BuildAuthorizationURL returns the Shopee OAuth authorization URL.
// Sprint 13: Returns ErrMissingCredentials because partner credentials are not configured yet.
func (a *ShopeeAdapter) BuildAuthorizationURL(storeID string, state string) (string, error) {
	// Future implementation will:
	// 1. Read SHOPEE_PARTNER_ID and SHOPEE_PARTNER_KEY from config
	// 2. Generate timestamp and HMAC-SHA256 signature
	// 3. Build authorization URL with partner_id, redirect, sign, timestamp, state
	return "", ErrMissingCredentials
}

// ExchangeAuthCode exchanges the authorization code for tokens.
// Sprint 13: Returns ErrNotImplemented.
func (a *ShopeeAdapter) ExchangeAuthCode(code string) (*TokenResult, error) {
	return nil, ErrNotImplemented
}

// RefreshToken refreshes the access token using the refresh token.
// Sprint 13: Returns ErrNotImplemented.
func (a *ShopeeAdapter) RefreshToken(refreshToken string) (*TokenResult, error) {
	return nil, ErrNotImplemented
}

// ValidateCredentials checks if Shopee API credentials are configured.
// Sprint 13: Returns ErrMissingCredentials.
func (a *ShopeeAdapter) ValidateCredentials() error {
	return ErrMissingCredentials
}

// PullOrders pulls orders from Shopee.
// Sprint 13: Returns ErrNotImplemented.
func (a *ShopeeAdapter) PullOrders() error {
	return ErrNotImplemented
}

// PullProducts pulls products from Shopee.
// Sprint 13: Returns ErrNotImplemented.
func (a *ShopeeAdapter) PullProducts() error {
	return ErrNotImplemented
}

// PushStock pushes stock levels to Shopee.
// Sprint 13: Returns ErrNotImplemented.
func (a *ShopeeAdapter) PushStock() error {
	return ErrNotImplemented
}
