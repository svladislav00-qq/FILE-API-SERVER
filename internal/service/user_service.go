package service

import (
	"context"
	"errors"
	"file-api-saver/internal/models"
	"file-api-saver/internal/repository"
	"time"

	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	Repo      *repository.UserRepository
	JWTSecret string
}

var (
	ErrInvalidRole        = errors.New("invalid role")
	ErrMailExists         = errors.New("email is already exists")
	ErrInvalidCredentials = errors.New("Invalid credentials")
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

func (u *UserService) LoginUser(ctx context.Context, user *models.User) (string, error) {
	userData, err := u.Repo.GetByEmail(ctx, user.Email)
	if err != nil {
		return "", err
	}

	if userData == nil {
		return "", err
	}

	err = bcrypt.CompareHashAndPassword([]byte(userData.Password), []byte(user.Password))
	if err != nil {
		return "", err
	}

	claims := jwt.MapClaims{
		"user_id": userData.ID,
		"email":   userData.Email,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(u.JWTSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
