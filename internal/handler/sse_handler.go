package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/claude-code/observability/internal/service"
)

// SSEHandler streams realtime trace events to browser clients.
type SSEHandler struct {
	bus *service.EventBus
}

// NewSSEHandler creates an instance of SSEHandler.
func NewSSEHandler(bus *service.EventBus) *SSEHandler {
	return &SSEHandler{bus: bus}
}

// ServeHTTP streams Server-Sent Events with keepalive pings.
func (h *SSEHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	events := h.bus.Subscribe()
	defer h.bus.Unsubscribe(events)

	// Send initial connection event
	fmt.Fprintf(w, "event: connected\ndata: {}\n\n")
	flusher.Flush()

	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
		case ev, open := <-events:
			if !open {
				return
			}
			bytes, err := json.Marshal(ev)
			if err == nil {
				fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.Type, string(bytes))
				flusher.Flush()
			}
		}
	}
}

