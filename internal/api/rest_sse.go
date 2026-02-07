package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/stashapp/stash/internal/log"
	"github.com/stashapp/stash/internal/manager"
	"github.com/stashapp/stash/pkg/job"
	"github.com/stashapp/stash/pkg/logger"
)

// SSEEvent represents a Server-Sent Event.
type SSEEvent struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// SSEBroker manages Server-Sent Event connections.
// It maintains a list of connected clients and broadcasts events to all of them.
type SSEBroker struct {
	mu         sync.RWMutex
	clients    map[chan SSEEvent]struct{}
	bufferSize int

	cancel context.CancelFunc
}

// NewSSEBroker creates a new SSE broker.
func NewSSEBroker() *SSEBroker {
	return &SSEBroker{
		clients:    make(map[chan SSEEvent]struct{}),
		bufferSize: 64,
	}
}

// Start begins listening for system events (jobs, logs, scan complete) and
// broadcasts them to all connected SSE clients. Call this after the manager
// is initialized.
func (b *SSEBroker) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	b.cancel = cancel

	mgr := manager.GetInstance()

	// Subscribe to job manager events
	jobSub := mgr.JobManager.Subscribe(ctx)
	go b.forwardJobEvents(ctx, jobSub)

	// Subscribe to log events
	stopLog := make(chan int, 1)
	logSub := mgr.Logger.SubscribeToLog(stopLog)
	go b.forwardLogEvents(ctx, logSub, stopLog)

	// Subscribe to scan complete events
	scanSub := mgr.ScanSubscribe(ctx)
	go b.forwardScanComplete(ctx, scanSub)
}

// Stop cancels all background goroutines.
func (b *SSEBroker) Stop() {
	if b.cancel != nil {
		b.cancel()
	}
}

// forwardJobEvents forwards job manager events to SSE clients.
func (b *SSEBroker) forwardJobEvents(ctx context.Context, sub *job.ManagerSubscription) {
	for {
		select {
		case j := <-sub.NewJob:
			b.Broadcast(SSEEvent{
				Type: "job.add",
				Data: &JobStatusUpdate{
					Type: JobStatusUpdateTypeAdd,
					Job:  jobToJobModel(j),
				},
			})
		case j := <-sub.RemovedJob:
			b.Broadcast(SSEEvent{
				Type: "job.remove",
				Data: &JobStatusUpdate{
					Type: JobStatusUpdateTypeRemove,
					Job:  jobToJobModel(j),
				},
			})
		case j := <-sub.UpdatedJob:
			b.Broadcast(SSEEvent{
				Type: "job.update",
				Data: &JobStatusUpdate{
					Type: JobStatusUpdateTypeUpdate,
					Job:  jobToJobModel(j),
				},
			})
		case <-ctx.Done():
			return
		}
	}
}

// SSELogEntry is the JSON representation of a log entry sent via SSE.
type SSELogEntry struct {
	Time    time.Time `json:"time"`
	Level   string    `json:"level"`
	Message string    `json:"message"`
}

func logItemToSSEEntry(item log.LogItem) SSELogEntry {
	return SSELogEntry{
		Time:    item.Time,
		Level:   item.Type,
		Message: item.Message,
	}
}

// forwardLogEvents forwards log entries to SSE clients.
func (b *SSEBroker) forwardLogEvents(ctx context.Context, logSub <-chan []log.LogItem, stop chan int) {
	defer func() {
		stop <- 0
	}()

	for {
		select {
		case items := <-logSub:
			entries := make([]SSELogEntry, len(items))
			for i, item := range items {
				entries[i] = logItemToSSEEntry(item)
			}
			b.Broadcast(SSEEvent{Type: "log.entries", Data: entries})
		case <-ctx.Done():
			return
		}
	}
}

// forwardScanComplete forwards scan complete notifications to SSE clients.
func (b *SSEBroker) forwardScanComplete(ctx context.Context, scanSub <-chan bool) {
	for {
		select {
		case <-scanSub:
			b.Broadcast(SSEEvent{Type: "scan.complete", Data: nil})
		case <-ctx.Done():
			return
		}
	}
}

// Subscribe creates a new client channel and registers it with the broker.
func (b *SSEBroker) Subscribe() chan SSEEvent {
	ch := make(chan SSEEvent, b.bufferSize)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

// Unsubscribe removes a client channel from the broker and closes it.
func (b *SSEBroker) Unsubscribe(ch chan SSEEvent) {
	b.mu.Lock()
	delete(b.clients, ch)
	b.mu.Unlock()
	close(ch)
}

// Broadcast sends an event to all connected clients.
// Non-blocking: if a client channel is full, the event is dropped for that client.
func (b *SSEBroker) Broadcast(event SSEEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch := range b.clients {
		select {
		case ch <- event:
		default:
			// Client buffer is full, skip this event for this client
			logger.Warnf("SSE: dropping event %s for slow client", event.Type)
		}
	}
}

// ClientCount returns the number of connected clients.
func (b *SSEBroker) ClientCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.clients)
}

// SSEHandler is the HTTP handler for the SSE endpoint.
// It streams events to connected clients using the text/event-stream format.
func (h *RESTHandler) SSEHandler(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // for nginx proxy

	events := h.sseBroker.Subscribe()
	defer h.sseBroker.Unsubscribe(events)

	// Send initial connection event
	fmt.Fprintf(w, "event: connected\ndata: {}\n\n")
	flusher.Flush()

	for {
		select {
		case event := <-events:
			data, err := json.Marshal(event.Data)
			if err != nil {
				logger.Errorf("SSE: error marshaling event data: %v", err)
				continue
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, string(data))
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}
