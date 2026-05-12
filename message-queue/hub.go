package main

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// LogEvent represents a structured log statement broadcasted to SSE clients
type LogEvent struct {
	Time    string `json:"time"`
	Source  string `json:"source"` // "producer", "worker-1", "worker-2", "worker-3", "system"
	Level   string `json:"level"`  // "INFO", "WARN", "ERROR"
	Message string `json:"message"`
}

// LogHub broadcasts logs to all registered web UI clients
type LogHub struct {
	clients   map[chan LogEvent]bool
	mu        sync.Mutex
	broadcast chan LogEvent
}

func NewLogHub() *LogHub {
	return &LogHub{
		clients:   make(map[chan LogEvent]bool),
		broadcast: make(chan LogEvent, 200),
	}
}

// Start listens to the broadcast channel and pushes logs to connected web browsers
func (lh *LogHub) Start() {
	for event := range lh.broadcast {
		lh.mu.Lock()
		for clientChan := range lh.clients {
			select {
			case clientChan <- event:
			default:
				// Avoid blocking if client buffer is full
			}
		}
		lh.mu.Unlock()
	}
}

func (lh *LogHub) Register(ch chan LogEvent) {
	lh.mu.Lock()
	lh.clients[ch] = true
	lh.mu.Unlock()
}

func (lh *LogHub) Unregister(ch chan LogEvent) {
	lh.mu.Lock()
	delete(lh.clients, ch)
	lh.mu.Unlock()
}

// Log prints to console and queues the log for SSE streaming
func (lh *LogHub) Log(source, level, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	event := LogEvent{
		Time:    time.Now().Format("15:04:05.000"),
		Source:  source,
		Level:   level,
		Message: msg,
	}
	log.Printf("[%s] [%s] %s", source, level, msg)
	select {
	case lh.broadcast <- event:
	default:
		// Drop log if broadcast queue is fully saturated
	}
}
