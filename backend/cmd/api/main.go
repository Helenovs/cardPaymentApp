package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"card-payment-app/backend/internal/httpapi"
	"card-payment-app/backend/internal/payments"
	"github.com/redis/go-redis/v9"
	"github.com/stripe/stripe-go/v85"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	secret := requireEnv(logger, "STRIPE_SECRET_KEY")
	publishable := requireEnv(logger, "STRIPE_PUBLISHABLE_KEY")
	stripe.Key = secret
	db, err := strconv.Atoi(env("REDIS_DB", "0"))
	if err != nil {
		logger.Error("invalid REDIS_DB", "error", err)
		os.Exit(1)
	}
	redisClient := redis.NewClient(&redis.Options{Addr: env("REDIS_ADDR", "localhost:6379"), Password: os.Getenv("REDIS_PASSWORD"), DB: db})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := payments.Ping(ctx, redisClient); err != nil {
		logger.Error("startup failed", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()
	api := httpapi.New(payments.NewRedisStore(redisClient), httpapi.Config{PublishableKey: publishable, WebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"), AllowedOrigin: env("APP_ORIGIN", "http://localhost:5173")}, logger)
	server := &http.Server{Addr: ":" + env("API_PORT", "8080"), Handler: api.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		logger.Info("API listening", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown failed", "error", err)
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
func requireEnv(logger *slog.Logger, key string) string {
	value := os.Getenv(key)
	if value == "" {
		logger.Error("required environment variable is missing", "name", key)
		os.Exit(1)
	}
	return value
}
