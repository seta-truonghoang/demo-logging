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
	StatusPreparePending OrderStatus = "PREPARE_PENDING"
	StatusConfirmed      OrderStatus = "CONFIRMED"
	StatusCancelled      OrderStatus = "CANCELLED"
)

type Order2PC struct {
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
	db.AutoMigrate(&Order2PC{})

	r := gin.Default()

	// Phase 1: Prepare
	r.POST("/prepare", func(c *gin.Context) {
		var req struct {
			AccountID string `json:"account_id" binding:"required"`
			Amount    int64  `json:"amount" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		order := Order2PC{
			AccountID: req.AccountID,
			Amount:    req.Amount,
			Status:    StatusPreparePending,
		}

		if err := db.Create(&order).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to prepare order"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"order_id": order.ID})
	})

	// Phase 2: Commit
	r.POST("/commit", func(c *gin.Context) {
		var req struct {
			OrderID uint `json:"order_id" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := db.Model(&Order2PC{}).Where("id = ?", req.OrderID).Update("status", StatusConfirmed).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit order"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "order committed"})
	})

	// Phase 2: Rollback
	r.POST("/rollback", func(c *gin.Context) {
		var req struct {
			OrderID uint `json:"order_id" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := db.Model(&Order2PC{}).Where("id = ?", req.OrderID).Update("status", StatusCancelled).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to rollback order"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "order rolled back"})
	})

	r.GET("/orders", func(c *gin.Context) {
		var orders []Order2PC
		db.Find(&orders)
		c.JSON(http.StatusOK, orders)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8083" // 2pc order service port
	}
	log.Printf("2PC Order Service running on port %s", port)
	r.Run(":" + port)
}
