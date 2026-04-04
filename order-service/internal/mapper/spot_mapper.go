package mapper

import (
	"exchange-system/order-service/internal/domain"
	spotV1 "exchange-system/proto/spot/v1"
	"github.com/shopspring/decimal"
)

func ToDomainMarket(pb *spotV1.Market) *domain.Market {
	if pb == nil {
		return nil
	}

	minSize, _ := decimal.NewFromString(pb.MinOrderSize)
	maxSize, _ := decimal.NewFromString(pb.MaxOrderSize)
	priceInc, _ := decimal.NewFromString(pb.PriceIncrement)
	sizeInc, _ := decimal.NewFromString(pb.SizeIncrement)

	m := &domain.Market{
		ID:             pb.MarketId,
		Symbol:         pb.Symbol,
		BaseCurrency:   pb.BaseCurrency,
		QuoteCurrency:  pb.QuoteCurrency,
		Enabled:        pb.Enabled,
		MinOrderSize:   minSize,
		MaxOrderSize:   maxSize,
		PriceIncrement: priceInc,
		SizeIncrement:  sizeInc,
		DeletedAt:      nil,
		AllowedRoles:   []string{},
	}

	return m
}
