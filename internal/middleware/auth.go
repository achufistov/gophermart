package middleware

import (
	"net/http"
	"strings"

	"gophermart/internal/utils"
)

// AuthMiddleware представляет middleware для аутентификации
type AuthMiddleware struct {
	jwtSecret string
}

// NewAuthMiddleware создает новый экземпляр middleware аутентификации
func NewAuthMiddleware(jwtSecret string) *AuthMiddleware {
	return &AuthMiddleware{
		jwtSecret: jwtSecret,
	}
}

// Auth проверяет аутентификацию пользователя
func (m *AuthMiddleware) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Получаем токен из заголовка
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			utils.SendError(w, http.StatusUnauthorized, "Authorization header is required")
			return
		}

		// Проверяем формат токена
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.SendError(w, http.StatusUnauthorized, "Invalid authorization header format")
			return
		}

		// Парсим и проверяем токен
		claims, err := utils.ParseToken(parts[1], m.jwtSecret)
		if err != nil {
			utils.LogError("Failed to parse token: %v", err)
			utils.SendError(w, http.StatusUnauthorized, "Invalid token")
			return
		}

		// Добавляем ID пользователя в контекст
		ctx := utils.WithUserID(r.Context(), claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
