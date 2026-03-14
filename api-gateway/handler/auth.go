package handler

import (
	"encoding/json"
	"net/http"

	"ecommerce/pkg/logger"
	userpb "ecommerce/proto/user"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthHandler struct {
	userClient userpb.UserServiceClient
	log        logger.Logger
}

func NewAuthHandler(c userpb.UserServiceClient, log logger.Logger) *AuthHandler {
	return &AuthHandler{userClient: c, log: log}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Role     string `json:"role"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	resp, err := h.userClient.CreateUser(r.Context(), &userpb.CreateUserRequest{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
		Role:     req.Role,
	})
	if err != nil {
		h.log.Error().Err(err).Str("email", req.Email).Msg("register failed")
		writeGRPCError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	resp, err := h.userClient.LoginUser(r.Context(), &userpb.LoginUserRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		h.log.Warn().Err(err).Str("email", req.Email).Msg("login failed")
		writeGRPCError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func writeGRPCError(w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	switch st.Code() {
	case codes.AlreadyExists:
		http.Error(w, st.Message(), http.StatusConflict)
	case codes.Unauthenticated:
		http.Error(w, st.Message(), http.StatusUnauthorized)
	case codes.NotFound:
		http.Error(w, st.Message(), http.StatusNotFound)
	case codes.InvalidArgument:
		http.Error(w, st.Message(), http.StatusBadRequest)
	default:
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}
