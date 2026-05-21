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
	"gorm.io/gorm/clause"
)

// Models
type Account struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	Name      string `gorm:"uniqueIndex" json:"name"`
	Balance   int64  `json:"balance"`
	CreatedAt time.Time
	UpdatedAt time.Time
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
	db.AutoMigrate(&Account{})

	// Seed some data for demo
	db.FirstOrCreate(&Account{Name: "alice", Balance: 1000}, Account{Name: "alice"})
	db.FirstOrCreate(&Account{Name: "bob", Balance: 500}, Account{Name: "bob"})

	// 2. Setup Kafka
	kafkaBroker := os.Getenv("KAFKA_BROKER")
	if kafkaBroker == "" {
		kafkaBroker = "localhost:9092"
	}

	orderTopic := "order_events"
	paymentTopic := "payment_events"

	writer := &kafka.Writer{
		Addr:     kafka.TCP(kafkaBroker),
		Topic:    paymentTopic,
		Balancer: &kafka.LeastBytes{},
	}
	defer writer.Close()

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{kafkaBroker},
		Topic:   orderTopic,
		GroupID: "payment-service-group",
	})
	defer reader.Close()

	// 3. Start Kafka Consumer (Listen to Order Events)
	go func() {
		for {
			m, err := reader.ReadMessage(context.Background())
			if err != nil {
				log.Printf("error reading message: %v", err)
				continue
			}

			var event OrderCreatedEvent
			if err := json.Unmarshal(m.Value, &event); err != nil {
				log.Printf("error unmarshalling event: %v", err)
				continue
			}

			log.Printf("Received order event: %+v", event)

			// Process Payment
			paymentStatus := "SUCCESS"
			reason := ""

			err = db.Transaction(func(tx *gorm.DB) error {
				var account Account
				// Lock the row for update
				if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("name = ?", event.AccountID).First(&account).Error; err != nil {
					return fmt.Errorf("account not found: %s", event.AccountID)
				}

				if account.Balance < event.Amount {
					return fmt.Errorf("insufficient balance")
				}

				account.Balance -= event.Amount
				if err := tx.Save(&account).Error; err != nil {
					return err
				}

				return nil
			})

			if err != nil {
				log.Printf("Payment failed for order %d: %v", event.OrderID, err)
				paymentStatus = "FAILED"
				reason = err.Error()
			} else {
				log.Printf("Payment successful for order %d", event.OrderID)
			}

			// Publish Payment Event
			paymentEvent := PaymentEvent{
				OrderID: event.OrderID,
				Status:  paymentStatus,
				Reason:  reason,
			}
			paymentEventBytes, _ := json.Marshal(paymentEvent)

			err = writer.WriteMessages(context.Background(),
				kafka.Message{
					Key:   []byte(fmt.Sprintf("%d", event.OrderID)),
					Value: paymentEventBytes,
				},
			)
			if err != nil {
				log.Printf("failed to write payment event: %v", err)
			}
		}
	}()

	// 4. Setup HTTP Server
	r := gin.Default()
	r.GET("/accounts", func(c *gin.Context) {
		var accounts []Account
		db.Find(&accounts)
		c.JSON(http.StatusOK, accounts)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082" // payment service port
	}
	log.Printf("Payment Service running on port %s", port)
	r.Run(":" + port)
}
