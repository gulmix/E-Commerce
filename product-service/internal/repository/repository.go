package repository

import (
	"context"

	"ecommerce/product-service/internal/domain"
)

type ProductRepository interface {
	Create(ctx context.Context, p *domain.Product) error
	GetByID(ctx context.Context, id string) (*domain.Product, error)
	List(ctx context.Context, page, pageSize int32, categoryID string) ([]*domain.Product, int32, error)
	Update(ctx context.Context, p *domain.Product) error
	Delete(ctx context.Context, id string) error
}
