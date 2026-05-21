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

type Account2PC struct {
	ID             uint   `gorm:"primaryKey" json:"id"`
	Name           string `gorm:"uniqueIndex" json:"name"`
	Balance        int64  `json:"balance"`
	LockedBalance  int64  `json:"locked_balance"` // Used for 2PC prepare phase
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
	db.AutoMigrate(&Account2PC{})

	// Seed data
	db.FirstOrCreate(&Account2PC{Name: "alice", Balance: 1000, LockedBalance: 0}, Account2PC{Name: "alice"})
	db.FirstOrCreate(&Account2PC{Name: "bob", Balance: 500, LockedBalance: 0}, Account2PC{Name: "bob"})

	r := gin.Default()

	// Phase 1: Prepare
	r.POST("/prepare", func(c *gin.Context) {
		var req struct {
			AccountID string `json:"account_id" binding:"required"`
			Amount    int64  `json:"amount" binding:"required"`
			TxID      string `json:"tx_id" binding:"required"` // Optional for tracing
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err = db.Transaction(func(tx *gorm.DB) error {
			var acc Account2PC
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("name = ?", req.AccountID).First(&acc).Error; err != nil {
				return fmt.Errorf("account not found")
			}

			available := acc.Balance - acc.LockedBalance
			if available < req.Amount {
				return fmt.Errorf("insufficient available balance")
			}

			// Lock the balance
			acc.LockedBalance += req.Amount
			return tx.Save(&acc).Error
		})

		if err != nil {
			c.JSON(http.StatusPaymentRequired, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "payment prepared"})
	})

	// Phase 2: Commit
	r.POST("/commit", func(c *gin.Context) {
		var req struct {
			AccountID string `json:"account_id" binding:"required"`
			Amount    int64  `json:"amount" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err = db.Transaction(func(tx *gorm.DB) error {
			var acc Account2PC
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("name = ?", req.AccountID).First(&acc).Error; err != nil {
				return err
			}

			acc.Balance -= req.Amount
			acc.LockedBalance -= req.Amount
			return tx.Save(&acc).Error
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "commit failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "payment committed"})
	})

	// Phase 2: Rollback
	r.POST("/rollback", func(c *gin.Context) {
		var req struct {
			AccountID string `json:"account_id" binding:"required"`
			Amount    int64  `json:"amount" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err = db.Transaction(func(tx *gorm.DB) error {
			var acc Account2PC
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("name = ?", req.AccountID).First(&acc).Error; err != nil {
				return err
			}

			// Unlock the balance
			acc.LockedBalance -= req.Amount
			return tx.Save(&acc).Error
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "rollback failed"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "payment rolled back"})
	})

	r.GET("/accounts", func(c *gin.Context) {
		var accounts []Account2PC
		db.Find(&accounts)
		c.JSON(http.StatusOK, accounts)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8084" // 2pc payment service port
	}
	log.Printf("2PC Payment Service running on port %s", port)
	r.Run(":" + port)
}
