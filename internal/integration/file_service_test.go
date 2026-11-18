package integration

import (
	"bytes"
	"context"
	"file-api-saver/internal/models"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestUploadFilePostgresError(t *testing.T) {
	env := setupTestEnvironment(t)

	testUser := &models.User{
		Email: uuid.NewString() + "testUser@gmail.test",
		Name:  "Tester2000",
		Role:  "user",
	}

	if err := env.DB.Create(testUser).Error; err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	if !env.DB.Migrator().HasTable(&models.FileMeta{}) {
		t.Fatal("file_metas table does not exist")
	}

	triggerSQL := `
		CREATE OR REPLACE FUNCTION reject_file_insert()
		RETURNS TRIGGER AS $$
		BEGIN
    		RAISE EXCEPTION 'simulated database error';
		END;
		$$ LANGUAGE plpgsql;

		CREATE TRIGGER reject_file_insert_trigger
		BEFORE INSERT ON file_meta
		FOR EACH ROW
		EXECUTE FUNCTION reject_file_insert();
		`

	if err := env.DB.Exec(triggerSQL).Error; err != nil {
		t.Fatalf("failed to create trigger: %v", err)
	}

	t.Cleanup(func() {
		if err := env.DB.Exec(
			"DROP TRIGGER IF EXISTS reject_file_insert_trigger ON file_meta",
		).Error; err != nil {
			t.Errorf("failed to drop trigger: %v", err)
		}

		if err := env.DB.Exec(
			"DROP FUNCTION IF EXISTS reject_file_insert()",
		).Error; err != nil {
			t.Errorf("failed to drop function: %v", err)
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	content := []byte("Hello, integration testing!")

	filename := uuid.NewString() + ".txt"

	file, header := createMultipartFile(t, filename, content)

	got, err := env.Service.UploadFile(ctx, file, header, testUser.ID)

	if got != nil {
		t.Fatalf("want empty meta, got %+v", got)
	}

	if err == nil {
		t.Fatalf("expected err, got nil")
	}

	found := false

	for object := range env.Minio.ListObjects(
		ctx,
		env.Bucket,
		minio.ListObjectsOptions{Recursive: true},
	) {
		if object.Err != nil {
			t.Fatalf("failed to list MinIO objects: %v", object.Err)
		}

		if strings.HasSuffix(object.Key, "-"+filename) {
			found = true
		}
	}

	if found {
		t.Fatal("object still exists in MinIO after database error")
	}

	var count int64

	err = env.DB.Model(&models.FileMeta{}).Where("object_name LIKE ?", "%-"+filename).Count(&count).Error
	if err != nil {
		t.Fatalf("failed to count file metadata: %v", err)
	}

	if count != 0 {
		t.Fatalf("expected 0 metadata records, got %d", count)
	}
}

func TestDeleteFile(t *testing.T) {
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

	err := env.DB.Create(testUser).Error
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	content := []byte("Hello, integration test")

	filename := uuid.NewString() + ".txt"

	file, header := createMultipartFile(t, filename, content)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	meta, err = env.Service.UploadFile(ctx, file, header, testUser.ID)
	if err != nil {
		t.Fatalf("failed to upload file: %v", err)
	}

	if meta == nil {
		t.Fatal("meta file can't be empty")
	}

	var savedMeta models.FileMeta

	if err := env.DB.First(&savedMeta, meta.ID).Error; err != nil {
		t.Fatalf("file meta was not saved: %v", err)
	}

	err = env.Service.DeleteFile(ctx, int(meta.ID), testUser.ID, testUser.Role)
	if err != nil {
		t.Fatalf("failed to delete file: %v", err)
	}

	found := false

	for object := range env.Minio.ListObjects(
		ctx,
		env.Bucket,
		minio.ListObjectsOptions{Recursive: true},
	) {
		if object.Err != nil {
			t.Fatalf("failed to list MinIO objects: %v", object.Err)
		}

		if strings.HasSuffix(object.Key, "-"+filename) {
			found = true
		}
	}

	if found {
		t.Fatal("object still exists in MinIO after deletion")
	}

	var count int64

	if err := env.DB.Model(&models.FileMeta{}).Where("id = ?", meta.ID).Count(&count).Error; err != nil {
		t.Fatalf("failed to check file metadata: %v", err)
	}

	if count != 0 {
		t.Fatalf("expected 0 metadata records, got %d", count)
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
