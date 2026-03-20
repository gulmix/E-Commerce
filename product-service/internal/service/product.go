package service

import (
	"context"

	apperr "ecommerce/pkg/errors"
	"ecommerce/product-service/internal/domain"
	"ecommerce/product-service/internal/repository"
)

type productService struct {
	repo repository.ProductRepository
}

func New(repo repository.ProductRepository) ProductService {
	return &productService{repo: repo}
}

func (s *productService) Create(ctx context.Context, name, description string, price float64, stock int32, categoryID string) (*domain.Product, error) {
	if name == "" {
		return nil, apperr.New(apperr.ErrInvalidInput, "name is required")
	}
	if price < 0 {
		return nil, apperr.New(apperr.ErrInvalidInput, "price must be non-negative")
	}
	if stock < 0 {
		return nil, apperr.New(apperr.ErrInvalidInput, "stock must be non-negative")
	}
	p := &domain.Product{
		Name:        name,
		Description: description,
		Price:       price,
		Stock:       stock,
		CategoryID:  categoryID,
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *productService) GetByID(ctx context.Context, id string) (*domain.Product, error) {
	if id == "" {
		return nil, apperr.New(apperr.ErrInvalidInput, "id is required")
	}
	return s.repo.GetByID(ctx, id)
}

func (s *productService) List(ctx context.Context, page, pageSize int32, categoryID string) ([]*domain.Product, int32, error) {
	return s.repo.List(ctx, page, pageSize, categoryID)
}

func (s *productService) Update(ctx context.Context, id, name, description string, price float64, stock int32, categoryID string, version int32) (*domain.Product, error) {
	if id == "" {
		return nil, apperr.New(apperr.ErrInvalidInput, "id is required")
	}
	if name == "" {
		return nil, apperr.New(apperr.ErrInvalidInput, "name is required")
	}
	if price < 0 {
		return nil, apperr.New(apperr.ErrInvalidInput, "price must be non-negative")
	}
	if stock < 0 {
		return nil, apperr.New(apperr.ErrInvalidInput, "stock must be non-negative")
	}
	p := &domain.Product{
		ID:          id,
		Name:        name,
		Description: description,
		Price:       price,
		Stock:       stock,
		CategoryID:  categoryID,
		Version:     version,
	}
	if err := s.repo.Update(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *productService) Delete(ctx context.Context, id string) error {
	if id == "" {
		return apperr.New(apperr.ErrInvalidInput, "id is required")
	}
	return s.repo.Delete(ctx, id)
}
