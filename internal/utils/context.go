package utils

import (
	"context"
)

type contextKey string

const (
	UserIDKey contextKey = "user_id"
)

// WithUserID добавляет ID пользователя в контекст
func WithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// GetUserID получает ID пользователя из контекста
func GetUserID(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(UserIDKey).(int64)
	return userID, ok
}
