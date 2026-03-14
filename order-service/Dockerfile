FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.work go.work.sum* ./
COPY pkg/          pkg/
COPY proto/gen/    proto/gen/
COPY order-service/ order-service/
RUN cd order-service && go build -o /bin/order-service .

FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /bin/order-service /usr/local/bin/order-service
EXPOSE 50053
ENTRYPOINT ["order-service"]
