package service_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	apperr "ecommerce/pkg/errors"
	"ecommerce/product-service/internal/domain"
	"ecommerce/product-service/internal/repository"
	"ecommerce/product-service/internal/service"
)

type mockRepo struct {
	mu   sync.Mutex
	byID map[string]*domain.Product
	seq  int
}

func newMockRepo() *mockRepo {
	return &mockRepo{byID: make(map[string]*domain.Product)}
}

func (m *mockRepo) Create(_ context.Context, p *domain.Product) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seq++
	p.ID = fmt.Sprintf("prod-%d", m.seq)
	p.Version = 1
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	cp := *p
	m.byID[p.ID] = &cp
	return nil
}

func (m *mockRepo) GetByID(_ context.Context, id string) (*domain.Product, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.byID[id]
	if !ok {
		return nil, apperr.New(apperr.ErrNotFound, "product not found")
	}
	cp := *p
	return &cp, nil
}

func (m *mockRepo) List(_ context.Context, page, pageSize int32, categoryID string) ([]*domain.Product, int32, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var all []*domain.Product
	for _, p := range m.byID {
		if categoryID == "" || p.CategoryID == categoryID {
			cp := *p
			all = append(all, &cp)
		}
	}
	total := int32(len(all))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	start := (page - 1) * pageSize
	if int(start) >= len(all) {
		return nil, total, nil
	}
	end := start + pageSize
	if int(end) > len(all) {
		end = int32(len(all))
	}
	return all[start:end], total, nil
}

func (m *mockRepo) Update(_ context.Context, p *domain.Product) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, ok := m.byID[p.ID]
	if !ok {
		return apperr.New(apperr.ErrNotFound, "product not found")
	}
	if existing.Version != p.Version {
		return apperr.New(apperr.ErrConflict, "product version conflict or not found")
	}
	p.Version = existing.Version + 1
	p.CreatedAt = existing.CreatedAt
	p.UpdatedAt = time.Now()
	cp := *p
	m.byID[p.ID] = &cp
	return nil
}

func (m *mockRepo) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.byID[id]; !ok {
		return apperr.New(apperr.ErrNotFound, "product not found")
	}
	delete(m.byID, id)
	return nil
}

var _ repository.ProductRepository = (*mockRepo)(nil)

func newTestService() service.ProductService {
	return service.New(newMockRepo())
}

func TestCreate_Success(t *testing.T) {
	svc := newTestService()
	p, err := svc.Create(context.Background(), "Widget", "A widget", 9.99, 100, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ID == "" {
		t.Error("expected non-empty ID")
	}
	if p.Version != 1 {
		t.Errorf("expected version 1, got %d", p.Version)
	}
}

func TestCreate_EmptyName(t *testing.T) {
	svc := newTestService()
	_, err := svc.Create(context.Background(), "", "desc", 1.0, 0, "")
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !apperr.Is(err, apperr.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreate_NegativePrice(t *testing.T) {
	svc := newTestService()
	_, err := svc.Create(context.Background(), "X", "", -1.0, 0, "")
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !apperr.Is(err, apperr.ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestGetByID_Success(t *testing.T) {
	svc := newTestService()
	created, _ := svc.Create(context.Background(), "Gadget", "", 5.0, 10, "")

	p, err := svc.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Name != "Gadget" {
		t.Errorf("name mismatch: %q", p.Name)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	svc := newTestService()
	_, err := svc.GetByID(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected not-found error")
	}
	if !apperr.Is(err, apperr.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestList_Pagination(t *testing.T) {
	svc := newTestService()
	for i := 0; i < 5; i++ {
		_, _ = svc.Create(context.Background(), fmt.Sprintf("Product %d", i), "", float64(i), 10, "")
	}
	products, total, err := svc.List(context.Background(), 1, 3, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(products) != 3 {
		t.Errorf("expected 3 products on page 1, got %d", len(products))
	}
}

func TestUpdate_OptimisticLock(t *testing.T) {
	svc := newTestService()
	created, _ := svc.Create(context.Background(), "Item", "", 1.0, 5, "")

	// Correct version → should succeed
	_, err := svc.Update(context.Background(), created.ID, "Item Updated", "", 2.0, 10, "", created.Version)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Stale version → should conflict
	_, err = svc.Update(context.Background(), created.ID, "Stale", "", 3.0, 5, "", created.Version)
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if !apperr.Is(err, apperr.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestDelete_Success(t *testing.T) {
	svc := newTestService()
	created, _ := svc.Create(context.Background(), "Doomed", "", 1.0, 1, "")

	if err := svc.Delete(context.Background(), created.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err := svc.GetByID(context.Background(), created.ID)
	if !apperr.Is(err, apperr.ErrNotFound) {
		t.Errorf("expected product to be deleted, got %v", err)
	}
}

func TestDelete_NotFound(t *testing.T) {
	svc := newTestService()
	err := svc.Delete(context.Background(), "ghost-id")
	if err == nil {
		t.Fatal("expected not-found error")
	}
	if !apperr.Is(err, apperr.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
