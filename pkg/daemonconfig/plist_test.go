package daemonconfig

import (
	"strings"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/userconfig"
)

func TestGeneratePlistXML(t *testing.T) {
	data := &PlistData{
		Label:            "com.orch.daemon",
		OrchPath:         "/Users/test/bin/orch",
		PollInterval:     15,
		MaxAgents:        3,
		IssueLabel:       "triage:ready",
		Verbose:          false,
		ReflectIssues:    true,
		ReflectOpen:      true,
		LogPath:          "/Users/test/.orch/daemon.log",
		WorkingDirectory: "/Users/test/Documents/personal/orch-go",
		PATH:             "/Users/test/bin:/usr/local/bin:/usr/bin:/bin",
		Home:             "/Users/test",
	}

	result, err := GeneratePlistXML(data)
	if err != nil {
		t.Fatalf("GeneratePlistXML() error = %v", err)
	}

	content := string(result)

	// Verify XML header
	if !strings.HasPrefix(content, "<?xml version=\"1.0\"") {
		t.Error("Expected XML header")
	}

	// Verify label
	if !strings.Contains(content, "<string>com.orch.daemon</string>") {
		t.Error("Expected label in plist")
	}

	// Verify orch path
	if !strings.Contains(content, "<string>/Users/test/bin/orch</string>") {
		t.Error("Expected orch path in plist")
	}

	// Verify poll interval
	if !strings.Contains(content, "<string>15</string>") {
		t.Error("Expected poll interval in plist")
	}

	// Verify max agents
	if !strings.Contains(content, "<string>3</string>") {
		t.Error("Expected max agents in plist")
	}

	// Verify issue label
	if !strings.Contains(content, "<string>triage:ready</string>") {
		t.Error("Expected issue label in plist")
	}

	// Verify verbose is NOT present when false
	if strings.Contains(content, "<string>--verbose</string>") {
		t.Error("Expected no --verbose flag when Verbose is false")
	}

	// Verify reflect flags
	if !strings.Contains(content, "<string>--reflect-issues=true</string>") {
		t.Error("Expected --reflect-issues=true")
	}
	if !strings.Contains(content, "<string>--reflect-open=true</string>") {
		t.Error("Expected --reflect-open=true")
	}

	// Verify log path
	if !strings.Contains(content, "<string>/Users/test/.orch/daemon.log</string>") {
		t.Error("Expected log path in plist")
	}

	// Verify working directory
	if !strings.Contains(content, "<string>/Users/test/Documents/personal/orch-go</string>") {
		t.Error("Expected working directory in plist")
	}

	// Verify PATH env var
	if !strings.Contains(content, "<string>/Users/test/bin:/usr/local/bin:/usr/bin:/bin</string>") {
		t.Error("Expected PATH in plist")
	}

	// Verify BEADS_NO_DAEMON
	if !strings.Contains(content, "<key>BEADS_NO_DAEMON</key>") {
		t.Error("Expected BEADS_NO_DAEMON env var")
	}
}

func TestGeneratePlistXMLWithVerbose(t *testing.T) {
	data := &PlistData{
		Label:            "com.orch.daemon",
		OrchPath:         "/Users/test/bin/orch",
		PollInterval:     60,
		MaxAgents:        5,
		IssueLabel:       "triage:ready",
		Verbose:          true,
		ReflectIssues:    false,
		ReflectOpen:      false,
		LogPath:          "/Users/test/.orch/daemon.log",
		WorkingDirectory: "/Users/test/project",
		PATH:             "/usr/bin:/bin",
		Home:             "/Users/test",
	}

	result, err := GeneratePlistXML(data)
	if err != nil {
		t.Fatalf("GeneratePlistXML() error = %v", err)
	}

	content := string(result)

	// Verify verbose IS present when true
	if !strings.Contains(content, "<string>--verbose</string>") {
		t.Error("Expected --verbose flag when Verbose is true")
	}

	// Verify reflect flags are false
	if !strings.Contains(content, "<string>--reflect-issues=false</string>") {
		t.Error("Expected --reflect-issues=false")
	}
	if !strings.Contains(content, "<string>--reflect-open=false</string>") {
		t.Error("Expected --reflect-open=false")
	}
}

func TestGetPlistPath(t *testing.T) {
	path := GetPlistPath()
	if path == "" {
		t.Error("Expected non-empty plist path")
	}
	if !strings.Contains(path, "com.orch.daemon.plist") {
		t.Errorf("Expected path to contain com.orch.daemon.plist, got %s", path)
	}
	if !strings.Contains(path, "Library/LaunchAgents") {
		t.Errorf("Expected path to contain Library/LaunchAgents, got %s", path)
	}
}

func TestFindOrchPath(t *testing.T) {
	// With a non-existent home, should fall back to default
	path := FindOrchPath("/nonexistent")
	if path == "" {
		t.Error("Expected non-empty orch path")
	}
	// Should default to ~/bin/orch for non-existent home
	if !strings.HasSuffix(path, "bin/orch") {
		t.Errorf("Expected path ending in bin/orch, got %s", path)
	}
}

func TestParsePlistValues(t *testing.T) {
	plistContent := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.orch.daemon</string>
    <key>ProgramArguments</key>
    <array>
        <string>/Users/test/bin/orch</string>
        <string>daemon</string>
        <string>run</string>
        <string>--poll-interval</string>
        <string>60</string>
        <string>--max-agents</string>
        <string>5</string>
        <string>--label</string>
        <string>triage:ready</string>
        <string>--verbose</string>
        <string>--reflect-issues=true</string>
        <string>--reflect-open=false</string>
    </array>
    <key>WorkingDirectory</key>
    <string>/Users/test/Documents/personal/orch-go</string>
</dict>
</plist>`

	values, err := ParsePlistValues(plistContent)
	if err != nil {
		t.Fatalf("ParsePlistValues() error = %v", err)
	}

	tests := []struct {
		key      string
		expected string
	}{
		{"poll_interval", "60"},
		{"max_agents", "5"},
		{"label", "triage:ready"},
		{"verbose", "true"},
		{"reflect_issues", "true"},
		{"reflect_open", "false"},
		{"working_directory", "/Users/test/Documents/personal/orch-go"},
	}

	for _, tt := range tests {
		got := values[tt.key]
		if got != tt.expected {
			t.Errorf("ParsePlistValues()[%q] = %q, want %q", tt.key, got, tt.expected)
		}
	}
}

func TestParsePlistValuesWithoutVerbose(t *testing.T) {
	plistContent := `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
    <key>ProgramArguments</key>
    <array>
        <string>/Users/test/bin/orch</string>
        <string>daemon</string>
        <string>run</string>
        <string>--poll-interval</string>
        <string>30</string>
        <string>--max-agents</string>
        <string>3</string>
    </array>
</dict>
</plist>`

	values, err := ParsePlistValues(plistContent)
	if err != nil {
		t.Fatalf("ParsePlistValues() error = %v", err)
	}

	if values["verbose"] != "false" {
		t.Errorf("ParsePlistValues() verbose = %q, want \"false\"", values["verbose"])
	}
	if values["poll_interval"] != "30" {
		t.Errorf("ParsePlistValues() poll_interval = %q, want \"30\"", values["poll_interval"])
	}
}

func TestParsePlistValuesReflectFlags(t *testing.T) {
	plistContent := `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
    <key>ProgramArguments</key>
    <array>
        <string>--reflect-issues=true</string>
        <string>--reflect-open=false</string>
    </array>
</dict>
</plist>`

	values, err := ParsePlistValues(plistContent)
	if err != nil {
		t.Fatalf("ParsePlistValues() error = %v", err)
	}

	if values["reflect_issues"] != "true" {
		t.Errorf("ParsePlistValues() reflect_issues = %q, want \"true\"", values["reflect_issues"])
	}
	if values["reflect_open"] != "false" {
		t.Errorf("ParsePlistValues() reflect_open = %q, want \"false\"", values["reflect_open"])
	}
}

func TestBuildPlistData(t *testing.T) {
	cfg := &userconfig.Config{}
	data, err := BuildPlistData(cfg)
	if err != nil {
		t.Fatalf("BuildPlistData() error = %v", err)
	}

	// Should set the standard label
	if data.Label != "com.orch.daemon" {
		t.Errorf("Label = %q, want %q", data.Label, "com.orch.daemon")
	}

	// OrchPath should be non-empty
	if data.OrchPath == "" {
		t.Error("OrchPath should not be empty")
	}

	// Should use defaults from FromUserConfig (which reads userconfig accessor defaults)
	dcfg := FromUserConfig(cfg)
	if data.PollInterval != int(dcfg.PollInterval.Seconds()) {
		t.Errorf("PollInterval = %d, want %d", data.PollInterval, int(dcfg.PollInterval.Seconds()))
	}
	if data.MaxAgents != dcfg.MaxAgents {
		t.Errorf("MaxAgents = %d, want %d", data.MaxAgents, dcfg.MaxAgents)
	}

	// LogPath should contain .orch/daemon.log
	if !strings.Contains(data.LogPath, ".orch/daemon.log") {
		t.Errorf("LogPath = %q, should contain .orch/daemon.log", data.LogPath)
	}

	// PATH should contain system paths
	if !strings.Contains(data.PATH, "/usr/bin") {
		t.Errorf("PATH = %q, should contain /usr/bin", data.PATH)
	}
}

func TestGeneratePlist(t *testing.T) {
	cfg := &userconfig.Config{}
	result, err := GeneratePlist(cfg)
	if err != nil {
		t.Fatalf("GeneratePlist() error = %v", err)
	}

	content := string(result)

	// Should be valid XML
	if !strings.HasPrefix(content, "<?xml version=\"1.0\"") {
		t.Error("Expected XML header")
	}

	// Should contain the standard label
	if !strings.Contains(content, "<string>com.orch.daemon</string>") {
		t.Error("Expected com.orch.daemon label")
	}

	// Should contain daemon run command
	if !strings.Contains(content, "<string>daemon</string>") {
		t.Error("Expected daemon command in plist")
	}
}

func TestBuildPATH(t *testing.T) {
	configPaths := []string{"/Users/test/bin", "/Users/test/.bun/bin"}
	result := BuildPATH(configPaths)

	// Should include config paths and system paths
	if !strings.Contains(result, "/Users/test/bin") {
		t.Error("Expected config path in result")
	}
	if !strings.Contains(result, "/usr/local/bin") {
		t.Error("Expected system path /usr/local/bin in result")
	}
	if !strings.Contains(result, "/usr/bin") {
		t.Error("Expected system path /usr/bin in result")
	}
	if !strings.Contains(result, "/bin") {
		t.Error("Expected system path /bin in result")
	}

	// Should be colon-separated
	parts := strings.Split(result, ":")
	if len(parts) < 4 {
		t.Errorf("Expected at least 4 path components, got %d", len(parts))
	}
}
