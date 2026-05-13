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
func (a *TikTokShopAdapter) ExchangeAuthCode(code string, shopID string) (*TokenResult, error) {
	return nil, ErrNotImplemented
}

// RefreshToken uses a refresh token to obtain a new access token.
// Sprint 13: Returns ErrNotImplemented.
func (a *TikTokShopAdapter) RefreshToken(refreshToken string, shopID string) (*TokenResult, error) {
	return nil, ErrNotImplemented
}

// ValidateCredentials checks if TikTok Shop API credentials are configured.
// Sprint 13: Returns ErrMissingCredentials.
func (a *TikTokShopAdapter) ValidateCredentials() error {
	return ErrMissingCredentials
}

func (a *TikTokShopAdapter) GetShopInfo(accessToken string, shopID string) (*ShopeeShopInfo, error) {
	return nil, ErrNotImplemented
}

// PullOrders pulls orders from TikTok Shop.
// Sprint 13: Returns ErrNotImplemented.
func (a *TikTokShopAdapter) PullOrders(accessToken, shopID string, timeFrom, timeTo int64, cursor string) (*ShopeeOrderListResponse, error) {
	return nil, ErrNotImplemented
}

func (a *TikTokShopAdapter) GetOrderDetails(accessToken, shopID string, orderSNs []string) ([]ShopeeOrderDetail, error) {
	return nil, ErrNotImplemented
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
