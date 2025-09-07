package db

import (
	"wallet_system/internal/domain" // Importing domain models

	"github.com/sirupsen/logrus"

	"gorm.io/driver/mysql" // MySQL driver for GORM
	"gorm.io/gorm"         // GORM ORM library
)

// Migrate performs automatic migration for the database schema
func Migrate(dsn string) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{}) // Open a connection to the database
	if err != nil {
		logrus.Fatalf("failed to connect database: %v", err) // Log fatal error if connection fails
	}
	// AutoMigrate will create tables, missing foreign keys, constraints, columns and indexes
	err = db.AutoMigrate(&domain.User{}, &domain.Wallet{}, &domain.Transaction{})
	if err != nil {
		logrus.Fatalf("migration failed: %v", err) // Log fatal error if migration fails
	}
	logrus.Info("Migration completed.") // Log successful migration
}
