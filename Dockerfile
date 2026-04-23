# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy source code
COPY . .

# Download dependencies and build
RUN go mod tidy && go mod download

ARG SERVICE
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/service ./cmd/${SERVICE}

# Run stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates

COPY --from=builder /app/service /service

ENTRYPOINT ["/service"]
