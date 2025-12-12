package main

import (
	_ "WalletAPI/m/docs"
	"WalletAPI/m/internal/config"
	"WalletAPI/m/internal/repository"
	"WalletAPI/m/internal/service"
	"context"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// @title Wallet API
// @version 1.0
// @description API for managing cryptocurrency wallets with deposit and withdrawal operations
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@walletapi.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /v1/
// @schemes http
func main() {
	cfg, err := config.InitConfig("WalletAPI")
	if err != nil {
		println(err)
		panic(err)
	}

	logger := cfg.Logger
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.PostgresURL)
	if err != nil {
		logger.Fatalf("FATAL: failed to connect to Postgres database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		logger.Fatalf("ping DB not pong: %v", err)
	}

	walletRepo := repository.NewWalletRepo(pool, logger)
	walletAPI := service.NewWalletAPI(walletRepo, logger)

	router := gin.Default()
	service.SetupRoutes(router, walletAPI)

	logger.Printf("INFO: API запущено на http://localhost:8080")
	logger.Printf("INFO: Swagger UI доступен по адресу: http://localhost:8080/swagger/index.html")

	if err := router.Run(":8080"); err != nil {
		logger.Printf("ERROR: ошибка запуска сервера: %v", err)
	}

}
