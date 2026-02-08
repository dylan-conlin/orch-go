package daemon

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/internal/testutil"
)

func TestCompletionService_StartStop_Lifecycle_NoLeakedConnections(t *testing.T) {
	var activeConnections atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/event" {
			http.NotFound(w, r)
			return
		}

		activeConnections.Add(1)
		defer activeConnections.Add(-1)

		w.Header().Set("Content-Type", "text/event-stream")
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}

		<-r.Context().Done()
	}))
	defer server.Close()

	const instances = 5
	services := make([]*CompletionService, 0, instances)
	startGoroutines := runtime.NumGoroutine()

	for i := 0; i < instances; i++ {
		cfg := DefaultCompletionServiceConfig()
		cfg.ServerURL = server.URL

		cs := NewCompletionService(cfg)
		cs.Start()
		services = append(services, cs)
	}

	testutil.WaitForWithTimeout(t, func() bool {
		return int(activeConnections.Load()) == instances
	}, 2*time.Second, "all completion service SSE connections to establish")

	for _, cs := range services {
		cs.Stop()
	}

	testutil.WaitForWithTimeout(t, func() bool {
		return activeConnections.Load() == 0
	}, 2*time.Second, "all completion service SSE connections to close")

	testutil.WaitForWithTimeout(t, func() bool {
		return runtime.NumGoroutine() <= startGoroutines+4
	}, 2*time.Second, "completion service goroutines to return to baseline")
}
