package service_test

import (
	"bytes"
	"context"
	"errors"
	"file-api-saver/internal/models"
	"file-api-saver/internal/service"
	"file-api-saver/internal/service/mocks"
	"mime/multipart"
	"testing"

	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

func TestUploadFileHappyCase(t *testing.T) {
	existingUser := &models.User{
		Email: "test@gmail.com",
		Model: gorm.Model{
			ID: 1,
		},
		Name: "Alex",
	}

	testCases := []struct {
		name         string
		existingUser *models.User
		bucket       string
		wantErr      error
		wantBucket   string
		wantName     string
	}{
		{
			name:         "Success saving",
			existingUser: existingUser,
			bucket:       "test-bucket",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := mocks.NewFileRepository(t)
			storage := mocks.NewFileStorage(t)

			fileService := service.FileService{
				Repo:    repo,
				Storage: storage,
			}

			storage.On(
				"Upload",
				mock.Anything,
				mock.Anything,
				mock.Anything,
				mock.Anything,
				mock.Anything,
			).Return(nil).Once()

			storage.On("GetBucket").Return(tc.bucket).Once()

			repo.On("Create", mock.Anything, mock.MatchedBy(func(meta *models.FileMeta) bool {
				return meta.UserID == tc.existingUser.ID
			})).Return(nil).Once()

			file, header := createTestFile(t)

			got, err := fileService.UploadFile(context.Background(), file, header, tc.existingUser.ID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got == nil {
				t.Fatal("expected file metadata, got nil")
			}

			if got.UserID != tc.existingUser.ID {
				t.Fatalf("expected userID %d, got %d", tc.existingUser.ID, got.ID)
			}

			if got.OriginalName != header.Filename {
				t.Fatalf("expected file name %s, got %s", header.Filename, got.ObjectName)
			}

			if got.Bucket != "test-bucket" {
				t.Fatalf("expected bucket test-bucket, got %s", got.Bucket)
			}

			if got.Size != int(header.Size) {
				t.Fatalf("expected size %d, got %d", int(header.Size), got.Size)
			}

			if got.ObjectName == "" {
				t.Fatal("expected not empty object name")
			}

		})
	}
}

func TestUploadFileMinioErr(t *testing.T) {
	minioErr := errors.New("minio error")

	repo := mocks.NewFileRepository(t)
	storage := mocks.NewFileStorage(t)

	fileService := &service.FileService{
		Repo:    repo,
		Storage: storage,
	}

	storage.On(
		"Upload",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).Return(minioErr).Once()

	file, header := createTestFile(t)

	got, err := fileService.UploadFile(context.Background(), file, header, 1)

	if err == nil {
		t.Fatal("expected err, got nil")
	}
	if got != nil {
		t.Fatalf("expected nil, god %+v", got)
	}

	if !errors.Is(err, minioErr) {
		t.Fatalf("expected %v, got %v", minioErr, err)
	}

}

func TestUploadFilePostgresErr(t *testing.T) {
	dbErr := errors.New("postgres error")

	var uploadedObjectName string

	repo := mocks.NewFileRepository(t)
	storage := mocks.NewFileStorage(t)

	fileService := &service.FileService{
		Repo:    repo,
		Storage: storage,
	}

	storage.On(
		"Upload",
		mock.Anything,
		mock.MatchedBy(func(objectName string) bool {
			uploadedObjectName = objectName
			return true
		}),
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).Return(nil).Once()

	storage.On("GetBucket").Return("test-bucket").Once()

	repo.On("Create", mock.Anything, mock.Anything).Return(dbErr).Once()

	storage.On("Delete", mock.Anything, mock.MatchedBy(func(objectName string) bool {
		return objectName == uploadedObjectName
	})).Return(nil).Once()

	file, header := createTestFile(t)

	got, err := fileService.UploadFile(context.Background(), file, header, 1)
	if uploadedObjectName == "" {
		t.Fatal("expected uploaded object name, got empty")
	}

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if got != nil {
		t.Fatalf("expected nil file meta, got %+v", got)
	}

	if !errors.Is(err, dbErr) {
		t.Fatalf("expected %v, got %v", dbErr, err)
	}
}

func TestDeleteFileSuccessUser(t *testing.T) {
	repo := mocks.NewFileRepository(t)
	storage := mocks.NewFileStorage(t)

	existingFile := &models.FileMeta{
		ObjectName:   "uuid-test.txt",
		OriginalName: "test.txt",
		Bucket:       "test-bucket",
		Size:         100,
		UserID:       1,
	}

	fileService := &service.FileService{
		Repo:    repo,
		Storage: storage,
	}

	var userID uint
	var fileID int

	repo.On("GetByIDByUser",
		mock.Anything,
		mock.MatchedBy(func(id int) bool {
			fileID = id
			return true
		}),
		mock.MatchedBy(func(id uint) bool {
			userID = id
			return true
		}),
	).Return(existingFile, nil).Once()

	storage.On(
		"Delete",
		mock.Anything,
		mock.MatchedBy(func(objectName string) bool {
			return objectName == existingFile.ObjectName
		}),
	).Return(nil).Once()

	repo.On(
		"DeleteByUser",
		mock.Anything,
		mock.MatchedBy(func(id int) bool {
			return fileID == id
		}),
		mock.MatchedBy(func(id uint) bool {
			return userID == id
		}),
	).Return(nil).Once()

	err := fileService.DeleteFile(context.Background(), 1, 1, "user")
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func createTestFile(t *testing.T) (multipart.File, *multipart.FileHeader) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "test.txt")
	if err != nil {
		t.Fatal(err)
	}

	_, err = part.Write([]byte("hello tests"))
	if err != nil {
		t.Fatal(err)
	}

	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	reader := multipart.NewReader(body, writer.Boundary())

	form, err := reader.ReadForm(1024 * 1024)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		_ = form.RemoveAll()
	})

	header := form.File["file"][0]

	file, err := header.Open()
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		_ = file.Close()
	})

	return file, header
}
