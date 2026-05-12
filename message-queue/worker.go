package main

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/segmentio/kafka-go"
)

// AssignmentInfo structures map partitions to active workers
type AssignmentInfo struct {
	WorkerID  string    `json:"worker_id"`
	Timestamp time.Time `json:"timestamp"`
}

// Worker handles reading from Kafka Topic inside a Consumer Group
type Worker struct {
	ID             string             `json:"id"`
	Topic          string             `json:"topic"`
	GroupID        string             `json:"group_id"`
	IsRunning      bool               `json:"is_running"`
	ProcessedCount int64              `json:"processed_count"`
	CurrentMsg     string             `json:"current_msg"`
	cancel         context.CancelFunc
	mu             sync.Mutex
}

// WorkerManager oversees creation and execution state of Consumer Workers
type WorkerManager struct {
	app           *App // Reference to main App to update global counters
	workers       map[string]*Worker
	mu            sync.RWMutex
	broker        string
	topic         string
	groupID       string
	logHub        *LogHub
	delayMs       int32 // atomic processing lag delay
	assignments   map[int]AssignmentInfo
	assignmentsMu sync.RWMutex
	appStats      func(consumed int64, rebalance int64)
}

func NewWorkerManager(broker, topic, groupID string, logHub *LogHub) *WorkerManager {
	return &WorkerManager{
		workers:     make(map[string]*Worker),
		assignments: make(map[int]AssignmentInfo),
		broker:      broker,
		topic:       topic,
		groupID:     groupID,
		logHub:      logHub,
		delayMs:     500, // 500ms baseline processing latency
	}
}

func (wm *WorkerManager) updatePartitionAssignment(partition int, workerID string) {
	wm.assignmentsMu.Lock()
	wm.assignments[partition] = AssignmentInfo{
		WorkerID:  workerID,
		Timestamp: time.Now(),
	}
	wm.assignmentsMu.Unlock()
}

func (wm *WorkerManager) GetActiveAssignments() map[string]string {
	wm.assignmentsMu.RLock()
	defer wm.assignmentsMu.RUnlock()

	result := make(map[string]string)
	now := time.Now()
	for p, info := range wm.assignments {
		if now.Sub(info.Timestamp) < 15*time.Second {
			result[strconv.Itoa(p)] = info.WorkerID
		}
	}
	return result
}

// StartWorker spins up a consumer thread inside the shared Kafka Consumer Group
func (wm *WorkerManager) StartWorker(id string) {
	wm.mu.Lock()
	w, exists := wm.workers[id]
	if !exists {
		w = &Worker{ID: id, Topic: wm.topic, GroupID: wm.groupID}
		wm.workers[id] = w
	}
	wm.mu.Unlock()

	w.mu.Lock()
	if w.IsRunning {
		w.mu.Unlock()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	w.cancel = cancel
	w.IsRunning = true
	w.mu.Unlock()

	wm.logHub.Log("system", "INFO", "Spinning up Consumer Worker %s in group: %s", id, wm.groupID)

	go func(w *Worker, ctx context.Context) {
		reader := kafka.NewReader(kafka.ReaderConfig{
			Brokers:        []string{wm.broker},
			GroupID:        wm.groupID,
			Topic:          wm.topic,
			MinBytes:       10,
			MaxBytes:       10e6,
			CommitInterval: 1 * time.Second,
			StartOffset:    kafka.LastOffset,
		})

		defer func() {
			reader.Close()
			w.mu.Lock()
			w.IsRunning = false
			w.CurrentMsg = ""
			w.mu.Unlock()
			wm.logHub.Log("system", "WARN", "Worker %s has cleanly closed connection and left the consumer group.", w.ID)
			
			if wm.app != nil {
				atomic.AddInt64(&wm.app.RebalanceCount, 1)
			}
		}()

		wm.logHub.Log(w.ID, "INFO", "Worker %s connected. Listening on partitions...", w.ID)

		for {
			m, err := reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return // Expected return on context cancellation
				}
				wm.logHub.Log(w.ID, "ERROR", "Failed to retrieve message: %v", err)
				time.Sleep(2 * time.Second)
				continue
			}

			keyStr := string(m.Key)
			if keyStr == "" {
				keyStr = "<empty>"
			}

			w.mu.Lock()
			w.ProcessedCount++
			w.CurrentMsg = fmt.Sprintf("Offset: %d | Key: %s", m.Offset, keyStr)
			w.mu.Unlock()

			wm.logHub.Log(w.ID, "INFO", "📥 Received from Partition #%d | Key: %s | Offset: %d", m.Partition, keyStr, m.Offset)

			// Apply adjustable network processing simulation latency
			delay := atomic.LoadInt32(&wm.delayMs)
			if delay > 0 {
				time.Sleep(time.Duration(delay) * time.Millisecond)
			}

			wm.logHub.Log(w.ID, "INFO", "✅ Successfully processed Key: %s in %dms", keyStr, delay)
			
			if wm.app != nil {
				atomic.AddInt64(&wm.app.TotalConsumedCount, 1)
			}

			w.mu.Lock()
			w.CurrentMsg = ""
			w.mu.Unlock()

			wm.updatePartitionAssignment(int(m.Partition), w.ID)
		}
	}(w, ctx)
}

func (wm *WorkerManager) StopWorker(id string) {
	wm.mu.Lock()
	w, exists := wm.workers[id]
	wm.mu.Unlock()

	if !exists {
		return
	}

	w.mu.Lock()
	if !w.IsRunning {
		w.mu.Unlock()
		return
	}
	wm.logHub.Log("system", "INFO", "Stopping Consumer Worker %s, sending leave-group signal...", id)
	w.cancel()
	w.mu.Unlock()

	time.Sleep(150 * time.Millisecond)
}

func (wm *WorkerManager) StopAll() {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	for _, w := range wm.workers {
		wm.StopWorker(w.ID)
	}
}

func (wm *WorkerManager) GetWorkersState() []Worker {
	wm.mu.RLock()
	defer wm.mu.RUnlock()

	states := make([]Worker, 0, len(wm.workers))
	for _, w := range wm.workers {
		w.mu.Lock()
		states = append(states, Worker{
			ID:             w.ID,
			Topic:          w.Topic,
			GroupID:        w.GroupID,
			IsRunning:      w.IsRunning,
			ProcessedCount: w.ProcessedCount,
			CurrentMsg:     w.CurrentMsg,
		})
		w.mu.Unlock()
	}
	return states
}
