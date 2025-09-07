package api

import (
	"context"                       // Context for Redis operations
	"net/http"                      // HTTP status codes
	"strconv"                       // String conversion
	"strings"                       // String manipulation
	"time"                          // Time durations
	"wallet_system/internal/domain" // Importing domain models
	"wallet_system/internal/utils"  // Utility functions

	"github.com/gin-gonic/gin"     // Gin web framework
	"github.com/redis/go-redis/v9" // Redis client
	"gorm.io/gorm"                 // GORM ORM library
)

// ListUsersHandler returns all users with their wallet info
func ListUsersHandler(db *gorm.DB, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.Background() // Use background context for Redis
		// Create a cache key based on pagination parameters
		cacheKey := "admin:users:page=" + c.DefaultQuery("page", "1") + ":size=" + c.DefaultQuery("page_size", "20")
		// Try to get cached response
		var cached struct {
			Users      []UserAdminResponse `json:"users"`       // List of users
			Page       int                 `json:"page"`        // Current page
			PageSize   int                 `json:"page_size"`   // Page size
			Total      int64               `json:"total"`       // Total number of users
			TotalPages int                 `json:"total_pages"` // Total pages
		}
		// If cached data found, return it
		found, err := utils.GetCache(ctx, rdb, cacheKey, &cached)
		if err == nil && found {
			c.JSON(http.StatusOK, gin.H{
				"users":       cached.Users,      // List of users
				"page":        cached.Page,       // Current page
				"page_size":   cached.PageSize,   // Page size
				"total":       cached.Total,      // Total number of users
				"total_pages": cached.TotalPages, // Total pages
				"cached":      true,              // Indicate response is from cache
			})
			return
		}
		page := 1      // Default page number
		pageSize := 20 // Default page size
		if p := c.Query("page"); p != "" {
			if v, err := strconv.Atoi(p); err == nil && v > 0 {
				page = v // Set page if valid
			}
		}
		// Check and set page size within limits
		if ps := c.Query("page_size"); ps != "" {
			// If valid, set page size
			if v, err := strconv.Atoi(ps); err == nil && v > 0 && v <= 100 {
				pageSize = v // Set page size
			}
		}
		offset := (page - 1) * pageSize // Calculate offset for pagination
		var total int64                 // Total user count
		// Fetch total user count and paginated users with wallet info
		if err := db.Model(&domain.User{}).Count(&total).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count users"}) // Return on error
			return
		}
		var users []domain.User // Slice to hold users
		// Preload Wallet relation, apply offset and limit for pagination
		if err := db.Preload("Wallet").Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"}) // Return on error
			return
		}
		// The total number of pages
		totalPages := (int(total) + pageSize - 1) / pageSize // Calculate total pages
		// Prepare response data
		resp := make([]UserAdminResponse, len(users))
		// Map users to response format
		for i, u := range users {
			resp[i] = UserAdminResponse{
				ID:       u.ID,       // User ID
				Username: u.Username, // Username
				Role:     u.Role,     // User role
				Wallet:   u.Wallet,   // Associated wallet
			}
		}
		// Prepare final response data
		respData := gin.H{
			"users":       resp,       // List of users
			"page":        page,       // Current page
			"page_size":   pageSize,   // Page size
			"total":       total,      // Total number of users
			"total_pages": totalPages, // Total pages
			"cached":      false,      // Indicate response is not from cache
		}
		// Cache the response for future requests
		_ = utils.SetCache(ctx, rdb, cacheKey, respData, 60*time.Second)
		c.JSON(http.StatusOK, respData) // Return the response
	}
}

// UserAdminResponse represents the user data returned to admin
type UserAdminResponse struct {
	ID       uint          `json:"id"`       // User ID
	Username string        `json:"username"` // Username
	Role     string        `json:"role"`     // User role
	Wallet   domain.Wallet `json:"wallet"`   // Associated wallet
}

// ListTransactionsHandler returns all transactions, with optional filtering by user, type, or date
func ListTransactionsHandler(db *gorm.DB, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.Background()
		// Build cache key from all query params
		var keyParts []string // Parts of the cache key
		// Append each query parameter to the key parts
		for _, k := range []string{"user_id", "type", "from", "to", "page", "page_size"} {
			keyParts = append(keyParts, k+"="+c.DefaultQuery(k, "")) // Append key-value pair
		}
		// Join key parts to form the final cache key
		cacheKey := "admin:txs:" + strings.Join(keyParts, ":")
		var cached struct {
			Transactions []domain.Transaction `json:"transactions"` // List of transactions
			Page         int                  `json:"page"`         // Current page
			PageSize     int                  `json:"page_size"`    // Page size
			Total        int64                `json:"total"`        // Total number of transactions
			TotalPages   int                  `json:"total_pages"`  // Total pages
		}

		// If cached data found, return it
		found, err := utils.GetCache(ctx, rdb, cacheKey, &cached)
		if err == nil && found {
			c.JSON(http.StatusOK, gin.H{
				"transactions": cached.Transactions, // List of transactions
				"page":         cached.Page,         // Current page
				"page_size":    cached.PageSize,     // Page size
				"total":        cached.Total,        // Total number of transactions
				"total_pages":  cached.TotalPages,   // Total pages
				"cached":       true,                // Indicate response is from cache
			})
			return
		}
		page := 1      // Default page number
		pageSize := 20 // Default page size
		// Check and set page number and size from query params
		if p := c.Query("page"); p != "" {
			// If valid, set page number
			if v, err := strconv.Atoi(p); err == nil && v > 0 {
				page = v // Set page if valid
			}
		}
		// Check and set page size within limits
		if ps := c.Query("page_size"); ps != "" {
			// If valid, set page size
			if v, err := strconv.Atoi(ps); err == nil && v > 0 && v <= 100 {
				pageSize = v // Set page size
			}
		}
		offset := (page - 1) * pageSize          // Calculate offset for pagination
		query := db.Model(&domain.Transaction{}) // Start building the query
		if userID := c.Query("user_id"); userID != "" {
			query = query.Where("from_wallet_id = ? OR to_wallet_id = ?", userID, userID) // Filter by user ID
		}
		if txType := c.Query("type"); txType != "" {
			query = query.Where("type = ?", txType) // Filter by transaction type
		}
		if from := c.Query("from"); from != "" {
			query = query.Where("created_at >= ?", from) // Filter by start date
		}
		if to := c.Query("to"); to != "" {
			query = query.Where("created_at <= ?", to) // Filter by end date
		}
		var total int64 // Total transaction count
		// Get total count of transactions matching the filters
		if err := query.Count(&total).Error; err != nil {
			// If error occurs, return internal server error
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count transactions"})
			return
		}
		var txs []domain.Transaction // Slice to hold transactions
		// Fetch paginated transactions with filters applied
		if err := query.Order("created_at desc").Offset(offset).Limit(pageSize).Find(&txs).Error; err != nil {
			// If error occurs, return internal server error
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transactions"})
			return
		}
		// The total number of pages
		totalPages := (int(total) + pageSize - 1) / pageSize
		respData := gin.H{
			"transactions": txs,        // List of transactions
			"page":         page,       // Current page
			"page_size":    pageSize,   // Page size
			"total":        total,      // Total number of transactions
			"total_pages":  totalPages, // Total pages
			"cached":       false,      // Indicate response is not from cache
		}
		// Cache the response for future requests
		_ = utils.SetCache(ctx, rdb, cacheKey, respData, 60*time.Second)
		c.JSON(http.StatusOK, respData) // Return the response
	}
}
