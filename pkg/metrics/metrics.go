package metrics

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func Handler() http.Handler {
	return promhttp.Handler()
}

func NewServerInterceptor(serviceName string) grpc.UnaryServerInterceptor {
	requests := promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "ecommerce",
		Subsystem: serviceName,
		Name:      "grpc_requests_total",
		Help:      "Total gRPC requests by method and gRPC status code.",
	}, []string{"method", "code"})

	duration := promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "ecommerce",
		Subsystem: serviceName,
		Name:      "grpc_request_duration_seconds",
		Help:      "gRPC request latency distribution.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"method"})

	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		code := status.Code(err).String()
		requests.WithLabelValues(info.FullMethod, code).Inc()
		duration.WithLabelValues(info.FullMethod).Observe(time.Since(start).Seconds())
		return resp, err
	}
}

func NewClientInterceptor(clientName string) grpc.UnaryClientInterceptor {
	requests := promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "ecommerce",
		Subsystem: clientName,
		Name:      "grpc_client_requests_total",
		Help:      "Total outbound gRPC requests by method and gRPC status code.",
	}, []string{"method", "code"})

	duration := promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "ecommerce",
		Subsystem: clientName,
		Name:      "grpc_client_request_duration_seconds",
		Help:      "Outbound gRPC request latency distribution.",
		Buckets:   prometheus.DefBuckets,
	}, []string{"method"})

	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		start := time.Now()
		err := invoker(ctx, method, req, reply, cc, opts...)
		code := status.Code(err).String()
		requests.WithLabelValues(method, code).Inc()
		duration.WithLabelValues(method).Observe(time.Since(start).Seconds())
		return err
	}
}
