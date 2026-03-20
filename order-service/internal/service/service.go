package service

import (
	"context"

	"ecommerce/order-service/internal/domain"
)

type OrderService interface {
	CreateOrder(ctx context.Context, userID string, items []domain.OrderItem) (*domain.Order, error)
	GetOrder(ctx context.Context, id string) (*domain.Order, error)
	ListUserOrders(ctx context.Context, userID string) ([]*domain.Order, error)
	UpdateOrderStatus(ctx context.Context, id string, status domain.OrderStatus) (*domain.Order, error)
}

type EventPublisher interface {
	Publish(ctx context.Context, routingKey string, body []byte) error
}
