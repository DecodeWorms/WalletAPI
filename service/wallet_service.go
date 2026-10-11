package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"time"
	"wallet-api/model"
	"wallet-api/storage"
)

// WalletRepository defines the database actions we need
type WalletRepository interface {
	// Step 2: Check for Idempotency
	TransactionExists(ctx context.Context, reference string) (bool, error)

	// Step 3 & 4: Get a wallet to check existence and balance
	GetWalletByID(ctx context.Context, walletID uuid.UUID) (*model.Wallet, error)

	// Create user wallet
	CreateWallet(ctx context.Context, data model.Wallet) error

	// Step 5: Execute the actual atomic database transaction
	ExecuteTransfer(ctx context.Context, fromWalletID, toWalletID uuid.UUID, amount int, reference string) error
}

type Wallet struct {
	UserID   string `json:"user_id"`
	Amount   int    `json:"amount"`
	Currency string `json:"currency"`
}
type UserBalance struct {
	WalletID uuid.UUID
	Balance  int
}

type WalletService struct {
	repo  WalletRepository
	cache storage.CacheStorage
}

func NewWalletService(repo WalletRepository, cache storage.CacheStorage) *WalletService {
	return &WalletService{repo: repo, cache: cache}
}

func (s *WalletService) GetWalletBalance(ctx context.Context, walletID string) (*UserBalance, error) {
	// Parse walletID
	walID, err := uuid.Parse(walletID)
	if err != nil {
		return nil, err
	}

	// Fetch user wallet balance from the cache to prevent the primary db from overwhelming with concurrent requests
	userWallets, err := s.cache.Get(ctx, "user:"+walletID)
	if err == nil {
		// Cache hit then return the result immediately
		var cacheWalletResult UserBalance
		if err := json.Unmarshal([]byte(userWallets), &cacheWalletResult); err != nil {

		}
		return &cacheWalletResult, nil
	}
	// if cache miss!
	dbWalletResult, err := s.repo.GetWalletByID(ctx, walID)
	if err != nil {
		return nil, err
	}
	// Update cache and return the result to the client == cache-aside
	d := UserBalance{
		WalletID: dbWalletResult.ID,
		Balance:  dbWalletResult.Balance,
	}
	data, err := json.Marshal(d)
	if err == nil {
		_ = s.cache.Set(ctx, "user:"+walletID, data, 30*time.Second)
	}
	return &d, nil
}

func (s *WalletService) CreateWallet(ctx context.Context, currency, userID string, amount int) error {
	// Validate amount is not a negative value
	if amount <= 0 {
		return errors.New("invalid amount, amount must be more than 0")
	}
	walletID := uuid.New()
	parseUserID, err := uuid.Parse(userID)
	if err != nil {
		return errors.New("error parsing userID")
	}
	storageData := model.Wallet{
		ID:        walletID,
		UserID:    parseUserID,
		Balance:   amount,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Currency:  currency,
	}
	if err := s.repo.CreateWallet(ctx, storageData); err != nil {
		return fmt.Errorf("error creating user wallet %v", err)
	}
	return nil
}

// TransferFunds moves money from one wallet to another
func (s *WalletService) TransferFunds(ctx context.Context, fromWalletID, toWalletID uuid.UUID, amount int, reference string) error {
	// Convert both the sender and receiver walletIDs to string
	fromWalID := fromWalletID.String()
	toWalID := toWalletID.String()
	// 1. Ensure the amount is valid
	if amount <= 0 {
		return errors.New("transfer amount must be greater than zero")
	}

	// 2. Check for Idempotency (Has this happened before?)
	exists, err := s.repo.TransactionExists(ctx, reference)
	if err != nil {
		return err // Real DB crash
	}
	if exists {
		return errors.New("duplicate transaction reference") // Idempotency caught it!
	}

	// 3. Validate sender wallet and check balance
	fromWallet, err := s.repo.GetWalletByID(ctx, fromWalletID)
	if err != nil {
		return errors.New("sender wallet not found")
	}
	if fromWallet.Balance < amount {
		return errors.New("insufficient funds")
	}

	// 4. Validate receiver wallet
	_, err = s.repo.GetWalletByID(ctx, toWalletID)
	if err != nil {
		return errors.New("receiver wallet not found")
	}

	// 5. Execute the ACID Database Transaction
	err = s.repo.ExecuteTransfer(ctx, fromWalletID, toWalletID, amount, reference)
	if err != nil {
		return errors.New("transfer failed during execution: " + err.Error())
	}

	/* If step 5 is successful then we invalidate the cache so that we do not
	have stale balances for both sender and receiver
	*/
	_ = s.cache.Del(ctx, "user:"+fromWalID)
	_ = s.cache.Del(ctx, "user:"+toWalID)

	return nil
}
