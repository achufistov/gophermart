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

// UserService представляет сервис для работы с пользователями
type UserService struct {
	repo *repository.Repository
}

// NewUserService создает новый экземпляр сервиса пользователей
func NewUserService(repo *repository.Repository) *UserService {
	return &UserService{repo: repo}
}

// Register регистрирует нового пользователя
func (s *UserService) Register(ctx context.Context, login, password string) (*models.User, error) {
	// Проверяем валидность логина
	if login == "" || utf8.RuneCountInString(login) < 3 {
		return nil, fmt.Errorf("invalid login")
	}

	// Проверяем валидность пароля
	if password == "" || utf8.RuneCountInString(password) < 6 {
		return nil, ErrInvalidPassword
	}

	// Хешируем пароль
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Создаем пользователя
	user := &models.User{
		Login:        login,
		PasswordHash: hashedPassword,
	}

	// Сохраняем пользователя в базу данных
	err = s.repo.CreateUser(ctx, user)
	if err != nil {
		if err.Error() == fmt.Sprintf("user with login %s already exists", login) {
			return nil, ErrUserExists
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// Authenticate аутентифицирует пользователя
func (s *UserService) Authenticate(ctx context.Context, login, password string) (*models.User, error) {
	// Проверяем входные данные
	if login == "" || password == "" {
		return nil, fmt.Errorf("login and password are required")
	}

	// Получаем пользователя из базы данных
	user, err := s.repo.GetUserByLogin(ctx, login)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	// Проверяем пароль
	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		return nil, ErrInvalidPassword
	}

	return user, nil
}
