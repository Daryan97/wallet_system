package middleware

import (
	"net/http"                      // HTTP status codes
	"wallet_system/internal/domain" // Importing domain models

	"github.com/gin-gonic/gin" // Gin web framework
	"gorm.io/gorm"             // GORM ORM library
)

// AdminOnlyMiddleware checks the user's role from the database on each request
func AdminOnlyMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("userID") // Get userID from context
		// Check if userID exists in context
		if !exists {
			// If not, abort with unauthorized status
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		var user domain.User // Fetch user from database
		if err := db.First(&user, userID).Error; err != nil {
			// If user not found or any error, abort with forbidden status
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}
		// Check if user role is admin
		if user.Role != "admin" {
			// If not admin, abort with forbidden status
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}
		// If admin, proceed to the next handler
		c.Next()
	}
}
