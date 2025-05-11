package handlers

import (
	"io"
	"net/http"

	"gophermart/internal/models"
	"gophermart/internal/services"
	"gophermart/internal/utils"
)

// OrderHandler представляет обработчик заказов
type OrderHandler struct {
	orderService *services.OrderService
}

// NewOrderHandler создает новый экземпляр обработчика заказов
func NewOrderHandler(orderService *services.OrderService) *OrderHandler {
	return &OrderHandler{
		orderService: orderService,
	}
}

// UploadOrder загружает номер заказа
func (h *OrderHandler) UploadOrder(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.GetUserID(r.Context())
	if !ok || userID == 0 {
		utils.SendError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Читаем номер заказа
	orderNumber, err := io.ReadAll(r.Body)
	if err != nil {
		utils.LogError("Failed to read request body: %v", err)
		utils.SendError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}
	defer r.Body.Close()

	// Загружаем заказ
	err = h.orderService.CreateOrder(r.Context(), userID, string(orderNumber))
	if err != nil {
		utils.LogError("Failed to upload order: %v", err)
		if err == services.ErrInvalidOrderNumber {
			utils.SendError(w, http.StatusUnprocessableEntity, "Invalid order number")
			return
		}
		if err == services.ErrOrderExists {
			utils.SendError(w, http.StatusConflict, "Order already exists")
			return
		}
		utils.SendError(w, http.StatusInternalServerError, "Failed to upload order")
		return
	}

	utils.SendJSON(w, http.StatusAccepted, map[string]string{"message": "Order uploaded successfully"})
}

// GetUserOrders получает список заказов пользователя
func (h *OrderHandler) GetUserOrders(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.GetUserID(r.Context())
	if !ok || userID == 0 {
		utils.SendError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	orders, err := h.orderService.GetUserOrders(r.Context(), userID)
	if err != nil {
		utils.LogError("Failed to get user orders: %v", err)
		utils.SendError(w, http.StatusInternalServerError, "Failed to get user orders")
		return
	}

	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	utils.SendJSON(w, http.StatusOK, orders)
}

// GetOrder получает информацию о конкретном заказе
func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.GetUserID(r.Context())
	if !ok || userID == 0 {
		utils.SendError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	// Получаем номер заказа из URL
	orderNumber := r.URL.Path[len("/api/orders/"):]
	if orderNumber == "" {
		utils.SendError(w, http.StatusBadRequest, "Order number is required")
		return
	}

	// Получаем все заказы пользователя
	orders, err := h.orderService.GetUserOrders(r.Context(), userID)
	if err != nil {
		utils.LogError("Failed to get user orders: %v", err)
		utils.SendError(w, http.StatusInternalServerError, "Failed to get user orders")
		return
	}

	// Ищем нужный заказ
	var targetOrder *models.Order
	for _, order := range orders {
		if order.Number == orderNumber {
			targetOrder = &order
			break
		}
	}

	if targetOrder == nil {
		utils.SendError(w, http.StatusNotFound, "Order not found")
		return
	}

	utils.SendJSON(w, http.StatusOK, targetOrder)
}
