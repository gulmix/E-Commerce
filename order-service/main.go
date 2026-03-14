package main

import (
	"net"
	"os"
	"os/signal"
	"syscall"

	"ecommerce/pkg/config"
	"ecommerce/pkg/logger"
	orderpb "ecommerce/proto/order"

	"google.golang.org/grpc"
)

type Config struct {
	GRPCPort string `mapstructure:"GRPC_PORT"`
	DBDsn    string `mapstructure:"DB_DSN"`
	LogLevel string `mapstructure:"LOG_LEVEL"`
}

func main() {
	cfg := Config{GRPCPort: ":50053", LogLevel: "info"}
	if err := config.Load(&cfg); err != nil {
		panic("failed to load config: " + err.Error())
	}

	log := logger.New(cfg.LogLevel, true)
	log.Info().Str("port", cfg.GRPCPort).Msg("starting order-service")

	lis, err := net.Listen("tcp", cfg.GRPCPort)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to listen")
	}

	srv := grpc.NewServer()
	orderpb.RegisterOrderServiceServer(srv, &orderServer{log: log})

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
