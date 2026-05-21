package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/segmentio/kafka-go"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Models
type OrderStatus string

const (
	StatusPending   OrderStatus = "PENDING"
	StatusConfirmed OrderStatus = "CONFIRMED"
	StatusCancelled OrderStatus = "CANCELLED"
)

type Order struct {
	ID        uint        `gorm:"primaryKey" json:"id"`
	AccountID string      `json:"account_id"`
	Amount    int64       `json:"amount"`
	Status    OrderStatus `json:"status"`
	CreatedAt time.Time   `json:"created_at"`
}

// Events
type OrderCreatedEvent struct {
	OrderID   uint   `json:"order_id"`
	AccountID string `json:"account_id"`
	Amount    int64  `json:"amount"`
}

type PaymentEvent struct {
	OrderID uint   `json:"order_id"`
	Status  string `json:"status"` // SUCCESS or FAILED
	Reason  string `json:"reason"`
}

func main() {
	// 1. Setup DB
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=demo password=demo dbname=demo port=5432 sslmode=disable TimeZone=UTC"
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	db.AutoMigrate(&Order{})

	// 2. Setup Kafka
	kafkaBroker := os.Getenv("KAFKA_BROKER")
	if kafkaBroker == "" {
		kafkaBroker = "localhost:9092"
	}

	orderTopic := "order_events"
	paymentTopic := "payment_events"

	writer := &kafka.Writer{
		Addr:     kafka.TCP(kafkaBroker),
		Topic:    orderTopic,
		Balancer: &kafka.LeastBytes{},
	}
	defer writer.Close()

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{kafkaBroker},
		Topic:   paymentTopic,
		GroupID: "order-service-group",
	})
	defer reader.Close()

	// 3. Start Kafka Consumer (Listen to Payment Events)
	go func() {
		for {
			m, err := reader.ReadMessage(context.Background())
			if err != nil {
				log.Printf("error reading message: %v", err)
				continue
			}

			var event PaymentEvent
			if err := json.Unmarshal(m.Value, &event); err != nil {
				log.Printf("error unmarshalling event: %v", err)
				continue
			}

			log.Printf("Received payment event for order %d: %s", event.OrderID, event.Status)

			var order Order
			if err := db.First(&order, event.OrderID).Error; err != nil {
				log.Printf("order not found: %d", event.OrderID)
				continue
			}

			if event.Status == "SUCCESS" {
				order.Status = StatusConfirmed
			} else {
				order.Status = StatusCancelled
			}
			db.Save(&order)
		}
	}()

	// 4. Setup HTTP Server
	r := gin.Default()
	r.POST("/orders", func(c *gin.Context) {
		var req struct {
			AccountID string `json:"account_id" binding:"required"`
			Amount    int64  `json:"amount" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		order := Order{
			AccountID: req.AccountID,
			Amount:    req.Amount,
			Status:    StatusPending,
		}

		// Save to DB
		if err := db.Create(&order).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create order"})
			return
		}

		// Publish Event
		event := OrderCreatedEvent{
			OrderID:   order.ID,
			AccountID: order.AccountID,
			Amount:    order.Amount,
		}
		eventBytes, _ := json.Marshal(event)

		err = writer.WriteMessages(context.Background(),
			kafka.Message{
				Key:   []byte(fmt.Sprintf("%d", order.ID)),
				Value: eventBytes,
			},
		)
		if err != nil {
			log.Printf("failed to write kafka message: %v", err)
			// Note: For a robust production system, use the Transactional Outbox pattern here.
		}

		c.JSON(http.StatusCreated, order)
	})

	r.GET("/orders/:id", func(c *gin.Context) {
		var order Order
		if err := db.First(&order, c.Param("id")).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		c.JSON(http.StatusOK, order)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081" // order service port
	}
	log.Printf("Order Service running on port %s", port)
	r.Run(":" + port)
}
