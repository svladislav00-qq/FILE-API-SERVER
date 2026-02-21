package handler

import (
	"file-api-saver/internal/service"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type FileHandler struct {
	Service *service.FileService
}

func (h *FileHandler) UploadFile(c *gin.Context) {

	f, fileHandler, err := c.Request.FormFile("file")
	if err != nil {
		slog.Error("Failed to get file from form: " + err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "file not provided"})
		return
	}
	defer f.Close()

	meta, err := h.Service.UploadFile(c.Request.Context(), f, fileHandler)
	if err != nil {
		slog.Error("upload failed: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "upload failed"})
		return
	}

	c.JSON(http.StatusCreated, meta)
}

func (h *FileHandler) DeleteFile(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing id parameter"})
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id parameter"})
		return
	}

	err = h.Service.DeleteFile(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *FileHandler) GetMeta(c *gin.Context) {
	metas, err := h.Service.GetMeta(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, metas)
}

func (h *FileHandler) GetObject(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing id parameter"})
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id parameter"})
		return
	}

	meta, obj, err := h.Service.GetObject(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	c.Header("Content-Disposition", "inline; filename="+meta.OriginalName)

	stat, _ := obj.Stat()

	c.DataFromReader(
		http.StatusOK,
		int64(meta.Size),
		stat.ContentType,
		obj,
		nil,
	)
}

func (h *FileHandler) DownloadObject(c *gin.Context) {
	idStr := c.Param("id")
	if idStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing id parametr"})
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id parametr"})
		return
	}

	meta, obj, err := h.Service.GetObject(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, meta.ObjectName))

	stat, _ := obj.Stat()

	c.DataFromReader(
		http.StatusOK,
		int64(meta.Size),
		stat.ContentType,
		obj,
		nil,
	)
}
