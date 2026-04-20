package main

import (
	"exchange-system/user-service/internal/di"

	"go.uber.org/fx"
)

func main() {
	app := fx.New(
		di.UserModule,
	)
	app.Run()
}
