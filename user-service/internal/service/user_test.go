package service_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	apperr "ecommerce/pkg/errors"
	"ecommerce/user-service/internal/domain"
	"ecommerce/user-service/internal/repository"
	"ecommerce/user-service/internal/service"
)

type mockRepo struct {
	byEmail map[string]*domain.User
	byID    map[string]*domain.User
	seq     int
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		byEmail: make(map[string]*domain.User),
		byID:    make(map[string]*domain.User),
	}
}

func (m *mockRepo) Create(_ context.Context, u *domain.User) error {
	if _, exists := m.byEmail[u.Email]; exists {
		return apperr.New(apperr.ErrConflict, "email already registered")
	}
	m.seq++
	u.ID = fmt.Sprintf("user-%d", m.seq)
	u.CreatedAt = time.Now()
	cp := *u
	m.byEmail[u.Email] = &cp
	m.byID[u.ID] = &cp
	return nil
}

func (m *mockRepo) GetByID(_ context.Context, id string) (*domain.User, error) {
	u, ok := m.byID[id]
	if !ok {
		return nil, apperr.New(apperr.ErrNotFound, "user not found")
	}
	cp := *u
	return &cp, nil
}

func (m *mockRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	u, ok := m.byEmail[email]
	if !ok {
		return nil, apperr.New(apperr.ErrNotFound, "user not found")
	}
	cp := *u
	return &cp, nil
}

var _ repository.UserRepository = (*mockRepo)(nil)

func newTestService() service.UserService {
	return service.New(newMockRepo(), service.Config{
		JWTSecret:     "test-secret-key-32-bytes-long!!",
		AccessExpiry:  15 * time.Minute,
		RefreshExpiry: 7 * 24 * time.Hour,
	})
}

func TestRegister_Success(t *testing.T) {
	svc := newTestService()
	u, err := svc.Register(context.Background(), "alice@example.com", "password123", "Alice", "customer")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID == "" {
		t.Error("expected non-empty ID")
	}
	if u.Email != "alice@example.com" {
		t.Errorf("email mismatch: %q", u.Email)
	}
	if u.PasswordHash == "password123" {
		t.Error("password must be hashed")
	}
}

func TestRegister_DefaultRole(t *testing.T) {
	svc := newTestService()
	u, err := svc.Register(context.Background(), "bob@example.com", "secret", "Bob", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Role != "customer" {
		t.Errorf("expected default role 'customer', got %q", u.Role)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	svc := newTestService()
	_, _ = svc.Register(context.Background(), "dup@example.com", "pass", "Dup", "customer")
	_, err := svc.Register(context.Background(), "dup@example.com", "pass2", "Dup2", "customer")
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if !apperr.Is(err, apperr.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestLogin_Success(t *testing.T) {
	svc := newTestService()
	_, _ = svc.Register(context.Background(), "carol@example.com", "mypassword", "Carol", "customer")

	access, refresh, u, err := svc.Login(context.Background(), "carol@example.com", "mypassword")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if access == "" || refresh == "" {
		t.Error("expected non-empty tokens")
	}
	if u.Email != "carol@example.com" {
		t.Errorf("email mismatch: %q", u.Email)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	svc := newTestService()
	_, _ = svc.Register(context.Background(), "dave@example.com", "rightpass", "Dave", "customer")

	_, _, _, err := svc.Login(context.Background(), "dave@example.com", "wrongpass")
	if err == nil {
		t.Fatal("expected unauthorized error")
	}
	if !apperr.Is(err, apperr.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestLogin_UnknownEmail(t *testing.T) {
	svc := newTestService()
	_, _, _, err := svc.Login(context.Background(), "ghost@example.com", "pass")
	if err == nil {
		t.Fatal("expected unauthorized error")
	}
	if !apperr.Is(err, apperr.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestValidateToken_Valid(t *testing.T) {
	svc := newTestService()
	_, _ = svc.Register(context.Background(), "eve@example.com", "pass", "Eve", "admin")
	access, _, _, err := svc.Login(context.Background(), "eve@example.com", "pass")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	claims, err := svc.ValidateToken(context.Background(), access)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims.Email != "eve@example.com" {
		t.Errorf("email mismatch: %q", claims.Email)
	}
	if claims.Role != "admin" {
		t.Errorf("role mismatch: %q", claims.Role)
	}
}

func TestValidateToken_Invalid(t *testing.T) {
	svc := newTestService()
	_, err := svc.ValidateToken(context.Background(), "not.a.valid.token")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
	if !apperr.Is(err, apperr.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestGetByID_Success(t *testing.T) {
	svc := newTestService()
	registered, _ := svc.Register(context.Background(), "frank@example.com", "pass", "Frank", "customer")

	u, err := svc.GetByID(context.Background(), registered.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.ID != registered.ID {
		t.Errorf("ID mismatch: %q vs %q", u.ID, registered.ID)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	svc := newTestService()
	_, err := svc.GetByID(context.Background(), "nonexistent-id")
	if err == nil {
		t.Fatal("expected not-found error")
	}
	if !apperr.Is(err, apperr.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
