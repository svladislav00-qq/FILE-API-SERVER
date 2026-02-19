package utils

import (
	"fmt"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func CreateCloud() {
	endpoint := "localhost:9000"
	accessKeyID := "minioadmin"
	secretAccessKeyID := "minioadmin"
	useSSL := false

	_, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKeyID, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Println("Connected to MinIO")
}
