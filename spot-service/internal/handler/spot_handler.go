package handler

import (
	"context"

	common "exchange-system/spot-service/proto/common"
	v1 "exchange-system/spot-service/proto/spot/v1"

	"exchange-system/spot-service/internal/mapper"
	"exchange-system/spot-service/internal/service"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Реализует gRPC сервер
type SpotHandler struct {
	v1.UnimplementedSpotInstrumentServiceServer
	service *service.SpotService
}

func NewSpotHandler(svc *service.SpotService) *SpotHandler {
	return &SpotHandler{service: svc}
}

// Обрабатывает запрос на получение списка рынков
func (h *SpotHandler) ViewMarkets(ctx context.Context, req *v1.ViewMarketsRequest) (*v1.ViewMarketsResponse, error) {
	markets, err := h.service.ViewMarkets(ctx, req.GetUserRoles())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failde to get markets: %v", err)
	}

	pbMarkets := mapper.ToProtoMarketList(markets)

	return &v1.ViewMarketsResponse{
		Markets: pbMarkets,

		PageInfo: &common.PageInfo{
			TotalCount: int32(len(markets)),
		},
	}, nil
}

func (h *SpotHandler) GetMarket(ctx context.Context, req *v1.GetMarketRequest) (*v1.GetMarketResponse, error) {
	if req.MarketId == "" {
		return nil, status.Error(codes.InvalidArgument, "market_id is required")
	}

	market, err := h.service.GetMarketByID(ctx, req.MarketId)
	if err != nil {
		if err.Error() == "market not found" {
			return nil, status.Error(codes.NotFound, "market not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get market: %v", err)
	}

	return &v1.GetMarketResponse{
		Market: mapper.ToProtoMarket(*market),
	}, nil
}
