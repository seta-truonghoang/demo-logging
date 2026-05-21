package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
)

func main() {
	orderServiceURL := os.Getenv("ORDER_SERVICE_URL")
	if orderServiceURL == "" {
		orderServiceURL = "http://localhost:8083"
	}

	paymentServiceURL := os.Getenv("PAYMENT_SERVICE_URL")
	if paymentServiceURL == "" {
		paymentServiceURL = "http://localhost:8084"
	}

	client := resty.New()
	r := gin.Default()

	r.POST("/checkout", func(c *gin.Context) {
		var req struct {
			AccountID string `json:"account_id" binding:"required"`
			Amount    int64  `json:"amount" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		txID := uuid.New().String()
		log.Printf("Starting 2PC Checkout - TxID: %s", txID)

		var orderID uint
		orderPrepared := false
		paymentPrepared := false

		// Phase 1: PREPARE
		// 1. Prepare Order
		var orderResp struct {
			OrderID uint `json:"order_id"`
		}
		resp, err := client.R().
			SetBody(map[string]interface{}{
				"account_id": req.AccountID,
				"amount":     req.Amount,
			}).
			SetResult(&orderResp).
			Post(orderServiceURL + "/prepare")

		if err == nil && resp.IsSuccess() {
			orderPrepared = true
			orderID = orderResp.OrderID
			log.Printf("[TxID: %s] Order prepared successfully. OrderID: %d", txID, orderID)
		} else {
			log.Printf("[TxID: %s] Order prepare failed: %v", txID, err)
		}

		// 2. Prepare Payment
		if orderPrepared {
			resp, err = client.R().
				SetBody(map[string]interface{}{
					"account_id": req.AccountID,
					"amount":     req.Amount,
					"tx_id":      txID,
				}).
				Post(paymentServiceURL + "/prepare")

			if err == nil && resp.IsSuccess() {
				paymentPrepared = true
				log.Printf("[TxID: %s] Payment prepared successfully.", txID)
			} else {
				log.Printf("[TxID: %s] Payment prepare failed: %v, body: %s", txID, err, string(resp.Body()))
			}
		}

		// Phase 2: COMMIT or ROLLBACK
		if orderPrepared && paymentPrepared {
			// COMMIT
			log.Printf("[TxID: %s] Both prepared successfully. Executing COMMIT.", txID)
			
			_, err1 := client.R().SetBody(map[string]interface{}{"order_id": orderID}).Post(orderServiceURL + "/commit")
			_, err2 := client.R().SetBody(map[string]interface{}{"account_id": req.AccountID, "amount": req.Amount}).Post(paymentServiceURL + "/commit")
			
			if err1 != nil || err2 != nil {
				// In a real 2PC, handling commit failures requires manual intervention or retry mechanisms
				log.Printf("[TxID: %s] CRITICAL: Commit failed! OrderErr: %v, PaymentErr: %v", txID, err1, err2)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "critical failure during commit"})
				return
			}

			c.JSON(http.StatusOK, gin.H{"message": "checkout successful", "order_id": orderID})
			return
		}

		// ROLLBACK
		log.Printf("[TxID: %s] Prepare phase failed. Executing ROLLBACK.", txID)
		if orderPrepared {
			_, _ = client.R().SetBody(map[string]interface{}{"order_id": orderID}).Post(orderServiceURL + "/rollback")
		}
		if paymentPrepared {
			_, _ = client.R().SetBody(map[string]interface{}{"account_id": req.AccountID, "amount": req.Amount}).Post(paymentServiceURL + "/rollback")
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "checkout failed and rolled back"})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8085" // coordinator port
	}
	log.Printf("2PC Coordinator running on port %s", port)
	r.Run(":" + port)
}
