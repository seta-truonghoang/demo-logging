FROM golang:1.21-alpine

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o main payment-service/main.go
EXPOSE 8087
CMD ["./main"]
