FROM golang:1.25-alpine
WORKDIR /app

RUN apk add --no-cache git curl bash
RUN go install github.com/air-verse/air@latest

COPY go.mod go.sum ./
RUN go mod download

CMD ["/go/bin/air", "-c", "/app/.air.toml"]
