package main

import (
	"context"
	"github.com/redis/go-redis/v9"
	"github.com/rs/cors"
	"log"
	"net/http"
	"time"
	"wallet-api/config"
	"wallet-api/handler"
	"wallet-api/middleware"
	"wallet-api/service"
	"wallet-api/storage"
)

func main() {
	// Load the system configuration
	cfg := config.NewConfig()
	ctx := context.Background()
	dataStorage, err := storage.NewPostgresStore(ctx, cfg.DbUrl)
	if err != nil {
		log.Printf("error connecting to database %v", err)
	}
	// Connect to redis client
	red := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	CacheStore := storage.NewCacStorage(red)

	// Initialize the rate limiter so that it be able to intercept the requests
	rateLimiter := middleware.RateLimiterMiddleware(CacheStore)
	// Initialize the Idempotency key extraction
	extractIdempotency := middleware.ExtractIdempotencyKey(CacheStore)
	// Initialize the wallet service
	walletService := service.NewWalletService(dataStorage, CacheStore)
	// Initialize the wallet handler
	walletHandler := handler.NewWalletHandler(walletService)

	// A server that intercepts FE calls
	router := http.NewServeMux()

	// Chain both the rate limiter and idempotency middleware
	finalHandler := rateLimiter(extractIdempotency(http.HandlerFunc(walletHandler.HandleTransfer)))

	// List the API(s) and the handler
	router.Handle("POST /api/v1/transfer", finalHandler)
	router.HandleFunc("POST /api/v1/wallet", walletHandler.Wallet)
	router.HandleFunc("GET /api/v1/wallet", walletHandler.UserWallet)

	// Config CORS so that the browser allows FE application request
	c := cors.New(cors.Options{
		AllowedOrigins: []string{
			"localhost:8080",
			"localhost:8081",
		},
		AllowedMethods: []string{
			"GET", "POST", "PATCH", "PUT"},
		AllowCredentials: true,
		MaxAge:           30,
	})

	// Bind mux router to the CORS
	httpHandler := c.Handler(router)

	// Config server settings
	server := &http.Server{
		Addr:         ":8080",
		Handler:      httpHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	log.Printf("Server starting on http://localhost%s\n", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}

}
