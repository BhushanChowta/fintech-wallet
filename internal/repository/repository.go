package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bhushanchowta/fintech-wallet/internal/models"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type UserRepository struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewUserRepository(db *gorm.DB, rdb *redis.Client) *UserRepository {
	return &UserRepository{db: db, rdb: rdb}
}

func (r *UserRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *UserRepository) FindByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.Where("email = ?", email).First(&user).Error
	return &user, err
}

func (r *UserRepository) FindByID(id uuid.UUID) (*models.User, error) {
	cacheKey := fmt.Sprintf("user:%s", id)
	ctx := context.Background()

	// Try Redis cache first
	cached, err := r.rdb.Get(ctx, cacheKey).Result()
	if err == nil {
		var user models.User
		if json.Unmarshal([]byte(cached), &user) == nil {
			return &user, nil
		}
	}

	// Fallback to DB
	var user models.User
	if err := r.db.Preload("Wallet").First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}

	// Cache for 5 minutes
	if data, err := json.Marshal(user); err == nil {
		r.rdb.Set(ctx, cacheKey, data, 5*time.Minute)
	}

	return &user, nil
}

func (r *UserRepository) InvalidateCache(id uuid.UUID) {
	r.rdb.Del(context.Background(), fmt.Sprintf("user:%s", id))
}

// WalletRepository handles wallet data access
type WalletRepository struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewWalletRepository(db *gorm.DB, rdb *redis.Client) *WalletRepository {
	return &WalletRepository{db: db, rdb: rdb}
}

func (r *WalletRepository) Create(wallet *models.Wallet) error {
	return r.db.Create(wallet).Error
}

func (r *WalletRepository) FindByUserID(userID uuid.UUID) (*models.Wallet, error) {
	var wallet models.Wallet
	err := r.db.Where("user_id = ?", userID).First(&wallet).Error
	return &wallet, err
}

func (r *WalletRepository) FindByID(id uuid.UUID) (*models.Wallet, error) {
	var wallet models.Wallet
	err := r.db.First(&wallet, "id = ?", id).Error
	return &wallet, err
}

func (r *WalletRepository) UpdateBalance(walletID uuid.UUID, newBalance float64) error {
	return r.db.Model(&models.Wallet{}).
		Where("id = ?", walletID).
		Update("balance", newBalance).Error
}

// TransactionRepository handles transaction data access
type TransactionRepository struct {
	db *gorm.DB
}

func NewTransactionRepository(db *gorm.DB) *TransactionRepository {
	return &TransactionRepository{db: db}
}

func (r *TransactionRepository) Create(tx *models.Transaction) error {
	return r.db.Create(tx).Error
}

func (r *TransactionRepository) FindByWalletID(walletID uuid.UUID) ([]models.Transaction, error) {
	var txns []models.Transaction
	err := r.db.Where("sender_wallet_id = ? OR receiver_wallet_id = ?", walletID, walletID).
		Order("created_at DESC").
		Limit(50).
		Find(&txns).Error
	return txns, err
}
