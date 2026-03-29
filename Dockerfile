# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/api

# Final stage
FROM alpine:3.19

WORKDIR /app

COPY --from=builder /app/server .
COPY --from=builder /app/.env.example .env

EXPOSE 8080

CMD ["./server"]
