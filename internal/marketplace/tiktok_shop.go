package marketplace

// TikTokShopAdapter is the placeholder adapter for TikTok Shop Partner Center integration.
// Real API calls will be implemented in a future sprint after credentials are obtained.
type TikTokShopAdapter struct{}

func (a *TikTokShopAdapter) MarketplaceName() string {
	return "tiktok_shop"
}

// BuildAuthorizationURL returns the TikTok Shop OAuth authorization URL.
// Sprint 13: Returns ErrMissingCredentials.
func (a *TikTokShopAdapter) BuildAuthorizationURL(storeID string, state string) (string, error) {
	return "", ErrMissingCredentials
}

// ExchangeAuthCode exchanges the authorization code for tokens.
// Sprint 13: Returns ErrNotImplemented.
func (a *TikTokShopAdapter) ExchangeAuthCode(code string) (*TokenResult, error) {
	return nil, ErrNotImplemented
}

// RefreshToken refreshes the access token.
// Sprint 13: Returns ErrNotImplemented.
func (a *TikTokShopAdapter) RefreshToken(refreshToken string) (*TokenResult, error) {
	return nil, ErrNotImplemented
}

// ValidateCredentials checks if TikTok Shop API credentials are configured.
// Sprint 13: Returns ErrMissingCredentials.
func (a *TikTokShopAdapter) ValidateCredentials() error {
	return ErrMissingCredentials
}

// PullOrders pulls orders from TikTok Shop.
// Sprint 13: Returns ErrNotImplemented.
func (a *TikTokShopAdapter) PullOrders() error {
	return ErrNotImplemented
}

// PullProducts pulls products from TikTok Shop.
// Sprint 13: Returns ErrNotImplemented.
func (a *TikTokShopAdapter) PullProducts() error {
	return ErrNotImplemented
}

// PushStock pushes stock levels to TikTok Shop.
// Sprint 13: Returns ErrNotImplemented.
func (a *TikTokShopAdapter) PushStock() error {
	return ErrNotImplemented
}
