package marketplace

// TokopediaShopAdapter is the placeholder adapter for Tokopedia Shop integration.
// Tokopedia has merged its API path under TikTok Shop Partner Center.
// Real API calls will be implemented in a future sprint.
type TokopediaShopAdapter struct{}

func (a *TokopediaShopAdapter) MarketplaceName() string {
	return "tokopedia_shop"
}

// BuildAuthorizationURL returns the Tokopedia Shop OAuth authorization URL.
// Sprint 13: Returns ErrMissingCredentials.
func (a *TokopediaShopAdapter) BuildAuthorizationURL(storeID string, state string) (string, error) {
	return "", ErrMissingCredentials
}

// ExchangeAuthCode exchanges the authorization code for tokens.
// Sprint 13: Returns ErrNotImplemented.
func (a *TokopediaShopAdapter) ExchangeAuthCode(code string) (*TokenResult, error) {
	return nil, ErrNotImplemented
}

// RefreshToken refreshes the access token.
// Sprint 13: Returns ErrNotImplemented.
func (a *TokopediaShopAdapter) RefreshToken(refreshToken string) (*TokenResult, error) {
	return nil, ErrNotImplemented
}

// ValidateCredentials checks if Tokopedia Shop API credentials are configured.
// Sprint 13: Returns ErrMissingCredentials.
func (a *TokopediaShopAdapter) ValidateCredentials() error {
	return ErrMissingCredentials
}

// PullOrders pulls orders from Tokopedia Shop.
// Sprint 13: Returns ErrNotImplemented.
func (a *TokopediaShopAdapter) PullOrders() error {
	return ErrNotImplemented
}

// PullProducts pulls products from Tokopedia Shop.
// Sprint 13: Returns ErrNotImplemented.
func (a *TokopediaShopAdapter) PullProducts() error {
	return ErrNotImplemented
}

// PushStock pushes stock levels to Tokopedia Shop.
// Sprint 13: Returns ErrNotImplemented.
func (a *TokopediaShopAdapter) PushStock() error {
	return ErrNotImplemented
}
