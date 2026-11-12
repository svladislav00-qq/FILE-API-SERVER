package main

import (
	"file-api-saver/internal/config"
	"file-api-saver/internal/utils"
	database "file-api-saver/internal/utils"
)

func init() {
	config.Load()
	database.GetDB()
}

func main() {
	if err := utils.Migrate(database.DB); err != nil {
		panic(err)
	}
}
