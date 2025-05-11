package services

import (
	"context"
	"errors"
	"fmt"

	"gophermart/internal/models"
	"gophermart/internal/repository"
)

var (
	ErrInsufficientFunds = errors.New("insufficient funds")
)

// BalanceService представляет сервис для работы с балансом
type BalanceService struct {
	repo *repository.Repository
}

// NewBalanceService создает новый экземпляр сервиса баланса
func NewBalanceService(repo *repository.Repository) *BalanceService {
	return &BalanceService{repo: repo}
}

// GetBalance получает баланс пользователя
func (s *BalanceService) GetBalance(ctx context.Context, userID int64) (*models.UserBalance, error) {
	return s.repo.GetUserBalance(ctx, userID)
}

// CreateWithdrawal создает списание средств
func (s *BalanceService) CreateWithdrawal(ctx context.Context, userID int64, orderNumber string, amount float64) error {
	// Проверяем номер заказа с помощью алгоритма Луна
	if !isValidLuhn(orderNumber) {
		return ErrInvalidOrderNumber
	}

	// Проверяем достаточность средств
	balance, err := s.repo.GetUserBalance(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user balance: %w", err)
	}

	if balance.Current < amount {
		return ErrInsufficientFunds
	}

	return s.repo.CreateWithdrawal(ctx, userID, orderNumber, amount)
}

// GetWithdrawals получает историю списаний пользователя
func (s *BalanceService) GetWithdrawals(ctx context.Context, userID int64) ([]models.Withdrawal, error) {
	return s.repo.GetUserWithdrawals(ctx, userID)
}
