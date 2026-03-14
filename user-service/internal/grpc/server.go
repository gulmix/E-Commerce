package grpc

import (
	"context"
	"errors"
	"time"

	apperr "ecommerce/pkg/errors"
	"ecommerce/pkg/logger"
	userpb "ecommerce/proto/user"
	"ecommerce/user-service/internal/service"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	userpb.UnimplementedUserServiceServer
	svc service.UserService
	log logger.Logger
}

func NewServer(svc service.UserService, log logger.Logger) *Server {
	return &Server{svc: svc, log: log}
}

func (s *Server) CreateUser(ctx context.Context, req *userpb.CreateUserRequest) (*userpb.CreateUserResponse, error) {
	u, err := s.svc.Register(ctx, req.Email, req.Password, req.Name, req.Role)
	if err != nil {
		s.log.Error().Err(err).Str("email", req.Email).Msg("Register failed")
		return nil, toGRPCError(err)
	}
	return &userpb.CreateUserResponse{
		UserId:    u.ID,
		Email:     u.Email,
		Name:      u.Name,
		Role:      u.Role,
		CreatedAt: u.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (s *Server) LoginUser(ctx context.Context, req *userpb.LoginUserRequest) (*userpb.LoginUserResponse, error) {
	access, refresh, u, err := s.svc.Login(ctx, req.Email, req.Password)
	if err != nil {
		s.log.Warn().Err(err).Str("email", req.Email).Msg("Login failed")
		return nil, toGRPCError(err)
	}
	return &userpb.LoginUserResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		UserId:       u.ID,
		Email:        u.Email,
		Role:         u.Role,
	}, nil
}

func (s *Server) GetUser(ctx context.Context, req *userpb.GetUserRequest) (*userpb.GetUserResponse, error) {
	u, err := s.svc.GetByID(ctx, req.UserId)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &userpb.GetUserResponse{
		UserId:    u.ID,
		Email:     u.Email,
		Name:      u.Name,
		Role:      u.Role,
		CreatedAt: u.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (s *Server) ValidateToken(ctx context.Context, req *userpb.ValidateTokenRequest) (*userpb.ValidateTokenResponse, error) {
	claims, err := s.svc.ValidateToken(ctx, req.Token)
	if err != nil {
		return &userpb.ValidateTokenResponse{Valid: false}, nil
	}
	return &userpb.ValidateTokenResponse{
		Valid:   true,
		UserId:  claims.UserID,
		Email:   claims.Email,
		Role:    claims.Role,
	}, nil
}

func toGRPCError(err error) error {
	var appErr *apperr.AppError
	if !errors.As(err, &appErr) {
		return status.Error(codes.Internal, "internal server error")
	}
	switch appErr.Code {
	case apperr.ErrNotFound:
		return status.Error(codes.NotFound, appErr.Message)
	case apperr.ErrUnauthorized:
		return status.Error(codes.Unauthenticated, appErr.Message)
	case apperr.ErrConflict:
		return status.Error(codes.AlreadyExists, appErr.Message)
	case apperr.ErrInvalidInput:
		return status.Error(codes.InvalidArgument, appErr.Message)
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}
