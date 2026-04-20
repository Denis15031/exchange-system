package mapper

import (
	"exchange-system/order-service/internal/domain"
	orderv1 "exchange-system/proto/order/v1"

	timestamppb "google.golang.org/protobuf/types/known/timestamppb"

	"github.com/shopspring/decimal"
)

func ToProto(o *domain.Order) *orderv1.Order {
	if o == nil {
		return nil
	}

	return &orderv1.Order{
		OrderId:        o.ID,
		UserId:         o.UserID,
		MarketId:       o.MarketID,
		Type:           OrderTypeToProto(o.Type),
		Status:         OrderStatusToProto(o.Status),
		Price:          o.Price.String(),
		Quantity:       o.Quantity.String(),
		FilledQuantity: o.FilledQuantity.String(),
		CreatedAt:      timestamppb.New(o.CreatedAt),
		UpdatedAt:      timestamppb.New(o.UpdatedAt),
	}
}

func ToDomain(p *orderv1.Order) *domain.Order {
	if p == nil {
		return nil
	}

	price, _ := decimal.NewFromString(p.Price)
	quantity, _ := decimal.NewFromString(p.Quantity)
	filledQty, _ := decimal.NewFromString(p.FilledQuantity)

	return &domain.Order{
		ID:             p.OrderId,
		UserID:         p.UserId,
		MarketID:       p.MarketId,
		Type:           OrderTypeFromProto(p.Type),
		Status:         OrderStatusFromProto(p.Status),
		Price:          price,
		Quantity:       quantity,
		FilledQuantity: filledQty,
		CreatedAt:      p.CreatedAt.AsTime(),
		UpdatedAt:      p.UpdatedAt.AsTime(),
	}
}

func OrderTypeToProto(t domain.OrderType) orderv1.OrderType {
	switch t {
	case domain.OrderTypeBuy:
		return orderv1.OrderType_ORDER_TYPE_BUY
	case domain.OrderTypeSell:
		return orderv1.OrderType_ORDER_TYPE_SELL
	default:
		return orderv1.OrderType_ORDER_TYPE_UNSPECIFIED
	}
}

func OrderTypeFromProto(t orderv1.OrderType) domain.OrderType {
	switch t {
	case orderv1.OrderType_ORDER_TYPE_BUY:
		return domain.OrderTypeBuy
	case orderv1.OrderType_ORDER_TYPE_SELL:
		return domain.OrderTypeSell
	default:
		return ""
	}
}

func OrderStatusToProto(s domain.OrderStatus) orderv1.OrderStatus {
	switch s {
	case domain.OrderStatusCreated:
		return orderv1.OrderStatus_ORDER_STATUS_CREATED
	case domain.OrderStatusPending:
		return orderv1.OrderStatus_ORDER_STATUS_PENDING
	case domain.OrderStatusFilled, domain.OrderStatusPartiallyFilled:
		return orderv1.OrderStatus_ORDER_STATUS_FILLED
	case domain.OrderStatusCanceled:
		return orderv1.OrderStatus_ORDER_STATUS_CANCELLED
	case domain.OrderStatusRejected:
		return orderv1.OrderStatus_ORDER_STATUS_REJECTED
	default:
		return orderv1.OrderStatus_ORDER_STATUS_UNSPECIFIED
	}
}

func OrderStatusFromProto(s orderv1.OrderStatus) domain.OrderStatus {
	switch s {
	case orderv1.OrderStatus_ORDER_STATUS_CREATED:
		return domain.OrderStatusCreated
	case orderv1.OrderStatus_ORDER_STATUS_PENDING:
		return domain.OrderStatusPending
	case orderv1.OrderStatus_ORDER_STATUS_FILLED:
		return domain.OrderStatusFilled
	case orderv1.OrderStatus_ORDER_STATUS_CANCELLED:
		return domain.OrderStatusCanceled
	case orderv1.OrderStatus_ORDER_STATUS_REJECTED:
		return domain.OrderStatusRejected
	default:
		return ""
	}
}
