package handler

import (
	"errors"
	"file-api-saver/internal/models"
	"file-api-saver/internal/service"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

type RegisterRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
	Name     string `json:"name" binding:"required"`
	Role     string `json:"role"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type UserHandler struct {
	Service *service.UserService
}

func (h *UserHandler) CreateUser(c *gin.Context) {
	var registerRequest RegisterRequest

	if err := c.BindJSON(&registerRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(registerRequest.Password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password must be at least 6 characters long"})
		return
	}

	user, err := h.Service.CreateUser(c.Request.Context(), &models.User{
		Email:    registerRequest.Email,
		Password: registerRequest.Password,
		Name:     registerRequest.Name,
		Role:     registerRequest.Role,
	})
	if err != nil {
		if errors.Is(err, service.ErrInvalidRole) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role"})
			return
		}
		if errors.Is(err, service.ErrMailExists) {
			c.JSON(http.StatusConflict, gin.H{"error": "email already exist"})
			return
		}

		slog.Error("failed to create user: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}

func (h *UserHandler) LoginUser(c *gin.Context) {
	var loginRequest LoginRequest
	if err := c.BindJSON(&loginRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, err := h.Service.LoginUser(c.Request.Context(), &models.User{
		Email:    loginRequest.Email,
		Password: loginRequest.Password,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, token)
}
