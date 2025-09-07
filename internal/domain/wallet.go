package domain

// Wallet Model
type Wallet struct {
	ID      uint    `gorm:"primaryKey"`         // Primary key
	UserID  uint    `gorm:"uniqueIndex"`        // Foreign key to User
	Balance float64 `gorm:"not null;default:0"` // Wallet balance
}
