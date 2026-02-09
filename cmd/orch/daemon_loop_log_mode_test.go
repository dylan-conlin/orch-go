package main

import "testing"

func TestDaemonVerboseForActive(t *testing.T) {
	tests := []struct {
		name    string
		verbose bool
		active  int
		want    bool
	}{
		{name: "verbose disabled", verbose: false, active: 0, want: false},
		{name: "verbose low concurrency", verbose: true, active: 0, want: true},
		{name: "verbose below threshold", verbose: true, active: daemonSwarmVerboseThreshold - 1, want: true},
		{name: "verbose at threshold", verbose: true, active: daemonSwarmVerboseThreshold, want: false},
		{name: "verbose above threshold", verbose: true, active: daemonSwarmVerboseThreshold + 5, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := daemonVerboseForActive(tt.verbose, tt.active); got != tt.want {
				t.Fatalf("daemonVerboseForActive(%v, %d) = %v, want %v", tt.verbose, tt.active, got, tt.want)
			}
		})
	}
}
