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

	"github.com/minio/minio-go/v7"
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
			fileService, repo, storage := createFileServiceWithStorageAndRepo(t)

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

	storage := mocks.NewFileStorage(t)

	fileService := &service.FileService{
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

	fileService, repo, storage := createFileServiceWithStorageAndRepo(t)

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

func TestDeleteFileSuccess(t *testing.T) {
	existingFile := &models.FileMeta{
		ObjectName:   "uuid-test.txt",
		OriginalName: "test.txt",
		Bucket:       "test-bucket",
		Size:         100,
		UserID:       1,
	}

	testCases := []struct {
		name         string
		role         string
		existingFile *models.FileMeta
	}{
		{
			name:         "Delete test for admin",
			role:         "admin",
			existingFile: existingFile,
		},
		{
			name:         "Delete test for user",
			role:         "user",
			existingFile: existingFile,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fileService, repo, storage := createFileServiceWithStorageAndRepo(t)

			var userID uint
			var fileID int

			if tc.role == "user" {

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
				).Return(tc.existingFile, nil).Once()
			} else {

				repo.On("GetByID",
					mock.Anything,
					mock.MatchedBy(func(id int) bool {
						fileID = id
						return true
					}),
				).Return(tc.existingFile, nil).Once()
			}

			storage.On(
				"Delete",
				mock.Anything,
				mock.MatchedBy(func(objectName string) bool {
					return objectName == tc.existingFile.ObjectName
				}),
			).Return(nil).Once()

			if tc.role == "user" {
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
			} else {
				repo.On(
					"Delete",
					mock.Anything,
					mock.MatchedBy(func(id int) bool {
						return fileID == id
					}),
				).Return(nil).Once()
			}
			err := fileService.DeleteFile(context.Background(), 1, 1, tc.role)
			if err != nil {
				t.Fatalf("expected nil, got %v", err)
			}
		})
	}
}

func TestDeleteFileGetUserError(t *testing.T) {
	dbErr := errors.New("data base error")

	testCases := []struct {
		name string
		role string
	}{
		{
			name: "Error in admin realization",
			role: "admin",
		},
		{
			name: "Error in user realization",
			role: "user",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fileService, repo := createFileServiceOnlyRepo(t)

			if tc.role == "admin" {
				repo.On(
					"GetByID",
					mock.Anything,
					mock.Anything,
				).Return(nil, dbErr).Once()
			} else {
				repo.On(
					"GetByIDByUser",
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return(nil, dbErr).Once()
			}

			err := fileService.DeleteFile(context.Background(), 1, 1, tc.role)
			if !errors.Is(err, dbErr) {
				t.Fatalf("want %v, got %v", dbErr, err)
			}
		})
	}
}

func TestDeleteFileMinioError(t *testing.T) {
	storageErr := errors.New("minio error")

	existingFile := &models.FileMeta{
		ObjectName:   "uuid-test.txt",
		OriginalName: "test.txt",
		Bucket:       "test-bucket",
		Size:         100,
		UserID:       1,
	}

	testCases := []struct {
		name         string
		role         string
		existingFile *models.FileMeta
	}{
		{
			name:         "Error in admin realization",
			role:         "admin",
			existingFile: existingFile,
		},
		{
			name:         "Error in user realization",
			role:         "user",
			existingFile: existingFile,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fileService, repo, storage := createFileServiceWithStorageAndRepo(t)

			if tc.role == "admin" {
				repo.On(
					"GetByID",
					mock.Anything,
					mock.Anything,
				).Return(tc.existingFile, nil).Once()
			} else {
				repo.On(
					"GetByIDByUser",
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return(tc.existingFile, nil).Once()
			}

			storage.On(
				"Delete",
				mock.Anything,
				mock.Anything,
			).Return(storageErr).Once()

			err := fileService.DeleteFile(context.Background(), 1, 1, tc.role)
			if err == nil {
				t.Fatal("expected err, got nil")
			}

			if !errors.Is(err, storageErr) {
				t.Fatalf("got %v, want %v", err, storageErr)
			}
		})
	}
}

func TestDeleteFileNilMetaError(t *testing.T) {
	testCases := []struct {
		name string
		role string
	}{
		{
			name: "Error in admin realization",
			role: "admin",
		},
		{
			name: "Error in user realization",
			role: "user",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fileService, repo := createFileServiceOnlyRepo(t)

			if tc.role == "admin" {
				repo.On(
					"GetByID",
					mock.Anything,
					mock.Anything,
				).Return(nil, nil).Once()
			} else {
				repo.On(
					"GetByIDByUser",
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return(nil, nil).Once()
			}

			err := fileService.DeleteFile(context.Background(), 1, 1, tc.role)
			if err == nil {
				t.Fatal("want err, got nil")
			}

			if !errors.Is(err, service.ErrFileNotFound) {
				t.Fatalf("want %v, got %v", service.ErrFileNotFound, err)
			}
		})
	}
}

func TestDeleteFilePostgresError(t *testing.T) {
	postgresErr := errors.New("postgres error")

	existingFile := &models.FileMeta{
		ObjectName:   "uuid-test.txt",
		OriginalName: "test.txt",
		Bucket:       "test-bucket",
		Size:         100,
		UserID:       1,
	}

	testCases := []struct {
		name         string
		role         string
		existingFile *models.FileMeta
	}{
		{
			name:         "Error in admin realization",
			role:         "admin",
			existingFile: existingFile,
		},
		{
			name:         "Error in user realization",
			role:         "user",
			existingFile: existingFile,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fileService, repo, storage := createFileServiceWithStorageAndRepo(t)

			if tc.role == "admin" {
				repo.On(
					"GetByID",
					mock.Anything,
					mock.Anything,
				).Return(tc.existingFile, nil).Once()
			} else {
				repo.On(
					"GetByIDByUser",
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return(tc.existingFile, nil).Once()
			}

			storage.On(
				"Delete",
				mock.Anything,
				mock.Anything,
			).Return(nil).Once()

			if tc.role == "user" {
				repo.On(
					"DeleteByUser",
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return(postgresErr).Once()
			} else {
				repo.On(
					"Delete",
					mock.Anything,
					mock.Anything,
				).Return(postgresErr).Once()
			}

			err := fileService.DeleteFile(context.Background(), 1, 1, tc.role)
			if err == nil {
				t.Fatal("want err, got nil")
			}

			if !errors.Is(err, postgresErr) {
				t.Fatalf("want %v, got %v", postgresErr, err)
			}

		})
	}
}

func TestGetAllDataSuccess(t *testing.T) {
	existingData := make([]models.FileMeta, 0)

	testCases := []struct {
		name         string
		role         string
		existingData []models.FileMeta
	}{
		{
			name:         "Success for admin",
			role:         "admin",
			existingData: existingData,
		},
		{
			name:         "Success for user",
			role:         "user",
			existingData: existingData,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fileService, repo := createFileServiceOnlyRepo(t)

			if tc.role == "admin" {

				repo.On(
					"GetAll",
					mock.Anything,
				).Return(tc.existingData, nil).Once()
			} else {
				repo.On(
					"GetAllByUser",
					mock.Anything,
					mock.Anything,
				).Return(tc.existingData, nil).Once()
			}

			got, err := fileService.GetAllData(context.Background(), 1, tc.role)
			if got == nil {
				t.Fatal("meta can't be empty")
			}

			if err != nil {
				t.Fatalf("want nil error, got %v", err)
			}
		})
	}
}

func TestGetAllDataError(t *testing.T) {
	dbErr := errors.New("data base error")

	testCases := []struct {
		name string
		role string
	}{
		{
			name: "Success for admin",
			role: "admin",
		},
		{
			name: "Success for user",
			role: "user",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := mocks.NewFileRepository(t)

			fileService := &service.FileService{
				Repo: repo,
			}

			if tc.role == "admin" {

				repo.On(
					"GetAll",
					mock.Anything,
				).Return(nil, dbErr).Once()
			} else {
				repo.On(
					"GetAllByUser",
					mock.Anything,
					mock.Anything,
				).Return(nil, dbErr).Once()
			}

			got, err := fileService.GetAllData(context.Background(), 1, tc.role)
			if got != nil {
				t.Fatalf("want nil meta, got %+v", got)
			}

			if !errors.Is(err, dbErr) {
				t.Fatalf("want %v, got %v", dbErr, err)
			}
		})
	}
}

func TestGetObjectSuccess(t *testing.T) {
	existingFile := &models.FileMeta{
		ObjectName:   "uuid-test.txt",
		OriginalName: "test.txt",
		Bucket:       "test-bucket",
		Size:         100,
		UserID:       1,
	}

	testObj := new(minio.Object)

	testCases := []struct {
		name         string
		role         string
		existingFile *models.FileMeta
		testObj      *minio.Object
	}{
		{
			name:         "Success getting for admin",
			role:         "admin",
			existingFile: existingFile,
			testObj:      testObj,
		},
		{
			name:         "Success getting for user",
			role:         "user",
			existingFile: existingFile,
			testObj:      testObj,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fileService, repo, storage := createFileServiceWithStorageAndRepo(t)

			if tc.role == "admin" {
				repo.On(
					"GetByID",
					mock.Anything,
					mock.Anything,
				).Return(tc.existingFile, nil).Once()
			} else {
				repo.On(
					"GetByIDByUser",
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return(tc.existingFile, nil).Once()
			}

			storage.On(
				"GetObject",
				mock.Anything,
				tc.existingFile.ObjectName,
			).Return(tc.testObj, nil).Once()

			meta, obj, err := fileService.GetObject(context.Background(), 1, 1, tc.role)
			if meta == nil {
				t.Fatal("meta data can't be empty")
			}

			if obj != tc.testObj {
				t.Fatal("returned object is not the expected object")
			}

			if err != nil {
				t.Fatalf("want nil error, got %v", err)
			}
		})
	}
}

func TestGetObjectMetaError(t *testing.T) {
	dbErr := errors.New("data base error")

	testCases := []struct {
		name string
		role string
	}{
		{
			name: "Error for getting meta for admin",
			role: "admin",
		},
		{
			name: "Error for getting meta for user",
			role: "user",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := mocks.NewFileRepository(t)

			fileService := &service.FileService{
				Repo: repo,
			}

			if tc.role == "admin" {
				repo.On(
					"GetByID",
					mock.Anything,
					mock.Anything,
				).Return(nil, dbErr).Once()
			} else {
				repo.On(
					"GetByIDByUser",
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return(nil, dbErr).Once()
			}

			meta, obj, err := fileService.GetObject(context.Background(), 1, 1, tc.role)
			if !errors.Is(err, dbErr) {
				t.Fatalf("want %v, got %v", dbErr, err)
			}

			if meta != nil {
				t.Fatalf("want nil meta, got %+v", meta)
			}

			if obj != nil {
				t.Fatalf("want nil obj, got %+v", obj)
			}
		})
	}
}

func TestGetObjectMetaNil(t *testing.T) {
	testCases := []struct {
		name string
		role string
	}{
		{
			name: "Error for getting meta for admin",
			role: "admin",
		},
		{
			name: "Error for getting meta for user",
			role: "user",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fileService, repo := createFileServiceOnlyRepo(t)

			if tc.role == "admin" {
				repo.On(
					"GetByID",
					mock.Anything,
					mock.Anything,
				).Return(nil, nil).Once()
			} else {
				repo.On(
					"GetByIDByUser",
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return(nil, nil).Once()
			}

			meta, obj, err := fileService.GetObject(context.Background(), 1, 1, tc.role)
			if !errors.Is(err, service.ErrFileNotFound) {
				t.Fatalf("want %v, got %v", service.ErrFileNotFound, err)
			}

			if meta != nil {
				t.Fatalf("want nil meta, got %+v", meta)
			}

			if obj != nil {
				t.Fatalf("want nil obj, got %+v", obj)
			}
		})
	}
}

func TestGetObjectMinioError(t *testing.T) {
	minioErr := errors.New("minio error")

	existingFile := &models.FileMeta{
		ObjectName:   "uuid-test.txt",
		OriginalName: "test.txt",
		Bucket:       "test-bucket",
		Size:         100,
		UserID:       1,
	}

	testCases := []struct {
		name         string
		role         string
		existingFile *models.FileMeta
	}{
		{
			name:         "Error for getting meta for admin",
			role:         "admin",
			existingFile: existingFile,
		},
		{
			name:         "Error for getting meta for user",
			role:         "user",
			existingFile: existingFile,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			fileService, repo, storage := createFileServiceWithStorageAndRepo(t)

			if tc.role == "admin" {
				repo.On(
					"GetByID",
					mock.Anything,
					mock.Anything,
				).Return(tc.existingFile, nil).Once()
			} else {
				repo.On(
					"GetByIDByUser",
					mock.Anything,
					mock.Anything,
					mock.Anything,
				).Return(tc.existingFile, nil).Once()
			}

			storage.On(
				"GetObject",
				mock.Anything,
				tc.existingFile.ObjectName,
			).Return(nil, minioErr).Once()

			meta, obj, err := fileService.GetObject(context.Background(), 1, 1, tc.role)
			if !errors.Is(err, minioErr) {
				t.Fatalf("want %v, got %v", minioErr, err)
			}

			if meta != nil {
				t.Fatalf("want nil meta, got %+v", meta)
			}

			if obj != nil {
				t.Fatalf("want nil object, got %+v", obj)
			}
		})
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

func createFileServiceWithStorageAndRepo(t *testing.T) (*service.FileService, *mocks.FileRepository, *mocks.FileStorage) {
	t.Helper()

	repo := mocks.NewFileRepository(t)
	storage := mocks.NewFileStorage(t)

	fileService := &service.FileService{
		Repo:    repo,
		Storage: storage,
	}

	return fileService, repo, storage
}

func createFileServiceOnlyRepo(t *testing.T) (*service.FileService, *mocks.FileRepository) {
	t.Helper()

	repo := mocks.NewFileRepository(t)

	fileService := &service.FileService{
		Repo: repo,
	}

	return fileService, repo
}
