package service

import (
	"context"
	"errors"
	"file-api-saver/internal/models"
	"file-api-saver/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	Repo *repository.UserRepository
}

var (
	ErrInvalidRole = errors.New("invalid role")
	ErrMailExists  = errors.New("email is already exists")
)

func (u *UserService) CreateUser(ctx context.Context, user *models.User) (*models.User, error) {
	existing, err := u.Repo.GetByEmail(ctx, user.Email)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		return nil, ErrMailExists
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	userRole := user.Role
	switch userRole {
	case "":
		userRole = "user"
	case "admin":
		userRole = "admin"
	case "user":
		userRole = "user"
	default:
		return nil, ErrInvalidRole
	}

	userData := &models.User{
		Email:    user.Email,
		Password: string(hashedPassword),
		Name:     user.Name,
		Role:     userRole,
	}

	err = u.Repo.CreateUser(ctx, userData)
	if err != nil {
		return nil, err
	}

	return userData, nil
}
