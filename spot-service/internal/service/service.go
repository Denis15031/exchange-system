package service

import (
	"context"
	"encoding/json"
	"time"

	commonv1 "exchange-system/proto/common"
	sharedports "exchange-system/shared/ports"
	"exchange-system/shared/uid"
	"exchange-system/spot-service/internal/domain"
	spotports "exchange-system/spot-service/internal/ports"

	"go.uber.org/zap"
)

type SpotService struct {
	repo   spotports.MarketRepository
	cache  sharedports.Cache
	logger *zap.Logger
}

func NewSpotService(
	repo spotports.MarketRepository,
	cache sharedports.Cache,
	logger *zap.Logger,
) *SpotService {
	return &SpotService{
		repo:   repo,
		cache:  cache,
		logger: logger,
	}
}

func (s *SpotService) ViewMarkets(
	ctx context.Context,
	userRoles []string,
	pagination *commonv1.CursorPaginationRequest,
) ([]domain.Market, string, bool, error) {
	start := time.Now()
	s.logger.Debug("ViewMarkets started",
		zap.Strings("user_roles", userRoles),
		zap.Any("pagination", pagination),
	)

	if err := ctx.Err(); err != nil {
		return nil, "", false, err
	}

	allMarkets, err := s.repo.GetAll(ctx)
	if err != nil {
		s.logger.Error("failed to fetch all markets", zap.Error(err))
		return nil, "", false, err
	}

	filtered := make([]domain.Market, 0, len(allMarkets))
	for _, market := range allMarkets {
		if !market.Enabled {
			continue
		}
		if market.DeletedAt != nil {
			continue
		}
		if len(userRoles) > 0 && len(market.AllowedRoles) > 0 {
			hasAccess := false
			for _, userRole := range userRoles {
				for _, allowedRole := range market.AllowedRoles {
					if userRole == allowedRole {
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
		filtered = append(filtered, market)
	}

	elapsed := time.Since(start)
	s.logger.Info("ViewMarkets finished",
		zap.Int("total_markets", len(allMarkets)),
		zap.Int("filtered_markets", len(filtered)),
		zap.Duration("took", elapsed),
	)

	return filtered, "", false, nil
}

func (s *SpotService) GetMarketByID(ctx context.Context, id string) (*domain.Market, error) {

	if err := uid.Validate(id, "market_id"); err != nil {
		return nil, err
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if s.cache != nil {
		data, found, err := s.cache.Get(ctx, id)
		if err != nil {
			s.logger.Warn("cache get failed, falling back to DB",
				zap.String("id", id),
				zap.Error(err),
			)
		} else if found {

			var market domain.Market
			if err := json.Unmarshal(data, &market); err != nil {

				s.logger.Warn("cache unmarshal failed, deleting key",
					zap.String("id", id),
					zap.Error(err),
				)
				_ = s.cache.Delete(ctx, id) // Игнорируем ошибку удаления
			} else {
				s.logger.Debug("cache hit", zap.String("id", id))
				return &market, nil
			}
		}
	}

	s.logger.Debug("repo lookup", zap.String("id", id))
	market, err := s.repo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("repo failed", zap.String("id", id), zap.Error(err))
		return nil, err
	}

	if s.cache != nil && market != nil {
		jsonBytes, marshalErr := json.Marshal(market)
		if marshalErr != nil {
			s.logger.Error("cache marshal failed", zap.Error(marshalErr))
		} else {

			_ = s.cache.Set(ctx, id, jsonBytes, 300)
		}
	}

	s.logger.Info("GetMarketByID success", zap.String("id", id))
	return market, nil
}
