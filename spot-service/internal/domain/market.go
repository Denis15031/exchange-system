package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// Представляет торговую пару
type Market struct {
	ID             string          `json:"id"`
	Symbol         string          `json:"symbol"`         // BTC_USD
	BaseCurrency   string          `json:"base_currency"`  // BTC
	QuoteCurrency  string          `json:"quote_currency"` // USD
	Enabled        bool            `json:"enabled"`
	DeletedAt      *time.Time      `json:"deleted_at,omitempty"` // nil если активен
	MinOrderSize   decimal.Decimal `json:"min_order_size"`
	MaxOrderSize   decimal.Decimal `json:"max_order_size"`
	PriceIncrement decimal.Decimal `json:"price_increment"`
	SizeIncrement  decimal.Decimal `json:"size_increment"`
	AllowedRoles   []string        `json:"allowed_roles"`
}

// Проверяет активен ли рынок
func (m *Market) IsActive() bool {
	return m.Enabled && m.DeletedAt == nil
}
