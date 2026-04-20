package service

import "errors"

var (
	ErrOrderNotFound       = errors.New("order not found")
	ErrInvalidMarket       = errors.New("invalid or inactive market")
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrInvalidPrice        = errors.New("invalid price format")
	ErrInvalidQuantity     = errors.New("invalid quantity format")
	ErrTooManyOrders       = errors.New("too many open orders")
)
