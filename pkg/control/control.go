// Package control provides immutable control plane enforcement.
// Lives outside repo at ~/.orch/ - agents cannot modify it.
package control

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var (
	// ControlDir is the immutable control plane directory
	ControlDir = filepath.Join(os.Getenv("HOME"), ".orch")

	// ConfigPath is the control plane config file
	ConfigPath = filepath.Join(ControlDir, "control-plane.conf")

	// HaltPath is the halt sentinel file
	HaltPath = filepath.Join(ControlDir, "halt")

	// MetricsDir is where metrics logs are stored
	MetricsDir = filepath.Join(ControlDir, "metrics")
)

// Config represents control plane configuration
type Config struct {
	MaxCommitsPerDay      int
	FixFeatRatioThreshold int
	ChurnRatioThreshold   int
	ProtectedPaths        []string
	CooldownMinutes       int
}

// HaltInfo represents halt sentinel data
type HaltInfo struct {
	Reason      string
	TriggeredBy string
	TriggeredAt time.Time
}

// StatusInfo represents control plane status
type StatusInfo struct {
	ConfigExists bool
	Config       *Config
	Halted       bool
	HaltInfo     *HaltInfo
	CommitsToday int
	Violations   []string
	FixFeatRatio string
	MetricsExist bool
}

// CheckHalt reads ~/.orch/halt and returns (halted, reason)
func CheckHalt() (bool, string) {
	data, err := os.ReadFile(HaltPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, ""
		}
		return false, fmt.Sprintf("error reading halt file: %v", err)
	}

	// Parse halt file (simple key: value format)
	lines := strings.Split(string(data), "\n")
	reason := "unknown"
	for _, line := range lines {
		if strings.HasPrefix(line, "reason:") {
			reason = strings.TrimSpace(strings.TrimPrefix(line, "reason:"))
			break
		}
	}

	return true, reason
}

// ClearHalt removes ~/.orch/halt
func ClearHalt() error {
	err := os.Remove(HaltPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to clear halt: %w", err)
	}
	return nil
}

// LoadConfig reads ~/.orch/control-plane.conf
func LoadConfig() (*Config, error) {
	data, err := os.ReadFile(ConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config not found: %s (run 'orch control init')", ConfigPath)
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	cfg := &Config{
		CooldownMinutes: 30, // default
	}

	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse key=value
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		value = strings.Trim(value, "\"")

		switch key {
		case "MAX_COMMITS_PER_DAY":
			if v, err := strconv.Atoi(value); err == nil {
				cfg.MaxCommitsPerDay = v
			}
		case "FIX_FEAT_RATIO_THRESHOLD":
			if v, err := strconv.Atoi(value); err == nil {
				cfg.FixFeatRatioThreshold = v
			}
		case "CHURN_RATIO_THRESHOLD":
			if v, err := strconv.Atoi(value); err == nil {
				cfg.ChurnRatioThreshold = v
			}
		case "PROTECTED_PATHS":
			// Split on spaces
			cfg.ProtectedPaths = strings.Fields(value)
		case "COOLDOWN_MINUTES":
			if v, err := strconv.Atoi(value); err == nil {
				cfg.CooldownMinutes = v
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}

	return cfg, nil
}

// ParseHaltInfo reads halt file and returns structured info
func ParseHaltInfo() (*HaltInfo, error) {
	data, err := os.ReadFile(HaltPath)
	if err != nil {
		return nil, err
	}

	info := &HaltInfo{}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "reason:") {
			info.Reason = strings.TrimSpace(strings.TrimPrefix(line, "reason:"))
		} else if strings.HasPrefix(line, "triggered_by:") {
			info.TriggeredBy = strings.TrimSpace(strings.TrimPrefix(line, "triggered_by:"))
		} else if strings.HasPrefix(line, "triggered_at:") {
			timeStr := strings.TrimSpace(strings.TrimPrefix(line, "triggered_at:"))
			if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
				info.TriggeredAt = t
			}
		}
	}

	return info, nil
}

// Status returns control plane state (config + halt + recent metrics)
func Status() (*StatusInfo, error) {
	status := &StatusInfo{}

	// Check config exists
	if _, err := os.Stat(ConfigPath); err == nil {
		status.ConfigExists = true
		if cfg, err := LoadConfig(); err == nil {
			status.Config = cfg
		}
	}

	// Check halt status
	if halted, _ := CheckHalt(); halted {
		status.Halted = true
		if info, err := ParseHaltInfo(); err == nil {
			status.HaltInfo = info
		}
	}

	// Read today's commit count from metrics
	commitsLogPath := filepath.Join(MetricsDir, "daily-commits.log")
	if data, err := os.ReadFile(commitsLogPath); err == nil {
		status.MetricsExist = true
		lines := strings.Split(string(data), "\n")
		today := time.Now().Format("2006-01-02")
		for i := len(lines) - 1; i >= 0; i-- {
			if strings.HasPrefix(lines[i], today) {
				parts := strings.Fields(lines[i])
				if len(parts) >= 2 {
					if count, err := strconv.Atoi(parts[1]); err == nil {
						status.CommitsToday = count
					}
					break
				}
			}
		}
	}

	// Read protected path violations from metrics
	violationsLogPath := filepath.Join(MetricsDir, "protected-violations.log")
	if data, err := os.ReadFile(violationsLogPath); err == nil {
		lines := strings.Split(string(data), "\n")
		today := time.Now().Format("2006-01-02")
		for _, line := range lines {
			if strings.Contains(line, today) {
				status.Violations = append(status.Violations, line)
			}
		}
	}

	// Read fix:feat ratio from metrics
	ratioLogPath := filepath.Join(MetricsDir, "fix-feat-ratio.log")
	if data, err := os.ReadFile(ratioLogPath); err == nil {
		lines := strings.Split(string(data), "\n")
		today := time.Now().Format("2006-01-02")
		for i := len(lines) - 1; i >= 0; i-- {
			if strings.HasPrefix(lines[i], today) {
				status.FixFeatRatio = strings.TrimPrefix(lines[i], today+" ")
				break
			}
		}
	}

	return status, nil
}

// InitConfig creates ~/.orch/control-plane.conf with defaults
func InitConfig() error {
	// Ensure directory exists
	if err := os.MkdirAll(ControlDir, 0755); err != nil {
		return fmt.Errorf("failed to create control dir: %w", err)
	}

	// Check if config already exists
	if _, err := os.Stat(ConfigPath); err == nil {
		return fmt.Errorf("config already exists: %s", ConfigPath)
	}

	defaultConfig := `# Control Plane Configuration
# Circuit breakers — daemon halts when threshold exceeded
MAX_COMMITS_PER_DAY=20
FIX_FEAT_RATIO_THRESHOLD=50    # percentage (50 = 0.5:1 fix:feat)
CHURN_RATIO_THRESHOLD=200      # percentage (200 = 2:1 created+deleted/net)

# Protected paths — violations logged, human notified
PROTECTED_PATHS="cmd/orch/ pkg/daemon/ pkg/spawn/ pkg/verify/ plugins/"

# Cooldown after circuit break
COOLDOWN_MINUTES=30
`

	if err := os.WriteFile(ConfigPath, []byte(defaultConfig), 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}
