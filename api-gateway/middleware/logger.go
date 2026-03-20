package middleware

import (
	"fmt"
	"net/http"
	"time"

	"ecommerce/pkg/logger"

	chimw "github.com/go-chi/chi/v5/middleware"
)

func Logger(log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := r.Header.Get("X-Request-ID")
			if reqID == "" {
				reqID = fmt.Sprintf("%d", time.Now().UnixNano())
			}
			w.Header().Set("X-Request-ID", reqID)

			ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()

			next.ServeHTTP(ww, r)

			latency := time.Since(start).Milliseconds()
			statusCode := ww.Status()

			entry := log.With("request_id", reqID).Logger.
				With().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Int("status", statusCode).
				Int64("latency_ms", latency).
				Str("remote_addr", r.RemoteAddr).
				Logger()

			switch {
			case statusCode >= 500:
				entry.Error().Msg("request")
			case statusCode >= 400:
				entry.Warn().Msg("request")
			default:
				entry.Info().Msg("request")
			}
		})
	}
}
