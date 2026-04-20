//go:build integration

package integration

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	userv1 "exchange-system/proto/user/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestIdempotency_RegisterSameKey(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	userServiceAddr := getEnv("USER_SERVICE_ADDR", "localhost:50053")

	conn, err := grpc.NewClient(userServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect to user-service: %v", err)
	}
	defer conn.Close()

	client := userv1.NewUserServiceClient(conn)

	testEmail := fmt.Sprintf("uniqueness-%d@example.com", time.Now().Unix())

	t.Logf("Testing email uniqueness with: %s", testEmail)

	req := &userv1.RegisterRequest{
		Email:    testEmail,
		Password: "UniquePass123!",
		Username: "uniqueuser",
	}

	resp1, err := client.Register(context.Background(), req)
	if err != nil {
		t.Fatalf("First Register() error = %v", err)
	}
	t.Logf("First registration: %s", resp1.User.UserId)

	_, err = client.Register(context.Background(), req)
	if err == nil {
		t.Error("Second Register() should fail with AlreadyExists")
	} else if !strings.Contains(err.Error(), "AlreadyExists") {
		t.Errorf("Expected AlreadyExists error, got: %v", err)
	} else {
		t.Logf("Duplicate email correctly rejected: %v", err)
	}

	t.Logf("User uniqueness constraint works!")
}
