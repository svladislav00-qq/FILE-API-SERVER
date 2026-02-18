package main

import (
	"file-api-saver/internal/config"
	"file-api-saver/internal/database"
	"file-api-saver/internal/models"
)

func init() {
	database.GetDB()
	config.Load()
}

func main() {
	database.DB.AutoMigrate(&models.FileMeta{})
}
