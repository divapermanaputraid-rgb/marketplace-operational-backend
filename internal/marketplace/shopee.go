package marketplace

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// ShopeeAdapter handles the Shopee Open Platform v2 integration.
type ShopeeAdapter struct{}

func (a *ShopeeAdapter) MarketplaceName() string {
	return "shopee"
}

// BuildAuthorizationURL returns the Shopee OAuth authorization URL.
func (a *ShopeeAdapter) BuildAuthorizationURL(storeID string, state string) (string, error) {
	cfg, err := LoadShopeeConfig()
	if err != nil {
		return "", err
	}

	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	path := "/api/v2/shop/auth_partner"
	baseString := BuildAuthBaseString(cfg.PartnerID, path, timestamp)
	sign := GenerateShopeeSignature(cfg.PartnerKey, baseString)

	url := fmt.Sprintf("%s%s?partner_id=%s&timestamp=%s&sign=%s&redirect=%s",
		cfg.AuthBaseURL, path, cfg.PartnerID, timestamp, sign, cfg.RedirectURI)
	// NOTE: Shopee docs don't explicitly document a `state` parameter in the auth URL in some versions,
	// but typically it can be passed or appended. We'll append it safely.
	// TODO: Verify if Shopee strictly supports `state` query param or if we must embed it in redirect_uri.
	url = fmt.Sprintf("%s&state=%s", url, state)

	return url, nil
}

type shopeeTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpireIn     int    `json:"expire_in"`
	Error        string `json:"error"`
	Message      string `json:"message"`
}

// ExchangeAuthCode exchanges the authorization code for tokens.
func (a *ShopeeAdapter) ExchangeAuthCode(code string, shopID string) (*TokenResult, error) {
	cfg, err := LoadShopeeConfig()
	if err != nil {
		return nil, err
	}

	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	path := "/api/v2/auth/token/get"
	baseString := BuildAuthBaseString(cfg.PartnerID, path, timestamp)
	sign := GenerateShopeeSignature(cfg.PartnerKey, baseString)

	url := fmt.Sprintf("%s%s?partner_id=%s&timestamp=%s&sign=%s", cfg.BaseURL, path, cfg.PartnerID, timestamp, sign)

	// TODO: Verify if partner_id and shop_id should be numbers instead of strings in the JSON payload.
	partnerIDInt, _ := strconv.Atoi(cfg.PartnerID)
	shopIDInt, _ := strconv.Atoi(shopID)

	payload := map[string]interface{}{
		"code":       code,
		"shop_id":    shopIDInt,
		"partner_id": partnerIDInt,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var tokenRes shopeeTokenResponse
	if err := json.NewDecoder(res.Body).Decode(&tokenRes); err != nil {
		return nil, err
	}

	if tokenRes.Error != "" {
		return nil, fmt.Errorf("shopee api error: %s - %s", tokenRes.Error, tokenRes.Message)
	}

	return &TokenResult{
		AccessToken:           tokenRes.AccessToken,
		RefreshToken:          tokenRes.RefreshToken,
		AccessTokenExpiresAt:  time.Now().Add(time.Duration(tokenRes.ExpireIn) * time.Second),
		RefreshTokenExpiresAt: time.Now().Add(30 * 24 * time.Hour), // Shopee refresh tokens are typically valid for 30 days
	}, nil
}

// RefreshToken refreshes the access token using the refresh token.
func (a *ShopeeAdapter) RefreshToken(refreshToken string, shopID string) (*TokenResult, error) {
	cfg, err := LoadShopeeConfig()
	if err != nil {
		return nil, err
	}

	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	path := "/api/v2/auth/access_token/get"
	baseString := BuildAuthBaseString(cfg.PartnerID, path, timestamp)
	sign := GenerateShopeeSignature(cfg.PartnerKey, baseString)

	url := fmt.Sprintf("%s%s?partner_id=%s&timestamp=%s&sign=%s", cfg.BaseURL, path, cfg.PartnerID, timestamp, sign)

	partnerIDInt, _ := strconv.Atoi(cfg.PartnerID)
	shopIDInt, _ := strconv.Atoi(shopID)

	payload := map[string]interface{}{
		"refresh_token": refreshToken,
		"shop_id":       shopIDInt,
		"partner_id":    partnerIDInt,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var tokenRes shopeeTokenResponse
	if err := json.NewDecoder(res.Body).Decode(&tokenRes); err != nil {
		return nil, err
	}

	if tokenRes.Error != "" {
		return nil, fmt.Errorf("shopee api error: %s - %s", tokenRes.Error, tokenRes.Message)
	}

	return &TokenResult{
		AccessToken:           tokenRes.AccessToken,
		RefreshToken:          tokenRes.RefreshToken,
		AccessTokenExpiresAt:  time.Now().Add(time.Duration(tokenRes.ExpireIn) * time.Second),
		RefreshTokenExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}, nil
}

// ValidateCredentials checks if Shopee API credentials are configured.
func (a *ShopeeAdapter) ValidateCredentials() error {
	_, err := LoadShopeeConfig()
	return err
}

// PullOrders pulls orders from Shopee.
func (a *ShopeeAdapter) PullOrders() error {
	return ErrNotImplemented
}

// PullProducts pulls products from Shopee.
func (a *ShopeeAdapter) PullProducts() error {
	return ErrNotImplemented
}

// PushStock pushes stock levels to Shopee.
func (a *ShopeeAdapter) PushStock() error {
	return ErrNotImplemented
}
