package main

import (
	"file-api-saver/internal/config"
	"file-api-saver/internal/database"

	"github.com/gin-gonic/gin"
)

func init() {
	database.GetDB()
	config.Load()
}

func main() {
	r := gin.Default()

	r.Run()
}
