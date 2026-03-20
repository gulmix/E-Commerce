package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"

	internalcfg "ecommerce/order-service/internal/config"
	grpcserver "ecommerce/order-service/internal/grpc"
	"ecommerce/order-service/internal/messaging"
	"ecommerce/order-service/internal/repository"
	"ecommerce/order-service/internal/service"
	"ecommerce/pkg/config"
	"ecommerce/pkg/logger"
	orderpb "ecommerce/proto/order"
	productpb "ecommerce/proto/product"
	userpb "ecommerce/proto/user"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cfg := internalcfg.Config{
		GRPCPort:           ":50053",
		LogLevel:           "info",
		UserServiceAddr:    "user-service:50051",
		ProductServiceAddr: "product-service:50052",
	}
	if err := config.Load(&cfg); err != nil {
		panic("failed to load config: " + err.Error())
	}

	log := logger.New(cfg.LogLevel, true)
	log.Info().Str("port", cfg.GRPCPort).Msg("starting order-service")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repo, err := repository.NewPostgres(ctx, cfg.DBDsn)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	if c, ok := repo.(interface{ Close() }); ok {
		defer c.Close()
	}

	userConn, err := grpc.NewClient(cfg.UserServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to dial user-service")
	}
	defer userConn.Close()

	productConn, err := grpc.NewClient(cfg.ProductServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to dial product-service")
	}
	defer productConn.Close()

	userClient := userpb.NewUserServiceClient(userConn)
	productClient := productpb.NewProductServiceClient(productConn)

	pub, err := messaging.NewPublisher(cfg.RabbitMQURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to RabbitMQ")
	}
	defer pub.Close()

	svc := service.New(repo, userClient, productClient, pub)

	lis, err := net.Listen("tcp", cfg.GRPCPort)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to listen")
	}

	srv := grpc.NewServer()
	orderpb.RegisterOrderServiceServer(srv, grpcserver.NewServer(svc, log))

	go func() {
		if err := srv.Serve(lis); err != nil {
			log.Fatal().Err(err).Msg("grpc server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	log.Info().Msg("shutting down order-service")
	srv.GracefulStop()
}
