package config

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseUrl string
	Port        string
}

func Load() (*Config, error) {
	var err error = godotenv.Load()

	if err != nil {
		slog.Error("Warning: env file not found, using enviroment variables")
	}

	var config *Config = &Config{
		DatabaseUrl: os.Getenv("DATABASE_URL"),
		Port:        os.Getenv("PORT"),
	}

	return config, nil
}
