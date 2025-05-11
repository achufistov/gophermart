package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"gophermart/internal/models"
	"gophermart/internal/repository"
)

var (
	ErrInvalidOrderNumber = errors.New("invalid order number")
	ErrOrderExists        = errors.New("order already exists")
)

type accrualResponse struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual,omitempty"`
}

// OrderService представляет сервис для работы с заказами
type OrderService struct {
	repo                *repository.Repository
	accrualSystemURL    string
	accrualCheckTimeout time.Duration
}

// NewOrderService создает новый экземпляр сервиса заказов
func NewOrderService(repo *repository.Repository, accrualSystemURL string) *OrderService {
	service := &OrderService{
		repo:                repo,
		accrualSystemURL:    accrualSystemURL,
		accrualCheckTimeout: 1 * time.Second,
	}

	// Запускаем горутину для проверки статусов заказов
	go service.startAccrualCheck()

	return service
}

// startAccrualCheck запускает периодическую проверку статусов заказов
func (s *OrderService) startAccrualCheck() {
	ticker := time.NewTicker(s.accrualCheckTimeout)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		orders, err := s.repo.GetProcessingOrders(ctx)
		if err != nil {
			continue
		}

		for _, order := range orders {
			status, accrual, err := s.checkAccrualStatus(ctx, order.Number)
			if err != nil {
				continue
			}

			if status == "PROCESSED" {
				err = s.repo.UpdateOrderStatus(ctx, order.Number, status, accrual)
				if err != nil {
					continue
				}
				// Получаем user_id для заказа
				userID, err := s.repo.GetOrderUserID(ctx, order.Number)
				if err != nil {
					continue
				}
				// Обновляем баланс пользователя
				err = s.repo.UpdateUserBalance(ctx, userID, accrual)
				if err != nil {
					continue
				}
			} else if status == "INVALID" {
				err = s.repo.UpdateOrderStatus(ctx, order.Number, status, 0)
				if err != nil {
					continue
				}
			}
		}
	}
}

// checkAccrualStatus проверяет статус заказа в системе начислений
func (s *OrderService) checkAccrualStatus(ctx context.Context, orderNumber string) (string, float64, error) {
	url := fmt.Sprintf("%s/api/orders/%s", s.accrualSystemURL, orderNumber)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", 0, err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return "NEW", 0, nil
	}

	if resp.StatusCode != http.StatusOK {
		return "", 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var accrualResp accrualResponse
	if err := json.NewDecoder(resp.Body).Decode(&accrualResp); err != nil {
		return "", 0, err
	}

	return accrualResp.Status, accrualResp.Accrual, nil
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
			return nil // Возвращаем nil, если заказ уже существует у этого пользователя
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
