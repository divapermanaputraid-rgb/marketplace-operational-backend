package marketplace

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// GenerateShopeeSignature generates the HMAC-SHA256 signature required for Shopee Open Platform v2.
func GenerateShopeeSignature(partnerKey, baseString string) string {
	h := hmac.New(sha256.New, []byte(partnerKey))
	h.Write([]byte(baseString))
	return hex.EncodeToString(h.Sum(nil))
}

// BuildAuthBaseString creates the base string for the authorization URL.
func BuildAuthBaseString(partnerID, path, timestamp string) string {
	return fmt.Sprintf("%s%s%s", partnerID, path, timestamp)
}

// BuildAPIBaseString creates the base string for standard API calls and token exchange/refresh.
func BuildAPIBaseString(partnerID, path, timestamp, accessToken, shopID string) string {
	return fmt.Sprintf("%s%s%s%s%s", partnerID, path, timestamp, accessToken, shopID)
}
