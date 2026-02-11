package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckDockerAvailable(t *testing.T) {
	tests := []struct {
		name        string
		setupSocket func() string
		wantErr     bool
		errContains string
	}{
		{
			name: "docker daemon running",
			setupSocket: func() string {
				// Create a temporary socket file
				tmpDir := t.TempDir()
				socketPath := filepath.Join(tmpDir, "docker.sock")
				f, err := os.Create(socketPath)
				if err != nil {
					t.Fatal(err)
				}
				f.Close()
				return socketPath
			},
			wantErr: false,
		},
		{
			name: "docker socket missing",
			setupSocket: func() string {
				return "/nonexistent/docker.sock"
			},
			wantErr:     true,
			errContains: "Docker daemon not running",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			socketPath := tt.setupSocket()

			// Override the docker socket path for testing
			originalDockerSocket := dockerSocketPath
			dockerSocketPath = socketPath
			defer func() { dockerSocketPath = originalDockerSocket }()

			err := checkDockerAvailable()

			if tt.wantErr {
				if err == nil {
					t.Errorf("checkDockerAvailable() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("checkDockerAvailable() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("checkDockerAvailable() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCheckClaudeAvailable(t *testing.T) {
	tests := []struct {
		name        string
		setupEnv    func()
		wantErr     bool
		errContains string
	}{
		{
			name: "claude CLI found in PATH",
			setupEnv: func() {
				// Assume claude exists in PATH (or skip test if not)
				_, err := exec.LookPath("claude")
				if err != nil {
					t.Skip("claude CLI not found in PATH, skipping test")
				}
			},
			wantErr: false,
		},
		{
			name: "claude CLI not found",
			setupEnv: func() {
				// Override PATH to empty
				t.Setenv("PATH", "")
			},
			wantErr:     true,
			errContains: "claude CLI not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()

			err := checkClaudeAvailable()

			if tt.wantErr {
				if err == nil {
					t.Errorf("checkClaudeAvailable() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("checkClaudeAvailable() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("checkClaudeAvailable() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCheckOpencodeAvailable(t *testing.T) {
	tests := []struct {
		name        string
		setupServer func() string
		wantErr     bool
		errContains string
	}{
		{
			name: "opencode API responding",
			setupServer: func() string {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte("OK"))
				}))
				t.Cleanup(server.Close)
				return server.URL
			},
			wantErr: false,
		},
		{
			name: "opencode API not responding",
			setupServer: func() string {
				return "http://localhost:99999" // Invalid port
			},
			wantErr:     true,
			errContains: "OpenCode API not responding",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverURL := tt.setupServer()

			err := checkOpencodeAvailable(serverURL)

			if tt.wantErr {
				if err == nil {
					t.Errorf("checkOpencodeAvailable() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("checkOpencodeAvailable() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("checkOpencodeAvailable() unexpected error: %v", err)
				}
			}
		})
	}
}
