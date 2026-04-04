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

	registerCmd := &cobra.Command{
		Use:   "register",
		Short: "Зарегистрировать нового пользователя",
		RunE:  runRegister,
	}

	loginCmd := &cobra.Command{
		Use:   "login",
		Short: "Войти в систему",
		RunE:  runLogin,
	}

	logoutCmd := &cobra.Command{
		Use:   "logout",
		Short: "Выйти из системы",
		RunE:  runLogout,
	}

	createCmd := &cobra.Command{
		Use:   "create-order",
		Short: "Создать новый ордер",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if orderType != "BUY" && orderType != "SELL" {
				return fmt.Errorf("--order-type must be BUY or SELL")
			}
			return nil
		},
		RunE: runCreateOrder,
	}

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

	rootCmd.AddCommand(registerCmd)
	rootCmd.AddCommand(loginCmd)
	rootCmd.AddCommand(logoutCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(statusCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

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

	fmt.Printf("Registration successful\n")
	fmt.Printf("User ID:  %s\n", resp.User.UserId)
	fmt.Printf("Email:    %s\n", resp.User.Email)
	fmt.Printf("Username: %s\n", resp.User.Username)

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

	fmt.Printf("✓ Login successful\n")
	fmt.Printf("  User ID:  %s\n", resp.User.UserId)
	fmt.Printf("  Email:    %s\n", resp.User.Email)
	fmt.Printf("  Token expires: %s\n", time.Unix(resp.Token.ExpiresAt, 0).Format(time.RFC3339))

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

	fmt.Printf("✓ Logout successful\n")
	return nil
}

func runCreateOrder(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logConfig := logger.DefaultConfig()
	logConfig.Format = "json"
	logConfig.Level = cfg.LogLevel

	appLogger, err := logger.New(logConfig)
	if err != nil {
		return err
	}
	defer func() { _ = appLogger.Sync() }()

	shutdownHandler := shutdown.New(10 * time.Second)
	shutdownHandler.RegisterFunc("logger", func() error {
		return appLogger.Sync()
	})

	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
		shutdownHandler.Trigger()
	}()

	tm, err := auth.NewTokenManager(cfg.UserServiceAddr, cfg.TokenStoragePath)
	if err != nil {
		return err
	}
	shutdownHandler.RegisterFunc("token-manager", func() error {
		return tm.Close()
	})

	if !tm.HasToken() {
		return fmt.Errorf("not authenticated. Please run 'login' first")
	}

	grpcClient, err := client.New(cfg, appLogger, tm)
	if err != nil {
		return err
	}
	shutdownHandler.RegisterFunc("grpc-client", func() error {
		return grpcClient.Close()
	})

	var protoOrderType orderv1.OrderType
	switch orderType {
	case "BUY":
		protoOrderType = orderv1.OrderType_ORDER_TYPE_BUY
	case "SELL":
		protoOrderType = orderv1.OrderType_ORDER_TYPE_SELL
	}

	appLogger.InfoRedact("creating order request",
		zap.String("market_id", marketID),
		zap.String("order_type", orderType),
	)

	resp, err := grpcClient.CreateOrder(ctx, marketID, protoOrderType, price, quantity)
	if err != nil {
		appLogger.ErrorRedact("order creation failed", err,
			zap.String("market_id", marketID),
		)
		return err
	}

	fmt.Printf("Order created successfully\n")
	fmt.Printf("Order ID: %s\n", resp.OrderId)
	fmt.Printf("Status:   %s\n", resp.Status)

	shutdownHandler.Trigger()
	shutdownHandler.WaitForCompletion()

	return nil
}
func runOrderStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logConfig := logger.DefaultConfig()
	logConfig.Format = "json"
	logConfig.Level = cfg.LogLevel

	appLogger, err := logger.New(logConfig)
	if err != nil {
		return err
	}
	defer func() { _ = appLogger.Sync() }()

	shutdownHandler := shutdown.New(10 * time.Second)
	shutdownHandler.RegisterFunc("logger", func() error {
		return appLogger.Sync()
	})

	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
		shutdownHandler.Trigger()
	}()

	tm, err := auth.NewTokenManager(cfg.UserServiceAddr, cfg.TokenStoragePath)
	if err != nil {
		return err
	}
	shutdownHandler.RegisterFunc("token-manager", func() error {
		return tm.Close()
	})

	if !tm.HasToken() {
		return fmt.Errorf("not authenticated. Please run 'login' first")
	}

	grpcClient, err := client.New(cfg, appLogger, tm)
	if err != nil {
		return err
	}
	shutdownHandler.RegisterFunc("grpc-client", func() error {
		return grpcClient.Close()
	})

	appLogger.Debug("requesting order status",
		zap.String("order_id", appLogger.Redact(orderID)),
	)

	resp, err := grpcClient.GetOrderStatus(ctx, orderID)
	if err != nil {
		appLogger.ErrorRedact("failed to get order status", err,
			zap.String("order_id", orderID),
		)
		return err
	}

	fmt.Printf("✓ Order status retrieved\n")

	if resp.Order != nil {
		fmt.Printf("  Order ID: %s\n", resp.Order.OrderId)
		fmt.Printf("  Status:   %s\n", resp.Order.Status)
		fmt.Printf("  Price:    %s\n", resp.Order.Price)
		fmt.Printf("  Quantity: %s\n", resp.Order.Quantity)
	} else {
		fmt.Printf("  Order data not available\n")
	}

	shutdownHandler.Trigger()
	shutdownHandler.WaitForCompletion()

	return nil
}
