package marketplace

import (
	"os"
)

type ShopeeConfig struct {
	PartnerID   string
	PartnerKey  string
	RedirectURI string
	BaseURL     string
	AuthBaseURL string
}

func LoadShopeeConfig() (*ShopeeConfig, error) {
	partnerID := os.Getenv("SHOPEE_PARTNER_ID")
	partnerKey := os.Getenv("SHOPEE_PARTNER_KEY")
	redirectURI := os.Getenv("SHOPEE_REDIRECT_URI")
	baseURL := os.Getenv("SHOPEE_BASE_URL")
	authBaseURL := os.Getenv("SHOPEE_AUTH_BASE_URL")

	if partnerID == "" || partnerKey == "" || redirectURI == "" || baseURL == "" || authBaseURL == "" {
		return nil, ErrMissingCredentials
	}

	return &ShopeeConfig{
		PartnerID:   partnerID,
		PartnerKey:  partnerKey,
		RedirectURI: redirectURI,
		BaseURL:     baseURL,
		AuthBaseURL: authBaseURL,
	}, nil
}
