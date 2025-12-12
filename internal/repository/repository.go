package repository

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WalletRepo struct {
	DB     *pgxpool.Pool
	logger *log.Logger
}

func NewWalletRepo(db *pgxpool.Pool, logger *log.Logger) *WalletRepo {
	return &WalletRepo{
		DB:     db,
		logger: logger,
	}
}

func (r *WalletRepo) CreateWallet(ctx context.Context) (string, error) {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	uuid := uuid.New()
	_, err = tx.Exec(ctx, `
        INSERT INTO wallets (uuid, balance)
        VALUES ($1, $2)`,
		uuid.String(), 0)

	if err != nil {
		return "", fmt.Errorf("error creating wallet: %v", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("error committing transaction: %v", err)
	}

	r.logger.Printf("INFO: New wallet %s successfully created!", uuid.String())
	return uuid.String(), nil
}
func (r *WalletRepo) Update(ctx context.Context, uuid, operationType string, amount int64) error {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	walletBalance, err := r.Balance(ctx, uuid)
	if err != nil {
		return fmt.Errorf("error updating balance: %v", err)
	}

	if operationType == "DEPOSIT" {
		walletBalance += amount
	} else if operationType == "WITHDRAW" {
		if amount > walletBalance {
			return fmt.Errorf("can't withdraw more than balance amount")
		}
		walletBalance -= amount
	} else {
		return fmt.Errorf("invalid operation type: %s", operationType)
	}

	_, err = tx.Exec(ctx, `
    UPDATE wallets SET balance = $1 WHERE uuid = $2`,
		walletBalance, uuid)

	if err != nil {
		return fmt.Errorf("error updating balance on wallet %s: %v", uuid, err)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("error committing transaction: %v", err)
	}

	r.logger.Printf("INFO: Wallet %s updated", uuid)
	return nil
}
func (r *WalletRepo) Balance(ctx context.Context, uuid string) (int64, error) {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	var balance int64
	err = r.DB.QueryRow(ctx, `
			SELECT balance FROM wallets 
			WHERE uuid = $1`, uuid).Scan(&balance)

	if err != nil {
		return 0, fmt.Errorf("error getting wallet %s balance: %v", uuid, err)
	}

	if err = tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("error committing transaction: %v", err)
	}

	r.logger.Printf("DEBUG: Got wallet %s balance: %d", uuid, balance)
	return balance, nil
}
