package api

import (
	"net/http"                      // HTTP status codes
	"regexp"                        // Regular expressions
	"strings"                       // String manipulation
	"wallet_system/internal/domain" // Importing domain models
	"wallet_system/internal/utils"  // Utility functions

	"github.com/gin-gonic/gin"   // Gin web framework
	"golang.org/x/crypto/bcrypt" // Password hashing
	"gorm.io/gorm"               // GORM ORM library
)

// Request and Response structs
type RegisterRequest struct {
	Username string `json:"username" binding:"required"` // Username must be provided
	Password string `json:"password" binding:"required"` // Password must be provided
}

// Request struct for login
type LoginRequest struct {
	Username string `json:"username" binding:"required"` // Username must be provided
	Password string `json:"password" binding:"required"` // Password must be provided
}

// Response struct for authentication
type AuthResponse struct {
	Token string `json:"token"` // JWT token
}

// isValidUsername checks if the username contains only alphabetic characters
func isValidUsername(username string) bool {
	matched, _ := regexp.MatchString(`^[A-Za-z]+$`, username) // Regex to match alphabetic characters only
	return matched                                            // Return whether it matched
}

// isValidPassword checks if the password length is between 8 and 15 characters
func isValidPassword(password string) bool {
	return len(password) >= 8 && len(password) <= 15 // Return true if length is valid
}

// isValidPassword checks if the password length is between 8 and 15 characters
func RegisterHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req RegisterRequest // Bind JSON request to struct
		if err := c.ShouldBindJSON(&req); err != nil {
			// If binding fails, return bad request
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}
		// Validate username and password
		if !isValidUsername(req.Username) {
			// If username is invalid, return bad request
			c.JSON(http.StatusBadRequest, gin.H{"error": "Username must be alphabetic only"})
			return
		}
		// Validate password length
		if !isValidPassword(req.Password) {
			// If password is invalid, return bad request
			c.JSON(http.StatusBadRequest, gin.H{"error": "Password must be 8-15 characters"})
			return
		}
		// Hash the password and create the user
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			// If hashing fails, return internal server error
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}
		// Create user with lowercase username to ensure uniqueness
		user := domain.User{Username: strings.ToLower(req.Username), Password: string(hash)}
		// Attempt to create the user in the database
		if err := db.Create(&user).Error; err != nil {
			// If creation fails (e.g., duplicate username), return bad request
			c.JSON(http.StatusBadRequest, gin.H{"error": "Username already exists"})
			return
		}
		// Return success response
		c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully"})
	}
}

// LoginHandler authenticates a user and returns a JWT token
func LoginHandler(db *gorm.DB, jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LoginRequest // Bind JSON request to struct
		if err := c.ShouldBindJSON(&req); err != nil {
			// If binding fails, return bad request
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}
		var user domain.User // Fetch user from database
		if err := db.Where("username = ?", strings.ToLower(req.Username)).First(&user).Error; err != nil {
			// If user not found, return unauthorized
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}
		// Compare provided password with stored hash
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}
		// Generate JWT token
		token, err := utils.GenerateJWT(user.ID, jwtSecret)
		if err != nil {
			// If token generation fails, return internal server error
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}
		// Return the token in the response
		c.JSON(http.StatusOK, AuthResponse{Token: token})
	}
}
