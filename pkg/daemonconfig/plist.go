package daemonconfig

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

// PlistData holds the template data for generating the launchd plist file.
type PlistData struct {
	Label            string
	OrchPath         string
	PollInterval     int
	MaxAgents        int
	IssueLabel       string
	Verbose          bool
	ReflectIssues    bool
	ReflectOpen      bool
	LogPath          string
	WorkingDirectory string
	PATH             string
	Home             string
}

// PlistTemplate is the launchd plist template for the orch daemon.
const PlistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>{{.Label}}</string>

    <key>ProgramArguments</key>
    <array>
        <string>{{.OrchPath}}</string>
        <string>daemon</string>
        <string>run</string>
        <string>--poll-interval</string>
        <string>{{.PollInterval}}</string>
        <string>--max-agents</string>
        <string>{{.MaxAgents}}</string>
        <string>--label</string>
        <string>{{.IssueLabel}}</string>{{if .Verbose}}
        <string>--verbose</string>{{end}}
        <string>--reflect-issues={{.ReflectIssues}}</string>
        <string>--reflect-open={{.ReflectOpen}}</string>
    </array>

    <key>RunAtLoad</key>
    <true/>

    <key>KeepAlive</key>
    <true/>

    <key>StandardOutPath</key>
    <string>{{.LogPath}}</string>

    <key>StandardErrorPath</key>
    <string>{{.LogPath}}</string>

    <key>WorkingDirectory</key>
    <string>{{.WorkingDirectory}}</string>

    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>{{.PATH}}</string>
        <key>BEADS_NO_DAEMON</key>
        <string>1</string>
    </dict>
</dict>
</plist>
`

// GeneratePlistXML renders the plist template with the given data and returns the XML bytes.
func GeneratePlistXML(data *PlistData) ([]byte, error) {
	tmpl, err := template.New("plist").Parse(PlistTemplate)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// GetPlistPath returns the path to the daemon's launchd plist file.
func GetPlistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", "com.orch.daemon.plist")
}

// FindOrchPath locates the orch binary by checking common locations.
func FindOrchPath(home string) string {
	candidates := []string{
		filepath.Join(home, "bin", "orch"),
		filepath.Join(home, "go", "bin", "orch"),
		filepath.Join(home, ".bun", "bin", "orch"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if path, err := exec.LookPath("orch"); err == nil {
		return path
	}
	return filepath.Join(home, "bin", "orch")
}

// BuildPATH constructs a colon-separated PATH string from config paths
// plus standard system paths.
func BuildPATH(configPaths []string) string {
	systemPaths := []string{"/usr/local/bin", "/usr/bin", "/bin"}
	allPaths := append(configPaths, systemPaths...)
	return strings.Join(allPaths, ":")
}

// ParsePlistValues extracts key values from the daemon plist content.
// Uses simple string parsing since the plist has a known structure.
func ParsePlistValues(content string) (map[string]string, error) {
	values := make(map[string]string)

	// Parse poll-interval
	if idx := strings.Index(content, "--poll-interval"); idx != -1 {
		remaining := content[idx:]
		if start := strings.Index(remaining, "</string>"); start != -1 {
			remaining = remaining[start+9:]
			if strings.HasPrefix(strings.TrimSpace(remaining), "<string>") {
				remaining = strings.TrimSpace(remaining)[8:]
				if end := strings.Index(remaining, "</string>"); end != -1 {
					values["poll_interval"] = remaining[:end]
				}
			}
		}
	}

	// Parse max-agents
	if idx := strings.Index(content, "--max-agents"); idx != -1 {
		remaining := content[idx:]
		if start := strings.Index(remaining, "</string>"); start != -1 {
			remaining = remaining[start+9:]
			if strings.HasPrefix(strings.TrimSpace(remaining), "<string>") {
				remaining = strings.TrimSpace(remaining)[8:]
				if end := strings.Index(remaining, "</string>"); end != -1 {
					values["max_agents"] = remaining[:end]
				}
			}
		}
	}

	// Parse label
	if idx := strings.Index(content, "--label"); idx != -1 {
		remaining := content[idx:]
		if start := strings.Index(remaining, "</string>"); start != -1 {
			remaining = remaining[start+9:]
			if strings.HasPrefix(strings.TrimSpace(remaining), "<string>") {
				remaining = strings.TrimSpace(remaining)[8:]
				if end := strings.Index(remaining, "</string>"); end != -1 {
					values["label"] = remaining[:end]
				}
			}
		}
	}

	// Parse verbose
	values["verbose"] = "false"
	if strings.Contains(content, "<string>--verbose</string>") {
		values["verbose"] = "true"
	}

	// Parse reflect-issues
	values["reflect_issues"] = "true"
	if idx := strings.Index(content, "--reflect-issues="); idx != -1 {
		remaining := content[idx+17:]
		if end := strings.Index(remaining, "</string>"); end != -1 {
			values["reflect_issues"] = remaining[:end]
		}
	}

	// Parse reflect-open
	values["reflect_open"] = "true"
	if idx := strings.Index(content, "--reflect-open="); idx != -1 {
		remaining := content[idx+15:]
		if end := strings.Index(remaining, "</string>"); end != -1 {
			values["reflect_open"] = remaining[:end]
		}
	}

	// Parse WorkingDirectory
	if idx := strings.Index(content, "<key>WorkingDirectory</key>"); idx != -1 {
		remaining := content[idx:]
		if start := strings.Index(remaining, "<string>"); start != -1 {
			remaining = remaining[start+8:]
			if end := strings.Index(remaining, "</string>"); end != -1 {
				values["working_directory"] = remaining[:end]
			}
		}
	}

	return values, nil
}
