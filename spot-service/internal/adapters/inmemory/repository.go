package inmemory

import (
	"context"
	"errors"
	"sync"
	"time"

	"exchange-system/spot-service/internal/domain"
	"exchange-system/spot-service/internal/ports"

	"github.com/shopspring/decimal"
)

var (
	ErrMarketNotFound = errors.New("market not found")
)

type Repository struct {
	mu      sync.RWMutex
	markets map[string]domain.Market
}

func NewRepository() ports.MarketRepository {
	repo := &Repository{
		markets: make(map[string]domain.Market),
	}

	repo.seedData()

	return repo
}

func (r *Repository) seedData() {
	now := time.Now()

	btcUsd := domain.Market{
		ID:             "btc_usd",
		Symbol:         "BTC_USD",
		BaseCurrency:   "BTC",
		QuoteCurrency:  "USD",
		Enabled:        true,
		DeletedAt:      nil,
		MinOrderSize:   decimal.RequireFromString("0.0001"),
		MaxOrderSize:   decimal.RequireFromString("100.0"),
		PriceIncrement: decimal.RequireFromString("0.01"),
		SizeIncrement:  decimal.RequireFromString("0.0001"),
	}

	ethUsd := domain.Market{
		ID:             "eth_usd",
		Symbol:         "ETH_USD",
		BaseCurrency:   "ETH",
		QuoteCurrency:  "USD",
		Enabled:        true,
		DeletedAt:      nil,
		MinOrderSize:   decimal.RequireFromString("0.0001"),
		MaxOrderSize:   decimal.RequireFromString("100.0"),
		PriceIncrement: decimal.RequireFromString("0.01"),
		SizeIncrement:  decimal.RequireFromString("0.0001"),
	}

	deletedTime := now.AddDate(-1, 0, 0)

	deletedMarket := domain.Market{
		ID:             "old_token",
		Symbol:         "OLD_USD",
		BaseCurrency:   "OLD",
		QuoteCurrency:  "USD",
		Enabled:        false,
		DeletedAt:      &deletedTime,
		MinOrderSize:   decimal.RequireFromString("1.0"),
		MaxOrderSize:   decimal.RequireFromString("10.0"),
		PriceIncrement: decimal.RequireFromString("0.01"),
		SizeIncrement:  decimal.RequireFromString("1.0"),
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.markets["btc_usd"] = btcUsd
	r.markets["eth_usd"] = ethUsd
	r.markets["old_token"] = deletedMarket
}

func (r *Repository) GetAll(ctx context.Context) ([]domain.Market, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]domain.Market, 0, len(r.markets))
	for _, m := range r.markets {
		cp := m
		result = append(result, cp)
	}
	return result, nil
}

func (r *Repository) GetByID(ctx context.Context, id string) (*domain.Market, error) {

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	market, ok := r.markets[id]
	if !ok {
		return nil, ErrMarketNotFound
	}

	cp := market
	return &cp, nil
}
