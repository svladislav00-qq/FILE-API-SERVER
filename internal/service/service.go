package service

import (
	"context"
	"file-api-saver/internal/models"
	"file-api-saver/internal/repository"
	"fmt"
	"mime/multipart"

	"github.com/google/uuid"
)

type FileService struct {
	Storage *repository.MinioStorage
	Repo    *repository.FileRepository
}

func (s *FileService) UploadFile(ctx context.Context, file multipart.File, header *multipart.FileHeader) (*models.FileMeta, error) {
	objectName := fmt.Sprintf("%s-%s", uuid.New().String(), header.Filename)

	err := s.Storage.Upload(
		ctx,
		objectName,
		file,
		header.Size,
		header.Header.Get("Content-Type"),
	)
	if err != nil {
		return nil, err
	}

	meta := &models.FileMeta{
		ObjectName:   objectName,
		OriginalName: header.Filename,
		Bucket:       s.Storage.Bucket,
		Size:         int(header.Size),
	}

	err = s.Repo.Create(ctx, meta)
	if err != nil {
		_ = s.Storage.Delete(ctx, objectName)
		return nil, err
	}

	return meta, nil
}

func (s *FileService) DeleteFile(ctx context.Context, id int) error {
	// 1. Получить метаданные
	meta, err := s.Repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 2. Удалить из MinIO
	err = s.Storage.Delete(ctx, meta.ObjectName)
	if err != nil {
		return err
	}

	// 3. Удалить из бд
	err = s.Repo.Delete(ctx, id)
	if err != nil {
		return err
	}

	return nil
}

func (s *FileService) GetMeta(ctx context.Context) ([]models.FileMeta, error) {
	metas, err := s.Repo.GetAllMeta(ctx)
	if err != nil {
		return nil, err
	}

	return metas, err
}
