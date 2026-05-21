package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
)

func main() {
	orderServiceURL := os.Getenv("ORDER_SERVICE_URL")
	if orderServiceURL == "" {
		orderServiceURL = "http://localhost:8086"
	}

	paymentServiceURL := os.Getenv("PAYMENT_SERVICE_URL")
	if paymentServiceURL == "" {
		paymentServiceURL = "http://localhost:8087"
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

		log.Printf("Orchestrating checkout for account %s, amount %d", req.AccountID, req.Amount)

		// Step 1: Create Order (Pending)
		var orderResp struct {
			ID uint `json:"id"`
		}
		resp1, err := client.R().
			SetBody(map[string]interface{}{
				"account_id": req.AccountID,
				"amount":     req.Amount,
			}).
			SetResult(&orderResp).
			Post(orderServiceURL + "/orders")

		if err != nil || !resp1.IsSuccess() {
			log.Printf("Failed to create order: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create order"})
			return
		}
		orderID := orderResp.ID
		log.Printf("Step 1: Order created with ID %d", orderID)

		// Step 2: Process Payment
		resp2, err := client.R().
			SetBody(map[string]interface{}{
				"account_id": req.AccountID,
				"amount":     req.Amount,
			}).
			Post(paymentServiceURL + "/payments")

		// Step 3: Handle Result (Confirm or Compensate)
		if err == nil && resp2.IsSuccess() {
			log.Printf("Step 2: Payment processed successfully")
			// Confirm order
			_, _ = client.R().Put(fmt.Sprintf("%s/orders/%d/confirm", orderServiceURL, orderID))
			log.Printf("Step 3: Order %d confirmed", orderID)
			c.JSON(http.StatusOK, gin.H{"message": "checkout successful", "order_id": orderID})
			return
		}

		// Payment failed, trigger compensating transaction (cancel order)
		log.Printf("Step 2: Payment failed, triggering compensation (Cancel Order)")
		_, _ = client.R().Put(fmt.Sprintf("%s/orders/%d/cancel", orderServiceURL, orderID))
		log.Printf("Step 3: Order %d cancelled", orderID)
		c.JSON(http.StatusPaymentRequired, gin.H{"error": "payment failed, order cancelled"})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8088" // orchestration service port
	}
	log.Printf("Orchestrator running on port %s", port)
	r.Run(":" + port)
}
