package utils

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var MinioClient *minio.Client
var MinioBucket string

func CreateCloud() {
	ctx := context.Background()

	endpoint := os.Getenv("ENDPOINT")
	accessKeyID := os.Getenv("ACCESS_KEY_ID")
	secretAccessKeyID := os.Getenv("SECRET_ACCESS_KEY_ID")
	useSSL, _ := strconv.ParseBool("USE_SSL")

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKeyID, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println("Connected to MinIO")

	MinioClient = client
	bucket := os.Getenv("BUCKET")
	MinioBucket = bucket
	location := os.Getenv("LOCATION")

	err = MinioClient.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: location})
	if err != nil {
		exists, errBucket := MinioClient.BucketExists(ctx, bucket)
		if errBucket == nil && exists {
			fmt.Println("Bucket already exists:", bucket)
		} else {
			log.Fatalln(err)
		}
	} else {
		fmt.Println("Created bucket:", bucket)
	}
}
