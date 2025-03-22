FROM golang:1.20-alpine AS build
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o media-service main.go

FROM alpine:3.17
WORKDIR /app
COPY --from=build /app/media-service /app/
EXPOSE 8002
ENTRYPOINT ["/app/media-service"]
