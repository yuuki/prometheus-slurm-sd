# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
RUN make build

# Run stage - using a minimal alpine image
FROM alpine:3.20

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/bin/prometheus-slurm-sd /app/
# Copy config file
COPY config.yaml /app/

# Create a non-root user to run the application
RUN adduser -D -u 10001 appuser
RUN chown -R appuser:appuser /app
USER appuser

# Expose the default port
EXPOSE 8080

# Set the entrypoint
ENTRYPOINT ["/app/prometheus-slurm-sd"]
CMD ["--config.file=/app/config.yaml"]
