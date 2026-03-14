FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.work go.work.sum* ./
COPY pkg/            pkg/
COPY proto/gen/      proto/gen/
COPY product-service/ product-service/
RUN cd product-service && go build -o /bin/product-service .

FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /bin/product-service /usr/local/bin/product-service
EXPOSE 50052
ENTRYPOINT ["product-service"]
