package auth

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"

	userv1 "exchange-system/proto/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type TokenManager struct {
	mu              sync.RWMutex
	accessToken     string
	refreshToken    string
	expiresAt       time.Time
	storagePath     string
	userServiceConn *grpc.ClientConn
}

func NewTokenManager(userServiceAddr, storagePath string) (*TokenManager, error) {
	conn, err := grpc.NewClient(userServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	tm := &TokenManager{
		storagePath:     storagePath,
		userServiceConn: conn,
	}

	_ = tm.loadFromDisk()

	return tm, nil
}

func (tm *TokenManager) Login(email, password string) (*userv1.LoginResponse, error) {
	client := userv1.NewUserServiceClient(tm.userServiceConn)

	resp, err := client.Login(context.Background(), &userv1.LoginRequest{
		Email:    email,
		Password: password,
	})
	if err != nil {
		return nil, err
	}

	tm.mu.Lock()
	tm.accessToken = resp.Token.AccessToken
	tm.refreshToken = resp.Token.RefreshToken
	tm.expiresAt = time.Unix(resp.Token.ExpiresAt, 0)
	tm.mu.Unlock()

	_ = tm.saveToDisk()

	return resp, nil
}

func (tm *TokenManager) Register(email, password, username string) (*userv1.RegisterResponse, error) {
	client := userv1.NewUserServiceClient(tm.userServiceConn)

	resp, err := client.Register(context.Background(), &userv1.RegisterRequest{
		Email:    email,
		Password: password,
		Username: username,
	})
	if err != nil {
		return nil, err
	}

	tm.mu.Lock()
	tm.accessToken = resp.Token.AccessToken
	tm.refreshToken = resp.Token.RefreshToken
	tm.expiresAt = time.Unix(resp.Token.ExpiresAt, 0)
	tm.mu.Unlock()

	_ = tm.saveToDisk()

	return resp, nil
}

func (tm *TokenManager) GetAccessToken() (string, error) {
	tm.mu.RLock()

	if time.Now().Add(1 * time.Minute).Before(tm.expiresAt) {
		token := tm.accessToken
		tm.mu.RUnlock()
		return token, nil
	}
	tm.mu.RUnlock()

	// Токен истёк — обновляем
	return tm.RefreshToken()
}

func (tm *TokenManager) RefreshToken() (string, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.refreshToken == "" {
		return "", errors.New("no refresh token available")
	}

	client := userv1.NewUserServiceClient(tm.userServiceConn)

	resp, err := client.RefreshToken(context.Background(), &userv1.RefreshTokenRequest{
		RefreshToken: tm.refreshToken,
	})
	if err != nil {
		return "", err
	}

	tm.accessToken = resp.Token.AccessToken
	tm.expiresAt = time.Unix(resp.Token.ExpiresAt, 0)

	_ = tm.saveToDisk()

	return tm.accessToken, nil
}

func (tm *TokenManager) Logout() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.refreshToken != "" {
		client := userv1.NewUserServiceClient(tm.userServiceConn)
		_, _ = client.Logout(context.Background(), &userv1.LogoutRequest{
			RefreshToken: tm.refreshToken,
		})
	}

	tm.accessToken = ""
	tm.refreshToken = ""
	tm.expiresAt = time.Time{}

	_ = os.Remove(tm.storagePath)

	return nil
}

func (tm *TokenManager) HasToken() bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.accessToken != ""
}

func (tm *TokenManager) saveToDisk() error {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	data := map[string]string{
		"access_token":  tm.accessToken,
		"refresh_token": tm.refreshToken,
		"expires_at":    tm.expiresAt.Format(time.RFC3339),
	}

	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(tm.storagePath, bytes, 0600)
}

func (tm *TokenManager) loadFromDisk() error {
	data, err := os.ReadFile(tm.storagePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Файла нет — это нормально
		}
		return err
	}

	var stored map[string]string
	if err := json.Unmarshal(data, &stored); err != nil {
		return err
	}

	tm.mu.Lock()
	tm.accessToken = stored["access_token"]
	tm.refreshToken = stored["refresh_token"]
	if expires, err := time.Parse(time.RFC3339, stored["expires_at"]); err == nil {
		tm.expiresAt = expires
	}
	tm.mu.Unlock()

	return nil
}

func (tm *TokenManager) Close() error {
	if tm.userServiceConn != nil {
		return tm.userServiceConn.Close()
	}
	return nil
}
