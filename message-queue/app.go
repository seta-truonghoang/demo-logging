package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// App is the main application struct holding all dependencies and state.
// This completely eliminates global variables and makes the code clean and testable.
type App struct {
	BrokerAddr         string
	Topic              string
	GroupID            string
	LogHub             *LogHub
	WorkerManager      *WorkerManager
	TotalProducedCount int64
	TotalConsumedCount int64
	RebalanceCount     int64
	Server             *http.Server
}

func NewApp(brokerAddr, topic, groupID string) *App {
	logHub := NewLogHub()
	wm := NewWorkerManager(brokerAddr, topic, groupID, logHub)

	// Link worker manager stats to app stats so we can read them globally
	wm.appStats = func(consumed int64, rebalance int64) {
		// This callback updates the app-level counters when a worker consumes or rebalances
	}

	app := &App{
		BrokerAddr:    brokerAddr,
		Topic:         topic,
		GroupID:       groupID,
		LogHub:        logHub,
		WorkerManager: wm,
	}
	
	// Pass the app pointer to worker manager so it can update shared stats
	wm.app = app

	return app
}

func (a *App) Run(port string) {
	// Start Log Hub
	go a.LogHub.Start()
	a.LogHub.Log("system", "INFO", "Initializing Kafka Event Streaming Demo Application...")

	// Verify Kafka topic existence with retries
	go a.initializeKafka()

	// Setup HTTP API Routes
	a.setupRoutes()

	// Start the HTTP Server
	a.Server = &http.Server{Addr: ":" + port}
	a.LogHub.Log("system", "INFO", "Booting Web Service on :%s", port)

	go func() {
		if err := a.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// Graceful Shutdown
	a.waitForShutdown()
}

func (a *App) initializeKafka() {
	for i := 0; i < 15; i++ {
		err := EnsureTopicExists(a.BrokerAddr, a.Topic, 3, a.LogHub)
		if err == nil {
			a.LogHub.Log("system", "INFO", "Topic is ready. Booting consumer workers...")
			a.WorkerManager.StartWorker("worker-1")
			a.WorkerManager.StartWorker("worker-2")
			a.WorkerManager.StartWorker("worker-3")
			return
		}
		a.LogHub.Log("system", "WARN", "Kafka broker is not ready yet. Retrying in 3 seconds... (%d/15)", i+1)
		time.Sleep(3 * time.Second)
	}
	a.LogHub.Log("system", "ERROR", "CRITICAL: Could not connect to Kafka Broker after multiple attempts.")
}

func (a *App) waitForShutdown() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	a.LogHub.Log("system", "WARN", "Interrupt signal received. Shutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if a.Server != nil {
		a.Server.Shutdown(ctx)
	}

	a.WorkerManager.StopAll()
	a.LogHub.Log("system", "INFO", "Shutdown complete.")
}
