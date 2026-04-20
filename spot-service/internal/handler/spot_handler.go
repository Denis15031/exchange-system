package handler

import (
	"context"

	commonv1 "exchange-system/proto/common"
	spotv1 "exchange-system/proto/spot/v1"
	"exchange-system/spot-service/internal/mapper"
	"exchange-system/spot-service/internal/service"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SpotHandler struct {
	spotv1.UnimplementedSpotInstrumentServiceServer
	spotService *service.SpotService
	logger      *zap.Logger
}

func NewSpotHandler(spotService *service.SpotService, logger *zap.Logger) *SpotHandler {
	return &SpotHandler{
		spotService: spotService,
		logger:      logger,
	}
}

func (h *SpotHandler) ViewMarkets(ctx context.Context, req *spotv1.ViewMarketsRequest) (*spotv1.ViewMarketsResponse, error) {
	h.logger.Debug("ViewMarkets request received",
		zap.Int("user_roles_count", len(req.GetUserRoles())),
	)

	userRoles := mapper.UserRolesFromProto(req.GetUserRoles())

	markets, nextPage, hasMore, err := h.spotService.ViewMarkets(ctx, userRoles, req.GetPagination())
	if err != nil {
		h.logger.Error("ViewMarkets failed", zap.Error(err))
		return nil, h.toGRPCError(err)
	}

	protoMarkets := make([]*spotv1.Market, 0, len(markets))
	for i := range markets {
		protoMarkets = append(protoMarkets, mapper.ToProto(&markets[i]))
	}

	h.logger.Debug("ViewMarkets success", zap.Int("markets_count", len(protoMarkets)))

	return &spotv1.ViewMarketsResponse{
		Markets: protoMarkets,
		Pagination: &commonv1.CursorPaginationResponse{
			NextPageToken: nextPage,
			HasMore:       hasMore,
		},
	}, nil
}

func (h *SpotHandler) GetMarket(ctx context.Context, req *spotv1.GetMarketRequest) (*spotv1.GetMarketResponse, error) {
	if req.MarketId == "" {
		return nil, status.Error(codes.InvalidArgument, "market_id is required")
	}

	h.logger.Debug("GetMarket request received", zap.String("market_id", req.MarketId))

	market, err := h.spotService.GetMarketByID(ctx, req.MarketId)
	if err != nil {
		h.logger.Warn("GetMarket failed",
			zap.String("market_id", req.MarketId),
			zap.Error(err),
		)
		return nil, h.toGRPCError(err)
	}

	h.logger.Debug("GetMarket success", zap.String("market_id", req.MarketId))

	return &spotv1.GetMarketResponse{
		Market: mapper.ToProto(market),
	}, nil
}

func (h *SpotHandler) toGRPCError(err error) error {
	if err == nil {
		return nil
	}

	switch err {
	case service.ErrMarketNotFound:
		return status.Error(codes.NotFound, "market not found")
	case service.ErrInvalidMarket, service.ErrInvalidRole:
		return status.Error(codes.InvalidArgument, "invalid request parameters")
	case service.ErrPermissionDenied:
		return status.Error(codes.PermissionDenied, "access denied")
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
