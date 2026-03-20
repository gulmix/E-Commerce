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
	"ecommerce/pkg/logger"
	orderpb "ecommerce/proto/order"
	productpb "ecommerce/proto/product"
	userpb "ecommerce/proto/user"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
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

	userConn, err := grpc.NewClient(cfg.UserServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to dial user-service")
	}
	defer userConn.Close()

	productConn, err := grpc.NewClient(cfg.ProductSvcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to dial product-service")
	}
	defer productConn.Close()

	orderConn, err := grpc.NewClient(cfg.OrderSvcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
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

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
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
		Handler:      r,
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
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
