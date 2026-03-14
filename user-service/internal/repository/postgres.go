package repository

import (
	"context"
	"errors"
	"fmt"

	apperr "ecommerce/pkg/errors"
	"ecommerce/user-service/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type postgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgres(ctx context.Context, dsn string) (UserRepository, error) {
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

func (r *postgresRepo) migrate(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email         TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			name          TEXT NOT NULL,
			role          TEXT NOT NULL DEFAULT 'customer',
			created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);
	`)
	return err
}

func (r *postgresRepo) Create(ctx context.Context, u *domain.User) error {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, name, role)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, created_at`,
		u.Email, u.PasswordHash, u.Name, u.Role,
	)
	if err := row.Scan(&u.ID, &u.CreatedAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return apperr.New(apperr.ErrConflict, "email already registered")
		}
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (r *postgresRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, name, role, created_at FROM users WHERE id = $1`,
		id,
	)
	u := &domain.User{}
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Role, &u.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.New(apperr.ErrNotFound, "user not found")
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return u, nil
}

func (r *postgresRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, name, role, created_at FROM users WHERE email = $1`,
		email,
	)
	u := &domain.User{}
	if err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Role, &u.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.New(apperr.ErrNotFound, "user not found")
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return u, nil
}
