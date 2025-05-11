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

// represents a balance service
type BalanceService struct {
	repo *repository.Repository
}

// creates a new balance service
func NewBalanceService(repo *repository.Repository) *BalanceService {
	return &BalanceService{repo: repo}
}

// gets a user balance
func (s *BalanceService) GetBalance(ctx context.Context, userID int) (*models.UserBalance, error) {
	return s.repo.GetUserBalance(ctx, userID)
}

// creates a withdrawal
func (s *BalanceService) CreateWithdrawal(ctx context.Context, userID int, orderNumber string, amount float32) error {
	// check if order number is valid
	if !isValidLuhn(orderNumber) {
		return ErrInvalidOrderNumber
	}

	// check if user has enough funds
	balance, err := s.repo.GetUserBalance(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user balance: %w", err)
	}

	if balance.Current < amount {
		return ErrInsufficientFunds
	}

	return s.repo.CreateWithdrawal(ctx, userID, orderNumber, amount)
}

// gets a user withdrawal history
func (s *BalanceService) GetWithdrawals(ctx context.Context, userID int) ([]models.Withdrawal, error) {
	return s.repo.GetUserWithdrawals(ctx, userID)
}
