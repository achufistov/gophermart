package repository

import (
	"context"
	"fmt"
	"os"
	"time"

	"gophermart/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// represents a data access layer
type Repository struct {
	db *pgxpool.Pool
}

func initDatabase(ctx context.Context, db *pgxpool.Pool) error {

	migrationSQL, err := os.ReadFile("migrations/001_init.sql")
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	_, err = db.Exec(ctx, string(migrationSQL))
	if err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	return nil
}

// creates a new repository
func NewRepository(databaseURI string) (*Repository, error) {
	pool, err := pgxpool.New(context.Background(), databaseURI)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Инициализируем базу данных
	if err := initDatabase(context.Background(), pool); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return &Repository{db: pool}, nil
}

// closes the connection to the database
func (r *Repository) Close() {
	if r.db != nil {
		r.db.Close()
	}
}

// creates a new user
func (r *Repository) CreateUser(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (login, password_hash, created_at)
		VALUES ($1, $2, $3)
		RETURNING id`

	err := r.db.QueryRow(ctx, query, user.Login, user.PasswordHash, time.Now()).Scan(&user.ID)
	if err != nil {
		if err.Error() == "ERROR: duplicate key value violates unique constraint \"users_login_key\" (SQLSTATE 23505)" {
			return fmt.Errorf("user with login %s already exists", user.Login)
		}
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// gets a user by login
func (r *Repository) GetUserByLogin(ctx context.Context, login string) (*models.User, error) {
	query := `
		SELECT id, login, password_hash, created_at
		FROM users
		WHERE login = $1`

	user := &models.User{}
	err := r.db.QueryRow(ctx, query, login).Scan(
		&user.ID,
		&user.Login,
		&user.PasswordHash,
		&user.CreatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by login: %w", err)
	}

	return user, nil
}

// creates a new order
func (r *Repository) CreateOrder(ctx context.Context, userID int, number string) error {
	// check if order exists
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM orders WHERE number = $1
		)`, number).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check order existence: %w", err)
	}

	if exists {
		// check if order belongs to the user
		var orderUserID int
		err := r.db.QueryRow(ctx, `
			SELECT user_id FROM orders WHERE number = $1`, number).Scan(&orderUserID)
		if err != nil {
			return fmt.Errorf("failed to get order user: %w", err)
		}

		if orderUserID == userID {
			return fmt.Errorf("order already exists")
		}
		return fmt.Errorf("order with number %s already exists for another user", number)
	}

	// create a new order
	_, err = r.db.Exec(ctx, `
		INSERT INTO orders (user_id, number, status, uploaded_at)
		VALUES ($1, $2, $3, $4)`,
		userID, number, "NEW", time.Now())
	if err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}

	return nil
}

// gets a list of user orders
func (r *Repository) GetUserOrders(ctx context.Context, userID int) ([]models.Order, error) {
	query := `
		SELECT number, status, accrual, uploaded_at
		FROM orders
		WHERE user_id = $1
		ORDER BY uploaded_at DESC`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders: %w", err)
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order
		err := rows.Scan(&order.Number, &order.Status, &order.Accrual, &order.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}
		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating orders: %w", err)
	}

	return orders, nil
}

// gets a user balance
func (r *Repository) GetUserBalance(ctx context.Context, userID int) (*models.UserBalance, error) {
	query := `
		SELECT current_balance, withdrawn_balance
		FROM user_balances
		WHERE user_id = $1`

	balance := &models.UserBalance{}
	err := r.db.QueryRow(ctx, query, userID).Scan(&balance.Current, &balance.Withdrawn)
	if err == pgx.ErrNoRows {
		// if record does not exist, create a new one with zero balance
		_, err = r.db.Exec(ctx, `
			INSERT INTO user_balances (user_id, current_balance, withdrawn_balance)
			VALUES ($1, 0, 0)`, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to create user balance: %w", err)
		}
		balance.Current = 0
		balance.Withdrawn = 0
		return balance, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user balance: %w", err)
	}

	return balance, nil
}

// creates a withdrawal
func (r *Repository) CreateWithdrawal(ctx context.Context, userID int, orderNumber string, sum float32) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// check if user has enough funds
	var currentBalance float32
	err = tx.QueryRow(ctx, `
		SELECT current_balance FROM user_balances WHERE user_id = $1`,
		userID).Scan(&currentBalance)
	if err != nil {
		return fmt.Errorf("failed to get user balance: %w", err)
	}

	if currentBalance < sum {
		return fmt.Errorf("insufficient funds")
	}

	// create a withdrawal record
	_, err = tx.Exec(ctx, `
		INSERT INTO withdrawals (user_id, order_number, sum, created_at)
		VALUES ($1, $2, $3, $4)`,
		userID, orderNumber, sum, time.Now())
	if err != nil {
		return fmt.Errorf("failed to create withdrawal: %w", err)
	}

	// update user balance
	_, err = tx.Exec(ctx, `
		UPDATE user_balances 
		SET current_balance = current_balance - $1, withdrawn_balance = withdrawn_balance + $1, updated_at = $2
		WHERE user_id = $3`,
		sum, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update user balance: %w", err)
	}

	return tx.Commit(ctx)
}

// gets a user withdrawal history
func (r *Repository) GetUserWithdrawals(ctx context.Context, userID int) ([]models.Withdrawal, error) {
	query := `
		SELECT order_number, sum, created_at
		FROM withdrawals
		WHERE user_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get withdrawals: %w", err)
	}
	defer rows.Close()

	var withdrawals []models.Withdrawal
	for rows.Next() {
		var w models.Withdrawal
		err := rows.Scan(&w.Order, &w.Sum, &w.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan withdrawal: %w", err)
		}
		withdrawals = append(withdrawals, w)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating withdrawals: %w", err)
	}

	return withdrawals, nil
}

// gets a list of orders in processing
func (r *Repository) GetProcessingOrders(ctx context.Context) ([]models.Order, error) {
	query := `
		SELECT number, status, accrual, uploaded_at
		FROM orders
		WHERE status IN ('NEW', 'PROCESSING')
		ORDER BY uploaded_at ASC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get processing orders: %w", err)
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order
		if err := rows.Scan(&order.Number, &order.Status, &order.Accrual, &order.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}
		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating orders: %w", err)
	}

	return orders, nil
}

// updates order status and accrual
func (r *Repository) UpdateOrderStatus(ctx context.Context, number string, status string, accrual float32) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// update order status and accrual
	_, err = tx.Exec(ctx, `
		UPDATE orders 
		SET status = $1, accrual = $2, updated_at = $3
		WHERE number = $4`,
		status, accrual, time.Now(), number)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	// if order is processed and there is an accrual, update user balance
	if status == "PROCESSED" && accrual > 0 {
		_, err = tx.Exec(ctx, `
			UPDATE user_balances 
			SET current_balance = current_balance + $1, updated_at = $2
			WHERE user_id = (
				SELECT user_id FROM orders WHERE number = $3
			)`,
			accrual, time.Now(), number)
		if err != nil {
			return fmt.Errorf("failed to update user balance: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// updates user balance
func (r *Repository) UpdateUserBalance(ctx context.Context, userID int, amount float32) error {
	// check if balance exists
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM user_balances WHERE user_id = $1)`, userID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check balance existence: %w", err)
	}

	if !exists {
		// if record does not exist, create a new one
		_, err = r.db.Exec(ctx, `
			INSERT INTO user_balances (user_id, current_balance, withdrawn_balance)
			VALUES ($1, $2, 0)`, userID, amount)
		if err != nil {
			return fmt.Errorf("failed to create user balance: %w", err)
		}
		return nil
	}

	// update existing balance
	_, err = r.db.Exec(ctx, `
		UPDATE user_balances
		SET current_balance = current_balance + $1, updated_at = $2
		WHERE user_id = $3`, amount, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update user balance: %w", err)
	}

	return nil
}

// checks if order exists and returns a user ID
func (r *Repository) CheckOrderExists(ctx context.Context, orderNumber string) (int, error) {
	query := `
		SELECT user_id
		FROM orders
		WHERE number = $1
	`

	var userID int
	err := r.db.QueryRow(ctx, query, orderNumber).Scan(&userID)
	if err == pgx.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to check order existence: %w", err)
	}

	return userID, nil
}
