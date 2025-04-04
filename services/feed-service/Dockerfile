FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o service cmd/main.go

FROM alpine:3.16
WORKDIR /app
COPY --from=builder /app/service /app/service
RUN chmod +x /app/service
EXPOSE 8085
CMD ["/app/service"]
