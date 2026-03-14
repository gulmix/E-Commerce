package service

import (
	"context"
	"fmt"
	"time"

	apperr "ecommerce/pkg/errors"
	"ecommerce/user-service/internal/domain"
	"ecommerce/user-service/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type jwtClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// Config holds JWT settings for the service.
type Config struct {
	JWTSecret     string
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
}

type userService struct {
	repo repository.UserRepository
	cfg  Config
}

func New(repo repository.UserRepository, cfg Config) UserService {
	return &userService{repo: repo, cfg: cfg}
}

func (s *userService) Register(ctx context.Context, email, password, name, role string) (*domain.User, error) {
	if role == "" {
		role = "customer"
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	u := &domain.User{
		Email:        email,
		PasswordHash: string(hash),
		Name:         name,
		Role:         role,
	}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *userService) Login(ctx context.Context, email, password string) (string, string, *domain.User, error) {
	u, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		// Don't leak whether the email exists
		return "", "", nil, apperr.New(apperr.ErrUnauthorized, "invalid credentials")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return "", "", nil, apperr.New(apperr.ErrUnauthorized, "invalid credentials")
	}
	access, err := s.issueToken(u, s.cfg.AccessExpiry)
	if err != nil {
		return "", "", nil, err
	}
	refresh, err := s.issueToken(u, s.cfg.RefreshExpiry)
	if err != nil {
		return "", "", nil, err
	}
	return access, refresh, u, nil
}

func (s *userService) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *userService) ValidateToken(ctx context.Context, token string) (*Claims, error) {
	t, err := jwt.ParseWithClaims(token, &jwtClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil || !t.Valid {
		return nil, apperr.New(apperr.ErrUnauthorized, "invalid or expired token")
	}
	c, ok := t.Claims.(*jwtClaims)
	if !ok {
		return nil, apperr.New(apperr.ErrUnauthorized, "invalid token claims")
	}
	return &Claims{UserID: c.UserID, Email: c.Email, Role: c.Role}, nil
}

func (s *userService) issueToken(u *domain.User, expiry time.Duration) (string, error) {
	now := time.Now()
	claims := &jwtClaims{
		UserID: u.ID,
		Email:  u.Email,
		Role:   u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}
