package marketplace

import (
	"errors"
	"time"
)

var (
	// ErrNotImplemented is returned when a marketplace adapter method is not yet implemented.
	ErrNotImplemented = errors.New("marketplace integration is not implemented yet")
	// ErrMissingCredentials is returned when required API credentials are not configured.
	ErrMissingCredentials = errors.New("marketplace API credentials are not configured")
	// ErrUnsupportedMarketplace is returned for unknown marketplace names.
	ErrUnsupportedMarketplace = errors.New("unsupported marketplace")
)

// TokenResult holds the result of an OAuth token exchange or refresh.
type TokenResult struct {
	AccessToken           string
	RefreshToken          string
	AccessTokenExpiresAt  time.Time
	RefreshTokenExpiresAt time.Time
	Scopes                string
}

// MarketplaceAdapter defines the interface that all marketplace integrations must implement.
// For Sprint 13, all methods return ErrNotImplemented or ErrMissingCredentials.
type MarketplaceAdapter interface {
	// MarketplaceName returns the canonical name of the marketplace (e.g., "shopee", "tiktok_shop").
	MarketplaceName() string

	// BuildAuthorizationURL generates the OAuth authorization URL for seller consent.
	BuildAuthorizationURL(storeID string, state string) (string, error)

	// ExchangeAuthCode exchanges an authorization code for access and refresh tokens.
	ExchangeAuthCode(code string) (*TokenResult, error)

	// RefreshToken uses a refresh token to obtain a new access token.
	RefreshToken(refreshToken string) (*TokenResult, error)

	// ValidateCredentials checks if the configured credentials are valid.
	ValidateCredentials() error

	// PullOrders pulls orders from the marketplace. Not implemented in Sprint 13.
	PullOrders() error

	// PullProducts pulls products/listings from the marketplace. Not implemented in Sprint 13.
	PullProducts() error

	// PushStock pushes inventory stock levels to the marketplace. Not implemented in Sprint 13.
	PushStock() error
}

// GetAdapter returns the appropriate marketplace adapter for a given marketplace name.
func GetAdapter(marketplace string) (MarketplaceAdapter, error) {
	switch marketplace {
	case "shopee":
		return &ShopeeAdapter{}, nil
	case "tiktok_shop":
		return &TikTokShopAdapter{}, nil
	case "tokopedia_shop":
		return &TokopediaShopAdapter{}, nil
	default:
		return nil, ErrUnsupportedMarketplace
	}
}

// SupportedMarketplaces returns the list of marketplace identifiers that the system supports.
func SupportedMarketplaces() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"id":                   "shopee",
			"name":                 "Shopee",
			"integration_status":   "not_implemented",
			"oauth_supported":      true,
			"description":          "Shopee Open Platform integration via OAuth 2.0 with HMAC-SHA256 signing.",
			"implementation_notes": "Adapter interface ready. Real API calls pending Sprint 14+.",
		},
		{
			"id":                   "tiktok_shop",
			"name":                 "TikTok Shop",
			"integration_status":   "not_implemented",
			"oauth_supported":      true,
			"description":          "TikTok Shop Partner Center integration via OAuth 2.0.",
			"implementation_notes": "Adapter interface ready. Requires approved developer account.",
		},
		{
			"id":                   "tokopedia_shop",
			"name":                 "Tokopedia Shop",
			"integration_status":   "not_implemented",
			"oauth_supported":      true,
			"description":          "Tokopedia Shop integration via TikTok Shop Partner Center APIs.",
			"implementation_notes": "Adapter interface ready. Uses unified TikTok/Tokopedia API path.",
		},
	}
}
