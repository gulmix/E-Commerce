.PHONY: proto tidy build k8s-apply k8s-delete helm-install helm-upgrade helm-uninstall

PROTO_OUT := proto/gen

proto:
	protoc \
		--go_out=$(PROTO_OUT) --go_opt=module=ecommerce/proto \
		--go-grpc_out=$(PROTO_OUT) --go-grpc_opt=module=ecommerce/proto \
		proto/user/user.proto proto/product/product.proto proto/order/order.proto

tidy:
	cd pkg            && go mod tidy
	cd proto/gen      && go mod tidy
	cd user-service   && go mod tidy
	cd product-service && go mod tidy
	cd order-service  && go mod tidy
	cd api-gateway    && go mod tidy

build:
	cd user-service    && go build ./cmd/app
	cd product-service && go build ./cmd/app
	cd order-service   && go build ./cmd/app
	cd api-gateway     && go build ./...

# ── Kubernetes (raw manifests) ────────────────────────────────────────────────
NAMESPACE ?= ecommerce

k8s-apply:
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

k8s-delete:
	kubectl delete -f k8s/ --recursive --ignore-not-found

# ── Helm ──────────────────────────────────────────────────────────────────────
HELM_RELEASE ?= ecommerce
HELM_CHART   := helm/ecommerce
IMAGE_TAG    ?= latest

helm-install:
	helm install $(HELM_RELEASE) $(HELM_CHART) \
		--namespace $(NAMESPACE) --create-namespace \
		--set global.imageTag=$(IMAGE_TAG)

helm-upgrade:
	helm upgrade $(HELM_RELEASE) $(HELM_CHART) \
		--namespace $(NAMESPACE) \
		--set global.imageTag=$(IMAGE_TAG) \
		--wait --timeout 5m

helm-uninstall:
	helm uninstall $(HELM_RELEASE) --namespace $(NAMESPACE)
