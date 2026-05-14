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
func (a *TikTokShopAdapter) PullOrders(accessToken, shopID string, timeFrom, timeTo int64, pageSize int, cursor string) (*ShopeeOrderListResponse, error) {
	return nil, ErrNotImplemented
}

func (a *TikTokShopAdapter) GetOrderDetails(accessToken, shopID string, orderSNs []string) ([]ShopeeOrderDetail, error) {
	return nil, ErrNotImplemented
}

// PullProducts pulls products from TikTok Shop.
func (a *TikTokShopAdapter) PullProducts(accessToken, shopID string, offset int, pageSize int, itemStatus string) (*ShopeeProductListResponse, error) {
	return nil, ErrNotImplemented
}

// GetProductDetails pulls detailed information for a list of product IDs.
func (a *TikTokShopAdapter) GetProductDetails(accessToken, shopID string, itemIDList []int64) ([]ShopeeProductDetail, error) {
	return nil, ErrNotImplemented
}

// PushStock pushes stock levels to TikTok Shop.
func (a *TikTokShopAdapter) UpdateStock(accessToken, shopID string, itemID int64, modelID int64, stock int) error {
	return ErrNotImplemented
}
