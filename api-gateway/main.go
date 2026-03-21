package main

import (
	"context"
	_ "embed"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ecommerce/api-gateway/handler"
	apimw "ecommerce/api-gateway/middleware"
	"ecommerce/pkg/config"
	"ecommerce/pkg/health"
	"ecommerce/pkg/logger"
	"ecommerce/pkg/metrics"
	"ecommerce/pkg/resilience"
	"ecommerce/pkg/telemetry"
	orderpb "ecommerce/proto/order"
	productpb "ecommerce/proto/product"
	userpb "ecommerce/proto/user"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/sony/gobreaker"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

//go:embed openapi.yaml
var openAPISpec []byte

type Config struct {
	HTTPPort        string `mapstructure:"HTTP_PORT"`
	UserServiceAddr string `mapstructure:"USER_SERVICE_ADDR"`
	ProductSvcAddr  string `mapstructure:"PRODUCT_SERVICE_ADDR"`
	OrderSvcAddr    string `mapstructure:"ORDER_SERVICE_ADDR"`
	LogLevel        string `mapstructure:"LOG_LEVEL"`
	RateLimitRPM    int    `mapstructure:"RATE_LIMIT_RPM"`
	OTELEndpoint    string `mapstructure:"OTEL_ENDPOINT"`
}

func main() {
	cfg := Config{
		HTTPPort:        ":8080",
		UserServiceAddr: "user-service:50051",
		ProductSvcAddr:  "product-service:50052",
		OrderSvcAddr:    "order-service:50053",
		LogLevel:        "info",
		RateLimitRPM:    100,
	}
	if err := config.Load(&cfg); err != nil {
		panic("failed to load config: " + err.Error())
	}
	if cfg.RateLimitRPM <= 0 {
		cfg.RateLimitRPM = 100
	}

	log := logger.New(cfg.LogLevel, true)
	log.Info().Str("port", cfg.HTTPPort).Msg("starting api-gateway")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tracerShutdown, err := telemetry.InitTracer(ctx, "api-gateway", cfg.OTELEndpoint)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to init tracer")
	}

	userConn, err := dialService(cfg.UserServiceAddr, resilience.NewBreaker("gw->user"))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to dial user-service")
	}
	defer userConn.Close()

	productConn, err := dialService(cfg.ProductSvcAddr, resilience.NewBreaker("gw->product"))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to dial product-service")
	}
	defer productConn.Close()

	orderConn, err := dialService(cfg.OrderSvcAddr, resilience.NewBreaker("gw->order"))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to dial order-service")
	}
	defer orderConn.Close()

	userClient := userpb.NewUserServiceClient(userConn)
	productClient := productpb.NewProductServiceClient(productConn)
	orderClient := orderpb.NewOrderServiceClient(orderConn)

	authH := handler.NewAuthHandler(userClient, log)
	productH := handler.NewProductHandler(productClient, log)
	orderH := handler.NewOrderHandler(orderClient, log)

	r := chi.NewRouter()

	r.Use(apimw.Logger(log))
	r.Use(chimw.Recoverer)
	r.Use(apimw.RateLimit(cfg.RateLimitRPM))

	r.Get("/health", health.LivenessHandler())
	r.Get("/ready", health.LivenessHandler())

	r.Handle("/metrics", metrics.Handler())

	r.Post("/auth/register", authH.Register)
	r.Post("/auth/login", authH.Login)

	r.Get("/swagger/openapi.yaml", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		_, _ = w.Write(openAPISpec)
	})
	r.Get("/swagger/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(swaggerUIHTML)
	})

	r.Get("/products", productH.List)
	r.Get("/products/{id}", productH.Get)

	r.Group(func(r chi.Router) {
		r.Use(apimw.Auth(userClient))
		r.Post("/products", productH.Create)
		r.Put("/products/{id}", productH.Update)
		r.Delete("/products/{id}", productH.Delete)
		r.Post("/orders", orderH.Create)
		r.Get("/orders", orderH.ListByUser)
		r.Get("/orders/{id}", orderH.Get)
		r.Patch("/orders/{id}/status", orderH.UpdateStatus)
	})

	srv := &http.Server{
		Addr:         cfg.HTTPPort,
		Handler:      otelhttp.NewHandler(r, "api-gateway"),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("http server error")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit

	log.Info().Msg("shutting down api-gateway")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)
	_ = tracerShutdown(shutdownCtx)
}

func dialService(addr string, cb *gobreaker.CircuitBreaker) (*grpc.ClientConn, error) {
	return grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		grpc.WithChainUnaryInterceptor(
			circuitBreakerInterceptor(cb),
			timeoutInterceptor(5*time.Second),
		),
	)
}

func circuitBreakerInterceptor(cb *gobreaker.CircuitBreaker) grpc.UnaryClientInterceptor {
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
		return err
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

var swaggerUIHTML = []byte(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>E-Commerce API — Swagger UI</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script>
  SwaggerUIBundle({
    url: "/swagger/openapi.yaml",
    dom_id: "#swagger-ui",
    presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
    layout: "BaseLayout",
    deepLinking: true,
  });
</script>
</body>
</html>`)
