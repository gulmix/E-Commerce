package service

import (
	"context"

	"ecommerce/user-service/internal/domain"
)

type UserService interface {
	Register(ctx context.Context, email, password, name, role string) (*domain.User, error)
	Login(ctx context.Context, email, password string) (accessToken, refreshToken string, user *domain.User, err error)
	GetByID(ctx context.Context, id string) (*domain.User, error)
	ValidateToken(ctx context.Context, token string) (*Claims, error)
}

type Claims struct {
	UserID string
	Email  string
	Role   string
}
