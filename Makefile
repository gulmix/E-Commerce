.PHONY: proto tidy build

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
	cd order-service   && go build ./...
	cd api-gateway     && go build ./...
