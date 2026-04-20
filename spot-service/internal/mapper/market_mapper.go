package mapper

import (
	"time"

	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/types/known/timestamppb"

	spotv1 "exchange-system/proto/spot/v1"
	userv1 "exchange-system/proto/user/v1"
	"exchange-system/spot-service/internal/domain"
)

func ToProto(m *domain.Market) *spotv1.Market {
	if m == nil {
		return nil
	}
	return &spotv1.Market{
		MarketId:       m.ID,
		Symbol:         m.Symbol,
		BaseCurrency:   m.BaseCurrency,
		QuoteCurrency:  m.QuoteCurrency,
		Enabled:        m.Enabled,
		MinOrderSize:   m.MinOrderSize.String(),
		MaxOrderSize:   m.MaxOrderSize.String(),
		PriceIncrement: m.PriceIncrement.String(),
		SizeIncrement:  m.SizeIncrement.String(),
		CreatedAt:      toProtoTimestamp(m.CreatedAt),
		UpdatedAt:      toProtoTimestamp(m.UpdatedAt),
		DeletedAt:      toProtoTimestampPtr(m.DeletedAt),
	}
}

func ToDomain(m *spotv1.Market) *domain.Market {
	if m == nil {
		return nil
	}
	return &domain.Market{
		ID:             m.MarketId,
		Symbol:         m.Symbol,
		BaseCurrency:   m.BaseCurrency,
		QuoteCurrency:  m.QuoteCurrency,
		Enabled:        m.Enabled,
		MinOrderSize:   mustDecimal(m.MinOrderSize),
		MaxOrderSize:   mustDecimal(m.MaxOrderSize),
		PriceIncrement: mustDecimal(m.PriceIncrement),
		SizeIncrement:  mustDecimal(m.SizeIncrement),
		CreatedAt:      toDomainTime(m.CreatedAt),
		UpdatedAt:      toDomainTime(m.UpdatedAt),
		DeletedAt:      toDomainTimePtr(m.DeletedAt),
	}
}

func toProtoTimestamp(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

func toProtoTimestampPtr(t *time.Time) *timestamppb.Timestamp {
	if t == nil || t.IsZero() {
		return nil
	}
	return timestamppb.New(*t)
}

func toDomainTime(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}

func toDomainTimePtr(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	t := ts.AsTime()
	return &t
}

func mustDecimal(s string) decimal.Decimal {
	d, _ := decimal.NewFromString(s)
	return d
}

func UserRolesFromProto(roles []userv1.UserRole) []string {
	result := make([]string, 0, len(roles))
	for _, r := range roles {
		result = append(result, r.String())
	}
	return result
}
