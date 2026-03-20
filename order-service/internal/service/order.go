package service

import (
	"context"
	"encoding/json"
	"fmt"

	"ecommerce/order-service/internal/domain"
	"ecommerce/order-service/internal/repository"
	apperr "ecommerce/pkg/errors"
	productpb "ecommerce/proto/product"
	userpb "ecommerce/proto/user"
)

type orderService struct {
	repo          repository.OrderRepository
	userClient    userpb.UserServiceClient
	productClient productpb.ProductServiceClient
	publisher     EventPublisher
}

func New(
	repo repository.OrderRepository,
	userClient userpb.UserServiceClient,
	productClient productpb.ProductServiceClient,
	publisher EventPublisher,
) OrderService {
	return &orderService{
		repo:          repo,
		userClient:    userClient,
		productClient: productClient,
		publisher:     publisher,
	}
}

func (s *orderService) CreateOrder(ctx context.Context, userID string, items []domain.OrderItem) (*domain.Order, error) {
	if userID == "" {
		return nil, apperr.New(apperr.ErrInvalidInput, "user_id is required")
	}
	if len(items) == 0 {
		return nil, apperr.New(apperr.ErrInvalidInput, "items must not be empty")
	}

	if _, err := s.userClient.GetUser(ctx, &userpb.GetUserRequest{UserId: userID}); err != nil {
		return nil, apperr.New(apperr.ErrInvalidInput, "user not found")
	}

	enrichedItems := make([]domain.OrderItem, len(items))
	var totalPrice float64
	for i, item := range items {
		if item.Quantity <= 0 {
			return nil, apperr.New(apperr.ErrInvalidInput, "item quantity must be positive")
		}

		prod, err := s.productClient.GetProduct(ctx, &productpb.GetProductRequest{ProductId: item.ProductID})
		if err != nil {
			return nil, apperr.New(apperr.ErrInvalidInput, fmt.Sprintf("product %s not found", item.ProductID))
		}
		if prod.Stock < item.Quantity {
			return nil, apperr.New(apperr.ErrInvalidInput, fmt.Sprintf("insufficient stock for product %s", item.ProductID))
		}

		_, err = s.productClient.UpdateProduct(ctx, &productpb.UpdateProductRequest{
			ProductId:   prod.ProductId,
			Name:        prod.Name,
			Description: prod.Description,
			Price:       prod.Price,
			Stock:       prod.Stock - item.Quantity,
			CategoryId:  prod.CategoryId,
			Version:     prod.Version,
		})
		if err != nil {
			return nil, apperr.New(apperr.ErrConflict, fmt.Sprintf("stock reservation failed for product %s: concurrent modification", item.ProductID))
		}

		enrichedItems[i] = domain.OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			UnitPrice: prod.Price,
		}
		totalPrice += prod.Price * float64(item.Quantity)
	}

	o := &domain.Order{
		UserID:     userID,
		Status:     domain.StatusPending,
		TotalPrice: totalPrice,
		Items:      enrichedItems,
	}
	if err := s.repo.Create(ctx, o); err != nil {
		return nil, err
	}

	s.publishEvent(ctx, "order.created", map[string]any{
		"order_id":    o.ID,
		"user_id":     o.UserID,
		"total_price": o.TotalPrice,
		"status":      string(o.Status),
		"created_at":  o.CreatedAt,
	})

	return o, nil
}

func (s *orderService) GetOrder(ctx context.Context, id string) (*domain.Order, error) {
	if id == "" {
		return nil, apperr.New(apperr.ErrInvalidInput, "id is required")
	}
	return s.repo.GetByID(ctx, id)
}

func (s *orderService) ListUserOrders(ctx context.Context, userID string) ([]*domain.Order, error) {
	if userID == "" {
		return nil, apperr.New(apperr.ErrInvalidInput, "user_id is required")
	}
	return s.repo.ListByUserID(ctx, userID)
}

func (s *orderService) UpdateOrderStatus(ctx context.Context, id string, status domain.OrderStatus) (*domain.Order, error) {
	if id == "" {
		return nil, apperr.New(apperr.ErrInvalidInput, "id is required")
	}

	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !current.Status.CanTransitionTo(status) {
		return nil, apperr.New(apperr.ErrInvalidInput,
			fmt.Sprintf("cannot transition order from %s to %s", current.Status, status))
	}

	updated, err := s.repo.UpdateStatus(ctx, id, status)
	if err != nil {
		return nil, err
	}

	if status == domain.StatusCancelled {
		s.publishEvent(ctx, "order.cancelled", map[string]any{
			"order_id":   id,
			"user_id":    current.UserID,
			"status":     string(status),
			"updated_at": updated.UpdatedAt,
		})
	}

	return updated, nil
}

func (s *orderService) publishEvent(ctx context.Context, key string, payload map[string]any) {
	body, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_ = s.publisher.Publish(ctx, key, body)
}
