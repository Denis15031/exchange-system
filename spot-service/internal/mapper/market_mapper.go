package mapper

import (
	"time"

	spotv1 "exchange-system/proto/spot/v1"
	userv1 "exchange-system/proto/user/v1"
	"exchange-system/spot-service/internal/domain"
	"github.com/shopspring/decimal"
)

func ToProto(m *domain.Market) *spotv1.Market {
	if m == nil {
		return nil
	}

	var deletedAt int64 = 0
	if m.DeletedAt != nil {
		deletedAt = m.DeletedAt.Unix()
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
		DeletedAt:      deletedAt,
	}
}

func ToDomain(p *spotv1.Market) *domain.Market {
	if p == nil {
		return nil
	}

	minSize, _ := decimal.NewFromString(p.MinOrderSize)
	maxSize, _ := decimal.NewFromString(p.MaxOrderSize)
	priceInc, _ := decimal.NewFromString(p.PriceIncrement)
	sizeInc, _ := decimal.NewFromString(p.SizeIncrement)

	var deletedAt *time.Time
	if p.DeletedAt != 0 {
		t := time.Unix(p.DeletedAt, 0)
		deletedAt = &t
	}

	return &domain.Market{
		ID:             p.MarketId,
		Symbol:         p.Symbol,
		BaseCurrency:   p.BaseCurrency,
		QuoteCurrency:  p.QuoteCurrency,
		Enabled:        p.Enabled,
		MinOrderSize:   minSize,
		MaxOrderSize:   maxSize,
		PriceIncrement: priceInc,
		SizeIncrement:  sizeInc,
		DeletedAt:      deletedAt,
		AllowedRoles:   []string{},
	}
}

func UserRolesFromProto(protoRoles []userv1.UserRole) []string {
	roles := make([]string, 0, len(protoRoles))
	for _, r := range protoRoles {
		switch r {
		case userv1.UserRole_USER_ROLE_ADMIN:
			roles = append(roles, "ADMIN")
		case userv1.UserRole_USER_ROLE_MODERATOR:
			roles = append(roles, "MODERATOR")
		case userv1.UserRole_USER_ROLE_PREMIUM:
			roles = append(roles, "PREMIUM")
		case userv1.UserRole_USER_ROLE_USER:
			roles = append(roles, "USER")
		default:
			roles = append(roles, "UNSPECIFIED")
		}
	}
	return roles
}
