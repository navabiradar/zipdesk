FROM golang:1.22-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux \
    go build -ldflags="-w -s" \
    -o bin/api cmd/api/main.go
RUN CGO_ENABLED=0 GOOS=linux \
    go build -ldflags="-w -s" \
    -o bin/worker cmd/worker/main.go

FROM alpine:3.19
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /app/bin/api .
COPY --from=builder /app/bin/worker .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/config ./config
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s \
    CMD wget -qO- http://localhost:8080/health || exit 1
CMD ["./api"]
