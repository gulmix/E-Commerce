package main

import (
	"context"

	"ecommerce/pkg/logger"
	orderpb "ecommerce/proto/order"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type orderServer struct {
	orderpb.UnimplementedOrderServiceServer
	log logger.Logger
}

func (s *orderServer) CreateOrder(_ context.Context, req *orderpb.CreateOrderRequest) (*orderpb.CreateOrderResponse, error) {
	s.log.Info().Str("user_id", req.UserId).Msg("CreateOrder called")
	return nil, status.Error(codes.Unimplemented, "not implemented yet")
}

func (s *orderServer) GetOrder(_ context.Context, req *orderpb.GetOrderRequest) (*orderpb.GetOrderResponse, error) {
	s.log.Info().Str("order_id", req.OrderId).Msg("GetOrder called")
	return nil, status.Error(codes.Unimplemented, "not implemented yet")
}

func (s *orderServer) UpdateOrderStatus(_ context.Context, req *orderpb.UpdateOrderStatusRequest) (*orderpb.UpdateOrderStatusResponse, error) {
	s.log.Info().Str("order_id", req.OrderId).Msg("UpdateOrderStatus called")
	return nil, status.Error(codes.Unimplemented, "not implemented yet")
}
