package services

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"gophermart/internal/models"
	"gophermart/internal/repository"
)

var (
	ErrInvalidOrderNumber = errors.New("invalid order number")
	ErrOrderExists        = errors.New("order already exists")
)

// OrderService представляет сервис для работы с заказами
type OrderService struct {
	repo *repository.Repository
}

// NewOrderService создает новый экземпляр сервиса заказов
func NewOrderService(repo *repository.Repository) *OrderService {
	return &OrderService{repo: repo}
}

// CreateOrder создает новый заказ
func (s *OrderService) CreateOrder(ctx context.Context, userID int64, orderNumber string) error {
	// Проверяем номер заказа с помощью алгоритма Луна
	if !isValidLuhn(orderNumber) {
		return ErrInvalidOrderNumber
	}

	// Проверяем существование заказа у пользователя
	orders, err := s.repo.GetUserOrders(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to check existing orders: %w", err)
	}

	for _, order := range orders {
		if order.Number == orderNumber {
			return ErrOrderExists
		}
	}

	err = s.repo.CreateOrder(ctx, userID, orderNumber)
	if err != nil {
		if err.Error() == fmt.Sprintf("order with number %s already exists", orderNumber) {
			return ErrOrderExists
		}
		return fmt.Errorf("failed to create order: %w", err)
	}

	return nil
}

// GetUserOrders получает список заказов пользователя
func (s *OrderService) GetUserOrders(ctx context.Context, userID int64) ([]models.Order, error) {
	return s.repo.GetUserOrders(ctx, userID)
}

// isValidLuhn проверяет номер заказа с помощью алгоритма Луна
func isValidLuhn(number string) bool {
	sum := 0
	alternate := false

	// Проходим по цифрам справа налево
	for i := len(number) - 1; i >= 0; i-- {
		// Преобразуем символ в число
		n, err := strconv.Atoi(string(number[i]))
		if err != nil {
			return false
		}

		if alternate {
			n *= 2
			if n > 9 {
				n = (n % 10) + 1
			}
		}

		sum += n
		alternate = !alternate
	}

	return sum%10 == 0
}
