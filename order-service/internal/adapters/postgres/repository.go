package postgres

import (
	"context"
	"fmt"

	"exchange-system/order-service/internal/ports"
	orderv1 "exchange-system/proto/order/v1"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) ports.OrderRepository {
	return &Repository{pool: pool}
}

func (r *Repository) Save(ctx context.Context, order *orderv1.Order) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	balanceQuery := `UPDATE users SET balance = balance - $1 WHERE user_id = $2 AND balance >= $1`
	res, err := tx.Exec(ctx, balanceQuery, order.Price, order.UserId)
	if err != nil {
		return fmt.Errorf("update balance: %w", err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("insufficient balance or user not found") // Откат транзакции
	}

	var createdAt, updatedAt time.Time
	if order.CreatedAt != nil {
		createdAt = order.CreatedAt.AsTime()
	} else {
		createdAt = time.Now()
	}
	if order.UpdatedAt != nil {
		updatedAt = order.UpdatedAt.AsTime()
	} else {
		updatedAt = time.Now()
	}

	insertQuery := `INSERT INTO orders (order_id, user_id, market_id, order_type, status, price, quantity, filled_quantity, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err = tx.Exec(ctx, insertQuery,
		order.OrderId, order.UserId, order.MarketId, order.Type, order.Status,
		order.Price, order.Quantity, order.FilledQuantity, createdAt, updatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}

	return tx.Commit(ctx)
}

func (r *Repository) GetByID(ctx context.Context, orderID string) (*orderv1.Order, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var order orderv1.Order
	var createdAt, updatedAt time.Time
	err := r.pool.QueryRow(ctx, `SELECT order_id, user_id, market_id, order_type, status, price, quantity, filled_quantity, created_at, updated_at FROM orders WHERE order_id = $1`, orderID).Scan(
		&order.OrderId, &order.UserId, &order.MarketId, &order.Type, &order.Status,
		&order.Price, &order.Quantity, &order.FilledQuantity, &createdAt, &updatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get order: %w", err)
	}
	order.CreatedAt = timestamppb.New(createdAt)
	order.UpdatedAt = timestamppb.New(updatedAt)
	return &order, nil
}

func (r *Repository) ListByUser(ctx context.Context, userID string) ([]*orderv1.Order, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `SELECT order_id, user_id, market_id, order_type, status, price, quantity, filled_quantity, created_at, updated_at FROM orders WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*orderv1.Order
	for rows.Next() {
		var o orderv1.Order
		var ct, ut time.Time
		err := rows.Scan(&o.OrderId, &o.UserId, &o.MarketId, &o.Type, &o.Status, &o.Price, &o.Quantity, &o.FilledQuantity, &ct, &ut)
		if err != nil {
			return nil, err
		}
		o.CreatedAt = timestamppb.New(ct)
		o.UpdatedAt = timestamppb.New(ut)
		orders = append(orders, &o)
	}
	return orders, nil
}

func (r *Repository) CountByUser(ctx context.Context, userID string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	var count int
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM orders WHERE user_id = $1`, userID).Scan(&count)
	return count, err
}
