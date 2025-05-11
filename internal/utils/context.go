package utils

import (
	"context"
)

type contextKey string

const (
	UserIDKey contextKey = "user_id"
)

// adds a user ID to the context
func WithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// gets a user ID from the context
func GetUserID(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(UserIDKey).(int64)
	return userID, ok
}
