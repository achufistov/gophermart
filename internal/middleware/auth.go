package middleware

import (
	"net/http"
	"strings"

	"gophermart/internal/utils"
)

// represents an auth middleware
type AuthMiddleware struct {
	jwtSecret string
}

// creates a new auth middleware
func NewAuthMiddleware(jwtSecret string) *AuthMiddleware {
	return &AuthMiddleware{
		jwtSecret: jwtSecret,
	}
}

// authenticates a user
func (m *AuthMiddleware) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			utils.SendError(w, http.StatusUnauthorized, "Authorization header is required")
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.SendError(w, http.StatusUnauthorized, "Invalid authorization header format")
			return
		}

		claims, err := utils.ParseToken(parts[1], m.jwtSecret)
		if err != nil {
			utils.LogError("Failed to parse token: %v", err)
			utils.SendError(w, http.StatusUnauthorized, "Invalid token")
			return
		}

		ctx := utils.WithUserID(r.Context(), claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
