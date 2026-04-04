package handler

import (
	"context"

	spotv1 "exchange-system/proto/spot/v1"
	"exchange-system/spot-service/internal/mapper"
	"exchange-system/spot-service/internal/service"
)

type SpotHandler struct {
	spotService *service.SpotService
	spotv1.UnimplementedSpotInstrumentServiceServer
}

func NewSpotHandler(spotService *service.SpotService) *SpotHandler {
	return &SpotHandler{
		spotService: spotService,
	}
}

func (h *SpotHandler) ViewMarkets(ctx context.Context, req *spotv1.ViewMarketsRequest) (*spotv1.ViewMarketsResponse, error) {
	userRoles := mapper.UserRolesFromProto(req.GetUserRoles())

	markets, err := h.spotService.ViewMarkets(ctx, userRoles)
	if err != nil {
		return nil, err
	}

	protoMarkets := make([]*spotv1.Market, 0, len(markets))
	for i := range markets {
		protoMarkets = append(protoMarkets, mapper.ToProto(&markets[i]))
	}

	return &spotv1.ViewMarketsResponse{
		Markets: protoMarkets,
	}, nil
}

func (h *SpotHandler) GetMarket(ctx context.Context, req *spotv1.GetMarketRequest) (*spotv1.GetMarketResponse, error) {
	market, err := h.spotService.GetMarketByID(ctx, req.MarketId)
	if err != nil {
		return nil, err
	}

	return &spotv1.GetMarketResponse{
		Market: mapper.ToProto(market),
	}, nil
}
