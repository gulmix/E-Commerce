package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"ecommerce/api-gateway/middleware"
	"ecommerce/pkg/logger"
	productpb "ecommerce/proto/product"

	"github.com/go-chi/chi/v5"
)

type ProductHandler struct {
	productClient productpb.ProductServiceClient
	log           logger.Logger
}

func NewProductHandler(c productpb.ProductServiceClient, log logger.Logger) *ProductHandler {
	return &ProductHandler{productClient: c, log: log}
}

type createProductRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Stock       int32   `json:"stock"`
	CategoryID  string  `json:"category_id"`
}

func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	if middleware.RoleFromCtx(r.Context()) != "admin" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	var req createProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	resp, err := h.productClient.CreateProduct(r.Context(), &productpb.CreateProductRequest{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
		CategoryId:  req.CategoryID,
	})
	if err != nil {
		h.log.Error().Err(err).Str("name", req.Name).Msg("CreateProduct failed")
		writeGRPCError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *ProductHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	resp, err := h.productClient.GetProduct(r.Context(), &productpb.GetProductRequest{ProductId: id})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var page, pageSize int32 = 1, 20
	if v := q.Get("page"); v != "" {
		var p int32
		if _, err := fmt.Sscan(v, &p); err == nil && p > 0 {
			page = p
		}
	}
	if v := q.Get("page_size"); v != "" {
		var ps int32
		if _, err := fmt.Sscan(v, &ps); err == nil && ps > 0 {
			pageSize = ps
		}
	}
	resp, err := h.productClient.ListProducts(r.Context(), &productpb.ListProductsRequest{
		Page:       page,
		PageSize:   pageSize,
		CategoryId: q.Get("category_id"),
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

type updateProductRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Stock       int32   `json:"stock"`
	CategoryID  string  `json:"category_id"`
	Version     int32   `json:"version"`
}

func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	if middleware.RoleFromCtx(r.Context()) != "admin" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	id := chi.URLParam(r, "id")
	var req updateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	resp, err := h.productClient.UpdateProduct(r.Context(), &productpb.UpdateProductRequest{
		ProductId:   id,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
		CategoryId:  req.CategoryID,
		Version:     req.Version,
	})
	if err != nil {
		h.log.Error().Err(err).Str("product_id", id).Msg("UpdateProduct failed")
		writeGRPCError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *ProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if middleware.RoleFromCtx(r.Context()) != "admin" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	id := chi.URLParam(r, "id")
	_, err := h.productClient.DeleteProduct(r.Context(), &productpb.DeleteProductRequest{ProductId: id})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
