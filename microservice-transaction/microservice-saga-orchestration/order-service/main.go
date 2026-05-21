package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type OrderStatus string

const (
	StatusPending   OrderStatus = "PENDING"
	StatusConfirmed OrderStatus = "CONFIRMED"
	StatusCancelled OrderStatus = "CANCELLED"
)

type OrderOrchestration struct {
	ID        uint        `gorm:"primaryKey" json:"id"`
	AccountID string      `json:"account_id"`
	Amount    int64       `json:"amount"`
	Status    OrderStatus `json:"status"`
}

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=demo password=demo dbname=demo port=5432 sslmode=disable TimeZone=UTC"
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	db.AutoMigrate(&OrderOrchestration{})

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

		order := OrderOrchestration{
			AccountID: req.AccountID,
			Amount:    req.Amount,
			Status:    StatusPending,
		}

		if err := db.Create(&order).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create order"})
			return
		}

		c.JSON(http.StatusOK, order)
	})

	r.PUT("/orders/:id/confirm", func(c *gin.Context) {
		id := c.Param("id")
		if err := db.Model(&OrderOrchestration{}).Where("id = ?", id).Update("status", StatusConfirmed).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to confirm order"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "order confirmed"})
	})

	r.PUT("/orders/:id/cancel", func(c *gin.Context) {
		id := c.Param("id")
		if err := db.Model(&OrderOrchestration{}).Where("id = ?", id).Update("status", StatusCancelled).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel order"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "order cancelled"})
	})

	r.GET("/orders", func(c *gin.Context) {
		var orders []OrderOrchestration
		db.Find(&orders)
		c.JSON(http.StatusOK, orders)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8086" // orchestration order service port
	}
	log.Printf("Orchestration Order Service running on port %s", port)
	r.Run(":" + port)
}
