package service_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"ecommerce/order-service/internal/domain"
	"ecommerce/order-service/internal/repository"
	"ecommerce/order-service/internal/service"
	apperr "ecommerce/pkg/errors"
	productpb "ecommerce/proto/product"
	userpb "ecommerce/proto/user"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

type mockRepo struct {
	mu     sync.Mutex
	orders map[string]*domain.Order
	seq    int
}

func newMockRepo() *mockRepo {
	return &mockRepo{orders: make(map[string]*domain.Order)}
}

func (m *mockRepo) Create(_ context.Context, o *domain.Order) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seq++
	o.ID = fmt.Sprintf("order-%d", m.seq)
	o.CreatedAt = time.Now()
	o.UpdatedAt = time.Now()
	for i := range o.Items {
		o.Items[i].ID = fmt.Sprintf("item-%d-%d", m.seq, i)
		o.Items[i].OrderID = o.ID
	}
	cp := *o
	cp.Items = append([]domain.OrderItem(nil), o.Items...)
	m.orders[o.ID] = &cp
	return nil
}

func (m *mockRepo) GetByID(_ context.Context, id string) (*domain.Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	o, ok := m.orders[id]
	if !ok {
		return nil, apperr.New(apperr.ErrNotFound, "order not found")
	}
	cp := *o
	cp.Items = append([]domain.OrderItem(nil), o.Items...)
	return &cp, nil
}

func (m *mockRepo) ListByUserID(_ context.Context, userID string) ([]*domain.Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*domain.Order
	for _, o := range m.orders {
		if o.UserID == userID {
			cp := *o
			cp.Items = append([]domain.OrderItem(nil), o.Items...)
			result = append(result, &cp)
		}
	}
	return result, nil
}

func (m *mockRepo) UpdateStatus(_ context.Context, id string, status domain.OrderStatus) (*domain.Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	o, ok := m.orders[id]
	if !ok {
		return nil, apperr.New(apperr.ErrNotFound, "order not found")
	}
	o.Status = status
	o.UpdatedAt = time.Now()
	cp := *o
	return &cp, nil
}

var _ repository.OrderRepository = (*mockRepo)(nil)

type mockUserClient struct {
	users map[string]*userpb.GetUserResponse
}

func (m *mockUserClient) CreateUser(_ context.Context, _ *userpb.CreateUserRequest, _ ...grpc.CallOption) (*userpb.CreateUserResponse, error) {
	return nil, grpcstatus.Error(codes.Unimplemented, "")
}
func (m *mockUserClient) LoginUser(_ context.Context, _ *userpb.LoginUserRequest, _ ...grpc.CallOption) (*userpb.LoginUserResponse, error) {
	return nil, grpcstatus.Error(codes.Unimplemented, "")
}
func (m *mockUserClient) GetUser(_ context.Context, in *userpb.GetUserRequest, _ ...grpc.CallOption) (*userpb.GetUserResponse, error) {
	u, ok := m.users[in.UserId]
	if !ok {
		return nil, grpcstatus.Error(codes.NotFound, "user not found")
	}
	return u, nil
}
func (m *mockUserClient) ValidateToken(_ context.Context, _ *userpb.ValidateTokenRequest, _ ...grpc.CallOption) (*userpb.ValidateTokenResponse, error) {
	return nil, grpcstatus.Error(codes.Unimplemented, "")
}

var _ userpb.UserServiceClient = (*mockUserClient)(nil)

type mockProductClient struct {
	mu       sync.Mutex
	products map[string]*productpb.GetProductResponse
}

func (m *mockProductClient) CreateProduct(_ context.Context, _ *productpb.CreateProductRequest, _ ...grpc.CallOption) (*productpb.CreateProductResponse, error) {
	return nil, grpcstatus.Error(codes.Unimplemented, "")
}
func (m *mockProductClient) GetProduct(_ context.Context, in *productpb.GetProductRequest, _ ...grpc.CallOption) (*productpb.GetProductResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.products[in.ProductId]
	if !ok {
		return nil, grpcstatus.Error(codes.NotFound, "product not found")
	}
	return p, nil
}
func (m *mockProductClient) ListProducts(_ context.Context, _ *productpb.ListProductsRequest, _ ...grpc.CallOption) (*productpb.ListProductsResponse, error) {
	return nil, grpcstatus.Error(codes.Unimplemented, "")
}
func (m *mockProductClient) UpdateProduct(_ context.Context, in *productpb.UpdateProductRequest, _ ...grpc.CallOption) (*productpb.UpdateProductResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.products[in.ProductId]
	if !ok {
		return nil, grpcstatus.Error(codes.NotFound, "product not found")
	}
	p.Stock = in.Stock
	p.Version++
	return &productpb.UpdateProductResponse{ProductId: in.ProductId, Stock: in.Stock}, nil
}
func (m *mockProductClient) DeleteProduct(_ context.Context, _ *productpb.DeleteProductRequest, _ ...grpc.CallOption) (*productpb.DeleteProductResponse, error) {
	return nil, grpcstatus.Error(codes.Unimplemented, "")
}

var _ productpb.ProductServiceClient = (*mockProductClient)(nil)

type mockPublisher struct {
	mu     sync.Mutex
	events []string
}

func (p *mockPublisher) Publish(_ context.Context, key string, _ []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, key)
	return nil
}

func (p *mockPublisher) published() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.events...)
}

var _ service.EventPublisher = (*mockPublisher)(nil)

func setup() (service.OrderService, *mockRepo, *mockUserClient, *mockProductClient, *mockPublisher) {
	repo := newMockRepo()
	userClient := &mockUserClient{users: map[string]*userpb.GetUserResponse{
		"user-1": {UserId: "user-1", Email: "a@b.com"},
	}}
	productClient := &mockProductClient{products: map[string]*productpb.GetProductResponse{
		"prod-1": {ProductId: "prod-1", Name: "Widget", Price: 10.0, Stock: 100, Version: 1},
		"prod-2": {ProductId: "prod-2", Name: "Gadget", Price: 25.0, Stock: 5, Version: 1},
	}}
	pub := &mockPublisher{}
	svc := service.New(repo, userClient, productClient, pub)
	return svc, repo, userClient, productClient, pub
}

func TestCreateOrder_Success(t *testing.T) {
	svc, _, _, _, pub := setup()
	o, err := svc.CreateOrder(context.Background(), "user-1", []domain.OrderItem{
		{ProductID: "prod-1", Quantity: 2},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if o.ID == "" {
		t.Error("expected non-empty order ID")
	}
	if o.Status != domain.StatusPending {
		t.Errorf("expected status pending, got %s", o.Status)
	}
	if o.TotalPrice != 20.0 {
		t.Errorf("expected total 20.0, got %f", o.TotalPrice)
	}
	if len(o.Items) != 1 || o.Items[0].UnitPrice != 10.0 {
		t.Error("item enrichment failed")
	}
	events := pub.published()
	if len(events) != 1 || events[0] != "order.created" {
		t.Errorf("expected order.created event, got %v", events)
	}
}

func TestCreateOrder_EmptyUserID(t *testing.T) {
	svc, _, _, _, _ := setup()
	_, err := svc.CreateOrder(context.Background(), "", []domain.OrderItem{{ProductID: "prod-1", Quantity: 1}})
	if !apperr.Is(err, apperr.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateOrder_EmptyItems(t *testing.T) {
	svc, _, _, _, _ := setup()
	_, err := svc.CreateOrder(context.Background(), "user-1", nil)
	if !apperr.Is(err, apperr.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateOrder_UserNotFound(t *testing.T) {
	svc, _, _, _, _ := setup()
	_, err := svc.CreateOrder(context.Background(), "unknown-user", []domain.OrderItem{{ProductID: "prod-1", Quantity: 1}})
	if !apperr.Is(err, apperr.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateOrder_InsufficientStock(t *testing.T) {
	svc, _, _, _, _ := setup()
	_, err := svc.CreateOrder(context.Background(), "user-1", []domain.OrderItem{
		{ProductID: "prod-2", Quantity: 10}, // only 5 in stock
	})
	if !apperr.Is(err, apperr.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateOrder_ProductNotFound(t *testing.T) {
	svc, _, _, _, _ := setup()
	_, err := svc.CreateOrder(context.Background(), "user-1", []domain.OrderItem{
		{ProductID: "nonexistent", Quantity: 1},
	})
	if !apperr.Is(err, apperr.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestGetOrder_Success(t *testing.T) {
	svc, _, _, _, _ := setup()
	created, _ := svc.CreateOrder(context.Background(), "user-1", []domain.OrderItem{{ProductID: "prod-1", Quantity: 1}})

	o, err := svc.GetOrder(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if o.ID != created.ID {
		t.Errorf("id mismatch: %q vs %q", o.ID, created.ID)
	}
}

func TestGetOrder_NotFound(t *testing.T) {
	svc, _, _, _, _ := setup()
	_, err := svc.GetOrder(context.Background(), "nonexistent")
	if !apperr.Is(err, apperr.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestListUserOrders(t *testing.T) {
	svc, _, _, _, _ := setup()
	_, _ = svc.CreateOrder(context.Background(), "user-1", []domain.OrderItem{{ProductID: "prod-1", Quantity: 1}})
	_, _ = svc.CreateOrder(context.Background(), "user-1", []domain.OrderItem{{ProductID: "prod-1", Quantity: 2}})

	orders, err := svc.ListUserOrders(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(orders) != 2 {
		t.Errorf("expected 2 orders, got %d", len(orders))
	}
}

func TestUpdateOrderStatus_ValidTransition(t *testing.T) {
	svc, _, _, _, _ := setup()
	o, _ := svc.CreateOrder(context.Background(), "user-1", []domain.OrderItem{{ProductID: "prod-1", Quantity: 1}})

	updated, err := svc.UpdateOrderStatus(context.Background(), o.ID, domain.StatusConfirmed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Status != domain.StatusConfirmed {
		t.Errorf("expected confirmed, got %s", updated.Status)
	}
}

func TestUpdateOrderStatus_InvalidTransition(t *testing.T) {
	svc, _, _, _, _ := setup()
	o, _ := svc.CreateOrder(context.Background(), "user-1", []domain.OrderItem{{ProductID: "prod-1", Quantity: 1}})

	_, err := svc.UpdateOrderStatus(context.Background(), o.ID, domain.StatusShipped)
	if !apperr.Is(err, apperr.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateOrderStatus_CancelledPublishesEvent(t *testing.T) {
	svc, _, _, _, pub := setup()
	o, _ := svc.CreateOrder(context.Background(), "user-1", []domain.OrderItem{{ProductID: "prod-1", Quantity: 1}})

	_, err := svc.UpdateOrderStatus(context.Background(), o.ID, domain.StatusCancelled)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	events := pub.published()
	if len(events) != 2 || events[1] != "order.cancelled" {
		t.Errorf("expected order.cancelled event, got %v", events)
	}
}

func TestUpdateOrderStatus_FullLifecycle(t *testing.T) {
	svc, _, _, _, _ := setup()
	o, _ := svc.CreateOrder(context.Background(), "user-1", []domain.OrderItem{{ProductID: "prod-1", Quantity: 1}})

	transitions := []domain.OrderStatus{domain.StatusConfirmed, domain.StatusShipped, domain.StatusDelivered}
	for _, s := range transitions {
		updated, err := svc.UpdateOrderStatus(context.Background(), o.ID, s)
		if err != nil {
			t.Fatalf("transition to %s failed: %v", s, err)
		}
		o = updated
	}
	if o.Status != domain.StatusDelivered {
		t.Errorf("expected delivered, got %s", o.Status)
	}
}
