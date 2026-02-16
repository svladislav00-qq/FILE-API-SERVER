package main

import (
	"file-api-saver/internal/config"
	"log/slog"
	"os"
)

func main() {
	var cfg *config.Config
	var err error
	cfg, err = config.Load()

	if err != nil {
		slog.Error("Failed to load configureation: ", err)
		os.Exit(1)
	}
}
