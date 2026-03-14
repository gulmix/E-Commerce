package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ecommerce/pkg/config"
	"ecommerce/pkg/logger"
	userpb "ecommerce/proto/user"
	internalcfg "ecommerce/user-service/internal/config"
	grpcserver "ecommerce/user-service/internal/grpc"
	"ecommerce/user-service/internal/repository"
	"ecommerce/user-service/internal/service"

	"google.golang.org/grpc"
)

func main() {
	cfg := internalcfg.Config{
		GRPCPort:           ":50051",
		LogLevel:           "info",
		JWTSecret:          "change-me-in-production",
		AccessExpiryMin:    15,
		RefreshExpiryHours: 168,
	}
	if err := config.Load(&cfg); err != nil {
		panic("failed to load config: " + err.Error())
	}

	log := logger.New(cfg.LogLevel, true)
	log.Info().Str("port", cfg.GRPCPort).Msg("starting user-service")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repo, err := repository.NewPostgres(ctx, cfg.DBDsn)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}

	if c, ok := repo.(interface{ Close() }); ok {
		defer c.Close()
	}

	svc := service.New(repo, service.Config{
		JWTSecret:     cfg.JWTSecret,
		AccessExpiry:  time.Duration(cfg.AccessExpiryMin) * time.Minute,
		RefreshExpiry: time.Duration(cfg.RefreshExpiryHours) * time.Hour,
	})

	lis, err := net.Listen("tcp", cfg.GRPCPort)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to listen")
	}

	srv := grpc.NewServer()
	userpb.RegisterUserServiceServer(srv, grpcserver.NewServer(svc, log))

	go func() {
		if err := srv.Serve(lis); err != nil {
			log.Fatal().Err(err).Msg("grpc server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	log.Info().Msg("shutting down user-service")
	srv.GracefulStop()
}
