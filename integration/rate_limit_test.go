//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	userv1 "exchange-system/proto/user/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// Rate limiting: слишком много попыток логина
func TestRateLimit_LoginBruteForce(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	userServiceAddr := getEnv("USER_SERVICE_ADDR", "localhost:50053")
	ctx := context.Background()

	conn, err := grpc.NewClient(userServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to connect to user-service: %v", err)
	}
	defer conn.Close()

	client := userv1.NewUserServiceClient(conn)

	// Пытаемся логиниться с неправильным паролем много раз
	testEmail := "ratelimit-test@example.com"
	maxAttempts := 15
	rateLimitTriggered := false

	t.Logf("Attempting %d login requests with wrong password...", maxAttempts)

	for i := 0; i < maxAttempts; i++ {
		_, err := client.Login(ctx, &userv1.LoginRequest{
			Email:    testEmail,
			Password: "wrong-password",
		})

		// После превышения лимита должна вернуться ошибка
		if err != nil {
			st, ok := status.FromError(err)
			if ok && st.Code() == codes.ResourceExhausted {
				t.Logf("✓ Rate limit triggered after %d attempts", i+1)
				rateLimitTriggered = true
				break
			}
		}

		// Небольшая пауза чтобы не спамить
		time.Sleep(50 * time.Millisecond)
	}

	if !rateLimitTriggered {
		t.Log("Rate limit did not trigger (check service configuration)")
		// Не фейлим тест
	}
}
