package main

import (
	"fmt"
	"log"

	"github.com/bhushanchowta/fintech-wallet/config"
	"github.com/bhushanchowta/fintech-wallet/internal/auth"
	"github.com/bhushanchowta/fintech-wallet/internal/handlers"
	"github.com/bhushanchowta/fintech-wallet/internal/middleware"
	"github.com/bhushanchowta/fintech-wallet/internal/models"
	"github.com/bhushanchowta/fintech-wallet/internal/repository"
	"github.com/bhushanchowta/fintech-wallet/internal/service"
	"github.com/bhushanchowta/fintech-wallet/pkg/database"
	"github.com/gin-gonic/gin"
)

func main() {
	// Load config
	cfg := config.Load()

	// Init DB & Redis
	db := database.NewPostgres(cfg)
	rdb := database.NewRedis(cfg)

	// Auto-migrate models
	if err := db.AutoMigrate(&models.User{}, &models.Wallet{}, &models.Transaction{}); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	// Wire dependencies
	jwtSvc := auth.NewJWTService(cfg.JWTSecret, cfg.JWTExpiryHours)

	userRepo := repository.NewUserRepository(db, rdb)
	walletRepo := repository.NewWalletRepository(db, rdb)
	txnRepo := repository.NewTransactionRepository(db)

	authSvc := service.NewAuthService(userRepo, jwtSvc, db)
	walletSvc := service.NewWalletService(walletRepo, txnRepo, userRepo, db)

	authHandler := handlers.NewAuthHandler(authSvc)
	walletHandler := handlers.NewWalletHandler(walletSvc)

	// Router setup
	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "fintech-wallet"})
	})

	// Public routes
	v1 := r.Group("/api/v1")
	{
		v1.POST("/auth/register", authHandler.Register)
		v1.POST("/auth/login", authHandler.Login)
	}

	// Protected routes
	protected := v1.Group("/")
	protected.Use(middleware.AuthMiddleware(jwtSvc))
	{
		protected.GET("/wallet", walletHandler.GetWallet)
		protected.POST("/wallet/deposit", walletHandler.Deposit)
		protected.POST("/wallet/transfer", walletHandler.Transfer)
		protected.GET("/wallet/transactions", walletHandler.GetTransactions)
	}

	addr := fmt.Sprintf(":%s", cfg.AppPort)
	log.Printf("Server starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
