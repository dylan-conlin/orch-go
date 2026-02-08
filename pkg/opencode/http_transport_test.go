package opencode

import (
	"net/http"
	"testing"
	"time"
)

func TestNewClientUsesSharedBoundedTransport(t *testing.T) {
	clientA := NewClient("http://127.0.0.1:4096")
	clientB := NewClientWithTimeout("http://127.0.0.1:4096", 2*time.Second)

	transportA, ok := clientA.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("clientA transport type = %T, want *http.Transport", clientA.httpClient.Transport)
	}
	transportB, ok := clientB.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("clientB transport type = %T, want *http.Transport", clientB.httpClient.Transport)
	}

	if transportA != transportB {
		t.Fatal("expected OpenCode clients to reuse shared HTTP transport")
	}
	if transportA.MaxIdleConns != maxIdleConns {
		t.Fatalf("MaxIdleConns = %d, want %d", transportA.MaxIdleConns, maxIdleConns)
	}
	if transportA.MaxIdleConnsPerHost != maxIdlePerHost {
		t.Fatalf("MaxIdleConnsPerHost = %d, want %d", transportA.MaxIdleConnsPerHost, maxIdlePerHost)
	}
	if transportA.IdleConnTimeout != idleConnTimeout {
		t.Fatalf("IdleConnTimeout = %s, want %s", transportA.IdleConnTimeout, idleConnTimeout)
	}
}

func TestNewSSEClientUsesSharedBoundedTransport(t *testing.T) {
	client := NewSSEClient("http://127.0.0.1:4096/event")

	transport, ok := client.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("SSE transport type = %T, want *http.Transport", client.httpClient.Transport)
	}

	if transport != sharedHTTPTransport {
		t.Fatal("expected SSE client to reuse shared OpenCode transport")
	}
	if transport.MaxIdleConns != maxIdleConns {
		t.Fatalf("MaxIdleConns = %d, want %d", transport.MaxIdleConns, maxIdleConns)
	}
	if transport.MaxIdleConnsPerHost != maxIdlePerHost {
		t.Fatalf("MaxIdleConnsPerHost = %d, want %d", transport.MaxIdleConnsPerHost, maxIdlePerHost)
	}
	if transport.IdleConnTimeout != idleConnTimeout {
		t.Fatalf("IdleConnTimeout = %s, want %s", transport.IdleConnTimeout, idleConnTimeout)
	}
}
