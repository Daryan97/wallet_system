package main

import (
	"context"                           // context package is needed for Redis operations
	"log"                               // log package is needed for logging
	"wallet_system/internal/api"        // Custom package for API handlers
	"wallet_system/internal/config"     // Custom package for configuration
	"wallet_system/internal/middleware" // Custom package for middleware

	// For loading .env files
	"github.com/gin-gonic/gin"     // Gin web framework
	"github.com/redis/go-redis/v9" // Redis client
	"github.com/sirupsen/logrus"   // Logrus for structured logging
	"gorm.io/driver/mysql"         // MySQL driver for GORM
	"gorm.io/gorm"                 // GORM ORM library
)

// Main function to set up and run the server
func main() {
	cfg := config.LoadConfig() // Load configuration

	// Setup logger
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	// Setup Data Source Name (DSN) and connect to the database
	dsn := cfg.DBUser + ":" + cfg.DBPassword + "@tcp(" + cfg.DBHost + ":" + cfg.DBPort + ")/" + cfg.DBName + "?parseTime=true"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		logrus.Fatalf("failed to connect to DB: %v", err) // Fatal error if DB connection fails
	}

	// Setup Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr, // Redis server address
		Password: cfg.RedisPass, // Redis password
		DB:       cfg.RedisDB,   // Redis database number
	})

	// Test Redis connection
	_, err = redisClient.Ping(context.Background()).Result()
	if err != nil {
		logrus.Fatalf("failed to connect to Redis: %v", err)
	}

	// Set Mode to Release if in production
	if cfg.IsProd {
		gin.SetMode(gin.ReleaseMode)
	}

	// Setup Gin
	r := gin.Default() // Gin router instance

	// Set trusted proxies for Gin
	if err := r.SetTrustedProxies([]string{"127.0.0.1"}); err != nil {
		logrus.Fatalf("failed to set trusted proxies: %v", err)
	}

	// Auth routes
	r.POST("/user", api.RegisterHandler(db))            // Registration endpoint
	r.GET("/user", api.LoginHandler(db, cfg.JWTSecret)) // Login endpoint

	// Wallet routes (protected by JWT)
	walletGroup := r.Group("/wallet")
	// Protect wallet routes with JWT middleware and inject Redis client into context
	walletGroup.Use(middleware.JWTAuthMiddleware(cfg.JWTSecret), func(c *gin.Context) {
		c.Set("redisClient", redisClient)
		c.Next()
	})
	walletGroup.POST("", api.CreateWalletHandler(db))                                   // Create wallet endpoint
	walletGroup.GET("", api.GetWalletHandler(db, redisClient))                          // Get wallet endpoint
	walletGroup.POST("/deposit", api.DepositHandler(db))                                // Deposit endpoint
	walletGroup.POST("/transfer", api.TransferHandler(db))                              // Transfer endpoint
	walletGroup.GET("/transactions", api.GetTransactionHistoryHandler(db, redisClient)) // Transaction history endpoint

	// Admin routes (protected, admin only)
	adminGroup := r.Group("/admin")
	// Protect admin routes with JWT and AdminOnly middleware
	adminGroup.Use(middleware.JWTAuthMiddleware(cfg.JWTSecret), middleware.AdminOnlyMiddleware(db))
	adminGroup.GET("/users", api.ListUsersHandler(db, redisClient))               // List users endpoint
	adminGroup.GET("/transactions", api.ListTransactionsHandler(db, redisClient)) // List transactions endpoint

	log.Println("Server running on " + cfg.AppPort) // Log server start
	r.Run(":" + cfg.AppPort)                        // Start the server on port cfg.AppPort
}
