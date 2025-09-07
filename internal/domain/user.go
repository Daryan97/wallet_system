package domain

// User Model
type User struct {
	ID       uint   `gorm:"primaryKey"`                                     // Primary key
	Username string `gorm:"unique;not null"`                                // Unique username
	Password string `gorm:"not null"`                                       // Hashed password
	Role     string `gorm:"default:user"`                                   // Role: user or admin
	Wallet   Wallet `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"` // One-to-one relationship with Wallet
}
