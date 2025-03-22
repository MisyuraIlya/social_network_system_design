# syntax=docker/dockerfile:1
FROM golang:1.20-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o feed-service main.go

FROM alpine:3.17
WORKDIR /app
COPY --from=build /app/feed-service /app/
EXPOSE 8003
ENTRYPOINT ["/app/feed-service"]
