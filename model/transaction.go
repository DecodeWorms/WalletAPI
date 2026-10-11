package model

import (
	"github.com/google/uuid"
	"time"
)

type Transaction struct {
	ID            uuid.UUID `json:"id" db:"id"`
	WalletID      uuid.UUID `json:"wallet_id" db:"wallet_id"`
	Amount        int       `json:"amount" db:"amount"`
	BalanceBefore int       `json:"balance_before" db:"balance_before"`
	BalanceNow    int       `json:"balance_now" db:"balance_now"`
	Type          string    `json:"type" db:"type"` // CREDIT or DEBIT
	Reference     string    `json:"reference" db:"reference"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	Status        string    `json:"status" db:"status"`
}
