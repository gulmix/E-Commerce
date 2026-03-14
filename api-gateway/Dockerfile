FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.work go.work.sum* ./
COPY pkg/         pkg/
COPY proto/gen/   proto/gen/
COPY api-gateway/ api-gateway/
RUN cd api-gateway && go build -o /bin/api-gateway .

FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /bin/api-gateway /usr/local/bin/api-gateway
EXPOSE 8080
ENTRYPOINT ["api-gateway"]
