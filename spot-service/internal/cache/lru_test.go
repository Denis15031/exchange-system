package cache

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"exchange-system/spot-service/internal/domain"

	"github.com/shopspring/decimal"
)

func newTestMarket(id, symbol string) *domain.Market {
	return &domain.Market{
		ID:             id,
		Symbol:         symbol,
		BaseCurrency:   "BTC",
		QuoteCurrency:  "USD",
		Enabled:        true,
		MinOrderSize:   decimal.NewFromFloat(0.001),
		MaxOrderSize:   decimal.NewFromFloat(100),
		PriceIncrement: decimal.NewFromFloat(0.01),
		SizeIncrement:  decimal.NewFromFloat(0.00001),
		AllowedRoles:   []string{"USER", "PREMIUM", "ADMIN"},
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

func marketToBytes(m *domain.Market) []byte {
	data, _ := json.Marshal(m)
	return data
}

func bytesToMarket(data []byte) *domain.Market {
	var m domain.Market
	_ = json.Unmarshal(data, &m)
	return &m
}

func TestLRUCache_GetSet(t *testing.T) {
	ctx := context.Background()
	cache := NewLRUCache(100)

	market := newTestMarket("BTC-USD", "BTC_USD")

	_ = cache.Set(ctx, "btc", marketToBytes(market), 300)

	data, found, err := cache.Get(ctx, "btc")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !found {
		t.Fatal("expected found=true")
	}

	got := bytesToMarket(data)
	if got.ID != "BTC-USD" {
		t.Errorf("got ID %q, want %q", got.ID, "BTC-USD")
	}
	if got.Symbol != "BTC_USD" {
		t.Errorf("got Symbol %q, want %q", got.Symbol, "BTC_USD")
	}
}

func TestLRUCache_GetMissing(t *testing.T) {
	ctx := context.Background()
	cache := NewLRUCache(100)

	_, found, err := cache.Get(ctx, "missing")
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}
	if found {
		t.Error("expected found=false for missing key")
	}
}

func TestLRUCache_TTL(t *testing.T) {
	ctx := context.Background()
	cache := NewLRUCache(100)

	cache.Set(ctx, "temp", marketToBytes(newTestMarket("TMP", "TMP_USD")), 1) // 1 секунда

	_, found, _ := cache.Get(ctx, "temp")
	if !found {
		t.Error("expected value immediately after Set")
	}

	time.Sleep(1100 * time.Millisecond)

	_, found, _ = cache.Get(ctx, "temp")
	if found {
		t.Error("expected value to expire after TTL")
	}
}

func TestLRUCache_Eviction(t *testing.T) {
	ctx := context.Background()

	cache := NewLRUCache(2)

	cache.Set(ctx, "first", marketToBytes(newTestMarket("1", "FIRST")), 300)
	cache.Set(ctx, "second", marketToBytes(newTestMarket("2", "SECOND")), 300)

	_, _, _ = cache.Get(ctx, "first")

	cache.Set(ctx, "third", marketToBytes(newTestMarket("3", "THIRD")), 300)

	_, found, _ := cache.Get(ctx, "second")
	if found {
		t.Error("expected 'second' to be evicted")
	}

	_, found, _ = cache.Get(ctx, "first")
	if !found {
		t.Error("expected 'first' to still be in cache")
	}

	_, found, _ = cache.Get(ctx, "third")
	if !found {
		t.Error("expected 'third' to be in cache")
	}
}

func TestLRUCache_Overwrite(t *testing.T) {
	ctx := context.Background()
	cache := NewLRUCache(100)

	cache.Set(ctx, "key", marketToBytes(newTestMarket("OLD", "OLD_USD")), 300)
	cache.Set(ctx, "key", marketToBytes(newTestMarket("NEW", "NEW_USD")), 300)

	data, found, err := cache.Get(ctx, "key")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !found {
		t.Fatal("expected found=true")
	}

	got := bytesToMarket(data)
	if got.ID != "NEW" {
		t.Errorf("got ID %q, want %q", got.ID, "NEW")
	}
	if got.Symbol != "NEW_USD" {
		t.Errorf("got Symbol %q, want %q", got.Symbol, "NEW_USD")
	}
}
