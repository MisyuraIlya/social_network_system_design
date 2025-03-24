# -----------------------------------------------------
# 1. Build Stage
# -----------------------------------------------------
    FROM golang:1.20 AS builder

    # Set the working directory inside the container
    WORKDIR /app
    
    # Copy go.mod and go.sum first (to leverage Docker layer caching)
    COPY go.mod go.sum ./
    RUN go mod download
    
    # Copy the rest of your application files
    COPY . .
    
    # Build the Go service
    RUN CGO_ENABLED=0 GOOS=linux go build -o /go/bin/user-service main.go
    
    # -----------------------------------------------------
    # 2. Run Stage
    # -----------------------------------------------------
    FROM alpine:3.18
    
    # Create a non-root user and group (optional but recommended)
    RUN addgroup -S appgroup && adduser -S appuser -G appgroup
    
    # Copy the compiled binary from the builder stage
    COPY --from=builder /go/bin/user-service /usr/local/bin/user-service
    
    # Switch to the non-root user
    USER appuser
    
    # Expose port 8080 (make sure it matches the port your service listens on)
    EXPOSE 8080
    
    # The command to run when the container starts
    ENTRYPOINT ["/usr/local/bin/user-service"]
    