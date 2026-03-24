package service

import (
	"context"
	"exchange-system/spot-service/internal/domain"
	"exchange-system/spot-service/internal/ports"
)

type SpotService struct {
	repo ports.MarketRepository
}

func NewSpotService(repo ports.MarketRepository) *SpotService {
	return &SpotService{repo: repo}
}

// Возвращает только активные
func (s *SpotService) ViewMarkets(ctx context.Context, userRoles []string) ([]domain.Market, error) {
	allMarkets, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	activeMarkets := make([]domain.Market, 0)
	for _, m := range allMarkets {
		if !m.IsActive() {
			continue
		}
		if len(m.AllowedRoles) > 0 {
			hasAccess := false
			for _, allowedRole := range m.AllowedRoles {
				for _, uRole := range userRoles {
					if allowedRole == uRole {
						hasAccess = true
						break
					}
				}
				if hasAccess {
					break
				}
			}
			if !hasAccess {
				continue
			}
		}

		activeMarkets = append(activeMarkets, m)
	}
	return activeMarkets, nil
}

// Возвращает рынок если он существует
func (s *SpotService) GetMarketByID(ctx context.Context, id string) (*domain.Market, error) {
	return s.repo.GetByID(ctx, id)
}
