package handlers

import (
	"encoding/json"
	"net/http"

	"gophermart/internal/services"
	"gophermart/internal/utils"
)

// represents a user handler
type UserHandler struct {
	userService *services.UserService
	jwtSecret   string
}

// creates a new user handler
func NewUserHandler(userService *services.UserService, jwtSecret string) *UserHandler {
	return &UserHandler{
		userService: userService,
		jwtSecret:   jwtSecret,
	}
}

// represents a register request
type RegisterRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// represents a register request
func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.LogError("Failed to decode request body: %v", err)
		utils.SendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	user, err := h.userService.Register(r.Context(), req.Login, req.Password)
	if err != nil {
		utils.LogError("Failed to register user: %v", err)
		switch err {
		case services.ErrUserExists:
			utils.SendError(w, http.StatusConflict, "User already exists")
		case services.ErrInvalidPassword:
			utils.SendError(w, http.StatusBadRequest, "Invalid password")
		default:
			utils.SendError(w, http.StatusInternalServerError, "Internal server error")
		}
		return
	}

	token, err := utils.GenerateToken(user.ID, h.jwtSecret)
	if err != nil {
		utils.LogError("Failed to generate token: %v", err)
		utils.SendError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	w.Header().Set("Authorization", "Bearer "+token)
	utils.SendSuccess(w, nil)
}

// represents a login request
type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// represents a login request
func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.LogError("Failed to decode request body: %v", err)
		utils.SendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	user, err := h.userService.Authenticate(r.Context(), req.Login, req.Password)
	if err != nil {
		utils.LogError("Failed to authenticate user: %v", err)
		switch err {
		case services.ErrUserNotFound, services.ErrInvalidPassword:
			utils.SendError(w, http.StatusUnauthorized, "Invalid credentials")
		default:
			utils.SendError(w, http.StatusInternalServerError, "Internal server error")
		}
		return
	}

	// create JWT token
	token, err := utils.GenerateToken(user.ID, h.jwtSecret)
	if err != nil {
		utils.LogError("Failed to generate token: %v", err)
		utils.SendError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	w.Header().Set("Authorization", "Bearer "+token)
	utils.SendSuccess(w, nil)
}
