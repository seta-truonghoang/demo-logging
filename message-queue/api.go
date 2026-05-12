package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/segmentio/kafka-go"
)

//go:embed web/index.html
var indexHTML []byte

type ProduceRequest struct {
	Key       string `json:"key"`
	Value     string `json:"value"`
	Strategy  string `json:"strategy"`
	Partition int    `json:"partition"`
}

func (a *App) setupRoutes() {
	http.HandleFunc("/", a.handleDashboard)
	http.HandleFunc("/api/state", a.handleState)
	http.HandleFunc("/api/produce", a.handleProduce)
	http.HandleFunc("/api/worker/toggle", a.handleWorkerToggle)
	http.HandleFunc("/api/delay", a.handleDelay)
	http.HandleFunc("/api/simulate/direct", a.handleSimulateDirect)
	http.HandleFunc("/api/simulate/kafka", a.handleSimulateKafka)
	http.HandleFunc("/api/logs", a.handleLogsStream)
}

func (a *App) handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.Write(indexHTML)
}

func (a *App) handleState(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	partitions := GetTopicPartitions(a.BrokerAddr, a.Topic)
	workers := a.WorkerManager.GetWorkersState()
	assignments := a.WorkerManager.GetActiveAssignments()

	state := map[string]interface{}{
		"broker":          a.BrokerAddr,
		"topic":           a.Topic,
		"group_id":        a.GroupID,
		"total_produced":  atomic.LoadInt64(&a.TotalProducedCount),
		"total_consumed":  atomic.LoadInt64(&a.TotalConsumedCount),
		"rebalance_count": atomic.LoadInt64(&a.RebalanceCount),
		"worker_delay":    atomic.LoadInt32(&a.WorkerManager.delayMs),
		"partitions":      partitions,
		"workers":         workers,
		"assignments":     assignments,
	}

	json.NewEncoder(w).Encode(state)
}

func (a *App) handleProduce(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ProduceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var balancer kafka.Balancer
	if req.Strategy == "key" {
		balancer = &kafka.Hash{}
	} else {
		balancer = &kafka.RoundRobin{}
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(a.BrokerAddr),
		Topic:        a.Topic,
		Balancer:     balancer,
		RequiredAcks: kafka.RequireOne,
	}
	defer writer.Close()

	msg := kafka.Message{
		Value: []byte(req.Value),
	}

	if req.Strategy == "key" && req.Key != "" {
		msg.Key = []byte(req.Key)
		a.LogHub.Log("producer", "INFO", "Dispatching to Topic '%s' | Key: '%s'", a.Topic, req.Key)
	} else if req.Strategy == "partition" {
		msg.Partition = req.Partition
		if req.Key != "" {
			msg.Key = []byte(req.Key)
		}
		a.LogHub.Log("producer", "INFO", "Dispatching direct to Partition #%d of Topic '%s'", req.Partition, a.Topic)
	} else {
		a.LogHub.Log("producer", "INFO", "Dispatching to Topic '%s' (Round Robin)", a.Topic)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := writer.WriteMessages(ctx, msg)
	if err != nil {
		a.LogHub.Log("producer", "ERROR", "Write message failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	atomic.AddInt64(&a.TotalProducedCount, 1)
	a.LogHub.Log("producer", "INFO", "Processed event receipt acknowledged by Kafka.")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success"})
}

func (a *App) handleWorkerToggle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	workerID := r.URL.Query().Get("id")
	action := r.URL.Query().Get("action")

	if workerID == "" || (action != "start" && action != "stop") {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if action == "start" {
		a.WorkerManager.StartWorker(workerID)
	} else {
		a.WorkerManager.StopWorker(workerID)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success"})
}

func (a *App) handleDelay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	delayStr := r.URL.Query().Get("delay")
	delayVal, err := strconv.Atoi(delayStr)
	if err != nil {
		http.Error(w, "Invalid delay value", http.StatusBadRequest)
		return
	}

	atomic.StoreInt32(&a.WorkerManager.delayMs, int32(delayVal))
	a.LogHub.Log("system", "INFO", "Processing delay adjusted to %dms", delayVal)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success"})
}

func (a *App) handleSimulateDirect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	countStr := r.URL.Query().Get("count")
	count, _ := strconv.Atoi(countStr)
	if count <= 0 {
		count = 1000
	}

	a.LogHub.Log("system", "WARN", "🚨 Direct Sync-API Simulation: %d writes...", count)

	go func() {
		maxConnections := 50
		var activeConnections int32
		var successCount int64
		var failCount int64
		var wg sync.WaitGroup

		startTime := time.Now()

		for i := 1; i <= count; i++ {
			wg.Add(1)
			go func(reqID int) {
				defer wg.Done()

				conn := atomic.AddInt32(&activeConnections, 1)
				defer atomic.AddInt32(&activeConnections, -1)

				if conn > int32(maxConnections) {
					atomic.AddInt64(&failCount, 1)
					if reqID%200 == 0 || reqID == count {
						a.LogHub.Log("system", "ERROR", "[Direct API] ❌ 500 ERROR: Pool Exhausted! Current Active: %d/%d. Req #%d rejected.", conn, maxConnections, reqID)
					}
					return
				}

				time.Sleep(80 * time.Millisecond)
				atomic.AddInt64(&successCount, 1)
			}(i)

			if i%100 == 0 {
				time.Sleep(5 * time.Millisecond)
			}
		}

		wg.Wait()
		duration := time.Since(startTime)
		a.LogHub.Log("system", "INFO", "📊 Direct Sync-API results: Handled %d calls in %v. Success: %d | Failures: %d (%.1f%%).", count, duration, successCount, failCount, float64(failCount)/float64(count)*100.0)
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success"})
}

func (a *App) handleSimulateKafka(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	countStr := r.URL.Query().Get("count")
	count, _ := strconv.Atoi(countStr)
	if count <= 0 {
		count = 10000
	}

	a.LogHub.Log("producer", "WARN", "🔥 Kafka Ingestion: Dispatching %d requests in batches...", count)

	go func() {
		bulkWriter := &kafka.Writer{
			Addr:         kafka.TCP(a.BrokerAddr),
			Topic:        a.Topic,
			Balancer:     &kafka.Hash{}, // Use Hash balancer so the same key ALWAYS goes to the same worker
			RequiredAcks: kafka.RequireNone,
			Async:        true,
			BatchSize:    1000,
			BatchTimeout: 10 * time.Millisecond,
		}
		defer bulkWriter.Close()

		startTime := time.Now()

		batchSize := 1000
		batch := make([]kafka.Message, 0, batchSize)

		for i := 1; i <= count; i++ {
			msgVal := fmt.Sprintf(`{"test_id":%d,"timestamp":%d,"action":"bulk_write"}`, i, time.Now().UnixNano())
			batch = append(batch, kafka.Message{
				Key:   []byte(fmt.Sprintf("bulk-key-%d", i%15)),
				Value: []byte(msgVal),
			})

			if len(batch) >= batchSize {
				err := bulkWriter.WriteMessages(context.Background(), batch...)
				if err != nil {
					a.LogHub.Log("producer", "ERROR", "Bulk batch error: %v", err)
					return
				}
				atomic.AddInt64(&a.TotalProducedCount, int64(len(batch)))
				batch = batch[:0]
				time.Sleep(1 * time.Millisecond)
			}
		}

		if len(batch) > 0 {
			bulkWriter.WriteMessages(context.Background(), batch...)
			atomic.AddInt64(&a.TotalProducedCount, int64(len(batch)))
		}

		duration := time.Since(startTime)
		a.LogHub.Log("producer", "INFO", "⚡ Kafka Ingestion Completed! Absorbed %d requests in %v.", count, duration)
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success"})
}

func (a *App) handleLogsStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	clientChan := make(chan LogEvent, 50)
	a.LogHub.Register(clientChan)
	defer a.LogHub.Unregister(clientChan)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case event := <-clientChan:
			data, err := json.Marshal(event)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "event: log\ndata: %s\n\n", string(data))
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}
