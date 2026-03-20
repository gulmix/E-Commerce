package service

import (
	"context"

	"ecommerce/product-service/internal/domain"
)

type ProductService interface {
	Create(ctx context.Context, name, description string, price float64, stock int32, categoryID string) (*domain.Product, error)
	GetByID(ctx context.Context, id string) (*domain.Product, error)
	List(ctx context.Context, page, pageSize int32, categoryID string) ([]*domain.Product, int32, error)
	Update(ctx context.Context, id, name, description string, price float64, stock int32, categoryID string, version int32) (*domain.Product, error)
	Delete(ctx context.Context, id string) error
}
