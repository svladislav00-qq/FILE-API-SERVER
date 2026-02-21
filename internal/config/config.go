package config

import (
	"log/slog"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseUrl string
	Port        string
}

func Load() {
	var err error = godotenv.Load()
	if err != nil {
		slog.Error("Warning: env file not found, using environment variables")
	}
}
