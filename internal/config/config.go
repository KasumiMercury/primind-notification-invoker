package config

import (
	"os"
)

type Config struct {
	Port              string
	FirebaseProjectID string
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return &Config{
		Port:              port,
		FirebaseProjectID: os.Getenv("FIREBASE_PROJECT_ID"),
	}
}
