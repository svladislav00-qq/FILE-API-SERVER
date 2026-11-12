package utils

import (
	"file-api-saver/internal/models"

	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&models.User{},
		&models.FileMeta{},
	)
}
