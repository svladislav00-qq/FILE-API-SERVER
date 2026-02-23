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
	db := r.DB.WithContext(ctx)

	if err := db.Create(f).Error; err != nil {
		return err
	}
	return db.Preload("User").First(f, f.ID).Error
}

func (r *FileRepository) Delete(ctx context.Context, id int) error {
	return r.DB.WithContext(ctx).Delete(&models.FileMeta{}, id).Error
}

func (r *FileRepository) DeleteByUser(ctx context.Context, id int, userID uint) error {
	return r.DB.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).Delete(&models.FileMeta{}).Error
}

func (r *FileRepository) GetByID(ctx context.Context, id int) (*models.FileMeta, error) {
	var meta models.FileMeta

	err := r.DB.WithContext(ctx).First(&meta, id).Error

	if err != nil {
		return nil, err
	}

	return &meta, nil
}

func (r *FileRepository) GetByIDByUser(ctx context.Context, id int, userID uint) (*models.FileMeta, error) {
	var meta models.FileMeta

	err := r.DB.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&meta).Error

	if err != nil {
		return nil, err
	}

	return &meta, nil
}

func (r *FileRepository) GetAll(ctx context.Context) ([]models.FileMeta, error) {
	var datas []models.FileMeta
	err := r.DB.WithContext(ctx).Unscoped().Preload("User").Find(&datas).Error
	return datas, err
}

func (r *FileRepository) GetAllByUser(ctx context.Context, userID uint) ([]models.FileMeta, error) {
	var datas []models.FileMeta
	err := r.DB.WithContext(ctx).Preload("User").Where("user_id = ?", userID).Find(&datas).Error
	if err != nil {
		return nil, err
	}
	return datas, nil
}
