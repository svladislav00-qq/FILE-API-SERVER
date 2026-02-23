package main

import (
	"file-api-saver/internal/config"
	"file-api-saver/internal/models"
	database "file-api-saver/internal/utils"
)

func init() {
	database.GetDB()
	config.Load()
}

func main() {
	database.DB.AutoMigrate(&models.User{}, &models.FileMeta{})
}
