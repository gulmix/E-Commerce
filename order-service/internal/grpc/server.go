package grpc

import (
	"context"
	"errors"
	"time"

	"ecommerce/order-service/internal/domain"
	"ecommerce/order-service/internal/service"
	apperr "ecommerce/pkg/errors"
	"ecommerce/pkg/logger"
	orderpb "ecommerce/proto/order"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	orderpb.UnimplementedOrderServiceServer
	svc service.OrderService
	log logger.Logger
}

func NewServer(svc service.OrderService, log logger.Logger) *Server {
	return &Server{svc: svc, log: log}
}

func (s *Server) CreateOrder(ctx context.Context, req *orderpb.CreateOrderRequest) (*orderpb.CreateOrderResponse, error) {
	items := make([]domain.OrderItem, len(req.Items))
	for i, it := range req.Items {
		items[i] = domain.OrderItem{
			ProductID: it.ProductId,
			Quantity:  it.Quantity,
		}
	}

	o, err := s.svc.CreateOrder(ctx, req.UserId, items)
	if err != nil {
		s.log.Error().Err(err).Str("user_id", req.UserId).Msg("CreateOrder failed")
		return nil, toGRPCError(err)
	}
	return &orderpb.CreateOrderResponse{
		OrderId:    o.ID,
		UserId:     o.UserID,
		Status:     toProtoStatus(o.Status),
		TotalPrice: o.TotalPrice,
		CreatedAt:  o.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (s *Server) GetOrder(ctx context.Context, req *orderpb.GetOrderRequest) (*orderpb.GetOrderResponse, error) {
	o, err := s.svc.GetOrder(ctx, req.OrderId)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return toGetOrderResponse(o), nil
}

func (s *Server) ListUserOrders(ctx context.Context, req *orderpb.ListUserOrdersRequest) (*orderpb.ListUserOrdersResponse, error) {
	orders, err := s.svc.ListUserOrders(ctx, req.UserId)
	if err != nil {
		return nil, toGRPCError(err)
	}
	resp := make([]*orderpb.GetOrderResponse, 0, len(orders))
	for _, o := range orders {
		resp = append(resp, toGetOrderResponse(o))
	}
	return &orderpb.ListUserOrdersResponse{Orders: resp}, nil
}

func (s *Server) UpdateOrderStatus(ctx context.Context, req *orderpb.UpdateOrderStatusRequest) (*orderpb.UpdateOrderStatusResponse, error) {
	domainStatus := fromProtoStatus(req.Status)
	o, err := s.svc.UpdateOrderStatus(ctx, req.OrderId, domainStatus)
	if err != nil {
		s.log.Error().Err(err).Str("order_id", req.OrderId).Msg("UpdateOrderStatus failed")
		return nil, toGRPCError(err)
	}
	return &orderpb.UpdateOrderStatusResponse{
		OrderId:   o.ID,
		Status:    toProtoStatus(o.Status),
		UpdatedAt: o.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func toGetOrderResponse(o *domain.Order) *orderpb.GetOrderResponse {
	items := make([]*orderpb.OrderItem, len(o.Items))
	for i, it := range o.Items {
		items[i] = &orderpb.OrderItem{
			ProductId: it.ProductID,
			Quantity:  it.Quantity,
			UnitPrice: it.UnitPrice,
		}
	}
	return &orderpb.GetOrderResponse{
		OrderId:    o.ID,
		UserId:     o.UserID,
		Status:     toProtoStatus(o.Status),
		TotalPrice: o.TotalPrice,
		Items:      items,
		CreatedAt:  o.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  o.UpdatedAt.Format(time.RFC3339),
	}
}

func toProtoStatus(s domain.OrderStatus) orderpb.OrderStatus {
	switch s {
	case domain.StatusPending:
		return orderpb.OrderStatus_ORDER_STATUS_PENDING
	case domain.StatusConfirmed:
		return orderpb.OrderStatus_ORDER_STATUS_CONFIRMED
	case domain.StatusShipped:
		return orderpb.OrderStatus_ORDER_STATUS_SHIPPED
	case domain.StatusDelivered:
		return orderpb.OrderStatus_ORDER_STATUS_DELIVERED
	case domain.StatusCancelled:
		return orderpb.OrderStatus_ORDER_STATUS_CANCELLED
	default:
		return orderpb.OrderStatus_ORDER_STATUS_UNSPECIFIED
	}
}

func fromProtoStatus(s orderpb.OrderStatus) domain.OrderStatus {
	switch s {
	case orderpb.OrderStatus_ORDER_STATUS_CONFIRMED:
		return domain.StatusConfirmed
	case orderpb.OrderStatus_ORDER_STATUS_SHIPPED:
		return domain.StatusShipped
	case orderpb.OrderStatus_ORDER_STATUS_DELIVERED:
		return domain.StatusDelivered
	case orderpb.OrderStatus_ORDER_STATUS_CANCELLED:
		return domain.StatusCancelled
	default:
		return domain.StatusPending
	}
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
