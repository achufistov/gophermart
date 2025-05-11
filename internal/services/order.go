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
	ErrInvalidOrderNumber      = errors.New("invalid order number")
	ErrOrderExists             = errors.New("order already exists")
	ErrOrderExistsForOtherUser = errors.New("order already exists for another user")
)

// represents an accrual response
type accrualResponse struct {
	Order   string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float32 `json:"accrual,omitempty"`
}

// represents an order service
type OrderService struct {
	repo                *repository.Repository
	accrualSystemURL    string
	accrualCheckTimeout time.Duration
}

// creates a new order service
func NewOrderService(repo *repository.Repository, accrualSystemURL string) *OrderService {
	service := &OrderService{
		repo:                repo,
		accrualSystemURL:    accrualSystemURL,
		accrualCheckTimeout: 1 * time.Second,
	}

	// start goroutine for checking order statuses
	go service.startAccrualCheck()

	return service
}

// starts a periodic check of order statuses
func (s *OrderService) startAccrualCheck() {
	ticker := time.NewTicker(s.accrualCheckTimeout)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		orders, err := s.repo.GetProcessingOrders(ctx)
		if err != nil {
			fmt.Printf("Failed to get processing orders: %v\n", err)
			continue
		}

		for _, order := range orders {
			status, accrual, err := s.checkAccrualStatus(ctx, order.Number)
			if err != nil {
				fmt.Printf("Failed to check accrual status for order %s: %v\n", order.Number, err)
				continue
			}

			fmt.Printf("Order %s status: %s, accrual: %f\n", order.Number, status, accrual)

			userID, err := s.repo.CheckOrderExists(ctx, order.Number)
			if err != nil {
				fmt.Printf("Failed to get user ID for order %s: %v\n", order.Number, err)
				continue
			}

			err = s.repo.UpdateOrderStatus(ctx, order.Number, status, accrual)
			if err != nil {
				fmt.Printf("Failed to update order status for order %s: %v\n", order.Number, err)
				continue
			}

			if status == "PROCESSED" && accrual > 0 {
				balance, err := s.repo.GetUserBalance(ctx, userID)
				if err != nil {
					fmt.Printf("Failed to get user balance for user %d: %v\n", userID, err)
					continue
				}

				fmt.Printf("Current balance for user %d: %f\n", userID, balance.Current)

				err = s.repo.UpdateUserBalance(ctx, userID, accrual)
				if err != nil {
					fmt.Printf("Failed to update user balance for user %d: %v\n", userID, err)
					continue
				}

				balance, err = s.repo.GetUserBalance(ctx, userID)
				if err != nil {
					fmt.Printf("Failed to get updated balance for user %d: %v\n", userID, err)
					continue
				}

				fmt.Printf("Updated balance for user %d: %f\n", userID, balance.Current)
			}
		}
	}
}

// checkAccrualStatus checks the status of an order in the accrual system
func (s *OrderService) checkAccrualStatus(ctx context.Context, orderNumber string) (string, float32, error) {
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

	if accrualResp.Order != orderNumber {
		return "", 0, fmt.Errorf("order number mismatch: expected %s, got %s", orderNumber, accrualResp.Order)
	}

	return accrualResp.Status, accrualResp.Accrual, nil
}

// creates a new order
func (s *OrderService) CreateOrder(ctx context.Context, userID int, orderNumber string) error {
	// check if order number is valid
	if !isValidLuhn(orderNumber) {
		return ErrInvalidOrderNumber
	}

	err := s.repo.CreateOrder(ctx, userID, orderNumber)
	if err != nil {
		if err.Error() == "order with number "+orderNumber+" already exists for another user" {
			return ErrOrderExistsForOtherUser
		}
		if err.Error() == "order already exists" {
			return ErrOrderExists
		}
		return fmt.Errorf("failed to create order: %w", err)
	}

	return nil
}

// gets a list of user orders
func (s *OrderService) GetUserOrders(ctx context.Context, userID int) ([]models.Order, error) {
	return s.repo.GetUserOrders(ctx, userID)
}

// checks if order number is valid
func isValidLuhn(number string) bool {
	sum := 0
	alternate := false

	for i := len(number) - 1; i >= 0; i-- {

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
