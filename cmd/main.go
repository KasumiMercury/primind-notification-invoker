package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KasumiMercury/primind-notification-invoker/internal/config"
	"github.com/KasumiMercury/primind-notification-invoker/internal/fcm"
	"github.com/KasumiMercury/primind-notification-invoker/internal/handler"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize slog with JSON handler
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))
	slog.SetDefault(logger)

	slog.Info("configuration loaded", slog.String("port", cfg.Port))

	ctx := context.Background()
	fcmClient, err := fcm.NewClient(ctx, cfg.FirebaseProjectID)
	if err != nil {
		slog.Error("failed to initialize FCM client", slog.String("error", err.Error()))
		os.Exit(1)
	}
	slog.Info("FCM client initialized")

	notificationHandler := handler.NewNotificationHandler(fcmClient)

	mux := http.NewServeMux()
	mux.HandleFunc("/notify", notificationHandler.SendNotification)
	mux.HandleFunc("/health", handler.Health)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("starting server", slog.String("port", cfg.Port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", slog.String("error", err.Error()))
		os.Exit(1)
	}

	slog.Info("server exited")
}
