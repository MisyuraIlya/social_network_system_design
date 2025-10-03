FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o api-gateway cmd/main.go

FROM alpine:3.16
WORKDIR /app
COPY --from=builder /app/api-gateway /app/api-gateway
EXPOSE 8080
CMD ["/app/api-gateway"]
