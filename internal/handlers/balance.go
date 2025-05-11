package handlers

import (
	"encoding/json"
	"net/http"

	"gophermart/internal/models"
	"gophermart/internal/services"
	"gophermart/internal/utils"
)

// represents a balance handler
type BalanceHandler struct {
	balanceService *services.BalanceService
}

// creates a new balance handler
func NewBalanceHandler(balanceService *services.BalanceService) *BalanceHandler {
	return &BalanceHandler{
		balanceService: balanceService,
	}
}

// gets a user balance
func (h *BalanceHandler) GetBalance(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.GetUserID(r.Context())
	if !ok || userID == 0 {
		utils.SendError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	balance, err := h.balanceService.GetBalance(r.Context(), int(userID))
	if err != nil {
		utils.LogError("Failed to get balance: %v", err)
		utils.SendError(w, http.StatusInternalServerError, "Failed to get balance")
		return
	}

	utils.SendJSON(w, http.StatusOK, balance)
}

// represents a withdrawal request
type withdrawalRequest struct {
	Order string  `json:"order"`
	Sum   float32 `json:"sum"`
}

// creates a withdrawal
func (h *BalanceHandler) CreateWithdrawal(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.GetUserID(r.Context())
	if !ok || userID == 0 {
		utils.SendError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req withdrawalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.LogError("Failed to decode request: %v", err)
		utils.SendError(w, http.StatusBadRequest, "Invalid request format")
		return
	}

	err := h.balanceService.CreateWithdrawal(r.Context(), int(userID), req.Order, req.Sum)
	if err != nil {
		utils.LogError("Failed to create withdrawal: %v", err)
		switch err {
		case services.ErrInvalidOrderNumber:
			utils.SendError(w, http.StatusUnprocessableEntity, "Invalid order number")
		case services.ErrInsufficientFunds:
			utils.SendError(w, http.StatusUnprocessableEntity, "Insufficient funds")
		default:
			utils.SendError(w, http.StatusInternalServerError, "Failed to create withdrawal")
		}
		return
	}

	utils.SendJSON(w, http.StatusOK, map[string]string{"message": "Withdrawal created successfully"})
}

func (h *BalanceHandler) GetWithdrawals(w http.ResponseWriter, r *http.Request) {
	userID, ok := utils.GetUserID(r.Context())
	if !ok || userID == 0 {
		utils.SendError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	withdrawals, err := h.balanceService.GetWithdrawals(r.Context(), int(userID))
	if err != nil {
		utils.LogError("Failed to get withdrawals: %v", err)
		utils.SendError(w, http.StatusInternalServerError, "Failed to get withdrawals")
		return
	}

	if len(withdrawals) == 0 {
		utils.SendJSON(w, http.StatusOK, []models.Withdrawal{})
		return
	}

	utils.SendJSON(w, http.StatusOK, withdrawals)
}
