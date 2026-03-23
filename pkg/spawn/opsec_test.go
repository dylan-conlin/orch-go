package spawn

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCheckOpsecProxy_NotRequired(t *testing.T) {
	err := CheckOpsecProxy(false, 8199)
	if err != nil {
		t.Errorf("expected nil error when opsec not required, got: %v", err)
	}
}

func TestCheckOpsecProxy_ProxyRunning(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	var port int
	_, err := fmt.Sscanf(server.URL, "http://127.0.0.1:%d", &port)
	if err != nil {
		t.Fatalf("failed to parse test server port: %v", err)
	}

	err = CheckOpsecProxy(true, port)
	if err != nil {
		t.Errorf("expected nil error when proxy is running, got: %v", err)
	}
}

func TestCheckOpsecProxy_ProxyNotRunning(t *testing.T) {
	err := CheckOpsecProxy(true, 59999)
	if err == nil {
		t.Error("expected error when proxy not running, got nil")
	}
}

func TestOpsecEnvPrefix(t *testing.T) {
	prefix := OpsecEnvPrefix(true, 8199)
	if prefix == "" {
		t.Error("expected non-empty opsec prefix when enabled")
	}

	expected := []string{
		"OPSEC_SANDBOX=1",
		"HTTP_PROXY=http://127.0.0.1:8199",
		"HTTPS_PROXY=http://127.0.0.1:8199",
		"ALL_PROXY=http://127.0.0.1:8199",
	}
	for _, env := range expected {
		if !strings.Contains(prefix, env) {
			t.Errorf("opsec prefix missing %q, got: %s", env, prefix)
		}
	}
}

func TestOpsecEnvPrefix_Disabled(t *testing.T) {
	prefix := OpsecEnvPrefix(false, 8199)
	if prefix != "" {
		t.Errorf("expected empty prefix when disabled, got: %s", prefix)
	}
}
