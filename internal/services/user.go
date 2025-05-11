package services

import (
	"context"
	"errors"
	"fmt"
	"unicode/utf8"

	"gophermart/internal/models"
	"gophermart/internal/repository"
	"gophermart/internal/utils"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrUserExists      = errors.New("user already exists")
	ErrInvalidPassword = errors.New("invalid password")
)

// represents a user service
type UserService struct {
	repo *repository.Repository
}

// creates a new user service
func NewUserService(repo *repository.Repository) *UserService {
	return &UserService{repo: repo}
}

// registers a new user
func (s *UserService) Register(ctx context.Context, login, password string) (*models.User, error) {

	if login == "" || utf8.RuneCountInString(login) < 3 {
		return nil, fmt.Errorf("invalid login")
	}

	if password == "" || utf8.RuneCountInString(password) < 6 {
		return nil, ErrInvalidPassword
	}

	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &models.User{
		Login:        login,
		PasswordHash: hashedPassword,
	}

	err = s.repo.CreateUser(ctx, user)
	if err != nil {
		if err.Error() == fmt.Sprintf("user with login %s already exists", login) {
			return nil, ErrUserExists
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// authenticates a user
func (s *UserService) Authenticate(ctx context.Context, login, password string) (*models.User, error) {

	if login == "" || password == "" {
		return nil, fmt.Errorf("login and password are required")
	}

	user, err := s.repo.GetUserByLogin(ctx, login)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		return nil, ErrInvalidPassword
	}

	return user, nil
}
