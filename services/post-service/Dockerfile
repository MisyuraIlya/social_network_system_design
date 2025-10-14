# services/post-service/Dockerfile
FROM golang:1.25-alpine
WORKDIR /app

RUN apk add --no-cache git curl bash
RUN go install github.com/air-verse/air@latest

COPY go.mod go.sum ./
RUN go mod download

# ✅ Copy the rest of the source including .air.toml
COPY . .

# Air will now find /app/.air.toml
CMD ["/go/bin/air", "-c", "/app/.air.toml"]
