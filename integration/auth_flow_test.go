//go:build integration

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	orderv1 "exchange-system/proto/order/v1"
	userv1 "exchange-system/proto/user/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func TestAuthFlow_RegisterLoginCreateOrder(t *testing.T) {
	// Пропускаем если не integration тест
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	userServiceAddr := getEnv("USER_SERVICE_ADDR", "localhost:50053")
	orderServiceAddr := getEnv("ORDER_SERVICE_ADDR", "localhost:50052")

	ctx := context.Background()

	//Регистрация
	t.Log("Step 1: Registering new user...")

	userConn, err := grpc.NewClient(userServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect to user-service: %v", err)
	}
	defer userConn.Close()

	userClient := userv1.NewUserServiceClient(userConn)

	// Генерируем уникальные данные для теста
	testEmail := fmt.Sprintf("integration-test-%d@example.com", time.Now().Unix())
	testPassword := "IntegrationTest123!"
	testUsername := "integrationtest"

	registerReq := &userv1.RegisterRequest{
		Email:    testEmail,
		Password: testPassword,
		Username: testUsername,
	}

	registerResp, err := userClient.Register(ctx, registerReq)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	userID := registerResp.User.UserId
	t.Logf("Registered user: %s", userID)
	t.Logf("Email:    %s", registerResp.User.Email)
	t.Logf("Username: %s", registerResp.User.Username)

	if userID == "" {
		t.Error("UserID should not be empty")
	}
	if registerResp.Token.AccessToken == "" {
		t.Error("AccessToken should not be empty after registration")
	}

	accessToken := registerResp.Token.AccessToken

	//Логин (проверяем что можем войти)
	t.Log("Step 2: Logging in with credentials...")

	loginReq := &userv1.LoginRequest{
		Email:    testEmail,
		Password: testPassword,
	}

	loginResp, err := userClient.Login(ctx, loginReq)
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	t.Logf("Login successful")
	t.Logf("UserID: %s", loginResp.User.UserId)

	if loginResp.User.UserId != userID {
		t.Errorf("Login UserID = %q, want %q", loginResp.User.UserId, userID)
	}
	if loginResp.Token.AccessToken == "" {
		t.Error("AccessToken should not be empty after login")
	}

	//Создание ордера с авторизацией
	t.Log("Step 3: Creating order with JWT token...")

	orderConn, err := grpc.NewClient(orderServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect to order-service: %v", err)
	}
	defer orderConn.Close()

	orderClient := orderv1.NewOrderServiceClient(orderConn)

	// Добавляем токен в metadata
	ctxWithAuth := metadata.NewOutgoingContext(ctx, metadata.Pairs("authorization", "Bearer "+accessToken))

	createReq := &orderv1.CreateOrderRequest{
		MarketId: "BTC_USD",
		Type:     orderv1.OrderType_ORDER_TYPE_BUY,
		Price:    "50000",
		Quantity: "0.001",
	}

	createResp, err := orderClient.CreateOrder(ctxWithAuth, createReq)
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}

	t.Logf("Order created: %s", createResp.OrderId)
	t.Logf("Status: %s", createResp.Status)

	if createResp.OrderId == "" {
		t.Error("OrderId should not be empty")
	}
	if createResp.Status == "" {
		t.Error("Status should not be empty")
	}
}

func TestAuthFlow_UnauthorizedRequestFails(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	orderServiceAddr := getEnv("ORDER_SERVICE_ADDR", "localhost:50052")
	ctx := context.Background()

	conn, err := grpc.NewClient(orderServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect to order-service: %v", err)
	}
	defer conn.Close()

	client := orderv1.NewOrderServiceClient(conn)

	// Пытаемся создать ордер БЕЗ токена
	t.Log("Attempting to create order without authentication...")

	_, err = client.CreateOrder(ctx, &orderv1.CreateOrderRequest{
		MarketId: "BTC_USD",
		Type:     orderv1.OrderType_ORDER_TYPE_BUY,
		Price:    "50000",
		Quantity: "0.001",
	})

	// Ожидаем ошибку аутентификации
	if err == nil {
		t.Error("CreateOrder() should fail without authentication")
	} else {
		t.Logf("✓ Expected error received: %v", err)
	}
}
