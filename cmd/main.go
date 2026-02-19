package main

import (
	"file-api-saver/internal/config"
	"file-api-saver/internal/utils"

	"github.com/gin-gonic/gin"
)

func init() {
	config.Load()
	utils.GetDB()
	utils.CreateCloud()
}

func main() {
	r := gin.Default()

	r.Run()
}
