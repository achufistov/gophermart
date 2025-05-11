package models

import (
	"time"
)

// represents a user
type User struct {
	ID           int64     `json:"-"`
	Login        string    `json:"login"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"-"`
}

// represents a user balance
type UserBalance struct {
	Current   float32 `json:"current"`
	Withdrawn float32 `json:"withdrawn"`
}

// represents an order
type Order struct {
	Number    string    `json:"number"`
	Status    string    `json:"status"`
	Accrual   float32   `json:"accrual,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// represents a withdrawal
type Withdrawal struct {
	Order     string    `json:"order"`
	Sum       float32   `json:"sum"`
	CreatedAt time.Time `json:"processed_at"`
}
