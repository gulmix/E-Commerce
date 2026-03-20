package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ecommerce/api-gateway/handler"
	"ecommerce/pkg/config"
	"ecommerce/pkg/logger"
	productpb "ecommerce/proto/product"
	userpb "ecommerce/proto/user"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Config struct {
	HTTPPort        string `mapstructure:"HTTP_PORT"`
	UserServiceAddr string `mapstructure:"USER_SERVICE_ADDR"`
	ProductSvcAddr  string `mapstructure:"PRODUCT_SERVICE_ADDR"`
	OrderSvcAddr    string `mapstructure:"ORDER_SERVICE_ADDR"`
	LogLevel        string `mapstructure:"LOG_LEVEL"`
}

func main() {
	cfg := Config{
		HTTPPort:        ":8080",
		UserServiceAddr: "user-service:50051",
		ProductSvcAddr:  "product-service:50052",
		OrderSvcAddr:    "order-service:50053",
		LogLevel:        "info",
	}
	if err := config.Load(&cfg); err != nil {
		panic("failed to load config: " + err.Error())
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

	userClient := userpb.NewUserServiceClient(userConn)

	productConn, err := grpc.NewClient(cfg.ProductSvcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to dial product-service")
	}
	defer productConn.Close()

	productClient := productpb.NewProductServiceClient(productConn)

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	authH := handler.NewAuthHandler(userClient, log)
	mux.HandleFunc("/auth/register", authH.Register)
	mux.HandleFunc("/auth/login", authH.Login)

	productH := handler.NewProductHandler(productClient, log)
	mux.Handle("/products", productH)
	mux.Handle("/products/", productH)

	srv := &http.Server{
		Addr:         cfg.HTTPPort,
		Handler:      mux,
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
