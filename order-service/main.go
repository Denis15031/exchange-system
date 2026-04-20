package main

import (
	"exchange-system/order-service/internal/di"

	"go.uber.org/fx"
)

func main() {
	fx.New(di.OrderModule).Run()
}
