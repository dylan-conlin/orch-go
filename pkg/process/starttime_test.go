package process

import (
	"os"
	"testing"
	"time"
)

func TestParseEtime(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{
			name:  "seconds only",
			input: "00:30",
			want:  30 * time.Second,
		},
		{
			name:  "minutes and seconds",
			input: "05:30",
			want:  5*time.Minute + 30*time.Second,
		},
		{
			name:  "hours minutes seconds",
			input: "02:05:30",
			want:  2*time.Hour + 5*time.Minute + 30*time.Second,
		},
		{
			name:  "days",
			input: "3-02:05:30",
			want:  3*24*time.Hour + 2*time.Hour + 5*time.Minute + 30*time.Second,
		},
		{
			name:  "days with MM:SS",
			input: "1-05:30",
			want:  1*24*time.Hour + 5*time.Minute + 30*time.Second,
		},
		{
			name:    "empty",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid",
			input:   "abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseEtime(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseEtime(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseEtime(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestProcessStartTime_CurrentProcess(t *testing.T) {
	// Our own process should have a start time
	pid := currentPID()
	if pid <= 0 {
		t.Skip("cannot determine current PID")
	}

	startTime, err := ProcessStartTime(pid)
	if err != nil {
		t.Fatalf("ProcessStartTime(%d) error: %v", pid, err)
	}

	// Start time should be before now
	if startTime.After(time.Now()) {
		t.Errorf("ProcessStartTime(%d) = %v, which is in the future", pid, startTime)
	}

	// Start time should be within the last hour (test process)
	age := time.Since(startTime)
	if age > time.Hour {
		t.Errorf("ProcessStartTime(%d) age = %v, expected within last hour", pid, age)
	}
}

func TestProcessStartTime_InvalidPID(t *testing.T) {
	_, err := ProcessStartTime(-1)
	if err == nil {
		t.Error("ProcessStartTime(-1) should return error")
	}
}

func currentPID() int {
	return os.Getpid()
}
