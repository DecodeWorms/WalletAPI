package storage

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"time"
	"wallet-api/model"
)

type PostgresStore struct {
	db *pgxpool.Pool
}

func NewPostgresStore(ctx context.Context, dbUrl string) (*PostgresStore, error) {
	// Scope the context to database handshake and ctx is an application scope
	initCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cfg, err := pgxpool.ParseConfig(dbUrl)
	if err != nil {
		return nil, fmt.Errorf("unable to Parse the connection string %v", err)
	}

	// Config database request connection
	cfg.MaxConns = 25
	cfg.MinConns = 5
	cfg.MaxConnLifetime = 1 * time.Hour

	pool, err := pgxpool.NewWithConfig(initCtx, cfg)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to the database %v", err)
	}

	// Ping the database
	if err := pool.Ping(initCtx); err != nil {
		return nil, fmt.Errorf("database ping failed %v", err)
	}
	log.Println("Connected to the database successfully")
	return &PostgresStore{
		db: pool,
	}, nil
}

// TransactionExists checks if a reference ID is already in the ledger
func (s *PostgresStore) TransactionExists(ctx context.Context, reference string) (bool, error) {
	query := "SELECT reference FROM transactions WHERE reference=$1"
	row := s.db.QueryRow(ctx, query, reference)

	err := row.Scan(&reference)
	if err == pgx.ErrNoRows {
		return false, nil // transaction has not happened yet
	} else if err != nil {
		return false, err // database crashed
	}
	return true, nil
}

// GetWalletByID fetches a wallet
func (s *PostgresStore) GetWalletByID(ctx context.Context, walletID uuid.UUID) (*model.Wallet, error) {
	// YOUR TURN: Write the SQL query logic here!
	query := "SELECT id, user_id, balance ,  currency , created_at ,updated_at FROM wallets WHERE id=$1"
	var result model.Wallet
	r := s.db.QueryRow(ctx, query, walletID)
	err := r.Scan(&result.ID, &result.UserID, &result.Balance, &result.Currency, &result.CreatedAt, &result.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &result, err
}

func (s *PostgresStore) CreateWallet(ctx context.Context, data model.Wallet) error {
	query := "INSERT INTO wallets(id, user_id,balance, currency,created_at,updated_at)" +
		"VALUES($1,$2,$3,$4,$5,$6)"
	_, err := s.db.Exec(ctx, query, data.ID, data.UserID, data.Balance, data.Currency, data.CreatedAt, data.UpdatedAt)
	if err != nil {
		return err
	}
	return nil
}

// ExecuteTransfer handles the ACID database transaction with Row-Level Locks
func (s *PostgresStore) ExecuteTransfer(ctx context.Context, fromWalletID, toWalletID uuid.UUID, amount int, reference string) error {
	// 1. Begin the transaction
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}

	// If anything fails, this guarantees the transaction rolls back safely!
	defer tx.Rollback(ctx)
	// 2. Lock the Sender's Wallet so nobody else can touch it (FOR UPDATE)
	var fromBalance int
	err = tx.QueryRow(ctx, "SELECT balance FROM wallets WHERE id=$1 FOR UPDATE", fromWalletID).Scan(&fromBalance)
	if err != nil {
		return fmt.Errorf("failed to lock sender wallet: %w", err)
	}
	// Validate the sender account is eligible to send money out
	frmRes, err := subCheck(fromBalance, amount)
	if err != nil {
		return err
	}
	// 3. Lock the Receiver's Wallet (FOR UPDATE)
	var toBalance int
	err = tx.QueryRow(ctx, "SELECT balance FROM wallets WHERE id=$1 FOR UPDATE", toWalletID).Scan(&toBalance)
	if err != nil {
		return fmt.Errorf("failed to lock receiver wallet: %w", err)
	}

	// Ensure the receiver amount does not crash our system
	subRes, err := addCheck(toBalance, amount)
	if err != nil {
		return err
	}
	// 4. Deduct from Sender (Notice the $1, $2 fix here!)
	_, err = tx.Exec(ctx, "UPDATE wallets SET balance = balance - $1 WHERE id=$2", amount, fromWalletID)
	if err != nil {
		return err
	}
	// 5. Add to Receiver
	_, err = tx.Exec(ctx, "UPDATE wallets SET balance = balance + $1 WHERE id=$2", amount, toWalletID)
	if err != nil {
		return err
	}
	// 6. Insert the audit logs into the transactions ledger
	_, err = tx.Exec(ctx, `
		INSERT INTO transactions (id, wallet_id, amount, balance_before, balance_now, type, reference, status, created_at)
		VALUES ($1, $2, $3, $4, $5, 'DEBIT', $6, 'SUCCESS', NOW()),
		       ($7, $8, $9, $10, $11, 'CREDIT', $6, 'SUCCESS', NOW())
	`, uuid.New(), fromWalletID, amount, fromBalance, frmRes, reference,
		uuid.New(), toWalletID, amount, toBalance, subRes)
	if err != nil {
		return err
	}
	// 7. If everything succeeded, commit!
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

func addCheck(a, b int) (int, error) {
	c := a + b
	if c < a || c < b {
		return 0, errors.New("amount over flow")
	}
	return c, nil
}

func subCheck(balance, amount int) (int, error) {
	if amount == 0 {
		return 0, errors.New("error invalid amount, amount cannot be 0")
	}
	if amount > balance {
		return 0, errors.New("error insufficient balance")
	}
	return balance - amount, nil
}
