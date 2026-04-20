package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

type Market struct {
	ID             string          `json:"id"`
	Symbol         string          `json:"symbol"`
	BaseCurrency   string          `json:"base_currency"`
	QuoteCurrency  string          `json:"quote_currency"`
	Enabled        bool            `json:"enabled"`
	DeletedAt      *time.Time      `json:"deleted_at,omitempty"`
	MinOrderSize   decimal.Decimal `json:"min_order_size"`
	MaxOrderSize   decimal.Decimal `json:"max_order_size"`
	PriceIncrement decimal.Decimal `json:"price_increment"`
	SizeIncrement  decimal.Decimal `json:"size_increment"`
	AllowedRoles   []string        `json:"allowed_roles"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

func (m *Market) IsActive() bool {
	return m.Enabled && m.DeletedAt == nil
}
