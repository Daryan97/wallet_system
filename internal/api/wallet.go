package api

import (
	"context"                       // Context for Redis operations
	"net/http"                      // HTTP status codes
	"strconv"                       // String conversion
	"time"                          // Time durations
	"wallet_system/internal/domain" // Importing domain models
	"wallet_system/internal/utils"  // Utility functions

	"github.com/gin-gonic/gin"     // Gin web framework
	"github.com/redis/go-redis/v9" // Redis client
	"gorm.io/gorm"                 // GORM ORM library

	"github.com/sirupsen/logrus" // Logging library
)

// TransferRequest represents a transfer request
type TransferRequest struct {
	ToUsername string  `json:"to_username" binding:"required"` // Target username
	Amount     float64 `json:"amount" binding:"required,gt=0"` // Transfer amount
}

// TransferHandler allows a user to transfer funds to another user's wallet
func TransferHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		fromUserID, exists := c.Get("userID") // Get userID from context
		// Check if userID exists in context
		if !exists {
			// If not, return unauthorized
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		var req TransferRequest // Bind JSON request to struct
		// Validate request
		if err := c.ShouldBindJSON(&req); err != nil || req.Amount <= 0 {
			// If invalid, return bad request
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}
		var toUser domain.User // Find target user
		// Query user by username
		if err := db.Where("username = ?", req.ToUsername).First(&toUser).Error; err != nil {
			// If user not found, return not found
			c.JSON(http.StatusNotFound, gin.H{"error": "Target user not found"})
			return
		}
		// Prevent transferring to self
		if toUser.ID == fromUserID {
			// If trying to transfer to self, return bad request
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot transfer to yourself"})
			return
		}
		var fromWallet, toWallet domain.Wallet // Find wallets
		// Query wallets
		if err := db.Where("user_id = ?", fromUserID).First(&fromWallet).Error; err != nil {
			// If sender wallet not found, return not found
			c.JSON(http.StatusNotFound, gin.H{"error": "Sender wallet not found"})
			return
		}
		// Query recipient wallet
		if err := db.Where("user_id = ?", toUser.ID).First(&toWallet).Error; err != nil {
			// If recipient wallet not found, return not found
			c.JSON(http.StatusNotFound, gin.H{"error": "Recipient wallet not found"})
			return
		}
		// Check sufficient funds
		if fromWallet.Balance < req.Amount {
			// If insufficient funds, return bad request
			c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient funds"})
			return
		}
		// Atomic transfer
		err := db.Transaction(func(tx *gorm.DB) error {
			// Deduct from sender
			if err := tx.Model(&fromWallet).Update("balance", gorm.Expr("balance - ?", req.Amount)).Error; err != nil {
				return err // Return error to rollback
			}
			// Add to recipient
			if err := tx.Model(&toWallet).Update("balance", gorm.Expr("balance + ?", req.Amount)).Error; err != nil {
				return err // Return error to rollback
			}
			// Create transaction record
			t := domain.Transaction{
				FromWalletID: &fromWallet.ID, // Pointer to handle nullability
				ToWalletID:   &toWallet.ID,   // Pointer to handle nullability
				Amount:       req.Amount,     // Transfer amount
				Type:         "transfer",     // Transaction type
			}
			// Save transaction
			if err := tx.Create(&t).Error; err != nil {
				return err // Return error to rollback
			}
			return nil // Commit transaction
		})
		// Handle transaction result
		if err != nil {
			// Log the error with context
			logrus.WithFields(logrus.Fields{
				"from_user_id": fromUserID,  // Sender user ID
				"to_user_id":   toUser.ID,   // Recipient user ID
				"amount":       req.Amount,  // Transfer amount
				"error":        err.Error(), // Error message
			}).Error("Transfer failed") // Log transfer failure
			// Return internal server error
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Transfer failed"})
			return
		}
		// Log successful transfer
		logrus.WithFields(logrus.Fields{
			"from_user_id": fromUserID,                      // Sender user ID
			"to_user_id":   toUser.ID,                       // Recipient user ID
			"amount":       req.Amount,                      // Transfer amount
			"type":         "transfer",                      // Transaction type
			"timestamp":    time.Now().Format(time.RFC3339), // Current timestamp
		}).Info("Transfer transaction") // Log transfer success
		// Invalidate wallet and transaction history cache for both users
		if rdb, ok := c.MustGet("redisClient").(*redis.Client); ok {
			ctx := context.Background()                                              // Context for Redis operations
			fromKey := "wallet:user:" + strconv.Itoa(int(fromUserID.(uint)))         // Cache key for sender
			toKey := "wallet:user:" + strconv.Itoa(int(toUser.ID))                   // Cache key for recipient
			fromTxPrefix := "txhistory:user:" + strconv.Itoa(int(fromUserID.(uint))) // Transaction history prefix for sender
			toTxPrefix := "txhistory:user:" + strconv.Itoa(int(toUser.ID))           // Transaction history prefix for recipient
			_ = utils.DeleteCache(ctx, rdb, fromKey)                                 // Invalidate sender wallet cache
			_ = utils.DeleteCache(ctx, rdb, toKey)                                   // Invalidate recipient wallet cache
			// Invalidate all paginated txhistory cache for both users (simple version: delete first 5 pages)
			for i := 1; i <= 5; i++ {
				// Delete cache entries for both users
				_ = utils.DeleteCache(ctx, rdb, fromTxPrefix+":page:"+strconv.Itoa(i)+":size:20")
				_ = utils.DeleteCache(ctx, rdb, toTxPrefix+":page:"+strconv.Itoa(i)+":size:20")
			}
		}
		// Return success response
		c.JSON(http.StatusOK, gin.H{"message": "Transfer successful"})
	}
}

// DepositRequest represents a deposit request
type DepositRequest struct {
	Amount float64 `json:"amount" binding:"required,gt=0"` // Deposit amount
}

// DepositHandler allows a user to deposit funds into their wallet
func DepositHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get userID from context
		userID, exists := c.Get("userID")
		// Check if userID exists in context
		if !exists {
			// If not, return unauthorized
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		var req DepositRequest // Bind JSON request to struct
		// Validate request
		if err := c.ShouldBindJSON(&req); err != nil || req.Amount <= 0 {
			// If invalid, return bad request
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid amount"})
			return
		}
		var wallet domain.Wallet // Find user's wallet
		// Query wallet by user ID
		if err := db.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
			// If wallet not found, return not found
			c.JSON(http.StatusNotFound, gin.H{"error": "Wallet not found"})
			return
		}
		// Update balance atomically
		err := db.Transaction(func(tx *gorm.DB) error {
			// Increment wallet balance
			if err := tx.Model(&wallet).Update("balance", gorm.Expr("balance + ?", req.Amount)).Error; err != nil {
				return err
			}
			// Create transaction record
			t := domain.Transaction{
				ToWalletID: &wallet.ID, // Pointer to handle nullability
				Amount:     req.Amount, // Deposit amount
				Type:       "deposit",  // Transaction type
			}
			// Save transaction
			if err := tx.Create(&t).Error; err != nil {
				return err // Return error to rollback
			}
			return nil // Commit transaction
		})
		// Handle transaction result
		if err != nil {
			// Log the error with context
			logrus.WithFields(logrus.Fields{
				"user_id": userID,      // User ID
				"amount":  req.Amount,  // Deposit amount
				"error":   err.Error(), // Error message
			}).Error("Deposit failed") // Log deposit failure
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Deposit failed"}) // Return internal server error
			return
		}
		// Log successful deposit
		logrus.WithFields(logrus.Fields{
			"user_id":   userID,                          // User ID
			"amount":    req.Amount,                      // Deposit amount
			"type":      "deposit",                       // Transaction type
			"timestamp": time.Now().Format(time.RFC3339), // Current timestamp
		}).Info("Deposit transaction") // Log deposit success
		// Invalidate wallet and transaction history cache
		if rdb, ok := c.MustGet("redisClient").(*redis.Client); ok {
			ctx := context.Background()                                         // Context for Redis operations
			userKey := "wallet:user:" + strconv.Itoa(int(userID.(uint)))        // Wallet cache key
			txKeyPrefix := "txhistory:user:" + strconv.Itoa(int(userID.(uint))) // Transaction history prefix
			_ = utils.DeleteCache(ctx, rdb, userKey)                            // Invalidate wallet cache
			// Invalidate all paginated txhistory cache for this user (simple version: delete first 5 pages)
			for i := 1; i <= 5; i++ {
				// Delete cache entries
				_ = utils.DeleteCache(ctx, rdb, txKeyPrefix+":page:"+strconv.Itoa(i)+":size:20")
			}
		}
		// Return success response
		c.JSON(http.StatusOK, gin.H{"message": "Deposit successful"})
	}
}

// CreateWalletHandler creates a wallet for a user (one wallet per user)
func CreateWalletHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get userID from context
		userID, exists := c.Get("userID")
		// Check if userID exists in context
		if !exists {
			// If not, return unauthorized
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		// Check if wallet already exists
		var wallet domain.Wallet
		// Query wallet by user ID
		if err := db.Where("user_id = ?", userID).First(&wallet).Error; err == nil {
			// If wallet exists, return bad request
			c.JSON(http.StatusBadRequest, gin.H{"error": "Wallet already exists"})
			return
		}
		// Create new wallet with zero balance
		wallet = domain.Wallet{UserID: userID.(uint), Balance: 0}
		// Save the new wallet
		if err := db.Create(&wallet).Error; err != nil {
			logrus.WithFields(logrus.Fields{
				"user_id": userID,      // User ID
				"error":   err.Error(), // Error message
			}).Error("Failed to create wallet") // Log failure
			// Return internal server error
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create wallet"})
			return
		}
		// Log successful wallet creation
		logrus.WithFields(logrus.Fields{
			"user_id":   userID,                          // User ID
			"wallet_id": wallet.ID,                       // Wallet ID
			"type":      "create_wallet",                 // Transaction type
			"timestamp": time.Now().Format(time.RFC3339), // Current timestamp
		}).Info("Wallet created") // Log wallet creation
		// Invalidate wallet cache
		if rdb, ok := c.MustGet("redisClient").(*redis.Client); ok {
			ctx := context.Background()                                  // Context for Redis operations
			userKey := "wallet:user:" + strconv.Itoa(int(userID.(uint))) // Wallet cache key
			_ = utils.DeleteCache(ctx, rdb, userKey)                     // Invalidate wallet cache
		}
		// Return success response
		c.JSON(http.StatusCreated, gin.H{"message": "Wallet created", "wallet": wallet})
	}
}

// GetWalletHandler returns wallet info for the authenticated user
func GetWalletHandler(db *gorm.DB, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get userID from context
		userID, exists := c.Get("userID") // Get userID from context
		// Check if userID exists in context
		if !exists {
			// If not, return unauthorized
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		ctx := context.Background()                                   // Context for Redis operations
		cacheKey := "wallet:user:" + strconv.Itoa(int(userID.(uint))) // Cache key for wallet
		var wallet domain.Wallet                                      // Wallet struct to hold data
		found, err := utils.GetCache(ctx, rdb, cacheKey, &wallet)     // Try to get from cache
		// If found in cache, return it
		if err == nil && found {
			// Return cached wallet
			c.JSON(http.StatusOK, gin.H{"wallet": wallet, "cached": true})
			return
		}
		// If not in cache, fetch from DB
		if err := db.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
			// Return not found if wallet doesn't exist
			c.JSON(http.StatusNotFound, gin.H{"error": "Wallet not found"})
			return
		}
		_ = utils.SetCache(ctx, rdb, cacheKey, wallet, 60*time.Second)  // Cache the wallet for 60 seconds
		c.JSON(http.StatusOK, gin.H{"wallet": wallet, "cached": false}) // Return wallet info
	}
}

// GetTransactionHistoryHandler returns all transactions for the authenticated user's wallet
func GetTransactionHistoryHandler(db *gorm.DB, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get userID from context
		userID, exists := c.Get("userID")
		// Check if userID exists in context
		if !exists {
			// If not, return unauthorized
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		var wallet domain.Wallet // Get user's wallet
		// Query wallet by user ID
		if err := db.Where("user_id = ?", userID).First(&wallet).Error; err != nil {
			// Return not found if wallet doesn't exist
			c.JSON(http.StatusNotFound, gin.H{"error": "Wallet not found"})
			return
		}
		page := 1      // Default page
		pageSize := 20 // Default page size
		// If page exists in query
		if p := c.Query("page"); p != "" {
			// Convert page to integer
			if v, err := strconv.Atoi(p); err == nil && v > 0 {
				page = v // Set page if valid
			}
		}
		// If page_size exists in query
		if ps := c.Query("page_size"); ps != "" {
			// Convert page_size to integer
			if v, err := strconv.Atoi(ps); err == nil && v > 0 && v <= 100 {
				pageSize = v // Set page size if valid
			}
		}
		offset := (page - 1) * pageSize // Calculate offset
		// Redis cache key
		cacheKey := "txhistory:user:" + strconv.Itoa(int(userID.(uint))) + ":page:" + strconv.Itoa(page) + ":size:" + strconv.Itoa(pageSize)
		ctx := context.Background() // Context for Redis operations
		var cached struct {
			Transactions []domain.Transaction `json:"transactions"` // List of transactions
			Page         int                  `json:"page"`         // Current page
			PageSize     int                  `json:"page_size"`    // Page size
			Total        int64                `json:"total"`        // Total transactions
			TotalPages   int                  `json:"total_pages"`  // Total pages
		}
		// Try to get from cache
		found, err := utils.GetCache(ctx, rdb, cacheKey, &cached)
		// If found in cache, return it
		if err == nil && found {
			c.JSON(http.StatusOK, gin.H{
				"transactions": cached.Transactions, // Cached transactions
				"page":         cached.Page,         // Current page
				"page_size":    cached.PageSize,     // Page size
				"total":        cached.Total,        // Total transactions
				"total_pages":  cached.TotalPages,   // Total pages
				"cached":       true,
			})
			return
		}
		var total int64 // Total count of transactions
		// Count total transactions for pagination
		if err := db.Model(&domain.Transaction{}).
			Where("from_wallet_id = ? OR to_wallet_id = ?", wallet.ID, wallet.ID).
			Count(&total).Error; err != nil {
			// If counting fails, return error
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count transactions"})
			return
		}
		var transactions []domain.Transaction // Slice to hold transactions
		// Fetch paginated transactions
		if err := db.Where("from_wallet_id = ? OR to_wallet_id = ?", wallet.ID, wallet.ID).
			Order("created_at desc").
			Offset(offset).
			Limit(pageSize).
			Find(&transactions).Error; err != nil {
			// If fetching fails, return error
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transactions"})
			return
		}
		// Calculate total pages
		totalPages := (int(total) + pageSize - 1) / pageSize
		resp := gin.H{
			"transactions": transactions, // List of transactions
			"page":         page,         // Current page
			"page_size":    pageSize,     // Page size
			"total":        total,        // Total transactions
			"total_pages":  totalPages,   // Total pages
			"cached":       false,        // Not from cache
		}
		// Cache the result for 60 seconds
		_ = utils.SetCache(ctx, rdb, cacheKey, resp, 60*time.Second)
		c.JSON(http.StatusOK, resp) // Return transaction history
	}
}
