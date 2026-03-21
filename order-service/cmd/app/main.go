package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	internalcfg "ecommerce/order-service/internal/config"
	grpcserver "ecommerce/order-service/internal/grpc"
	"ecommerce/order-service/internal/messaging"
	"ecommerce/order-service/internal/repository"
	"ecommerce/order-service/internal/service"
	"ecommerce/pkg/config"
	"ecommerce/pkg/health"
	"ecommerce/pkg/logger"
	"ecommerce/pkg/metrics"
	"ecommerce/pkg/resilience"
	"ecommerce/pkg/telemetry"
	orderpb "ecommerce/proto/order"
	productpb "ecommerce/proto/product"
	userpb "ecommerce/proto/user"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func main() {
	cfg := internalcfg.Config{
		GRPCPort:           ":50053",
		MetricsPort:        ":9093",
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

	tracerShutdown, err := telemetry.InitTracer(ctx, "order-service", cfg.OTELEndpoint)
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

	userCB := resilience.NewBreaker("order->user")
	productCB := resilience.NewBreaker("order->product")

	userConn, err := grpc.NewClient(cfg.UserServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		grpc.WithChainUnaryInterceptor(
			metrics.NewClientInterceptor("order_user_client"),
			circuitBreakerInterceptor(userCB, log),
			retryInterceptor(3),
			timeoutInterceptor(5*time.Second),
		),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to dial user-service")
	}
	defer userConn.Close()

	productConn, err := grpc.NewClient(cfg.ProductServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		grpc.WithChainUnaryInterceptor(
			metrics.NewClientInterceptor("order_product_client"),
			circuitBreakerInterceptor(productCB, log),
			retryInterceptor(3),
			timeoutInterceptor(5*time.Second),
		),
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

	grpcSrv := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(metrics.NewServerInterceptor("order_service")),
	)
	orderpb.RegisterOrderServiceServer(grpcSrv, grpcserver.NewServer(svc, log))

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

	log.Info().Msg("shutting down order-service")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	_ = httpSrv.Shutdown(shutdownCtx)
	grpcSrv.GracefulStop()
	_ = tracerShutdown(shutdownCtx)
}

func circuitBreakerInterceptor(cb interface {
	Execute(func() (interface{}, error)) (interface{}, error)
	Name() string
}, log logger.Logger) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		_, err := cb.Execute(func() (interface{}, error) {
			return nil, invoker(ctx, method, req, reply, cc, opts...)
		})
		if err != nil {
			log.Warn().Str("method", method).Str("breaker", cb.Name()).Err(err).Msg("circuit breaker")
		}
		return err
	}
}

func retryInterceptor(maxAttempts int) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		var lastErr error
		for attempt := 1; attempt <= maxAttempts; attempt++ {
			lastErr = invoker(ctx, method, req, reply, cc, opts...)
			if lastErr == nil {
				return nil
			}
			code := status.Code(lastErr)
			if code != codes.Unavailable && code != codes.ResourceExhausted {
				return lastErr
			}
			if attempt < maxAttempts {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(time.Duration(attempt) * 200 * time.Millisecond):
				}
			}
		}
		return lastErr
	}
}

func timeoutInterceptor(timeout time.Duration) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		if _, ok := ctx.Deadline(); !ok {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
