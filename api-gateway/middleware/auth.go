package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	userpb "ecommerce/proto/user"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ctxKey int

const (
	ctxKeyUserID ctxKey = iota
	ctxKeyRole
)

func UserIDFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyUserID).(string)
	return v
}

func RoleFromCtx(ctx context.Context) string {
	v, _ := ctx.Value(ctxKeyRole).(string)
	return v
}

func Auth(userClient userpb.UserServiceClient) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			token, ok := strings.CutPrefix(authHeader, "Bearer ")
			if !ok || token == "" {
				writeJSONError(w, http.StatusUnauthorized, "missing or invalid authorization header")
				return
			}

			resp, err := userClient.ValidateToken(r.Context(), &userpb.ValidateTokenRequest{Token: token})
			if err != nil {
				st, _ := status.FromError(err)
				if st.Code() == codes.Unauthenticated || st.Code() == codes.InvalidArgument {
					writeJSONError(w, http.StatusUnauthorized, "invalid token")
					return
				}
				writeJSONError(w, http.StatusServiceUnavailable, "authentication service unavailable")
				return
			}
			if !resp.Valid {
				writeJSONError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), ctxKeyUserID, resp.UserId)
			ctx = context.WithValue(ctx, ctxKeyRole, resp.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeJSONError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
