package domain

// Transaction Model
type Transaction struct {
	ID           uint    `gorm:"primaryKey"` // Primary key
	FromWalletID *uint   // Foreign key to Wallet of the sender
	ToWalletID   *uint   // Foreign key to Wallet of the receiver
	Amount       float64 // Amount of the transaction
	Type         string  // Transaction type: deposit, transfer
	CreatedAt    int64   `gorm:"autoCreateTime:milli"` // Timestamp of creation in milliseconds
}
