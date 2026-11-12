package integration

import (
	"bytes"
	"context"
	"file-api-saver/internal/models"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

func TestSetupTestEnvironment(t *testing.T) {
	env := setupTestEnvironment(t)

	if env.DB == nil {
		t.Fatal("database can't be nil")
	}

	if env.Minio == nil {
		t.Fatal("minio can't be nil")
	}

	if env.Service == nil {
		t.Fatal("service can't be nil")
	}

	sqlDb, err := env.DB.DB()
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = sqlDb.PingContext(ctx); err != nil {
		t.Fatalf("failed to ping db: %v", err)
	}

	exists, err := env.Minio.BucketExists(ctx, env.Bucket)
	if err != nil {
		t.Fatalf("failed to check Minio bucket: %v", err)
	}

	if !exists {
		t.Fatalf("bucket %q does not exists", env.Bucket)
	}
}

func TestUploadFile(t *testing.T) {
	env := setupTestEnvironment(t)

	testUser := &models.User{
		Email: uuid.NewString() + "testUser@gmail.test",
		Name:  "Tester2000",
		Role:  "user",
	}

	var meta *models.FileMeta

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()

		if meta != nil {

			if err := env.Minio.RemoveObject(
				cleanupCtx,
				env.Bucket,
				meta.ObjectName,
				minio.RemoveObjectOptions{},
			); err != nil {
				t.Errorf("failed to remove MinIO object: %v", err)
			}

			if err := env.DB.Unscoped().Delete(&models.FileMeta{}, meta.ID).Error; err != nil {
				t.Errorf("failed to delete file metadata: %v", err)
			}

		}
		if err := env.DB.Unscoped().Delete(&models.User{}, testUser.ID).Error; err != nil {
			t.Errorf("failed to delete test user: %v", err)
		}
	})

	if err := env.DB.Create(testUser).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	content := []byte("Hello, integration testing!")

	file, header := createMultipartFile(t, "test.txt", content)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	meta, err := env.Service.UploadFile(ctx, file, header, testUser.ID)
	if err != nil {
		t.Fatalf("want nil error, got %v", err)
	}

	if meta == nil {
		t.Fatal("expected meta, got nil")
	}

	if meta.OriginalName != header.Filename {
		t.Errorf("wrong original name: want %q, got %q", header.Filename, meta.OriginalName)
	}

	if meta.UserID != testUser.ID {
		t.Errorf("wrong user ID: want %d, got %d", testUser.ID, meta.UserID)
	}

	if meta.Bucket != env.Bucket {
		t.Errorf("wrong bucket: want %q, got %q", env.Bucket, meta.Bucket)
	}

	if meta.Size != len(content) {
		t.Errorf("wrong file size: want %d, got %d", len(content), meta.Size)
	}

	if meta.ObjectName == "" {
		t.Fatal("object name can't be empty")
	}

	var savedMeta models.FileMeta

	if err := env.DB.First(&savedMeta, meta.ID).Error; err != nil {
		t.Fatalf("file metadata not found in data base: %v", err)
	}

	if savedMeta.ObjectName != meta.ObjectName {
		t.Errorf(
			"wrong object name in database: want %q, got %q",
			meta.ObjectName,
			savedMeta.ObjectName,
		)
	}

	obj, err := env.Minio.GetObject(ctx, env.Bucket, meta.ObjectName, minio.GetObjectOptions{})
	if err != nil {
		t.Fatalf("failed to get object from MinIO: %v", err)
	}

	defer obj.Close()

	uploadedContent, err := io.ReadAll(obj)
	if err != nil {
		t.Fatalf("failed to read object content: %v", err)
	}

	if !bytes.Equal(content, uploadedContent) {
		t.Errorf("uploaded content doesn't match original content")
	}

}

func createMultipartFile(t *testing.T, filename string, content []byte) (multipart.File, *multipart.FileHeader) {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}

	if _, err := part.Write(content); err != nil {
		t.Fatalf("failed to write content: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	if err := req.ParseMultipartForm(10 << 20); err != nil {
		t.Fatalf("failed to parse multipart form: %v", err)
	}

	t.Cleanup(func() {
		if req.MultipartForm != nil {
			if err := req.MultipartForm.RemoveAll(); err != nil {
				t.Errorf("failed to remove multipart temp files: %v", err)
			}
		}
	})

	file, header, err := req.FormFile("file")
	if err != nil {
		t.Fatalf("failed to read multipart part: %v", err)
	}

	t.Cleanup(func() {
		if err := file.Close(); err != nil {
			t.Errorf("failed to close multipart file")
		}
	})

	return file, header
}
