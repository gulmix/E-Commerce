# E-Commerce Microservices Architecture

Go-based microservices platform for e-commerce: services for users, products, and orders.

---

## Tech Stack

- **Language:** Go
- **Transport:** gRPC (inter-service) + REST/HTTP (API Gateway)
- **Database:** PostgreSQL (per service, isolated schemas)
- **Message Broker:** RabbitMQ / Kafka (async events)
- **Auth:** JWT
- **Containerization:** Docker + Docker Compose
- **Orchestration (later):** Kubernetes

---

## Project Structure

```
E-Commerce/
├── api-gateway/          # HTTP entrypoint, routing, auth middleware
├── user-service/         # Registration, login, JWT, profiles
├── product-service/      # Catalog, categories, inventory
├── order-service/        # Cart, orders, order lifecycle
├── proto/                # Shared .proto files (gRPC contracts)
├── pkg/                  # Shared libraries (logger, errors, config)
├── docker-compose.yml
└── README.md
```

---

## Implementation Plan

### Phase 1 — Foundation (Week 1)

**Goal:** Skeleton of all services, Docker environment, shared tooling.

1. Initialize Go modules for each service (`go mod init`)
2. Write shared `proto/` contracts:
   - `user.proto` — CreateUser, GetUser, ValidateToken
   - `product.proto` — CreateProduct, GetProduct, ListProducts
   - `order.proto` — CreateOrder, GetOrder, UpdateOrderStatus
3. Generate gRPC code with `protoc`
4. Set up `pkg/` with:
   - Structured logger (zerolog / zap)
   - Config loader (viper / envconfig)
   - Common error types
5. Write `docker-compose.yml`:
   - PostgreSQL (3 separate databases)
   - RabbitMQ
   - Each service container

---

### Phase 2 — User Service (Week 2)

**Endpoints (gRPC):** RegisterUser, LoginUser, GetUser, ValidateToken

1. Database schema: `users` table (id, email, password_hash, role, created_at)
2. Password hashing with `bcrypt`
3. JWT issue + validate (access + refresh tokens)
4. gRPC server implementation
5. Unit tests for auth logic
6. REST wrapper in API Gateway: `POST /auth/register`, `POST /auth/login`

---

### Phase 3 — Product Service (Week 3)

**Endpoints (gRPC):** CreateProduct, GetProduct, ListProducts, UpdateProduct, DeleteProduct

1. Database schema: `products`, `categories` tables
2. CRUD handlers with pagination and filtering
3. Inventory field (stock count) with optimistic locking
4. gRPC server implementation
5. Unit + integration tests
6. REST wrapper in API Gateway: `GET /products`, `GET /products/:id`, `POST /products`

---

### Phase 4 — Order Service (Week 4)

**Endpoints (gRPC):** CreateOrder, GetOrder, ListUserOrders, UpdateOrderStatus

1. Database schema: `orders`, `order_items` tables
2. On order creation:
   - Call **Product Service** (gRPC) to validate products + reserve stock
   - Call **User Service** (gRPC) to validate user
3. Order status machine: `pending` → `confirmed` → `shipped` → `delivered` / `cancelled`
4. Publish events to RabbitMQ on status change (`order.created`, `order.cancelled`)
5. Unit + integration tests
6. REST wrapper in API Gateway: `POST /orders`, `GET /orders/:id`, `PATCH /orders/:id/status`

---

### Phase 5 — API Gateway (Week 5)

**Goal:** Single HTTP entrypoint for all clients.

1. HTTP router (chi / gin)
2. Auth middleware — validates JWT via User Service gRPC call
3. Route table: maps REST endpoints to gRPC calls on respective services
4. Request/response logging and tracing headers (X-Request-ID)
5. Rate limiting middleware
6. Swagger/OpenAPI spec generation

---

### Phase 6 — Observability & Hardening (Week 6)

1. **Metrics:** Prometheus + Grafana dashboards (request rate, latency, errors)
2. **Tracing:** OpenTelemetry + Jaeger (distributed trace propagation across services)
3. **Health checks:** `GET /health` and `GET /ready` on each service
4. **Graceful shutdown** on SIGTERM in every service
5. **Circuit breaker** on inter-service gRPC calls (sony/gobreaker)
6. **Retry + timeout** policies for all outbound calls

---

### Phase 7 — Kubernetes (Week 7+)

1. Write `Deployment`, `Service`, `ConfigMap`, `Secret` manifests per service
2. Horizontal Pod Autoscaler for product and order services
3. Ingress controller (nginx) replacing API Gateway's Docker role
4. Helm chart for full stack deployment
5. CI/CD pipeline (GitHub Actions): lint → test → build image → push → deploy

---

## Service Communication Map

```
Client
  │
  ▼
API Gateway (REST/HTTP :8080)
  ├──gRPC──► User Service    (:50051) ──► users_db
  ├──gRPC──► Product Service (:50052) ──► products_db
  └──gRPC──► Order Service   (:50053) ──► orders_db
                 │                            │
                 └──gRPC──► Product Service   │
                 └──gRPC──► User Service      │
                                              │
                                     RabbitMQ (events)
```

---

## Running Locally

```bash
# Start all infrastructure and services
docker-compose up --build

# API is available at
http://localhost:8080
```

---

## Development Conventions

- Each service is an independent Go module with its own `go.mod`
- No shared database — services communicate only via gRPC or events
- All config via environment variables (no hardcoded values)
- Every public gRPC method has a corresponding unit test
- Migrations managed per-service with `golang-migrate`