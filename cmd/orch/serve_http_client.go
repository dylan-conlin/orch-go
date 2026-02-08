package main

import (
	"crypto/tls"
	"net/http"
	"time"
)

const (
	serveHTTPMaxIdleConns        = 64
	serveHTTPMaxIdleConnsPerHost = 16
	serveHTTPIdleConnTimeout     = 30 * time.Second
)

var (
	serveOutboundTransport         = newServeTransport(nil)
	serveOutboundInsecureTLS       = tlsConfigSkipVerify()
	serveOutboundTLSTransport      = newServeTransport(serveOutboundInsecureTLS)
	serveOpenCodeSSEHTTPClient     = &http.Client{Transport: serveOutboundTransport}
	serveLoopbackDefaultHTTPClient = &http.Client{
		Timeout:   5 * time.Second,
		Transport: serveOutboundTLSTransport,
	}
)

func newServeTransport(tlsConfig *tls.Config) *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = serveHTTPMaxIdleConns
	transport.MaxIdleConnsPerHost = serveHTTPMaxIdleConnsPerHost
	transport.IdleConnTimeout = serveHTTPIdleConnTimeout
	if tlsConfig != nil {
		transport.TLSClientConfig = tlsConfig
	}
	return transport
}
