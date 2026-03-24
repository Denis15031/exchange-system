package service

type OrderService struct {
}

func NewOrderService(repo interface{}, spotClient interface{}) *OrderService {
	return &OrderService{}
}
