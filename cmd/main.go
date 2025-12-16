package main

import (
	"context"
	"log"
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
	cfg := config.Load()

	ctx := context.Background()
	fcmClient, err := fcm.NewClient(ctx, cfg.FirebaseProjectID)
	if err != nil {
		log.Fatalf("Failed to initialize FCM client: %v", err)
	}

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
		log.Printf("Starting server on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
