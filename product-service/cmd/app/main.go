package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"

	"ecommerce/pkg/config"
	"ecommerce/pkg/logger"
	productpb "ecommerce/proto/product"
	internalcfg "ecommerce/product-service/internal/config"
	grpcserver "ecommerce/product-service/internal/grpc"
	"ecommerce/product-service/internal/repository"
	"ecommerce/product-service/internal/service"

	"google.golang.org/grpc"
)

func main() {
	cfg := internalcfg.Config{
		GRPCPort: ":50052",
		LogLevel: "info",
	}
	if err := config.Load(&cfg); err != nil {
		panic("failed to load config: " + err.Error())
	}

	log := logger.New(cfg.LogLevel, true)
	log.Info().Str("port", cfg.GRPCPort).Msg("starting product-service")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repo, err := repository.NewPostgres(ctx, cfg.DBDsn)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}

	if c, ok := repo.(interface{ Close() }); ok {
		defer c.Close()
	}

	svc := service.New(repo)

	lis, err := net.Listen("tcp", cfg.GRPCPort)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to listen")
	}

	srv := grpc.NewServer()
	productpb.RegisterProductServiceServer(srv, grpcserver.NewServer(svc, log))

	go func() {
		if err := srv.Serve(lis); err != nil {
			log.Fatal().Err(err).Msg("grpc server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	log.Info().Msg("shutting down product-service")
	srv.GracefulStop()
}
