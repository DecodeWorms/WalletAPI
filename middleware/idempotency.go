package middleware

import (
	"net/http"
	"time"
	"wallet-api/storage"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func ExtractIdempotencyKey(cache storage.CacheStorage) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			idKKey := r.Header.Get("X-Idempotency-key")
			if idKKey == "" {
				http.Error(w, "Missing X-Idempotency-ID header", http.StatusBadRequest)
				return
			}

			// Create idempotency key
			idemptKey := "idempotency_key:" + idKKey
			acquired, err := cache.SetNX(r.Context(), idemptKey, "lock", 24*time.Hour)
			if err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			if !acquired {
				http.Error(w, "Request conflicts error", http.StatusConflict)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
