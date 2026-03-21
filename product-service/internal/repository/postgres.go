package repository

import (
	"context"
	"errors"
	"fmt"

	apperr "ecommerce/pkg/errors"
	"ecommerce/product-service/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgres(ctx context.Context, dsn string) (ProductRepository, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("pgxpool.New: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("db ping: %w", err)
	}
	repo := &postgresRepo{pool: pool}
	if err := repo.migrate(ctx); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return repo, nil
}

func (r *postgresRepo) Close() {
	r.pool.Close()
}

func (r *postgresRepo) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

func (r *postgresRepo) migrate(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS categories (
			id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT NOT NULL UNIQUE
		);
		CREATE TABLE IF NOT EXISTS products (
			id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name        TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			price       DOUBLE PRECISION NOT NULL,
			stock       INTEGER NOT NULL DEFAULT 0,
			version     INTEGER NOT NULL DEFAULT 1,
			category_id UUID REFERENCES categories(id),
			created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_products_category ON products (category_id);
	`)
	return err
}

func (r *postgresRepo) Create(ctx context.Context, p *domain.Product) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO products (name, description, price, stock, category_id)
		 VALUES ($1, $2, $3, $4, NULLIF($5, ''))
		 RETURNING id, version, created_at, updated_at`,
		p.Name, p.Description, p.Price, p.Stock, p.CategoryID,
	)
	return row.Scan(&p.ID, &p.Version, &p.CreatedAt, &p.UpdatedAt)
}

func (r *postgresRepo) GetByID(ctx context.Context, id string) (*domain.Product, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, name, description, price, stock, version, COALESCE(category_id::text,''), created_at, updated_at
		 FROM products WHERE id = $1`,
		id,
	)
	p := &domain.Product{}
	if err := row.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.Version, &p.CategoryID, &p.CreatedAt, &p.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.New(apperr.ErrNotFound, "product not found")
		}
		return nil, fmt.Errorf("get product: %w", err)
	}
	return p, nil
}

func (r *postgresRepo) List(ctx context.Context, page, pageSize int32, categoryID string) ([]*domain.Product, int32, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var total int32
	var countErr error
	if categoryID != "" {
		countErr = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM products WHERE category_id = $1`, categoryID).Scan(&total)
	} else {
		countErr = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM products`).Scan(&total)
	}
	if countErr != nil {
		return nil, 0, fmt.Errorf("count products: %w", countErr)
	}

	var rows pgx.Rows
	var err error
	if categoryID != "" {
		rows, err = r.pool.Query(ctx,
			`SELECT id, name, description, price, stock, version, COALESCE(category_id::text,''), created_at, updated_at
			 FROM products WHERE category_id = $1
			 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
			categoryID, pageSize, offset,
		)
	} else {
		rows, err = r.pool.Query(ctx,
			`SELECT id, name, description, price, stock, version, COALESCE(category_id::text,''), created_at, updated_at
			 FROM products
			 ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
			pageSize, offset,
		)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("list products: %w", err)
	}
	defer rows.Close()

	var products []*domain.Product
	for rows.Next() {
		p := &domain.Product{}
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.Stock, &p.Version, &p.CategoryID, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan product: %w", err)
		}
		products = append(products, p)
	}
	return products, total, rows.Err()
}

func (r *postgresRepo) Update(ctx context.Context, p *domain.Product) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE products
		 SET name=$1, description=$2, price=$3, stock=$4, category_id=NULLIF($5,''),
		     version=version+1, updated_at=now()
		 WHERE id=$6 AND version=$7`,
		p.Name, p.Description, p.Price, p.Stock, p.CategoryID, p.ID, p.Version,
	)
	if err != nil {
		return fmt.Errorf("update product: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.New(apperr.ErrConflict, "product version conflict or not found")
	}
	return nil
}

func (r *postgresRepo) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM products WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete product: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.New(apperr.ErrNotFound, "product not found")
	}
	return nil
}
