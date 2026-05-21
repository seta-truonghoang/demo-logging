package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AccountOrchestration struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	Name    string `gorm:"uniqueIndex" json:"name"`
	Balance int64  `json:"balance"`
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
	db.AutoMigrate(&AccountOrchestration{})

	// Seed data
	db.FirstOrCreate(&AccountOrchestration{Name: "alice", Balance: 1000}, AccountOrchestration{Name: "alice"})
	db.FirstOrCreate(&AccountOrchestration{Name: "bob", Balance: 500}, AccountOrchestration{Name: "bob"})

	r := gin.Default()

	r.POST("/payments", func(c *gin.Context) {
		var req struct {
			AccountID string `json:"account_id" binding:"required"`
			Amount    int64  `json:"amount" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err = db.Transaction(func(tx *gorm.DB) error {
			var acc AccountOrchestration
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("name = ?", req.AccountID).First(&acc).Error; err != nil {
				return fmt.Errorf("account not found")
			}

			if acc.Balance < req.Amount {
				return fmt.Errorf("insufficient balance")
			}

			acc.Balance -= req.Amount
			return tx.Save(&acc).Error
		})

		if err != nil {
			c.JSON(http.StatusPaymentRequired, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "payment processed"})
	})

	r.GET("/accounts", func(c *gin.Context) {
		var accounts []AccountOrchestration
		db.Find(&accounts)
		c.JSON(http.StatusOK, accounts)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8087" // orchestration payment service port
	}
	log.Printf("Orchestration Payment Service running on port %s", port)
	r.Run(":" + port)
}
