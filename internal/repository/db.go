package repository

import (
	"context"
	"file-api-saver/internal/models"

	"gorm.io/gorm"
)

type FileRepository struct {
	DB *gorm.DB
}

func (r *FileRepository) Create(ctx context.Context, f *models.FileMeta) error {
	return r.DB.WithContext(ctx).Create(f).Error
}

func (r *FileRepository) Delete(ctx context.Context, id int) error {
	return r.DB.WithContext(ctx).Delete(&models.FileMeta{}, id).Error
}

func (r *FileRepository) GetByID(ctx context.Context, id int) (*models.FileMeta, error) {
	var meta models.FileMeta

	err := r.DB.WithContext(ctx).First(&meta, id).Error

	if err != nil {
		return nil, err
	}

	return &meta, nil
}

func (r *FileRepository) GetAllMeta(ctx context.Context) ([]models.FileMeta, error) {
	var metas []models.FileMeta
	if err := r.DB.WithContext(ctx).Find(&metas).Error; err != nil {
		return nil, err
	}
	return metas, nil
}
