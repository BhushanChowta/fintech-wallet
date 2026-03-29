package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Name         string    `gorm:"not null" json:"name"`
	Email        string    `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash string    `gorm:"not null" json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Wallet       Wallet    `gorm:"foreignKey:UserID" json:"wallet,omitempty"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

type Wallet struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"user_id"`
	Balance   float64   `gorm:"type:decimal(15,2);default:0" json:"balance"`
	Currency  string    `gorm:"default:'INR'" json:"currency"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (w *Wallet) BeforeCreate(tx *gorm.DB) error {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	return nil
}

// TransactionType represents the type of a transaction
type TransactionType string

const (
	TransactionTypeCredit   TransactionType = "CREDIT"
	TransactionTypeDebit    TransactionType = "DEBIT"
	TransactionTypeTransfer TransactionType = "TRANSFER"
)

// TransactionStatus represents the status of a transaction
type TransactionStatus string

const (
	TransactionStatusSuccess TransactionStatus = "SUCCESS"
	TransactionStatusFailed  TransactionStatus = "FAILED"
	TransactionStatusPending TransactionStatus = "PENDING"
)

type Transaction struct {
	ID              uuid.UUID         `gorm:"type:uuid;primaryKey" json:"id"`
	SenderWalletID  *uuid.UUID        `gorm:"type:uuid" json:"sender_wallet_id,omitempty"`
	ReceiverWalletID *uuid.UUID       `gorm:"type:uuid" json:"receiver_wallet_id,omitempty"`
	Amount          float64           `gorm:"type:decimal(15,2);not null" json:"amount"`
	Type            TransactionType   `gorm:"not null" json:"type"`
	Status          TransactionStatus `gorm:"not null;default:'PENDING'" json:"status"`
	Description     string            `json:"description"`
	CreatedAt       time.Time         `json:"created_at"`
}

func (t *Transaction) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}
