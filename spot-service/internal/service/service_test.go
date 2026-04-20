package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"exchange-system/spot-service/internal/domain"

	"go.uber.org/zap/zaptest"
)

type MockCache struct {
	data map[string][]byte
}

func (m *MockCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if val, ok := m.data[key]; ok {
		return val, true, nil
	}
	return nil, false, nil
}

func (m *MockCache) Set(ctx context.Context, key string, value []byte, ttlSeconds int) error {
	if m.data == nil {
		m.data = make(map[string][]byte)
	}
	m.data[key] = value
	return nil
}

func (m *MockCache) Delete(ctx context.Context, key string) error {
	if m.data != nil {
		delete(m.data, key)
	}
	return nil
}

func (m *MockCache) Close() error                  { return nil }
func (m *MockCache) Stats() map[string]interface{} { return nil }

type MockRepo struct {
	markets map[string]*domain.Market
}

func NewMockRepo() *MockRepo {
	return &MockRepo{markets: make(map[string]*domain.Market)}
}

func (m *MockRepo) GetAll(ctx context.Context) ([]domain.Market, error) {
	result := make([]domain.Market, 0, len(m.markets))
	for _, mk := range m.markets {
		result = append(result, *mk)
	}
	return result, nil
}

func (m *MockRepo) GetByID(ctx context.Context, id string) (*domain.Market, error) {
	if mk, ok := m.markets[id]; ok {
		return mk, nil
	}
	return nil, nil
}

func (m *MockRepo) Add(market *domain.Market) {
	m.markets[market.ID] = market
}

func TestViewMarkets_FilterEnabled(t *testing.T) {
	t.Parallel()

	repo := NewMockRepo()
	repo.Add(&domain.Market{ID: "BTC", Symbol: "BTC_USD", Enabled: true, DeletedAt: nil, AllowedRoles: []string{"USER"}})
	repo.Add(&domain.Market{ID: "ETH", Symbol: "ETH_USD", Enabled: false, DeletedAt: nil, AllowedRoles: []string{"USER"}}) // выключен
	repo.Add(&domain.Market{ID: "LTC", Symbol: "LTC_USD", Enabled: true, DeletedAt: nil, AllowedRoles: []string{"USER"}})

	cache := &MockCache{data: make(map[string][]byte)}
	logger := zaptest.NewLogger(t)

	svc := NewSpotService(repo, cache, logger)

	result, _, _, err := svc.ViewMarkets(context.Background(), []string{"USER"}, nil)
	if err != nil {
		t.Fatalf("ViewMarkets() error = %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 active markets, got %d", len(result))
	}
	for _, m := range result {
		if m.ID == "ETH" {
			t.Error("Disabled market should not be in result")
		}
	}
}

func TestViewMarkets_FilterDeleted(t *testing.T) {
	t.Parallel()

	deletedTime := time.Now().Add(-1 * time.Hour)
	repo := NewMockRepo()
	repo.Add(&domain.Market{ID: "ACTIVE", Symbol: "ACTIVE", Enabled: true, DeletedAt: nil, AllowedRoles: []string{"USER"}})
	repo.Add(&domain.Market{ID: "DEL", Symbol: "DEL", Enabled: true, DeletedAt: &deletedTime, AllowedRoles: []string{"USER"}})

	cache := &MockCache{data: make(map[string][]byte)}
	logger := zaptest.NewLogger(t)

	svc := NewSpotService(repo, cache, logger)

	result, _, _, err := svc.ViewMarkets(context.Background(), []string{"USER"}, nil)
	if err != nil {
		t.Fatalf("ViewMarkets() error = %v", err)
	}

	if len(result) != 1 {
		t.Errorf("Expected 1 non-deleted market, got %d", len(result))
	}
	if result[0].ID != "ACTIVE" {
		t.Errorf("Expected ACTIVE market, got %s", result[0].ID)
	}
}

func TestViewMarkets_FilterByRoles(t *testing.T) {
	t.Parallel()

	repo := NewMockRepo()
	repo.Add(&domain.Market{ID: "PUB", Symbol: "PUB", Enabled: true, AllowedRoles: []string{"USER", "ADMIN"}})
	repo.Add(&domain.Market{ID: "ADM", Symbol: "ADM", Enabled: true, AllowedRoles: []string{"ADMIN"}})

	cache := &MockCache{data: make(map[string][]byte)}
	logger := zaptest.NewLogger(t)

	svc := NewSpotService(repo, cache, logger)

	result, _, _, err := svc.ViewMarkets(context.Background(), []string{"USER"}, nil)
	if err != nil {
		t.Fatalf("ViewMarkets() error = %v", err)
	}
	if len(result) != 1 || result[0].ID != "PUB" {
		t.Errorf("USER should see only PUBLIC, got %d markets", len(result))
	}
}

func TestGetMarketByID_CacheHit(t *testing.T) {
	t.Parallel()

	// ✅ Используем валидный UUID
	marketID := "550e8400-e29b-41d4-a716-446655440000"

	repo := NewMockRepo()
	originalMarket := &domain.Market{ID: marketID, Symbol: "BTC_USD", Enabled: true}
	repo.Add(originalMarket)

	cache := &MockCache{data: make(map[string][]byte)}

	// Предзаполняем кэш (эмуляция предыдущего запроса)
	jsonBytes, _ := json.Marshal(originalMarket)
	cache.data[marketID] = jsonBytes

	logger := zaptest.NewLogger(t)
	svc := NewSpotService(repo, cache, logger)

	market, err := svc.GetMarketByID(context.Background(), marketID)
	if err != nil {
		t.Fatalf("GetMarketByID() error = %v", err)
	}
	if market.ID != marketID {
		t.Errorf("Expected market ID %s from cache, got %s", marketID, market.ID)
	}
}

func TestGetMarketByID_CacheMissAndSet(t *testing.T) {
	t.Parallel()

	// ✅ Используем валидный UUID
	marketID := "6ba7b810-9dad-11d1-80b4-00c04fd430c8"

	repo := NewMockRepo()
	originalMarket := &domain.Market{ID: marketID, Symbol: "ETH_USD", Enabled: true}
	repo.Add(originalMarket)

	cache := &MockCache{data: make(map[string][]byte)}
	logger := zaptest.NewLogger(t)
	svc := NewSpotService(repo, cache, logger)

	// Первый запрос: кэш пуст, идём в репо, сохраняем в кэш
	market, err := svc.GetMarketByID(context.Background(), marketID)
	if err != nil {
		t.Fatalf("GetMarketByID() error = %v", err)
	}
	if market.ID != marketID {
		t.Errorf("Expected market ID %s from repo, got %s", marketID, market.ID)
	}

	// Проверяем, что данные попали в кэш
	cachedBytes, found, _ := cache.Get(context.Background(), marketID)
	if !found {
		t.Error("Expected market to be saved in cache")
	}

	// Проверяем, что в кэше лежит валидный JSON
	var cachedMarket domain.Market
	if err := json.Unmarshal(cachedBytes, &cachedMarket); err != nil {
		t.Fatalf("Cache contains invalid JSON: %v", err)
	}
	if cachedMarket.ID != marketID {
		t.Errorf("Cache JSON mismatch: expected %s, got %s", marketID, cachedMarket.ID)
	}
}
