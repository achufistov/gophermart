package handlers

import (
	"io"
	"net/http"

	"gophermart/internal/models"
	"gophermart/internal/services"
	"gophermart/internal/utils"
)

// represents an order handler
type OrderHandler struct {
	orderService *services.OrderService
}

// creates a new order handler
func NewOrderHandler(orderService *services.OrderService) *OrderHandler {
	return &OrderHandler{
		orderService: orderService,
	}
}

// uploads an order
func (h *OrderHandler) UploadOrder(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.GetUserID(r.Context())
	if !ok || userID == 0 {
		utils.SendError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	orderNumber, err := io.ReadAll(r.Body)
	if err != nil {
		utils.LogError("Failed to read request body: %v", err)
		utils.SendError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}
	defer r.Body.Close()

	err = h.orderService.CreateOrder(r.Context(), int(userID), string(orderNumber))
	if err != nil {
		utils.LogError("Failed to upload order: %v", err)
		switch err {
		case services.ErrInvalidOrderNumber:
			utils.SendError(w, http.StatusUnprocessableEntity, "Invalid order number")
		case services.ErrOrderExistsForOtherUser:
			utils.SendError(w, http.StatusConflict, "Order already exists for another user")
		case services.ErrOrderExists:
			w.WriteHeader(http.StatusOK)
			return
		default:
			utils.SendError(w, http.StatusInternalServerError, "Failed to upload order")
		}
		return
	}

	utils.SendJSON(w, http.StatusAccepted, map[string]string{"message": "Order uploaded successfully"})
}

// gets a list of user orders
func (h *OrderHandler) GetUserOrders(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.GetUserID(r.Context())
	if !ok || userID == 0 {
		utils.SendError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	orders, err := h.orderService.GetUserOrders(r.Context(), int(userID))
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

// gets a user order
func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.GetUserID(r.Context())
	if !ok || userID == 0 {
		utils.SendError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	orderNumber := r.URL.Path[len("/api/orders/"):]
	if orderNumber == "" {
		utils.SendError(w, http.StatusBadRequest, "Order number is required")
		return
	}

	orders, err := h.orderService.GetUserOrders(r.Context(), int(userID))
	if err != nil {
		utils.LogError("Failed to get user orders: %v", err)
		utils.SendError(w, http.StatusInternalServerError, "Failed to get user orders")
		return
	}

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
