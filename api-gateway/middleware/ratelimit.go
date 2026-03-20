package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/httprate"
)

func RateLimit(requestsPerMin int) func(http.Handler) http.Handler {
	return httprate.LimitByIP(requestsPerMin, time.Minute)
}
