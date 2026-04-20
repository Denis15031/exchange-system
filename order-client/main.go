package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"exchange-system/order-client/internal/auth"
	"exchange-system/order-client/internal/client"
	"exchange-system/order-client/internal/config"
	"exchange-system/order-client/internal/ports"
	orderv1 "exchange-system/proto/order/v1"
	"exchange-system/shared/logger"
	"exchange-system/shared/shutdown"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	email     string
	password  string
	username  string
	marketID  string
	orderType string
	price     string
	quantity  string
	orderID   string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "order-client",
		Short: "CLI клиент для OrderService биржевой системы",
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
	}

	rootCmd.PersistentFlags().StringVar(&email, "email", "", "Email для аутентификации")
	rootCmd.PersistentFlags().StringVar(&password, "password", "", "Пароль для аутентификации")
	rootCmd.PersistentFlags().StringVar(&username, "username", "", "Имя пользователя (для регистрации)")
	rootCmd.PersistentFlags().StringVar(&marketID, "market-id", "", "Market ID")
	rootCmd.PersistentFlags().StringVar(&orderType, "order-type", "BUY", "Order type: BUY or SELL")
	rootCmd.PersistentFlags().StringVar(&price, "price", "", "Order price")
	rootCmd.PersistentFlags().StringVar(&quantity, "quantity", "", "Order quantity")
	rootCmd.PersistentFlags().StringVar(&orderID, "order-id", "", "Order ID (для статуса)")

	rootCmd.AddCommand(&cobra.Command{Use: "register", Short: "Зарегистрировать нового пользователя", RunE: runRegister})
	rootCmd.AddCommand(&cobra.Command{Use: "login", Short: "Войти в систему", RunE: runLogin})
	rootCmd.AddCommand(&cobra.Command{Use: "logout", Short: "Выйти из системы", RunE: runLogout})

	createCmd := &cobra.Command{
		Use:   "create-order",
		Short: "Создать новый ордер",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if marketID == "" || price == "" || quantity == "" {
				return fmt.Errorf("--market-id, --price and --quantity are required")
			}
			if orderType != "BUY" && orderType != "SELL" {
				return fmt.Errorf("--order-type must be BUY or SELL")
			}
			return nil
		},
		RunE: runCreateOrder,
	}
	rootCmd.AddCommand(createCmd)

	statusCmd := &cobra.Command{
		Use:   "order-status",
		Short: "Получить статус ордера",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if orderID == "" {
				return fmt.Errorf("--order-id is required")
			}
			return nil
		},
		RunE: runOrderStatus,
	}
	rootCmd.AddCommand(statusCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

//Auth Commands

func runRegister(cmd *cobra.Command, args []string) error {
	cfg, _ := config.Load()
	if email == "" || password == "" {
		reader := bufio.NewReader(os.Stdin)
		if email == "" {
			fmt.Print("Email: ")
			email, _ = reader.ReadString('\n')
			email = email[:len(email)-1]
		}
		if password == "" {
			fmt.Print("Password: ")
			password, _ = reader.ReadString('\n')
			password = password[:len(password)-1]
		}
		if username == "" {
			fmt.Print("Username: ")
			username, _ = reader.ReadString('\n')
			username = username[:len(username)-1]
		}
	}
	tm, err := auth.NewTokenManager(cfg.UserServiceAddr, cfg.TokenStoragePath)
	if err != nil {
		return err
	}
	defer func() { _ = tm.Close() }()
	resp, err := tm.Register(email, password, username)
	if err != nil {
		return fmt.Errorf("registration failed: %w", err)
	}
	fmt.Printf("Registration successful\nUser ID: %s\nEmail: %s\nUsername: %s\n", resp.User.UserId, resp.User.Email, resp.User.Username)
	return nil
}

func runLogin(cmd *cobra.Command, args []string) error {
	cfg, _ := config.Load()
	if email == "" || password == "" {
		reader := bufio.NewReader(os.Stdin)
		if email == "" {
			fmt.Print("Email: ")
			email, _ = reader.ReadString('\n')
			email = email[:len(email)-1]
		}
		if password == "" {
			fmt.Print("Password: ")
			password, _ = reader.ReadString('\n')
			password = password[:len(password)-1]
		}
	}
	tm, err := auth.NewTokenManager(cfg.UserServiceAddr, cfg.TokenStoragePath)
	if err != nil {
		return err
	}
	defer func() { _ = tm.Close() }()
	resp, err := tm.Login(email, password)
	if err != nil {
		return fmt.Errorf("login failed: %w", err)
	}
	fmt.Printf("Login successful\nUser ID: %s\nEmail: %s\nToken expires: %s\n", resp.User.UserId, resp.User.Email, time.Now().Add(15*time.Minute).Format(time.RFC3339))
	return nil
}

func runLogout(cmd *cobra.Command, args []string) error {
	cfg, _ := config.Load()
	tm, err := auth.NewTokenManager(cfg.UserServiceAddr, cfg.TokenStoragePath)
	if err != nil {
		return err
	}
	defer func() { _ = tm.Close() }()
	if err := tm.Logout(); err != nil {
		return fmt.Errorf("logout failed: %w", err)
	}
	fmt.Printf("Logout successful\n")
	return nil
}

// Order Commands
func runCreateOrder(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	appLogger, err := logger.New(logger.Config{Level: cfg.LogLevel, Format: "json"})
	if err != nil {
		return err
	}
	defer func() { _ = appLogger.Sync() }()

	shutdownHandler := shutdown.New(10 * time.Second)
	shutdownHandler.RegisterFunc("logger", func() error { return appLogger.Sync() })

	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() { <-sigCh; cancel(); shutdownHandler.Trigger() }()

	tm, err := auth.NewTokenManager(cfg.UserServiceAddr, cfg.TokenStoragePath)
	if err != nil {
		return err
	}
	defer func() { _ = tm.Close() }()
	shutdownHandler.RegisterFunc("token-manager", func() error { return tm.Close() })

	orderClient, err := client.New(cfg, appLogger, tm)
	if err != nil {
		return err
	}
	defer func() { _ = orderClient.Close() }()
	shutdownHandler.RegisterFunc("grpc-client", func() error { return orderClient.Close() })

	protoType := toProtoOrderType(orderType)
	return executeCreateOrder(ctx, tm, orderClient, appLogger, marketID, protoType, price, quantity)
}

func executeCreateOrder(
	ctx context.Context,
	tm interface{ HasToken() bool },
	client ports.OrderServiceClient,
	logger *logger.Logger,
	marketID string,
	orderType orderv1.OrderType,
	price, quantity string,
) error {
	if !tm.HasToken() {
		return fmt.Errorf("not authenticated. Please run 'login' first")
	}
	logger.Info("creating order",
		zap.String("market_id", marketID),
		zap.String("order_type", orderType.String()),
	)
	resp, err := client.CreateOrder(ctx, marketID, orderType, price, quantity)
	if err != nil {
		logger.Error("order creation failed", zap.Error(err))
		return err
	}
	fmt.Printf("Order created: %s (Status: %s)\n", resp.OrderId, resp.Status.String())
	return nil
}

func runOrderStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	appLogger, err := logger.New(logger.Config{Level: cfg.LogLevel, Format: "json"})
	if err != nil {
		return err
	}
	defer func() { _ = appLogger.Sync() }()

	shutdownHandler := shutdown.New(10 * time.Second)
	shutdownHandler.RegisterFunc("logger", func() error { return appLogger.Sync() })

	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() { <-sigCh; cancel(); shutdownHandler.Trigger() }()

	tm, err := auth.NewTokenManager(cfg.UserServiceAddr, cfg.TokenStoragePath)
	if err != nil {
		return err
	}
	defer func() { _ = tm.Close() }()
	shutdownHandler.RegisterFunc("token-manager", func() error { return tm.Close() })

	orderClient, err := client.New(cfg, appLogger, tm)
	if err != nil {
		return err
	}
	defer func() { _ = orderClient.Close() }()
	shutdownHandler.RegisterFunc("grpc-client", func() error { return orderClient.Close() })

	return executeOrderStatus(ctx, tm, orderClient, appLogger, orderID)
}

func executeOrderStatus(
	ctx context.Context,
	tm interface{ HasToken() bool },
	client ports.OrderServiceClient,
	logger *logger.Logger,
	orderID string,
) error {
	if !tm.HasToken() {
		return fmt.Errorf("not authenticated")
	}
	resp, err := client.GetOrderStatus(ctx, orderID)
	if err != nil {
		logger.Error("failed to get order status", zap.Error(err))
		return err
	}
	if resp.Order != nil {
		fmt.Printf("Order %s | Status: %s\n", resp.Order.OrderId, resp.Order.Status.String())
	}
	return nil
}

func toProtoOrderType(s string) orderv1.OrderType {
	switch s {
	case "BUY":
		return orderv1.OrderType_ORDER_TYPE_BUY
	case "SELL":
		return orderv1.OrderType_ORDER_TYPE_SELL
	default:
		return orderv1.OrderType_ORDER_TYPE_BUY
	}
}
