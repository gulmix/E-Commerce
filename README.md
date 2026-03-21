# E-Commerce Microservices Architecture

Go-based microservices platform for e-commerce: services for users, products, and orders.

---

## Tech Stack

- **Language:** Go
- **Transport:** gRPC (inter-service) + REST/HTTP (API Gateway)
- **Database:** PostgreSQL (per service, isolated schemas)
- **Message Broker:** RabbitMQ (async events)
- **Auth:** JWT
- **Metrics:** Prometheus + Grafana
- **Tracing:** OpenTelemetry + Jaeger (OTLP)
- **Resilience:** circuit breaker (sony/gobreaker), retry, per-call timeout
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
├── pkg/                  # Shared libraries (logger, errors, config, metrics, telemetry, health, resilience)
├── prometheus/           # Prometheus scrape config
├── grafana/              # Grafana datasource provisioning
├── docker-compose.yml
└── README.md
```

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

# Observability UIs
http://localhost:16686   # Jaeger traces
http://localhost:9090    # Prometheus
http://localhost:3000    # Grafana  (admin / admin)
```

---

## Development Conventions

- Each service is an independent Go module with its own `go.mod`
- No shared database — services communicate only via gRPC or events
- All config via environment variables (no hardcoded values)
- Every public gRPC method has a corresponding unit test
- Migrations managed per-service with `golang-migrate`