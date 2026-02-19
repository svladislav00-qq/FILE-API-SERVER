package repository

import (
	"context"
	"io"

	"github.com/minio/minio-go/v7"
)

type MinioStorage struct {
	Client *minio.Client
	Bucket string
}

func (s *MinioStorage) Upload(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) error {
	_, err := s.Client.PutObject(
		ctx,
		s.Bucket,
		objectName,
		reader,
		size,
		minio.PutObjectOptions{
			ContentType: contentType,
		},
	)

	return err
}

func (s *MinioStorage) Delete(ctx context.Context, objectName string) error {
	return s.Client.RemoveObject(ctx, s.Bucket, objectName, minio.RemoveObjectOptions{})
}
