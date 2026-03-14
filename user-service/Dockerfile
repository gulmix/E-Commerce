FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.work go.work.sum* ./
COPY pkg/         pkg/
COPY proto/gen/   proto/gen/
COPY user-service/ user-service/
RUN cd user-service && go build -o /bin/user-service ./cmd/app

FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
COPY --from=builder /bin/user-service /usr/local/bin/user-service
EXPOSE 50051
ENTRYPOINT ["user-service"]
