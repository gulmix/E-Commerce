package repository

import (
	"context"
	"errors"
	"fmt"

	apperr "ecommerce/pkg/errors"
	"ecommerce/order-service/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgres(ctx context.Context, dsn string) (OrderRepository, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("pgxpool.New: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("db ping: %w", err)
	}
	r := &postgresRepo{pool: pool}
	if err := r.migrate(ctx); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return r, nil
}

func (r *postgresRepo) Close() {
	r.pool.Close()
}

func (r *postgresRepo) migrate(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS orders (
			id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id     UUID NOT NULL,
			status      TEXT NOT NULL DEFAULT 'pending',
			total_price DOUBLE PRECISION NOT NULL DEFAULT 0,
			created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_orders_user ON orders (user_id);

		CREATE TABLE IF NOT EXISTS order_items (
			id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			order_id   UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
			product_id UUID NOT NULL,
			quantity   INTEGER NOT NULL,
			unit_price DOUBLE PRECISION NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_order_items_order ON order_items (order_id);
	`)
	return err
}

func (r *postgresRepo) Create(ctx context.Context, o *domain.Order) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	row := tx.QueryRow(ctx,
		`INSERT INTO orders (user_id, status, total_price)
		 VALUES ($1, $2, $3)
		 RETURNING id, created_at, updated_at`,
		o.UserID, string(o.Status), o.TotalPrice,
	)
	if err := row.Scan(&o.ID, &o.CreatedAt, &o.UpdatedAt); err != nil {
		return fmt.Errorf("insert order: %w", err)
	}

	for i := range o.Items {
		item := &o.Items[i]
		item.OrderID = o.ID
		row := tx.QueryRow(ctx,
			`INSERT INTO order_items (order_id, product_id, quantity, unit_price)
			 VALUES ($1, $2, $3, $4)
			 RETURNING id`,
			o.ID, item.ProductID, item.Quantity, item.UnitPrice,
		)
		if err := row.Scan(&item.ID); err != nil {
			return fmt.Errorf("insert order_item: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *postgresRepo) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, user_id, status, total_price, created_at, updated_at
		 FROM orders WHERE id = $1`,
		id,
	)
	o := &domain.Order{}
	var statusStr string
	if err := row.Scan(&o.ID, &o.UserID, &statusStr, &o.TotalPrice, &o.CreatedAt, &o.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.New(apperr.ErrNotFound, "order not found")
		}
		return nil, fmt.Errorf("get order: %w", err)
	}
	o.Status = domain.OrderStatus(statusStr)

	rows, err := r.pool.Query(ctx,
		`SELECT id, order_id, product_id, quantity, unit_price
		 FROM order_items WHERE order_id = $1`,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("get order items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item domain.OrderItem
		if err := rows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.Quantity, &item.UnitPrice); err != nil {
			return nil, fmt.Errorf("scan order item: %w", err)
		}
		o.Items = append(o.Items, item)
	}
	return o, rows.Err()
}

func (r *postgresRepo) ListByUserID(ctx context.Context, userID string) ([]*domain.Order, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, status, total_price, created_at, updated_at
		 FROM orders WHERE user_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()

	var orders []*domain.Order
	for rows.Next() {
		o := &domain.Order{}
		var statusStr string
		if err := rows.Scan(&o.ID, &o.UserID, &statusStr, &o.TotalPrice, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		o.Status = domain.OrderStatus(statusStr)
		orders = append(orders, o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Fetch items for each order
	for _, o := range orders {
		itemRows, err := r.pool.Query(ctx,
			`SELECT id, order_id, product_id, quantity, unit_price
			 FROM order_items WHERE order_id = $1`,
			o.ID,
		)
		if err != nil {
			return nil, fmt.Errorf("get items for order %s: %w", o.ID, err)
		}
		for itemRows.Next() {
			var item domain.OrderItem
			if err := itemRows.Scan(&item.ID, &item.OrderID, &item.ProductID, &item.Quantity, &item.UnitPrice); err != nil {
				itemRows.Close()
				return nil, fmt.Errorf("scan order item: %w", err)
			}
			o.Items = append(o.Items, item)
		}
		itemRows.Close()
		if err := itemRows.Err(); err != nil {
			return nil, err
		}
	}
	return orders, nil
}

func (r *postgresRepo) UpdateStatus(ctx context.Context, id string, status domain.OrderStatus) (*domain.Order, error) {
	row := r.pool.QueryRow(ctx,
		`UPDATE orders SET status=$1, updated_at=now()
		 WHERE id=$2
		 RETURNING id, user_id, status, total_price, created_at, updated_at`,
		string(status), id,
	)
	o := &domain.Order{}
	var statusStr string
	if err := row.Scan(&o.ID, &o.UserID, &statusStr, &o.TotalPrice, &o.CreatedAt, &o.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.New(apperr.ErrNotFound, "order not found")
		}
		return nil, fmt.Errorf("update order status: %w", err)
	}
	o.Status = domain.OrderStatus(statusStr)
	return o, nil
}
