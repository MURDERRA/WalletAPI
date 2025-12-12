package service

import (
	"WalletAPI/m/internal/model"
	"WalletAPI/m/internal/repository"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Структура для API
type WalletAPI struct {
	WalletRepo *repository.WalletRepo
	logger     *log.Logger
}

// Конструктор WalletAPI
func NewWalletAPI(walletRepo *repository.WalletRepo, logger *log.Logger) *WalletAPI {
	return &WalletAPI{
		WalletRepo: walletRepo,
		logger:     logger,
	}
}

// CreateWallet godoc
// @Summary Create a new wallet
// @Description Creates a new wallet with zero balance and returns its UUID
// @Tags Wallets
// @Accept json
// @Produce json
// @Success 200 {object} model.Response{data=map[string]string} "Wallet created successfully"
// @Failure 500 {object} model.Response "Internal server error"
// @Router /create [post]
func (api *WalletAPI) CreateWallet(c *gin.Context) {
	walletUUID, err := api.WalletRepo.CreateWallet(c.Request.Context())
	if err != nil {
		api.logger.Printf("ERROR: Failed to create wallet: %v", err)
		c.JSON(http.StatusInternalServerError, model.Response{
			Success: false,
			Error:   "Internal Error",
		})
		return
	}

	api.logger.Printf("INFO: Created wallet %s", walletUUID)
	c.JSON(http.StatusOK, model.Response{
		Success: true,
		Data:    map[string]string{"walletId": walletUUID}, // возвращаем UUID кошелька
	})
}

// UpdateBalance godoc
// @Summary Update wallet balance
// @Description Deposits or withdraws funds from a wallet
// @Tags Wallets
// @Accept json
// @Produce json
// @Param request body model.UpdateBalance true "Update balance request"
// @Success 200 {object} model.Response{data=map[string]string} "Balance updated successfully"
// @Failure 400 {object} model.Response "Invalid request body"
// @Failure 500 {object} model.Response "Internal server error"
// @Router /wallet [post]
func (api *WalletAPI) UpdateBalance(c *gin.Context) {
	var req model.UpdateBalance
	if err := c.ShouldBindJSON(&req); err != nil {
		api.logger.Printf("ERROR: Invalid request body: %v", err)
		c.JSON(http.StatusBadRequest, model.Response{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	api.logger.Printf("INFO: Wallet %s requested %s , amount %d", req.WalletId, req.OperationType, req.Amount)

	err := api.WalletRepo.Update(c.Request.Context(), req.WalletId, req.OperationType, req.Amount)
	if err != nil {
		api.logger.Printf("ERROR: Failed to update wallet %s: %v", req.WalletId, err)
		c.JSON(http.StatusInternalServerError, model.Response{
			Success: false,
			Error:   "Internal Error",
		})
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Success: true,
		Data:    map[string]string{"message": "Wallet updated successfully!"},
	})
}

// GetBalance godoc
// @Summary Get wallet balance
// @Description Returns the current balance of a wallet by its UUID
// @Tags Wallets
// @Accept json
// @Produce json
// @Param WALLET_UUID path string true "Wallet UUID"
// @Success 200 {object} model.Response{data=map[string]interface{}} "Balance retrieved successfully"
// @Failure 400 {object} model.Response "Wallet UUID not provided"
// @Failure 404 {object} model.Response "Wallet not found"
// @Router /wallets/{WALLET_UUID} [get]
func (api *WalletAPI) GetBalance(c *gin.Context) {
	walletUUID := c.Param("WALLET_UUID") // param берёт значение WALLET_UUID из url запроса
	if walletUUID == "" {
		api.logger.Printf("ERROR: Wallet UUID not provided")
		c.JSON(http.StatusBadRequest, model.Response{
			Success: false,
			Error:   "Wallet UUID is required",
		})
		return
	}

	balance, err := api.WalletRepo.Balance(c.Request.Context(), walletUUID)
	if err != nil {
		api.logger.Printf("ERROR: Failed to get balance for wallet %s: %v", walletUUID, err)
		c.JSON(http.StatusNotFound, model.Response{
			Success: false,
			Error:   "Wallet not found",
		})
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Success: true,
		Data:    map[string]interface{}{"balance": balance}, // возврашаемый баланс, собственно
	})
}

// Настройка ручек для API
func SetupRoutes(router *gin.Engine, api *WalletAPI) {
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler)) // для swagger документации, в логах есть ссылка на неё

	// базовая безопасность
	router.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Next()
	})

	router.POST("/v1/create", api.CreateWallet)
	router.POST("/v1/wallet", api.UpdateBalance)
	router.GET("/v1/wallets/:WALLET_UUID", api.GetBalance)
}
