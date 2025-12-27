package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KasumiMercury/primind-notification-invoker/internal/config"
	"github.com/KasumiMercury/primind-notification-invoker/internal/fcm"
	"github.com/KasumiMercury/primind-notification-invoker/internal/handler"
	"github.com/KasumiMercury/primind-notification-invoker/internal/observability/logging"
	"github.com/KasumiMercury/primind-notification-invoker/internal/observability/metrics"
	"github.com/KasumiMercury/primind-notification-invoker/internal/observability/middleware"
)

// Version is set via ldflags at build time
var Version = "dev"

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	obs, err := initObservability(ctx)
	if err != nil {
		slog.Error("failed to initialize observability", slog.String("error", err.Error()))

		return err
	}

	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		if err := obs.Shutdown(shutdownCtx); err != nil {
			slog.Warn("observability shutdown error", slog.String("error", err.Error()))
		}
	}()

	slog.SetDefault(obs.Logger())

	cfg := config.Load()

	slog.Info("configuration loaded", slog.String("port", cfg.Port))

	httpMetrics, err := metrics.NewHTTPMetrics()
	if err != nil {
		slog.Error("failed to initialize HTTP metrics", slog.String("error", err.Error()))

		return err
	}

	fcmClient, err := fcm.NewClient(ctx, cfg.FirebaseProjectID)
	if err != nil {
		slog.Error("failed to initialize FCM client", slog.String("error", err.Error()))

		return err
	}

	slog.Info("FCM client initialized")

	notificationHandler := handler.NewNotificationHandler(fcmClient)

	mux := http.NewServeMux()
	mux.HandleFunc("/notify", notificationHandler.SendNotification)
	mux.HandleFunc("/health", handler.Health)

	// Wrap with observability middleware
	wrappedHandler := middleware.HTTP(mux, middleware.HTTPConfig{
		SkipPaths:  []string{"/health", "/metrics"},
		Module:     logging.Module("notification-invoker"),
		Worker:     true,
		TracerName: "github.com/KasumiMercury/primind-notification-invoker/internal/observability/middleware",
		JobNameResolver: func(r *http.Request) string {
			if messageType := r.Header.Get("message_type"); messageType != "" {
				return messageType
			}
			if eventType := r.Header.Get("event_type"); eventType != "" {
				return eventType
			}

			return r.URL.Path
		},
		HTTPMetrics: httpMetrics,
	})
	wrappedHandler = middleware.PanicRecoveryHTTP(wrappedHandler)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           wrappedHandler,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	slog.InfoContext(ctx, "starting server",
		slog.String("event", "server.start"),
		slog.String("port", cfg.Port),
		slog.String("version", Version),
	)

	go func() {
		<-ctx.Done()

		slog.InfoContext(ctx, "shutdown signal received",
			slog.String("event", "server.shutdown.start"),
		)

		shutdownCtx, shutdownCancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("failed to shutdown server", slog.String("error", err.Error()))
		}
	}()

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.ErrorContext(ctx, "server exited with error",
			slog.String("event", "server.exit.fail"),
			slog.String("error", err.Error()),
		)

		return err
	}

	slog.InfoContext(ctx, "server stopped",
		slog.String("event", "server.stop"),
	)

	return nil
}
