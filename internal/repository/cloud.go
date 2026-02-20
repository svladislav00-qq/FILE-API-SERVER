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

func (s *MinioStorage) GetObject(ctx context.Context, objectName string) (*minio.Object, error) {
	obj, err := s.Client.GetObject(ctx, s.Bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}

	return obj, err
}
