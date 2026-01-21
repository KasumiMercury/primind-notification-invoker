package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"connectrpc.com/grpchealth"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/KasumiMercury/primind-notification-invoker/internal/config"
	"github.com/KasumiMercury/primind-notification-invoker/internal/fcm"
	"github.com/KasumiMercury/primind-notification-invoker/internal/handler"
	"github.com/KasumiMercury/primind-notification-invoker/internal/health"
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

	fcmClient, err := fcm.NewClient(ctx, cfg.FirebaseProjectID, cfg.WebAppBaseURL)
	if err != nil {
		slog.Error("failed to initialize FCM client", slog.String("error", err.Error()))

		return err
	}

	slog.Info("FCM client initialized",
		slog.String("web_app_base_url", cfg.WebAppBaseURL),
	)

	notificationHandler := handler.NewNotificationHandler(fcmClient)

	// Health check setup
	healthChecker := health.NewChecker(fcmClient, Version)

	mux := http.NewServeMux()
	mux.HandleFunc("/notify", notificationHandler.SendNotification)
	mux.HandleFunc("/health/live", healthChecker.LiveHandler)
	mux.HandleFunc("/health/ready", healthChecker.ReadyHandler)
	mux.HandleFunc("/health", healthChecker.ReadyHandler)

	// gRPC Health Checking Protocol (grpc.health.v1.Health/Check)
	grpcHealthChecker := health.NewGRPCChecker(healthChecker)
	grpcHealthPath, grpcHealthHandler := grpchealth.NewHandler(grpcHealthChecker)

	// Wrap with observability middleware
	wrappedHandler := middleware.HTTP(mux, middleware.HTTPConfig{
		SkipPaths:  []string{"/health", "/health/live", "/health/ready", "/metrics"},
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

	// Create multiplexed handler for HTTP + gRPC health
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if strings.HasPrefix(req.URL.Path, grpcHealthPath) {
			grpcHealthHandler.ServeHTTP(w, req)
			return
		}
		wrappedHandler.ServeHTTP(w, req)
	})

	// Create HTTP/2 server for h2c (HTTP/2 Cleartext) support
	// This enables grpc_health_probe to work in Docker environments without TLS
	h2s := &http2.Server{}

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           h2c.NewHandler(finalHandler, h2s),
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
