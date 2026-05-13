package services

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWTService handles JWT token generation and validation.
type JWTService struct {
	secret     string
	expiration time.Duration
}

// JWTClaims holds the custom claims stored in the JWT.
type JWTClaims struct {
	AdminID uuid.UUID `json:"admin_id"`
	Email   string    `json:"email"`
	Role    string    `json:"role"`
	jwt.RegisteredClaims
}

// NewJWTService creates a new JWTService with the given secret.
// Default token expiration is 24 hours.
func NewJWTService(secret string) *JWTService {
	return &JWTService{
		secret:     secret,
		expiration: 24 * time.Hour,
	}
}

// GenerateToken creates a signed JWT for the given admin.
func (s *JWTService) GenerateToken(adminID uuid.UUID, email string, role string) (string, error) {
	now := time.Now()

	claims := JWTClaims{
		AdminID: adminID,
		Email:   email,
		Role:    role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.expiration)),
			Issuer:    "marketplace-ops-api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.secret))
}

// ValidateToken parses and validates a JWT string.
// Returns the claims if valid, or an error if invalid/expired.
func (s *JWTService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Ensure the signing method is HMAC
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.secret), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}
