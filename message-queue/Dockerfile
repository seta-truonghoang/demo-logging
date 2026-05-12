# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and embedded resources
COPY . .

# Build statically linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/demo-app .

# Run stage (minimal distroless-like Alpine image)
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /

COPY --from=builder /app/demo-app /demo-app

EXPOSE 8080

ENTRYPOINT ["/demo-app"]
