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

// ShopeeShopInfo holds safe metadata about the connected shop.
type ShopeeShopInfo struct {
	ShopID   string
	ShopName string
	Region   string
	Status   string
}

// Order data types exposed to integration handler
type ShopeeOrderSummary struct {
	OrderSN     string
	OrderStatus string
}

type ShopeeOrderListResponse struct {
	More       bool
	NextCursor string
	Orders     []ShopeeOrderSummary
}

type ShopeeOrderDetail struct {
	OrderSN     string
	Region      string
	Currency    string
	TotalAmount float64
	OrderStatus string
	CreateTime  int64
	PayTime     int64
	ItemList    []ShopeeOrderItem
	RawPayload  string
}

type ShopeeOrderItem struct {
	ItemID     int64   `json:"item_id"`
	ModelID    int64   `json:"model_id"`
	ItemName   string  `json:"item_name"`
	ModelName  string  `json:"model_name"`
	ItemSKU    string  `json:"item_sku"`
	ModelSKU   string  `json:"model_sku"`
	ModelQty   int     `json:"model_qty"`
	ModelPrice float64 `json:"model_price"`
}

// Product data types for Shopee
type ShopeeProductSummary struct {
	ItemID     int64  `json:"item_id"`
	ItemStatus string `json:"item_status"`
	UpdateTime int64  `json:"update_time"`
}

type ShopeeProductListResponse struct {
	HasNextPage bool                   `json:"has_next_page"`
	NextOffset  int                    `json:"next_offset"`
	Items       []ShopeeProductSummary `json:"item_list"`
}

type ShopeeProductDetail struct {
	ItemID      int64                  `json:"item_id"`
	ItemName    string                 `json:"item_name"`
	Description string                 `json:"description"`
	ItemSKU     string                 `json:"item_sku"`
	CreateTime  int64                  `json:"create_time"`
	UpdateTime  int64                  `json:"update_time"`
	ItemStatus  string                 `json:"item_status"`
	PriceInfo   []ShopeePriceInfo      `json:"price_info"`
	StockInfo   []ShopeeStockInfo      `json:"stock_info"`
	Models      []ShopeeProductVariant `json:"model"`
	Images      struct {
		ImageIDList  []string `json:"image_id_list"`
		ImageUrlList []string `json:"image_url_list"`
	} `json:"image"`
}

type ShopeePriceInfo struct {
	Currency      string  `json:"currency"`
	OriginalPrice float64 `json:"original_price"`
}

type ShopeeStockInfo struct {
	StockType      int `json:"stock_type"`
	NormalStock    int `json:"normal_stock"`
	ReservedStock  int `json:"reserved_stock"`
	TotalAvailable int `json:"total_available"`
}

type ShopeeProductVariant struct {
	ModelID   int64             `json:"model_id"`
	ModelSKU  string            `json:"model_sku"`
	ModelName string            `json:"model_name"`
	PriceInfo []ShopeePriceInfo `json:"price_info"`
	StockInfo []ShopeeStockInfo `json:"stock_info"`
}

// MarketplaceAdapter defines the interface that all marketplace integrations must implement.
// For Sprint 13, all methods return ErrNotImplemented or ErrMissingCredentials.
type MarketplaceAdapter interface {
	// MarketplaceName returns the canonical name of the marketplace (e.g., "shopee", "tiktok_shop").
	MarketplaceName() string

	// BuildAuthorizationURL generates the OAuth authorization URL for seller consent.
	BuildAuthorizationURL(storeID string, state string) (string, error)

	// ExchangeAuthCode exchanges an authorization code for access and refresh tokens.
	ExchangeAuthCode(code string, shopID string) (*TokenResult, error)

	// RefreshToken uses a refresh token to obtain a new access token.
	RefreshToken(refreshToken string, shopID string) (*TokenResult, error)

	// ValidateCredentials checks if the configured credentials are valid.
	ValidateCredentials() error

	// GetShopInfo pulls shop information to validate connection.
	GetShopInfo(accessToken string, shopID string) (*ShopeeShopInfo, error)

	// PullOrders pulls orders from the marketplace.
	PullOrders(accessToken, shopID string, timeFrom, timeTo int64, cursor string) (*ShopeeOrderListResponse, error)

	// GetOrderDetails pulls detailed information for a list of order IDs.
	GetOrderDetails(accessToken, shopID string, orderSNs []string) ([]ShopeeOrderDetail, error)

	// PullProducts pulls products/listings from the marketplace.
	PullProducts(accessToken, shopID string, offset int, pageSize int, itemStatus string) (*ShopeeProductListResponse, error)

	// GetProductDetails pulls detailed information for a list of product IDs.
	GetProductDetails(accessToken, shopID string, itemIDList []int64) ([]ShopeeProductDetail, error)

	// UpdateStock updates inventory stock levels for a specific product/model on the marketplace.
	UpdateStock(accessToken, shopID string, itemID int64, modelID int64, stock int) error
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
