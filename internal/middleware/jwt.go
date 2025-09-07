package middleware

import (
	"net/http"                     // HTTP status codes
	"strings"                      // String manipulation
	"wallet_system/internal/utils" // JWT utility functions

	"github.com/gin-gonic/gin" // Gin web framework
)

// JWTAuthMiddleware validates JWT tokens and extracts user information
func JWTAuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization") // Get Authorization header
		// Check if the Authorization header is present and properly formatted
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			// If not, abort with unauthorized status
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid Authorization header"})
			return
		}
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ") // Extract the token string and parse it
		claims, err := utils.ParseJWT(tokenStr, secret)       // Parse the JWT token
		if err != nil {
			// If parsing fails, abort with unauthorized status
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}
		c.Set("userID", claims.UserID) // Store userID in context
		c.Next()                       // Proceed to the next handler
	}
}
