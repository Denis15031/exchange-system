package ports

import (
	"context"
)

// Это позволяет нам делать моки в тестах и не зависеть от конкретной реализации gRPC клиента
type MarketClient interface {
	CheckMarketActive(ctx context.Context, marketID string) (bool, error)
}
