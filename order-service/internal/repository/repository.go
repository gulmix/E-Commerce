package repository

import (
	"context"

	"ecommerce/order-service/internal/domain"
)

type OrderRepository interface {
	Create(ctx context.Context, o *domain.Order) error
	GetByID(ctx context.Context, id string) (*domain.Order, error)
	ListByUserID(ctx context.Context, userID string) ([]*domain.Order, error)
	UpdateStatus(ctx context.Context, id string, status domain.OrderStatus) (*domain.Order, error)
}
