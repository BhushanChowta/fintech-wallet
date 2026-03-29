package service

import (
	"errors"
	"fmt"

	"github.com/bhushanchowta/fintech-wallet/internal/auth"
	"github.com/bhushanchowta/fintech-wallet/internal/models"
	"github.com/bhushanchowta/fintech-wallet/internal/repository"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// --- Auth Service ---

type AuthService struct {
	userRepo *repository.UserRepository
	jwtSvc   *auth.JWTService
	db       *gorm.DB
}

func NewAuthService(userRepo *repository.UserRepository, jwtSvc *auth.JWTService, db *gorm.DB) *AuthService {
	return &AuthService{userRepo: userRepo, jwtSvc: jwtSvc, db: db}
}

type RegisterRequest struct {
	Name     string `json:"name" binding:"required,min=2"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token string      `json:"token"`
	User  models.User `json:"user"`
}

func (s *AuthService) Register(req RegisterRequest) (*AuthResponse, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &models.User{
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: string(hash),
	}

	// Create user + wallet in a transaction
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return err
		}
		wallet := &models.Wallet{UserID: user.ID, Currency: "INR"}
		return tx.Create(wallet).Error
	})
	if err != nil {
		return nil, fmt.Errorf("registration failed: %w", err)
	}

	token, err := s.jwtSvc.GenerateToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{Token: token, User: *user}, nil
}

func (s *AuthService) Login(req LoginRequest) (*AuthResponse, error) {
	user, err := s.userRepo.FindByEmail(req.Email)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	token, err := s.jwtSvc.GenerateToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{Token: token, User: *user}, nil
}

// --- Wallet Service ---

type WalletService struct {
	walletRepo *repository.WalletRepository
	txnRepo    *repository.TransactionRepository
	userRepo   *repository.UserRepository
	db         *gorm.DB
}

func NewWalletService(
	walletRepo *repository.WalletRepository,
	txnRepo *repository.TransactionRepository,
	userRepo *repository.UserRepository,
	db *gorm.DB,
) *WalletService {
	return &WalletService{walletRepo: walletRepo, txnRepo: txnRepo, userRepo: userRepo, db: db}
}

type DepositRequest struct {
	Amount      float64 `json:"amount" binding:"required,gt=0"`
	Description string  `json:"description"`
}

type TransferRequest struct {
	ReceiverUserID uuid.UUID `json:"receiver_user_id" binding:"required"`
	Amount         float64   `json:"amount" binding:"required,gt=0"`
	Description    string    `json:"description"`
}

func (s *WalletService) GetWallet(userID uuid.UUID) (*models.Wallet, error) {
	return s.walletRepo.FindByUserID(userID)
}

func (s *WalletService) Deposit(userID uuid.UUID, req DepositRequest) (*models.Transaction, error) {
	wallet, err := s.walletRepo.FindByUserID(userID)
	if err != nil {
		return nil, errors.New("wallet not found")
	}

	var txn *models.Transaction

	err = s.db.Transaction(func(tx *gorm.DB) error {
		newBalance := wallet.Balance + req.Amount
		if err := s.walletRepo.UpdateBalance(wallet.ID, newBalance); err != nil {
			return err
		}

		txn = &models.Transaction{
			ReceiverWalletID: &wallet.ID,
			Amount:           req.Amount,
			Type:             models.TransactionTypeCredit,
			Status:           models.TransactionStatusSuccess,
			Description:      req.Description,
		}
		return s.txnRepo.Create(txn)
	})

	return txn, err
}

func (s *WalletService) Transfer(senderUserID uuid.UUID, req TransferRequest) (*models.Transaction, error) {
	if senderUserID == req.ReceiverUserID {
		return nil, errors.New("cannot transfer to yourself")
	}

	senderWallet, err := s.walletRepo.FindByUserID(senderUserID)
	if err != nil {
		return nil, errors.New("sender wallet not found")
	}

	if senderWallet.Balance < req.Amount {
		return nil, errors.New("insufficient balance")
	}

	receiverWallet, err := s.walletRepo.FindByUserID(req.ReceiverUserID)
	if err != nil {
		return nil, errors.New("receiver wallet not found")
	}

	var txn *models.Transaction

	// Atomic DB transaction — critical for fintech correctness
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.walletRepo.UpdateBalance(senderWallet.ID, senderWallet.Balance-req.Amount); err != nil {
			return err
		}
		if err := s.walletRepo.UpdateBalance(receiverWallet.ID, receiverWallet.Balance+req.Amount); err != nil {
			return err
		}

		txn = &models.Transaction{
			SenderWalletID:   &senderWallet.ID,
			ReceiverWalletID: &receiverWallet.ID,
			Amount:           req.Amount,
			Type:             models.TransactionTypeTransfer,
			Status:           models.TransactionStatusSuccess,
			Description:      req.Description,
		}
		return s.txnRepo.Create(txn)
	})

	return txn, err
}

func (s *WalletService) GetTransactions(userID uuid.UUID) ([]models.Transaction, error) {
	wallet, err := s.walletRepo.FindByUserID(userID)
	if err != nil {
		return nil, errors.New("wallet not found")
	}
	return s.txnRepo.FindByWalletID(wallet.ID)
}
