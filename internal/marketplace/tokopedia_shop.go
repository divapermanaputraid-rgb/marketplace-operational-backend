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
func (a *TokopediaShopAdapter) ExchangeAuthCode(code string, shopID string) (*TokenResult, error) {
	return nil, ErrNotImplemented
}

// RefreshToken uses a refresh token to obtain a new access token.
// Sprint 13: Returns ErrNotImplemented.
func (a *TokopediaShopAdapter) RefreshToken(refreshToken string, shopID string) (*TokenResult, error) {
	return nil, ErrNotImplemented
}

// ValidateCredentials checks if Tokopedia Shop API credentials are configured.
// Sprint 13: Returns ErrMissingCredentials.
func (a *TokopediaShopAdapter) ValidateCredentials() error {
	return ErrMissingCredentials
}

func (a *TokopediaShopAdapter) GetShopInfo(accessToken string, shopID string) (*ShopeeShopInfo, error) {
	return nil, ErrNotImplemented
}

// PullOrders pulls orders from Tokopedia Shop.
// Sprint 13: Returns ErrNotImplemented.
func (a *TokopediaShopAdapter) PullOrders(accessToken, shopID string, timeFrom, timeTo int64, cursor string) (*ShopeeOrderListResponse, error) {
	return nil, ErrNotImplemented
}

func (a *TokopediaShopAdapter) GetOrderDetails(accessToken, shopID string, orderSNs []string) ([]ShopeeOrderDetail, error) {
	return nil, ErrNotImplemented
}

// PullProducts pulls products from Tokopedia Shop.
func (a *TokopediaShopAdapter) PullProducts(accessToken, shopID string, offset int, pageSize int, itemStatus string) (*ShopeeProductListResponse, error) {
	return nil, ErrNotImplemented
}

// GetProductDetails pulls detailed information for a list of product IDs.
func (a *TokopediaShopAdapter) GetProductDetails(accessToken, shopID string, itemIDList []int64) ([]ShopeeProductDetail, error) {
	return nil, ErrNotImplemented
}

// PushStock pushes stock levels to Tokopedia Shop.
func (a *TokopediaShopAdapter) UpdateStock(accessToken, shopID string, itemID int64, modelID int64, stock int) error {
	return ErrNotImplemented
}
