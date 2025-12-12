package repository

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Структура для работы с базой данных
type WalletRepo struct {
	DB     *pgxpool.Pool
	logger *log.Logger
}

// Конструктор WalletRepo
func NewWalletRepo(db *pgxpool.Pool, logger *log.Logger) *WalletRepo {
	return &WalletRepo{
		DB:     db,
		logger: logger,
	}
}

/*
Создание кошелька (в ТЗ нет, но без UUID и кошелька не будет, а так тестить легче)

Возвращает:

walletUUID string - UUID кошелька

error - error
*/
func (r *WalletRepo) CreateWallet(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	walletUUID := uuid.New().String()

	_, err = tx.Exec(ctx, `
        INSERT INTO wallets (uuid, balance)
        VALUES ($1, $2)`,
		walletUUID, 0)
	if err != nil {
		return "", fmt.Errorf("error creating wallet: %v", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("error committing transaction: %v", err)
	}

	r.logger.Printf("INFO: New wallet %s successfully created!", walletUUID)
	return walletUUID, nil
}

/*
Обновление кошелька

Вызов SELECT внутри транзакции, а не вызов уже существующей функции - потому что так postgres лучше справляется
с конкурентными задачами

Принимает:

walletUUID string - UUID кошелька

operationType string - тип оперции, DEPOSIT либо WITHDRAW

amount int64 - сумма, на которую пополняется/списывается с кошелька

Возвращает:

error - error
*/
func (r *WalletRepo) Update(ctx context.Context, walletUUID, operationType string, amount int64) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	var currentBalance int64
	err = tx.QueryRow(ctx, `
        SELECT balance FROM wallets 
        WHERE uuid = $1
        FOR UPDATE`, // предотвращает race conditions
		walletUUID).Scan(&currentBalance)
	if err != nil {
		return fmt.Errorf("error getting balance: %v", err)
	}

	var newBalance int64
	if operationType == "DEPOSIT" {
		newBalance = currentBalance + amount
	} else if operationType == "WITHDRAW" {
		if amount > currentBalance {
			return fmt.Errorf("insufficient funds: have %d, need %d", currentBalance, amount)
		}
		newBalance = currentBalance - amount
	} else {
		return fmt.Errorf("invalid operation type: %s", operationType)
	}

	_, err = tx.Exec(ctx, `
        UPDATE wallets 
        SET balance = $1 
        WHERE uuid = $2`,
		newBalance, walletUUID)
	if err != nil {
		return fmt.Errorf("error updating balance: %v", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("error committing transaction: %v", err)
	}

	r.logger.Printf("INFO: Wallet %s updated: %s %d (new balance: %d)",
		walletUUID, operationType, amount, newBalance)
	return nil
}

/*
Получение баланса кошелька

Принимает:

walletUUID string - UUID кошелька

Возвращает:

balance int64 - баланс

error - error
*/
func (r *WalletRepo) Balance(ctx context.Context, walletUUID string) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var balance int64
	err := r.DB.QueryRow(ctx, `
        SELECT balance FROM wallets 
        WHERE uuid = $1`,
		walletUUID).Scan(&balance)
	if err != nil {
		return 0, fmt.Errorf("error getting wallet %s balance: %v", walletUUID, err)
	}

	return balance, nil
}
