//go:build gcloud

package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/KasumiMercury/primind-notification-invoker/internal/observability"
	"github.com/KasumiMercury/primind-notification-invoker/internal/observability/logging"
)

func initObservability(ctx context.Context) (*observability.Resources, error) {
	serviceName := os.Getenv("K_SERVICE")
	if serviceName == "" {
		serviceName = "notification-invoker"
	}

	env := logging.EnvProd
	if e := os.Getenv("ENV"); e != "" {
		env = logging.Environment(e)
	}

	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		projectID = os.Getenv("GCLOUD_PROJECT_ID")
	}

	res, err := observability.Init(ctx, observability.Config{
		ServiceInfo: logging.ServiceInfo{
			Name:     serviceName,
			Version:  Version,
			Revision: os.Getenv("K_REVISION"),
		},
		Environment:   env,
		GCPProjectID:  projectID,
		SamplingRate:  1.0,
		DefaultModule: logging.Module("notification-invoker"),
	})
	if err != nil {
		return nil, err
	}

	slog.Info("observability initialized",
		slog.String("service", serviceName),
		slog.String("version", Version),
		slog.String("environment", string(env)),
	)

	return res, nil
}
