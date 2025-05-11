package handlers

import (
	"encoding/json"
	"net/http"

	"gophermart/internal/services"
	"gophermart/internal/utils"
)

// UserHandler представляет обработчики для работы с пользователями
type UserHandler struct {
	userService *services.UserService
	jwtSecret   string
}

// NewUserHandler создает новый экземпляр обработчика пользователей
func NewUserHandler(userService *services.UserService, jwtSecret string) *UserHandler {
	return &UserHandler{
		userService: userService,
		jwtSecret:   jwtSecret,
	}
}

// RegisterRequest представляет запрос на регистрацию
type RegisterRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// Register обрабатывает регистрацию нового пользователя
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

	// Создаем JWT токен
	token, err := utils.GenerateToken(user.ID, h.jwtSecret)
	if err != nil {
		utils.LogError("Failed to generate token: %v", err)
		utils.SendError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	w.Header().Set("Authorization", "Bearer "+token)
	utils.SendSuccess(w, nil)
}

// LoginRequest представляет запрос на вход
type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// Login обрабатывает вход пользователя
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

	// Создаем JWT токен
	token, err := utils.GenerateToken(user.ID, h.jwtSecret)
	if err != nil {
		utils.LogError("Failed to generate token: %v", err)
		utils.SendError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	w.Header().Set("Authorization", "Bearer "+token)
	utils.SendSuccess(w, nil)
}
