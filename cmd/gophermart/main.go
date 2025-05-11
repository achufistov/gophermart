package main

import (
	"log"
	"net/http"

	"gophermart/internal/config"
	"gophermart/internal/handlers"
	"gophermart/internal/middleware"
	"gophermart/internal/repository"
	"gophermart/internal/services"
)

func main() {
	cfg := config.NewConfig()

	// init repo
	repo, err := repository.NewRepository(cfg.DatabaseURI)
	if err != nil {
		log.Fatalf("Failed to initialize repository: %v", err)
	}
	defer repo.Close()

	// init services
	userService := services.NewUserService(repo)
	orderService := services.NewOrderService(repo, cfg.AccrualSystemAddress)
	balanceService := services.NewBalanceService(repo)

	// init handlers
	userHandler := handlers.NewUserHandler(userService, cfg.JWTSecret)
	orderHandler := handlers.NewOrderHandler(orderService)
	balanceHandler := handlers.NewBalanceHandler(balanceService)

	// init middleware
	authMiddleware := middleware.NewAuthMiddleware(cfg.JWTSecret)

	// creates a router
	mux := http.NewServeMux()

	// public routes
	mux.HandleFunc("/api/user/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		userHandler.Register(w, r)
	})

	mux.HandleFunc("/api/user/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		userHandler.Login(w, r)
	})

	// protected routes
	mux.HandleFunc("/api/user/orders", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			authMiddleware.Auth(http.HandlerFunc(orderHandler.UploadOrder)).ServeHTTP(w, r)
		case http.MethodGet:
			authMiddleware.Auth(http.HandlerFunc(orderHandler.GetUserOrders)).ServeHTTP(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/orders/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		authMiddleware.Auth(http.HandlerFunc(orderHandler.GetOrder)).ServeHTTP(w, r)
	})

	mux.HandleFunc("/api/user/balance", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		authMiddleware.Auth(http.HandlerFunc(balanceHandler.GetBalance)).ServeHTTP(w, r)
	})

	mux.HandleFunc("/api/user/balance/withdraw", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		authMiddleware.Auth(http.HandlerFunc(balanceHandler.CreateWithdrawal)).ServeHTTP(w, r)
	})

	mux.HandleFunc("/api/user/withdrawals", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		authMiddleware.Auth(http.HandlerFunc(balanceHandler.GetWithdrawals)).ServeHTTP(w, r)
	})

	log.Printf("Starting server on %s", cfg.RunAddress)
	if err := http.ListenAndServe(cfg.RunAddress, mux); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
