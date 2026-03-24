package mapper

import (
	"exchange-system/spot-service/internal/domain"
	v1 "exchange-system/spot-service/proto/spot/v1"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func ToProtoMarket(m domain.Market) *v1.Market {
	pbMarket := &v1.Market{
		Id:             m.ID,
		Symbol:         m.Symbol,
		BaseCurrency:   m.BaseCurrency,
		QuoteCurrency:  m.QuoteCurrency,
		Enabled:        m.Enabled,
		MinOrderSize:   m.MinOrderSize.String(),
		MaxOrderSize:   m.MaxOrderSize.String(),
		PriceIncrement: m.PriceIncrement.String(),
		SizeIncrement:  m.SizeIncrement.String(),
		AllowedRoles:   m.AllowedRoles,
	}

	if m.DeletedAt != nil {
		pbMarket.DeletedAt = timestamppb.New(*m.DeletedAt)
	}

	return pbMarket
}

func ToProtoMarketList(markets []domain.Market) []*v1.Market {
	if len(markets) == 0 {
		return []*v1.Market{}
	}

	pbMarkets := make([]*v1.Market, 0, len(markets))
	for _, m := range markets {
		pbMarkets = append(pbMarkets, ToProtoMarket(m))
	}
	return pbMarkets
}
