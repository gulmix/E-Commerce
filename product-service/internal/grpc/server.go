package grpc

import (
	"context"
	"errors"
	"time"

	apperr "ecommerce/pkg/errors"
	"ecommerce/pkg/logger"
	productpb "ecommerce/proto/product"
	"ecommerce/product-service/internal/service"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	productpb.UnimplementedProductServiceServer
	svc service.ProductService
	log logger.Logger
}

func NewServer(svc service.ProductService, log logger.Logger) *Server {
	return &Server{svc: svc, log: log}
}

func (s *Server) CreateProduct(ctx context.Context, req *productpb.CreateProductRequest) (*productpb.CreateProductResponse, error) {
	p, err := s.svc.Create(ctx, req.Name, req.Description, req.Price, req.Stock, req.CategoryId)
	if err != nil {
		s.log.Error().Err(err).Str("name", req.Name).Msg("CreateProduct failed")
		return nil, toGRPCError(err)
	}
	return &productpb.CreateProductResponse{
		ProductId:   p.ID,
		Name:        p.Name,
		Description: p.Description,
		Price:       p.Price,
		Stock:       p.Stock,
		CategoryId:  p.CategoryID,
		CreatedAt:   p.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (s *Server) GetProduct(ctx context.Context, req *productpb.GetProductRequest) (*productpb.GetProductResponse, error) {
	p, err := s.svc.GetByID(ctx, req.ProductId)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &productpb.GetProductResponse{
		ProductId:   p.ID,
		Name:        p.Name,
		Description: p.Description,
		Price:       p.Price,
		Stock:       p.Stock,
		CategoryId:  p.CategoryID,
		CreatedAt:   p.CreatedAt.Format(time.RFC3339),
		Version:     p.Version,
	}, nil
}

func (s *Server) ListProducts(ctx context.Context, req *productpb.ListProductsRequest) (*productpb.ListProductsResponse, error) {
	products, total, err := s.svc.List(ctx, req.Page, req.PageSize, req.CategoryId)
	if err != nil {
		return nil, toGRPCError(err)
	}
	items := make([]*productpb.ProductItem, 0, len(products))
	for _, p := range products {
		items = append(items, &productpb.ProductItem{
			ProductId:  p.ID,
			Name:       p.Name,
			Price:      p.Price,
			Stock:      p.Stock,
			CategoryId: p.CategoryID,
		})
	}
	return &productpb.ListProductsResponse{
		Products: items,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

func (s *Server) UpdateProduct(ctx context.Context, req *productpb.UpdateProductRequest) (*productpb.UpdateProductResponse, error) {
	p, err := s.svc.Update(ctx, req.ProductId, req.Name, req.Description, req.Price, req.Stock, req.CategoryId, req.Version)
	if err != nil {
		s.log.Error().Err(err).Str("product_id", req.ProductId).Msg("UpdateProduct failed")
		return nil, toGRPCError(err)
	}
	return &productpb.UpdateProductResponse{
		ProductId:   p.ID,
		Name:        p.Name,
		Description: p.Description,
		Price:       p.Price,
		Stock:       p.Stock,
		CategoryId:  p.CategoryID,
		UpdatedAt:   p.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (s *Server) DeleteProduct(ctx context.Context, req *productpb.DeleteProductRequest) (*productpb.DeleteProductResponse, error) {
	if err := s.svc.Delete(ctx, req.ProductId); err != nil {
		return nil, toGRPCError(err)
	}
	return &productpb.DeleteProductResponse{Success: true}, nil
}

func toGRPCError(err error) error {
	var appErr *apperr.AppError
	if !errors.As(err, &appErr) {
		return status.Error(codes.Internal, "internal server error")
	}
	switch appErr.Code {
	case apperr.ErrNotFound:
		return status.Error(codes.NotFound, appErr.Message)
	case apperr.ErrConflict:
		return status.Error(codes.AlreadyExists, appErr.Message)
	case apperr.ErrInvalidInput:
		return status.Error(codes.InvalidArgument, appErr.Message)
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
