package mapper

import (
	"exchange-system/order-service/internal/domain"
	orderV1 "exchange-system/order-service/proto/order/v1"

	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func ToProtoOrder(o *domain.Order) *orderV1.Order {
	if o == nil {
		return nil
	}

	return &orderV1.Order{
		Id:             o.ID,
		UserId:         o.UserID,
		MarketId:       o.MarketID,
		Type:           orderV1.OrderType(orderV1.OrderType_value[string(o.Type)]),
		Price:          o.Price.String(),
		Quantity:       o.Quantity.String(),
		FilledQuantity: o.FilledQuantity.String(),
		Status:         orderV1.OrderStatus(orderV1.OrderStatus_value[string(o.Status)]),
		CreatedAt:      timestamppb.New(o.CreatedAt),
		UpdatedAt:      timestamppb.New(o.UpdatedAt),
	}
}

func ToDomainOrder(pb *orderV1.Order) *domain.Order {
	if pb == nil {
		return nil
	}

	price, _ := decimal.NewFromString(pb.Price)
	qty, _ := decimal.NewFromString(pb.Quantity)
	filledQty, _ := decimal.NewFromString(pb.FilledQuantity)

	return &domain.Order{
		ID:             pb.Id,
		UserID:         pb.UserId,
		MarketID:       pb.MarketId,
		Type:           domain.OrderType(pb.Type),
		Price:          price,
		Quantity:       qty,
		FilledQuantity: filledQty,
		Status:         domain.OrderStatus(pb.Status),
		CreatedAt:      pb.CreatedAt.AsTime(),
		UpdatedAt:      pb.UpdatedAt.AsTime(),
	}
}
