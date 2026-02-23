package main

import (
	"file-api-saver/internal/config"
	"file-api-saver/internal/handler"
	"file-api-saver/internal/middleware"
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

	userService := &service.UserService{
		Repo:      &repository.UserRepository{DB: utils.DB},
		JWTSecret: config.AppConfig.JWTSecret,
	}

	userHandler := &handler.UserHandler{
		Service: userService,
	}

	r.POST("/auth/register", userHandler.CreateUser)
	r.POST("/auth/login", userHandler.LoginUser)

	protected := r.Group("/file")
	protected.Use(middleware.AuthMiddleware(&config.AppConfig))
	{
		protected.POST("", fileHandler.UploadFile)
		protected.DELETE("/:id", fileHandler.DeleteFile)
		protected.GET("/allfiles", fileHandler.GetFileData)
		protected.GET("/:id", fileHandler.GetObject)
		protected.GET("/:id/download", fileHandler.DownloadObject)
	}

	admin := r.Group("/admin/file", middleware.AuthMiddleware(&config.AppConfig), middleware.RoleMiddleware("admin"))
	{
		admin.GET("/allfiles", fileHandler.GetFileData)
		admin.DELETE("/:id", fileHandler.DeleteFile)
		admin.GET("/:id", fileHandler.GetObject)
	}

	r.Run()
}
