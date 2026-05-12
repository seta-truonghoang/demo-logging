# ==========================================
# STAGE 1: Build Environment
# ==========================================
FROM golang:1.22-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Install build dependencies (needed if compiling static CGO)
RUN apk add --no-cache git ca-certificates

# Copy dependency definition files first to leverage Docker cache layers
COPY go.mod ./
# COPY go.sum ./

# Download dependencies
RUN go mod download

# Copy the entire source code
COPY . .

# Compile the application optimally:
# - CGO_ENABLED=0: Creates a standalone static binary with no runtime dependencies on host libc
# - GOOS=linux: Explicitly targets Linux operating systems
# - -ldflags="-s -w": Strips debug symbols and DWARF tables, reducing final binary size by 50-70%
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /app/server main.go

# ==========================================
# STAGE 2: Runtime Environment
# ==========================================
FROM alpine:3.19.1

# Create a non-privileged group and user to prevent privilege escalation vulnerabilities
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

# Copy SSL certificates from builder stage for secure outbound HTTPS calls
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# Copy the compiled binary from the builder stage
COPY --from=builder /app/server /app/server

# Change ownership of the binary to the non-root user
RUN chown -R appuser:appgroup /app

# Run as the non-privileged user
USER appuser

# Configure default environment variables
ENV PORT=8080

# Expose port
EXPOSE 8080

# Run the application
ENTRYPOINT ["/app/server"]
