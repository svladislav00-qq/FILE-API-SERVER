package integration

import (
	"context"
	"file-api-saver/internal/repository"
	"file-api-saver/internal/service"
	"file-api-saver/internal/utils"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gorm.io/gorm"
)

type testEnvironment struct {
	DB      *gorm.DB
	Minio   *minio.Client
	Bucket  string
	Service *service.FileService
}

func setupTestEnvironment(t *testing.T) *testEnvironment {
	t.Helper()

	if err := godotenv.Load("../../.env.test"); err != nil {
		t.Fatalf("failed to load .env.test: %v", err)
	}

	dsn := os.Getenv("DATABASE_URL_TEST")
	if dsn == "" {
		t.Fatal("DATABASE_URL_TEST is not set")
	}

	db, err := utils.ConnectDB(dsn)
	if err != nil {
		t.Fatalf("failed to connect db: %v", err)
	}

	sqlDb, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get underlying sql.DB: %v", err)
	}

	t.Cleanup(func() {
		if err := sqlDb.Close(); err != nil {
			t.Errorf("failed to close database: %v", err)
		}
	})

	if err = utils.Migrate(db); err != nil {
		t.Fatalf("failed to create migrations: %v", err)
	}

	minioClient, bucket := setupMinio(t)

	storage := &repository.MinioStorage{
		Client: minioClient,
		Bucket: bucket,
	}

	repo := &repository.FileRepository{
		DB: db,
	}

	fileService := &service.FileService{
		Storage: storage,
		Repo:    repo,
	}

	return &testEnvironment{
		DB:      db,
		Minio:   minioClient,
		Bucket:  bucket,
		Service: fileService,
	}
}

func setupMinio(t *testing.T) (*minio.Client, string) {
	endpoint := os.Getenv("ENDPOINT")
	if endpoint == "" {
		t.Fatal("failed to set endpoint")
	}

	accessKey := os.Getenv("ACCESS_KEY_ID")
	if accessKey == "" {
		t.Fatal("failed to set access key")
	}

	secretKey := os.Getenv("SECRET_ACCESS_KEY_ID")
	if secretKey == "" {
		t.Fatal("failed to set secret key")
	}

	bucket := os.Getenv("BUCKET")
	if bucket == "" {
		t.Fatal("failed to set bucket")
	}

	useSSL, err := strconv.ParseBool(os.Getenv("USE_SSL"))
	if err != nil {
		t.Fatalf("invalid USE_SSL value: %v", err)
	}

	minioOpt := &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	}

	minioClient, err := minio.New(endpoint, minioOpt)
	if err != nil {
		t.Fatalf("failed to create minio client %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exists, err := minioClient.BucketExists(ctx, bucket)
	if err != nil {
		t.Fatalf("failed to check bucket: %v", err)
	}

	bucketOpts := minio.MakeBucketOptions{
		Region: "us-east-1",
	}

	if !exists {
		err = minioClient.MakeBucket(ctx, bucket, bucketOpts)
		if err != nil {
			t.Fatalf("failed to create bucket: %v", err)
		}

	}

	return minioClient, bucket
}
