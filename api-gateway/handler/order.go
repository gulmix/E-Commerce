package handler

import (
	"encoding/json"
	"net/http"

	"ecommerce/api-gateway/middleware"
	"ecommerce/pkg/logger"
	orderpb "ecommerce/proto/order"

	"github.com/go-chi/chi/v5"
)

type OrderHandler struct {
	orderClient orderpb.OrderServiceClient
	log         logger.Logger
}

func NewOrderHandler(c orderpb.OrderServiceClient, log logger.Logger) *OrderHandler {
	return &OrderHandler{orderClient: c, log: log}
}

type createOrderRequest struct {
	Items []orderItemRequest `json:"items"`
}

type orderItemRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int32  `json:"quantity"`
}

func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	var req createOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	items := make([]*orderpb.OrderItem, len(req.Items))
	for i, it := range req.Items {
		items[i] = &orderpb.OrderItem{ProductId: it.ProductID, Quantity: it.Quantity}
	}
	resp, err := h.orderClient.CreateOrder(r.Context(), &orderpb.CreateOrderRequest{
		UserId: userID,
		Items:  items,
	})
	if err != nil {
		h.log.Error().Err(err).Str("user_id", userID).Msg("CreateOrder failed")
		writeGRPCError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *OrderHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	resp, err := h.orderClient.GetOrder(r.Context(), &orderpb.GetOrderRequest{OrderId: id})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *OrderHandler) ListByUser(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	resp, err := h.orderClient.ListUserOrders(r.Context(), &orderpb.ListUserOrdersRequest{UserId: userID})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

var protoStatusMap = map[string]orderpb.OrderStatus{
	"confirmed": orderpb.OrderStatus_ORDER_STATUS_CONFIRMED,
	"shipped":   orderpb.OrderStatus_ORDER_STATUS_SHIPPED,
	"delivered": orderpb.OrderStatus_ORDER_STATUS_DELIVERED,
	"cancelled": orderpb.OrderStatus_ORDER_STATUS_CANCELLED,
}

type updateOrderStatusRequest struct {
	Status string `json:"status"`
}

func (h *OrderHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req updateOrderStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	protoStatus, ok := protoStatusMap[req.Status]
	if !ok {
		http.Error(w, "invalid status value", http.StatusBadRequest)
		return
	}
	resp, err := h.orderClient.UpdateOrderStatus(r.Context(), &orderpb.UpdateOrderStatusRequest{
		OrderId: id,
		Status:  protoStatus,
	})
	if err != nil {
		h.log.Error().Err(err).Str("order_id", id).Msg("UpdateOrderStatus failed")
		writeGRPCError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
