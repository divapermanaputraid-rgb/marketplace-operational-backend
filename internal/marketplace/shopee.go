package marketplace

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

type shopeeShopInfoResponse struct {
	Error    string `json:"error"`
	Message  string `json:"message"`
	Response struct {
		ShopName string `json:"shop_name"`
		Region   string `json:"region"`
		Status   string `json:"status"`
	} `json:"response"`
}

// GetShopInfo pulls shop information to validate connection.
func (a *ShopeeAdapter) GetShopInfo(accessToken string, shopID string) (*ShopeeShopInfo, error) {
	cfg, err := LoadShopeeConfig()
	if err != nil {
		return nil, err
	}

	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	path := "/api/v2/shop/get_shop_info"
	baseString := BuildAPIBaseString(cfg.PartnerID, path, timestamp, accessToken, shopID)
	sign := GenerateShopeeSignature(cfg.PartnerKey, baseString)

	url := fmt.Sprintf("%s%s?partner_id=%s&timestamp=%s&access_token=%s&shop_id=%s&sign=%s",
		cfg.BaseURL, path, cfg.PartnerID, timestamp, accessToken, shopID, sign)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var infoRes shopeeShopInfoResponse
	if err := json.NewDecoder(res.Body).Decode(&infoRes); err != nil {
		return nil, err
	}

	if infoRes.Error != "" {
		return nil, fmt.Errorf("shopee api error: %s - %s", infoRes.Error, infoRes.Message)
	}

	return &ShopeeShopInfo{
		ShopID:   shopID,
		ShopName: infoRes.Response.ShopName,
		Region:   infoRes.Response.Region,
		Status:   infoRes.Response.Status,
	}, nil
}

// PullOrders pulls orders from Shopee.
// Note: This implements a simplified manual pull for Sprint 17.
// It uses GetOrderList then GetOrderDetail to retrieve full data.
func (a *ShopeeAdapter) PullOrders(accessToken, shopID string, timeFrom, timeTo int64, cursor string) (*ShopeeOrderListResponse, error) {
	cfg, err := LoadShopeeConfig()
	if err != nil {
		return nil, err
	}

	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	path := "/api/v2/order/get_order_list"
	baseString := BuildAPIBaseString(cfg.PartnerID, path, timestamp, accessToken, shopID)
	sign := GenerateShopeeSignature(cfg.PartnerKey, baseString)

	url := fmt.Sprintf("%s%s?partner_id=%s&timestamp=%s&access_token=%s&shop_id=%s&sign=%s",
		cfg.BaseURL, path, cfg.PartnerID, timestamp, accessToken, shopID, sign)

	// Add query params for time range. Shopee typically uses time_range_field="create_time" and time_from/time_to.
	url += fmt.Sprintf("&time_range_field=create_time&time_from=%d&time_to=%d&page_size=50", timeFrom, timeTo)
	if cursor != "" {
		url += fmt.Sprintf("&cursor=%s", cursor)
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	type shopeeOrderListResponse struct {
		Error    string `json:"error"`
		Message  string `json:"message"`
		Response struct {
			More       bool                 `json:"more"`
			NextCursor string               `json:"next_cursor"`
			OrderList  []ShopeeOrderSummary `json:"order_list"`
		} `json:"response"`
	}

	var listRes shopeeOrderListResponse
	if err := json.NewDecoder(res.Body).Decode(&listRes); err != nil {
		return nil, err
	}

	if listRes.Error != "" {
		return nil, fmt.Errorf("shopee api error: %s - %s", listRes.Error, listRes.Message)
	}

	var mappedOrders []ShopeeOrderSummary
	for _, o := range listRes.Response.OrderList {
		mappedOrders = append(mappedOrders, ShopeeOrderSummary{
			OrderSN:     o.OrderSN,
			OrderStatus: o.OrderStatus,
		})
	}

	return &ShopeeOrderListResponse{
		More:       listRes.Response.More,
		NextCursor: listRes.Response.NextCursor,
		Orders:     mappedOrders,
	}, nil
}

type shopeeApiOrderDetail struct {
	OrderSN     string            `json:"order_sn"`
	Region      string            `json:"region"`
	Currency    string            `json:"currency"`
	TotalAmount float64           `json:"total_amount"`
	OrderStatus string            `json:"order_status"`
	CreateTime  int64             `json:"create_time"`
	PayTime     int64             `json:"pay_time"`
	ItemList    []ShopeeOrderItem `json:"item_list"`
}

type shopeeOrderDetailResponse struct {
	Error    string `json:"error"`
	Message  string `json:"message"`
	Response struct {
		OrderList []shopeeApiOrderDetail `json:"order_list"`
	} `json:"response"`
}

func (a *ShopeeAdapter) GetOrderDetails(accessToken, shopID string, orderSNs []string) ([]ShopeeOrderDetail, error) {
	if len(orderSNs) == 0 {
		return nil, nil
	}

	cfg, err := LoadShopeeConfig()
	if err != nil {
		return nil, err
	}

	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	path := "/api/v2/order/get_order_detail"
	baseString := BuildAPIBaseString(cfg.PartnerID, path, timestamp, accessToken, shopID)
	sign := GenerateShopeeSignature(cfg.PartnerKey, baseString)

	// Combine SNs
	snList := ""
	for i, sn := range orderSNs {
		if i > 0 {
			snList += ","
		}
		snList += sn
	}

	url := fmt.Sprintf("%s%s?partner_id=%s&timestamp=%s&access_token=%s&shop_id=%s&sign=%s&order_sn_list=%s",
		cfg.BaseURL, path, cfg.PartnerID, timestamp, accessToken, shopID, sign, snList)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	// Read body into buffer to retain raw payload
	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var detailRes shopeeOrderDetailResponse
	if err := json.Unmarshal(bodyBytes, &detailRes); err != nil {
		return nil, err
	}

	if detailRes.Error != "" {
		return nil, fmt.Errorf("shopee api error: %s - %s", detailRes.Error, detailRes.Message)
	}

	var mappedDetails []ShopeeOrderDetail
	for _, d := range detailRes.Response.OrderList {

		// Map items
		var items []ShopeeOrderItem
		for _, item := range d.ItemList {
			items = append(items, ShopeeOrderItem{
				ItemID:     item.ItemID,
				ModelID:    item.ModelID,
				ItemName:   item.ItemName,
				ModelName:  item.ModelName,
				ItemSKU:    item.ItemSKU,
				ModelSKU:   item.ModelSKU,
				ModelQty:   item.ModelQty,
				ModelPrice: item.ModelPrice,
			})
		}

		rawJSON, _ := json.Marshal(d)

		mappedDetails = append(mappedDetails, ShopeeOrderDetail{
			OrderSN:     d.OrderSN,
			Region:      d.Region,
			Currency:    d.Currency,
			TotalAmount: d.TotalAmount,
			OrderStatus: d.OrderStatus,
			CreateTime:  d.CreateTime,
			PayTime:     d.PayTime,
			ItemList:    items,
			RawPayload:  string(rawJSON),
		})
	}

	return mappedDetails, nil
}

// PullProducts pulls products from Shopee.
func (a *ShopeeAdapter) PullProducts() error {
	return ErrNotImplemented
}

// PushStock pushes stock levels to Shopee.
func (a *ShopeeAdapter) PushStock() error {
	return ErrNotImplemented
}
