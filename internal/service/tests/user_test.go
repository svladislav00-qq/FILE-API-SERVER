package service_test

import (
	"context"
	"errors"
	"testing"

	"file-api-saver/internal/models"
	"file-api-saver/internal/service"
	"file-api-saver/internal/service/mocks"

	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

func TestCreateUser(t *testing.T) {
	dbErr := errors.New("database unavailable")

	testCases := []struct {
		name         string
		input        *models.User
		existingUser *models.User
		getErr       error
		createErr    error
		wantErr      error
		wantRole     string
	}{
		{
			name: "Success registration with empty role",
			input: &models.User{
				Email:    "test@gmail.com",
				Password: "password123",
				Name:     "Alex",
			},
			wantRole: "user",
		},
		{
			name: "Success registration with role user",
			input: &models.User{
				Email:    "test@gmail.com",
				Password: "password123",
				Name:     "Alex",
				Role:     "user",
			},
			wantRole: "user",
		},
		{
			name: "Success registration with role admin",
			input: &models.User{
				Email:    "test@gmail.com",
				Password: "password123",
				Name:     "Alex",
				Role:     "admin",
			},
			wantRole: "admin",
		},
		{
			name: "Invalid role",
			input: &models.User{
				Email:    "test@gmail.com",
				Password: "password123",
				Name:     "Alex",
				Role:     "anybody",
			},
			wantErr: service.ErrInvalidRole,
		},
		{
			name: "Email already exists",
			input: &models.User{
				Email:    "test@gmail.com",
				Password: "password123",
				Name:     "Alex",
			},
			existingUser: &models.User{
				Email: "test@gmail.com",
			},
			wantErr: service.ErrMailExists,
		},
		{
			name: "Repository get error",
			input: &models.User{
				Email:    "test@gmail.com",
				Password: "password123",
				Name:     "Alex",
			},
			getErr:  dbErr,
			wantErr: dbErr,
		},
		{
			name: "Repository create error",
			input: &models.User{
				Email:    "test@gmail.com",
				Password: "password123",
				Name:     "Alex",
			},
			createErr: dbErr,
			wantErr:   dbErr,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := mocks.NewUserRepository(t)

			repo.On("GetByEmail", mock.Anything, tc.input.Email).Return(tc.existingUser, tc.getErr).Once()

			shouldCreate := tc.getErr == nil && tc.existingUser == nil && tc.input.Role != "anybody"

			if shouldCreate {
				repo.On("CreateUser", mock.Anything, mock.AnythingOfType("*models.User")).Return(tc.createErr).Once()
			}

			userService := &service.UserService{
				Repo: repo,
			}

			got, err := userService.CreateUser(
				context.Background(),
				tc.input,
			)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected error %v, got %v", tc.wantErr, err)
				}

				if got != nil {
					t.Errorf("expected nil user, got %+v", got)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got == nil {
				t.Fatalf("expected user, got nil")
			}

			if got.Role != tc.wantRole {
				t.Errorf(
					"expected role %s, got %s",
					tc.wantRole,
					got.Role,
				)
			}

			if err := bcrypt.CompareHashAndPassword(
				[]byte(got.Password),
				[]byte(tc.input.Password),
			); err != nil {
				t.Errorf("password was not hashed correctly: %v", err)
			}
		})
	}
}

func TestLoginUser(t *testing.T) {
	dbErr := errors.New("database unavailable")
	password := "password123"

	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		t.Fatal(err)
	}

	invalidHashUser := &models.User{
		Email:    "test@gmail.com",
		Password: "not-a-valid-bcrypt-hash",
		Name:     "Alex",
		Role:     "user",
	}

	existingUser := &models.User{
		Email:    "test@gmail.com",
		Password: string(hashedPassword),
		Name:     "Alex",
		Role:     "user",
	}

	testCases := []struct {
		name         string
		input        *models.User
		existingUser *models.User
		wantErr      error
		getErr       error
	}{
		{
			name: "Success login",
			input: &models.User{
				Email:    "test@gmail.com",
				Password: "password123",
			},
			existingUser: existingUser,
		},
		{
			name: "User not found",
			input: &models.User{
				Email:    "test1@gmail.com",
				Password: "password123",
			},
			existingUser: nil,
			wantErr:      service.ErrInvalidCredentials,
		},
		{
			name: "Invalid password",
			input: &models.User{
				Email:    "test@gmail.com",
				Password: "wrong_password",
			},
			existingUser: existingUser,
			wantErr:      service.ErrInvalidCredentials,
		},
		{
			name: "Error in search user in repo",
			input: &models.User{
				Email:    "test@gmail.com",
				Password: "password123",
			},
			getErr:  dbErr,
			wantErr: dbErr,
		},
		{
			name: "Invalid hash",
			input: &models.User{
				Email:    "test@gmail.com",
				Password: "password123",
			},
			existingUser: invalidHashUser,
			wantErr:      service.ErrInvalidCredentials,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := mocks.NewUserRepository(t)

			repo.On("GetByEmail", mock.Anything, tc.input.Email).Return(tc.existingUser, tc.getErr).Once()

			userService := &service.UserService{
				Repo:      repo,
				JWTSecret: "test-secret",
			}

			got, err := userService.LoginUser(context.Background(), tc.input)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected error %v, got %v", tc.wantErr, err)
				}

				if got != "" {
					t.Errorf("expected empty token, got %s", got)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got == "" {
				t.Fatalf("wanted token, got %s", got)
			}

		})
	}
}
