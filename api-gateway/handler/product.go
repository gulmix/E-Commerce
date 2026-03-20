package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"ecommerce/pkg/logger"
	productpb "ecommerce/proto/product"
)

type ProductHandler struct {
	productClient productpb.ProductServiceClient
	log           logger.Logger
}

func NewProductHandler(c productpb.ProductServiceClient, log logger.Logger) *ProductHandler {
	return &ProductHandler{productClient: c, log: log}
}

func (h *ProductHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/products")
	path = strings.TrimPrefix(path, "/")

	if path == "" {
		switch r.Method {
		case http.MethodGet:
			h.listProducts(w, r)
		case http.MethodPost:
			h.createProduct(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getProduct(w, r, path)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

type createProductRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Stock       int32   `json:"stock"`
	CategoryID  string  `json:"category_id"`
}

func (h *ProductHandler) createProduct(w http.ResponseWriter, r *http.Request) {
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
		h.log.Error().Err(err).Msg("CreateProduct failed")
		writeGRPCError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *ProductHandler) getProduct(w http.ResponseWriter, r *http.Request, id string) {
	resp, err := h.productClient.GetProduct(r.Context(), &productpb.GetProductRequest{ProductId: id})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *ProductHandler) listProducts(w http.ResponseWriter, r *http.Request) {
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
	categoryID := q.Get("category_id")

	resp, err := h.productClient.ListProducts(r.Context(), &productpb.ListProductsRequest{
		Page:       page,
		PageSize:   pageSize,
		CategoryId: categoryID,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
