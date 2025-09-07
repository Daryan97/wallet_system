package main

import (
	"wallet_system/internal/config" // Custom import path (Config)
	"wallet_system/internal/db"     // Custom import path (Database)
)

// Main entry point for migration
func main() {
	cfg := config.LoadConfig() // Load configuration

	// Database Source Name (DSN) for MySQL connection
	dsn := cfg.DBUser + ":" + cfg.DBPassword + "@tcp(" + cfg.DBHost + ":" + cfg.DBPort + ")/" + cfg.DBName + "?parseTime=true"
	db.Migrate(dsn)
}
