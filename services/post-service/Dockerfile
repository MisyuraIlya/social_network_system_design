# syntax=docker/dockerfile:1
FROM golang:1.20-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o post-service main.go

FROM alpine:3.17
WORKDIR /app
COPY --from=build /app/post-service /app/
EXPOSE 8001
ENTRYPOINT ["/app/post-service"]
