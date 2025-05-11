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

// Repository представляет слой доступа к данным
type Repository struct {
	db *pgxpool.Pool
}

// initDatabase инициализирует базу данных, создавая необходимые таблицы
func initDatabase(ctx context.Context, db *pgxpool.Pool) error {
	// Читаем файл миграции
	migrationSQL, err := os.ReadFile("migrations/001_init.sql")
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	// Выполняем миграцию
	_, err = db.Exec(ctx, string(migrationSQL))
	if err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	return nil
}

// NewRepository создает новый экземпляр репозитория
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

// Close закрывает соединение с базой данных
func (r *Repository) Close() {
	if r.db != nil {
		r.db.Close()
	}
}

// CreateUser создает нового пользователя
func (r *Repository) CreateUser(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (login, password_hash, created_at)
		VALUES ($1, $2, $3)
		RETURNING id`

	err := r.db.QueryRow(ctx, query, user.Login, user.PasswordHash, time.Now()).Scan(&user.ID)
	if err != nil {
		// Проверяем на ошибку уникального ограничения
		if err.Error() == "ERROR: duplicate key value violates unique constraint \"users_login_key\" (SQLSTATE 23505)" {
			return fmt.Errorf("user with login %s already exists", user.Login)
		}
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// GetUserByLogin получает пользователя по логину
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

// CreateOrder создает новый заказ
func (r *Repository) CreateOrder(ctx context.Context, userID int64, orderNumber string) error {
	// Проверяем существование заказа
	var exists bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM orders WHERE number = $1)`, orderNumber).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check existing order: %w", err)
	}
	if exists {
		return fmt.Errorf("order with number %s already exists", orderNumber)
	}

	query := `
		INSERT INTO orders (user_id, number, status, created_at)
		VALUES ($1, $2, $3, $4)`

	_, err = r.db.Exec(ctx, query, userID, orderNumber, "NEW", time.Now())
	if err != nil {
		// Проверяем на ошибку уникального ограничения
		if err.Error() == "ERROR: duplicate key value violates unique constraint \"orders_number_key\" (SQLSTATE 23505)" {
			return fmt.Errorf("order with number %s already exists", orderNumber)
		}
		return fmt.Errorf("failed to create order: %w", err)
	}
	return nil
}

// GetUserOrders получает список заказов пользователя
func (r *Repository) GetUserOrders(ctx context.Context, userID int64) ([]models.Order, error) {
	query := `
		SELECT number, status, accrual, created_at
		FROM orders
		WHERE user_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order
		err := rows.Scan(&order.Number, &order.Status, &order.Accrual, &order.CreatedAt)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	return orders, rows.Err()
}

// GetUserBalance получает баланс пользователя
func (r *Repository) GetUserBalance(ctx context.Context, userID int64) (*models.UserBalance, error) {
	query := `
		SELECT current_balance, withdrawn_balance
		FROM user_balances
		WHERE user_id = $1`

	balance := &models.UserBalance{}
	err := r.db.QueryRow(ctx, query, userID).Scan(&balance.Current, &balance.Withdrawn)
	if err == pgx.ErrNoRows {
		// Если записи нет, создаем новую с нулевым балансом
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

// CreateWithdrawal создает списание средств
func (r *Repository) CreateWithdrawal(ctx context.Context, userID int64, orderNumber string, amount float64) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Проверяем достаточность средств
	var currentBalance float64
	err = tx.QueryRow(ctx, `
		SELECT current_balance
		FROM user_balances
		WHERE user_id = $1`, userID).Scan(&currentBalance)
	if err != nil {
		return fmt.Errorf("failed to get current balance: %w", err)
	}

	if currentBalance < amount {
		return fmt.Errorf("insufficient funds")
	}

	// Создаем запись о списании
	_, err = tx.Exec(ctx, `
		INSERT INTO withdrawals (user_id, order_number, sum)
		VALUES ($1, $2, $3)`,
		userID, orderNumber, amount)
	if err != nil {
		return fmt.Errorf("failed to create withdrawal record: %w", err)
	}

	// Обновляем баланс пользователя
	_, err = tx.Exec(ctx, `
		UPDATE user_balances
		SET current_balance = current_balance - $1,
			withdrawn_balance = withdrawn_balance + $1
		WHERE user_id = $2`,
		amount, userID)
	if err != nil {
		return fmt.Errorf("failed to update user balance: %w", err)
	}

	return tx.Commit(ctx)
}

// GetUserWithdrawals получает историю списаний пользователя
func (r *Repository) GetUserWithdrawals(ctx context.Context, userID int64) ([]models.Withdrawal, error) {
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
		err := rows.Scan(&w.Order, &w.Sum, &w.ProcessedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan withdrawal: %w", err)
		}
		withdrawals = append(withdrawals, w)
	}

	return withdrawals, rows.Err()
}
