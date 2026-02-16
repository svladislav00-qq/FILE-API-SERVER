package main

import (
	"file-api-saver/internal/config"
	"file-api-saver/internal/database"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	var cfg *config.Config
	var err error
	cfg, err = config.Load()

	if err != nil {
		log.Fatal("Failed to load configureation: ", err)
	}

	var pool *pgxpool.Pool
	pool, err = database.Connect(cfg.DatabaseUrl)

	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	defer pool.Close()

	var router *gin.Engine = gin.Default()
	router.SetTrustedProxies(nil)
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message":  "File api saver is running well!",
			"status":   "success",
			"database": "connected",
		})
	})

	router.Run(":" + cfg.Port)
}
