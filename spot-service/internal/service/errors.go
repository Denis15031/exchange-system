package service

import "errors"

var (
	ErrMarketNotFound   = errors.New("market not found")
	ErrInvalidMarket    = errors.New("invalid market data")
	ErrInvalidRole      = errors.New("invalid user role")
	ErrPermissionDenied = errors.New("permission denied")
)
