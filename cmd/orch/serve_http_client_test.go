package main

import (
	"net/http"
	"testing"
	"time"
)

func TestNewServeTransportHasBoundedIdlePool(t *testing.T) {
	transport := newServeTransport(nil)

	if transport.MaxIdleConns != serveHTTPMaxIdleConns {
		t.Fatalf("MaxIdleConns = %d, want %d", transport.MaxIdleConns, serveHTTPMaxIdleConns)
	}
	if transport.MaxIdleConnsPerHost != serveHTTPMaxIdleConnsPerHost {
		t.Fatalf("MaxIdleConnsPerHost = %d, want %d", transport.MaxIdleConnsPerHost, serveHTTPMaxIdleConnsPerHost)
	}
	if transport.IdleConnTimeout != serveHTTPIdleConnTimeout {
		t.Fatalf("IdleConnTimeout = %s, want %s", transport.IdleConnTimeout, serveHTTPIdleConnTimeout)
	}
	if transport.TLSClientConfig != nil && transport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("expected default transport to keep TLS verification enabled")
	}
}

func TestNewServeTransportAppliesTLSConfig(t *testing.T) {
	tlsConfig := tlsConfigSkipVerify()
	transport := newServeTransport(tlsConfig)

	if transport.TLSClientConfig != tlsConfig {
		t.Fatal("expected transport to use provided TLS config")
	}
}

func TestServeHTTPClientsReuseSharedTransports(t *testing.T) {
	sseTransport, ok := serveOpenCodeSSEHTTPClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("SSE transport type = %T, want *http.Transport", serveOpenCodeSSEHTTPClient.Transport)
	}
	if sseTransport != serveOutboundTransport {
		t.Fatal("expected SSE client to reuse shared outbound transport")
	}

	loopbackTransport, ok := serveLoopbackDefaultHTTPClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("loopback transport type = %T, want *http.Transport", serveLoopbackDefaultHTTPClient.Transport)
	}
	if loopbackTransport != serveOutboundTLSTransport {
		t.Fatal("expected loopback client to reuse shared TLS transport")
	}
	if serveLoopbackDefaultHTTPClient.Timeout != 5*time.Second {
		t.Fatalf("loopback timeout = %s, want %s", serveLoopbackDefaultHTTPClient.Timeout, 5*time.Second)
	}
}
