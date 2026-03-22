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
- **Orchestration:** Kubernetes

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
├── k8s/                  # Raw Kubernetes manifests (per service + infra)
├── helm/ecommerce/       # Helm chart for full-stack deployment
├── .github/workflows/    # CI (lint→test→build) + CD (push→deploy)
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

## Kubernetes Deployment

### Raw manifests

```bash
# Apply all manifests in dependency order
make k8s-apply

# Or individually:
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/postgres-users.yaml
kubectl apply -f k8s/postgres-products.yaml
kubectl apply -f k8s/postgres-orders.yaml
kubectl apply -f k8s/rabbitmq.yaml
kubectl apply -f k8s/observability/
kubectl apply -f k8s/user-service.yaml
kubectl apply -f k8s/product-service.yaml
kubectl apply -f k8s/order-service.yaml
kubectl apply -f k8s/api-gateway.yaml
kubectl apply -f k8s/ingress.yaml
```

### Helm (recommended)

```bash
# Install nginx ingress controller first
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.10.0/deploy/static/provider/cloud/deploy.yaml

# Install the full stack
make helm-install IMAGE_TAG=latest

# Upgrade after a new image push
make helm-upgrade IMAGE_TAG=<git-sha>

# Override values for production
helm upgrade --install ecommerce helm/ecommerce \
  --namespace ecommerce --create-namespace \
  --set global.imageRegistry=ghcr.io/OWNER \
  --set global.imageTag=<git-sha> \
  --set userService.secrets.jwtSecret=<real-secret>
```

### CI/CD (GitHub Actions)

| Workflow | Trigger | Steps |
|---|---|---|
| `ci.yml` | push / PR | lint → test → docker build (no push) |
| `deploy.yml` | push to `main` | build → push to GHCR → helm upgrade → verify rollout |

**Required GitHub secrets:**
- `KUBECONFIG` — kubeconfig YAML for your cluster

**Required GitHub variables:**
- Images are pushed to `ghcr.io/<repository_owner>/ecommerce-<service>:<sha>`

---

## Development Conventions

- Each service is an independent Go module with its own `go.mod`
- No shared database — services communicate only via gRPC or events
- All config via environment variables (no hardcoded values)
- Every public gRPC method has a corresponding unit test
- Migrations managed per-service with `golang-migrate`