package handler

import (
	"encoding/json"
	"github.com/google/uuid"
	"net/http"
	"wallet-api/service"
)

// TransferRequest is what the mobile app sends us
type TransferRequest struct {
	FromWalletID string `json:"from_wallet_id"`
	ToWalletID   string `json:"to_wallet_id"`
	Amount       int    `json:"amount"`
	Reference    string `json:"reference"`
}

// Wallet is the data sends for creating a new wallet
type Wallet struct {
	UserID   string `json:"user_id"`
	Amount   int    `json:"amount"`
	Currency string `json:"currency"`
}

type WalletHandler struct {
	service *service.WalletService
}

func NewWalletHandler(service *service.WalletService) *WalletHandler {
	return &WalletHandler{service: service}
}

// UserWallet fetches user wallet
func (h *WalletHandler) UserWallet(w http.ResponseWriter, r *http.Request) {
	walletID := r.URL.Query().Get("wallet_id")
	if walletID == "" {
		http.Error(w, "error url parameter is incorrect", 400)
	}
	// 3. Call our Brain (The Service Layer)
	res, err := h.service.GetWalletBalance(r.Context(), walletID)
	if err != nil {
		// Failed to create a user wallet
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 4. Success!
	w.WriteHeader(http.StatusOK)
	// Convert the 'res' struct to JSON and send it
	if err := json.NewEncoder(w).Encode(res); err != nil {
		http.Error(w, "failed to format response", http.StatusInternalServerError)
	}
}

// Wallet creates a new user wallet
func (h *WalletHandler) Wallet(w http.ResponseWriter, r *http.Request) {
	// 1. Decode the JSON from the Mobile App
	var req Wallet
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// 3. Call our Brain (The Service Layer)
	err := h.service.CreateWallet(r.Context(), req.Currency, req.UserID, req.Amount)
	if err != nil {
		// Failed to create a user wallet
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 4. Success!
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Wallet creation was successful!"}`))
}

// HandleTransfer is the API endpoint
func (h *WalletHandler) HandleTransfer(w http.ResponseWriter, r *http.Request) {
	// 1. Decode the JSON from the Mobile App
	var req TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// 2. Convert the string IDs to actual UUIDs for our database
	fromID, err := uuid.Parse(req.FromWalletID)
	if err != nil {
		http.Error(w, "Invalid from_wallet_id format", http.StatusBadRequest)
		return
	}

	toID, err := uuid.Parse(req.ToWalletID)
	if err != nil {
		http.Error(w, "Invalid to_wallet_id format", http.StatusBadRequest)
		return
	}

	// 3. Call our Brain (The Service Layer)
	err = h.service.TransferFunds(r.Context(), fromID, toID, req.Amount, req.Reference)
	if err != nil {
		// If anything failed (Insufficient funds, Idempotency hit, DB crash)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// 4. Success!
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Transfer successful!"}`))
}
