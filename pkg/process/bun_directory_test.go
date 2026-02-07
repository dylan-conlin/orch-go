package process

import "testing"

func TestParsePIDArgsLine(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		wantPID    int
		wantCmd    string
		wantParsed bool
	}{
		{
			name:       "valid pid and command",
			line:       "12345 /opt/homebrew/bin/bun run dev",
			wantPID:    12345,
			wantCmd:    "/opt/homebrew/bin/bun run dev",
			wantParsed: true,
		},
		{
			name:       "header line",
			line:       "PID COMMAND",
			wantParsed: false,
		},
		{
			name:       "empty line",
			line:       "",
			wantParsed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPID, gotCmd, gotParsed := parsePIDArgsLine(tt.line)
			if gotParsed != tt.wantParsed {
				t.Fatalf("parsePIDArgsLine() parsed = %v, want %v", gotParsed, tt.wantParsed)
			}
			if !tt.wantParsed {
				return
			}
			if gotPID != tt.wantPID {
				t.Errorf("parsePIDArgsLine() pid = %d, want %d", gotPID, tt.wantPID)
			}
			if gotCmd != tt.wantCmd {
				t.Errorf("parsePIDArgsLine() command = %q, want %q", gotCmd, tt.wantCmd)
			}
		})
	}
}

func TestIsBunCommandLine(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    bool
	}{
		{name: "plain bun", command: "bun run dev", want: true},
		{name: "absolute path bun", command: "/opt/homebrew/bin/bun run dev", want: true},
		{name: "env wrapper", command: "env NODE_ENV=development bun run dev", want: true},
		{name: "non bun", command: "node server.js", want: false},
		{name: "empty", command: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isBunCommandLine(tt.command); got != tt.want {
				t.Errorf("isBunCommandLine(%q) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}

func TestParseLsofCWD(t *testing.T) {
	output := "p12345\nfcwd\nn/Users/dylanconlin/Documents/personal/orch-go/web\n"
	got, err := parseLsofCWD(output)
	if err != nil {
		t.Fatalf("parseLsofCWD() returned error: %v", err)
	}
	if got != "/Users/dylanconlin/Documents/personal/orch-go/web" {
		t.Errorf("parseLsofCWD() = %q, want %q", got, "/Users/dylanconlin/Documents/personal/orch-go/web")
	}
}

func TestIsWithinDirectory(t *testing.T) {
	directory := "/Users/dylanconlin/Documents/personal/orch-go/web"

	tests := []struct {
		name string
		path string
		want bool
	}{
		{name: "exact directory", path: "/Users/dylanconlin/Documents/personal/orch-go/web", want: true},
		{name: "subdirectory", path: "/Users/dylanconlin/Documents/personal/orch-go/web/.svelte-kit", want: true},
		{name: "outside directory", path: "/Users/dylanconlin/Documents/personal/orch-go", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isWithinDirectory(tt.path, directory); got != tt.want {
				t.Errorf("isWithinDirectory(%q, %q) = %v, want %v", tt.path, directory, got, tt.want)
			}
		})
	}
}
