package middleware

import (
	"net/http"
	"time"
	"wallet-api/storage" // Update this if your module name is different
)

func RateLimiterMiddleware(cache storage.CacheStorage) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// For testing, let's pretend the User ID is passed in a header
			userID := r.Header.Get("X-User-ID")
			if userID == "" {
				http.Error(w, "Missing X-User-ID header", http.StatusUnauthorized)
				return
			}

			cacheKey := "rate_limit:transfer:" + userID

			// TODO 1: Increment the cacheKey using cache.Incr()
			reqCounter, _ := cache.Incr(r.Context(), cacheKey)

			// TODO 2: If the count is exactly 1, set the expiration to 60 seconds using cache.Expire()
			if reqCounter == 1 {
				_ = cache.Expire(r.Context(), cacheKey, 30*time.Second)
			}

			// TODO 3: If the count is greater than 5, return http.StatusTooManyRequests (429) and STOP (return)
			if reqCounter > 5 {
				http.Error(w, "too many request", http.StatusTooManyRequests)
				return
			}

			// If they pass all checks, allow the request to proceed!
			next.ServeHTTP(w, r)
		})
	}
}
