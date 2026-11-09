package service

import (
	"context"
	"errors"
	"file-api-saver/internal/models"
	"time"

	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	Repo      UserRepository
	JWTSecret string
}

// UserRepository contains exactly the database operations needed by UserService.
// Keeping this interface next to its consumer makes the service easy to test
// without PostgreSQL and keeps the production repository compatible by default.
//
//go:generate mockery --name=UserRepository --output=./mocks --outpkg=mocks
type UserRepository interface {
	GetByEmail(context.Context, string) (*models.User, error)
	CreateUser(context.Context, *models.User) error
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
		return "", ErrInvalidCredentials
	}

	err = bcrypt.CompareHashAndPassword([]byte(userData.Password), []byte(user.Password))
	if err != nil {
		return "", ErrInvalidCredentials
	}

	claims := jwt.MapClaims{
		"user_id": userData.ID,
		"email":   userData.Email,
		"role":    userData.Role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(u.JWTSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
