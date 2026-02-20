package main

import (
	"file-api-saver/internal/config"
	"file-api-saver/internal/handler"
	"file-api-saver/internal/repository"
	"file-api-saver/internal/service"
	"file-api-saver/internal/utils"

	"github.com/gin-gonic/gin"
)

func init() {
	config.Load()
	utils.GetDB()
	utils.CreateCloud()
}

func main() {

	r := gin.Default()
	r.SetTrustedProxies(nil)

	storage := &repository.MinioStorage{
		Client: utils.MinioClient,
		Bucket: utils.MinioBucket,
	}

	repo := &repository.FileRepository{
		DB: utils.DB,
	}

	fileService := &service.FileService{
		Storage: storage,
		Repo:    repo,
	}

	fileHandler := &handler.FileHandler{
		Service: fileService,
	}

	r.POST("/file", fileHandler.UploadFile)
	r.DELETE("/file/:id", fileHandler.DeleteFile)
	r.GET("/files", fileHandler.GetMeta)

	r.Run()
}
