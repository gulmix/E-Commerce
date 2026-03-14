package main

import (
	"context"

	"ecommerce/pkg/logger"
	productpb "ecommerce/proto/product"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type productServer struct {
	productpb.UnimplementedProductServiceServer
	log logger.Logger
}

func (s *productServer) CreateProduct(_ context.Context, req *productpb.CreateProductRequest) (*productpb.CreateProductResponse, error) {
	s.log.Info().Str("name", req.Name).Msg("CreateProduct called")
	return nil, status.Error(codes.Unimplemented, "not implemented yet")
}

func (s *productServer) GetProduct(_ context.Context, req *productpb.GetProductRequest) (*productpb.GetProductResponse, error) {
	s.log.Info().Str("product_id", req.ProductId).Msg("GetProduct called")
	return nil, status.Error(codes.Unimplemented, "not implemented yet")
}

func (s *productServer) ListProducts(_ context.Context, req *productpb.ListProductsRequest) (*productpb.ListProductsResponse, error) {
	s.log.Info().Int32("page", req.Page).Msg("ListProducts called")
	return nil, status.Error(codes.Unimplemented, "not implemented yet")
}
