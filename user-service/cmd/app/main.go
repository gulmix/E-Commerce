package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ecommerce/pkg/config"
	"ecommerce/pkg/health"
	"ecommerce/pkg/logger"
	"ecommerce/pkg/metrics"
	"ecommerce/pkg/telemetry"
	userpb "ecommerce/proto/user"
	internalcfg "ecommerce/user-service/internal/config"
	grpcserver "ecommerce/user-service/internal/grpc"
	"ecommerce/user-service/internal/repository"
	"ecommerce/user-service/internal/service"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

func main() {
	cfg := internalcfg.Config{
		GRPCPort:           ":50051",
		MetricsPort:        ":9091",
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

	tracerShutdown, err := telemetry.InitTracer(ctx, "user-service", cfg.OTELEndpoint)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to init tracer")
	}

	repo, err := repository.NewPostgres(ctx, cfg.DBDsn)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}

	type pinger interface{ Ping(context.Context) error }
	var dbPing health.Checker
	if p, ok := repo.(pinger); ok {
		dbPing = p.Ping
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

	grpcSrv := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(metrics.NewServerInterceptor("user_service")),
	)
	userpb.RegisterUserServiceServer(grpcSrv, grpcserver.NewServer(svc, log))

	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.Handler())
	mux.HandleFunc("/health", health.LivenessHandler())
	if dbPing != nil {
		mux.HandleFunc("/ready", health.ReadinessHandler(dbPing))
	} else {
		mux.HandleFunc("/ready", health.LivenessHandler())
	}
	httpSrv := &http.Server{Addr: cfg.MetricsPort, Handler: mux}

	go func() {
		if err := grpcSrv.Serve(lis); err != nil {
			log.Fatal().Err(err).Msg("grpc server error")
		}
	}()
	go func() {
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("metrics/health server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	log.Info().Msg("shutting down user-service")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	_ = httpSrv.Shutdown(shutdownCtx)
	grpcSrv.GracefulStop()
	_ = tracerShutdown(shutdownCtx)
}
