package main

import (
	"os"
)

func main() {
	// Setup broker address from environment variable or fallback to localhost
	brokerAddr := os.Getenv("KAFKA_BROKERS")
	if brokerAddr == "" {
		brokerAddr = "localhost:9094"
	}

	app := NewApp(brokerAddr, "ecommerce-events", "order-processing-group")
	app.Run("8080")
}
