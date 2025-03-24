# service-user/Dockerfile

# Stage 1: build
FROM golang:1.20-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o service cmd/main.go

# Stage 2: run
FROM alpine:3.16
WORKDIR /app
COPY --from=builder /app/service /app/service
EXPOSE 8081
CMD ["/app/service"]
