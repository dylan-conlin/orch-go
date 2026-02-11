package tmux

import (
	"strings"
	"testing"
	"time"
)

func TestProbeWindowForErrors(t *testing.T) {
	tests := []struct {
		name        string
		paneContent string
		wantErr     bool
		errContains string
	}{
		{
			name:        "no errors - clean output",
			paneContent: "Loading...\nAgent started successfully\nReady for input",
			wantErr:     false,
		},
		{
			name:        "docker daemon not running",
			paneContent: "Cannot connect to the Docker daemon at unix:///var/run/docker.sock",
			wantErr:     true,
			errContains: "Cannot connect to the Docker daemon",
		},
		{
			name:        "docker command not found",
			paneContent: "bash: docker: command not found",
			wantErr:     true,
			errContains: "command not found",
		},
		{
			name:        "claude command not found",
			paneContent: "bash: claude: command not found",
			wantErr:     true,
			errContains: "command not found",
		},
		{
			name:        "connection refused",
			paneContent: "Failed to connect to localhost:3000\nconnection refused",
			wantErr:     true,
			errContains: "connection refused",
		},
		{
			name:        "generic error",
			paneContent: "Error: Something went wrong",
			wantErr:     true,
			errContains: "Error:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock GetPaneContent to return test content
			originalGetPaneContent := getPaneContentFunc
			getPaneContentFunc = func(windowTarget string) (string, error) {
				return tt.paneContent, nil
			}
			defer func() { getPaneContentFunc = originalGetPaneContent }()

			cfg := LivenessConfig{
				WaitDuration:  1 * time.Millisecond, // Short duration for tests
				ErrorPatterns: DefaultLivenessConfig().ErrorPatterns,
			}

			err := ProbeWindowForErrors("test:window.0", cfg)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ProbeWindowForErrors() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ProbeWindowForErrors() error = %v, want error containing %q", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("ProbeWindowForErrors() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestDefaultLivenessConfig(t *testing.T) {
	cfg := DefaultLivenessConfig()

	if cfg.WaitDuration == 0 {
		t.Error("DefaultLivenessConfig() WaitDuration should not be zero")
	}

	if len(cfg.ErrorPatterns) == 0 {
		t.Error("DefaultLivenessConfig() should have error patterns")
	}

	// Verify some key patterns are included
	keyPatterns := []string{
		"command not found",
		"Connection refused",
		"Cannot connect",
	}

	for _, pattern := range keyPatterns {
		found := false
		for _, p := range cfg.ErrorPatterns {
			if p == pattern {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("DefaultLivenessConfig() should include pattern %q", pattern)
		}
	}
}
