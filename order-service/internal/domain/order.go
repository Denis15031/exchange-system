package domain

import (
	"github.com/shopspring/decimal"
	"time"
)

type OrderType string

const (
	OrderTypeBuy  OrderType = "BUY"
	OrderTypeSell OrderType = "SELL"
)

type OrderStatus string

const (
	OrderStatusCreated         OrderStatus = "CREATED"
	OrderStatusPending         OrderStatus = "PENDING"
	OrderStatusFilled          OrderStatus = "FILLED"
	OrderStatusPartiallyFilled OrderStatus = "PARTIALLY_FILLED"
	OrderStatusCanceled        OrderStatus = "CANCELLED"
	OrderStatusRejected        OrderStatus = "REJECTED"
)

type Order struct {
	ID             string          `json:"id"`
	UserID         string          `json:"user_id"`
	MarketID       string          `json:"market_id"`
	Type           OrderType       `json:"type"`
	Price          decimal.Decimal `json:"price"`
	Quantity       decimal.Decimal `json:"quantity"`
	FilledQuantity decimal.Decimal `json:"filled_quantity"`
	Status         OrderStatus     `json:"status"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type Market struct {
	ID             string
	Symbol         string
	BaseCurrency   string
	QuoteCurrency  string
	Enabled        bool
	DeletedAt      *time.Time
	MinOrderSize   decimal.Decimal
	MaxOrderSize   decimal.Decimal
	PriceIncrement decimal.Decimal
	SizeIncrement  decimal.Decimal
	AllowedRoles   []string
}

func (m *Market) IsActive() bool {
	return m.Enabled && m.DeletedAt == nil
}
