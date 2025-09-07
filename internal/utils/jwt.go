package utils

import (
	"time" // Time for token expiration

	"github.com/golang-jwt/jwt/v5" // JWT library
)

// JWT Claims
type Claims struct {
	UserID               uint `json:"user_id"` // Custom claim for user ID
	jwt.RegisteredClaims      // Standard JWT claims
}

// GenerateJWT creates a JWT token for a given user ID
func GenerateJWT(userID uint, secret string) (string, error) {
	// Set token claims
	claims := Claims{
		UserID: userID, // Custom claim for user ID
		// Standard claims
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // Token expires in 24 hours
			IssuedAt:  jwt.NewNumericDate(time.Now()),                     // Issued at current time
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims) // Create token with claims
	return token.SignedString([]byte(secret))                  // Sign the token with the secret
}

// ParseJWT parses and validates a JWT token string
func ParseJWT(tokenStr, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (any, error) {
		return []byte(secret), nil // Return the secret key for validation
	})
	// Check for parsing errors
	if err != nil {
		return nil, err // Return error if parsing fails
	}
	// Validate token and extract claims
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil // Return claims if valid
	}
	// Return error if token is invalid
	return nil, jwt.ErrSignatureInvalid
}
