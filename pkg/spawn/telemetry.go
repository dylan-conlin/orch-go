// Package spawn provides spawn configuration, context generation, and telemetry.
package spawn

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// EventTypeSpawnTelemetry is the event type for spawn telemetry in events.jsonl.
const EventTypeSpawnTelemetry = "spawn.telemetry"

// SpawnTelemetry captures observability data at spawn time.
// This is logged to events.jsonl to enable analysis of spawn patterns,
// context size trends, and outcome correlation.
type SpawnTelemetry struct {
	// BeadsID is the beads issue ID associated with this spawn (if tracked).
	BeadsID string `json:"beads_id,omitempty"`
	// WorkspaceName is the generated workspace directory name.
	WorkspaceName string `json:"workspace_name"`
	// Skill is the skill name used for this spawn.
	Skill string `json:"skill"`
	// Tier is the spawn tier: "light" or "full".
	Tier string `json:"tier"`
	// ContextSizeChars is the total character count of the generated SPAWN_CONTEXT.md.
	ContextSizeChars int `json:"context_size_chars"`
	// ContextSizeTokensEst is the estimated token count (chars / CharsPerToken).
	ContextSizeTokensEst int `json:"context_size_tokens_est"`
	// KBContextStats contains statistics about the kb context injection (if any).
	KBContextStats *KBContextStats `json:"kb_context_stats,omitempty"`
	// BehavioralPatternsCount is the number of behavioral patterns injected.
	BehavioralPatternsCount int `json:"behavioral_patterns_count"`
	// EcosystemInjected indicates whether ecosystem context was included.
	EcosystemInjected bool `json:"ecosystem_context_injected"`
	// ServerContextInjected indicates whether server context was included.
	ServerContextInjected bool `json:"server_context_injected"`
}

// KBContextStats captures statistics about kb context included in spawn.
type KBContextStats struct {
	// Query is the keywords used for kb context lookup.
	Query string `json:"query,omitempty"`
	// MatchCount is the number of matches included (after truncation if any).
	MatchCount int `json:"match_count"`
	// WasTruncated indicates if context was truncated due to token limits.
	WasTruncated bool `json:"was_truncated"`
	// ConstraintsCount is the number of constraints included.
	ConstraintsCount int `json:"constraints_count"`
	// DecisionsCount is the number of decisions included.
	DecisionsCount int `json:"decisions_count"`
	// InvestigationsCount is the number of investigations included.
	InvestigationsCount int `json:"investigations_count"`
}

// CollectSpawnTelemetry gathers telemetry data from spawn configuration and generated context.
// This should be called after GenerateContext() to capture accurate size metrics.
// Accepts optional KBContextFormatResult for detailed truncation info; otherwise uses GapAnalysis from Config.
func CollectSpawnTelemetry(cfg *Config, generatedContext string, kbResult *KBContextFormatResult) SpawnTelemetry {
	telemetry := SpawnTelemetry{
		BeadsID:               cfg.BeadsID,
		WorkspaceName:         cfg.WorkspaceName,
		Skill:                 cfg.SkillName,
		Tier:                  cfg.Tier,
		ContextSizeChars:      len(generatedContext),
		ContextSizeTokensEst:  EstimateTokens(len(generatedContext)),
		EcosystemInjected:     cfg.EcosystemContext != "",
		ServerContextInjected: cfg.ServerContext != "",
	}

	// Count behavioral patterns if present
	if cfg.BehavioralPatterns != "" {
		// Count lines starting with emoji indicators (🚫 or ⚠️)
		count := 0
		for _, line := range splitLines(cfg.BehavioralPatterns) {
			if len(line) > 0 && (hasPrefix(line, "🚫") || hasPrefix(line, "⚠️")) {
				count++
			}
		}
		telemetry.BehavioralPatternsCount = count
	}

	// Populate KB context stats from KBContextFormatResult if available
	if kbResult != nil {
		telemetry.KBContextStats = &KBContextStats{
			MatchCount:   kbResult.TruncatedMatches,
			WasTruncated: kbResult.WasTruncated,
		}
	} else if cfg.GapAnalysis != nil {
		// Fall back to GapAnalysis if KBContextFormatResult not provided
		telemetry.KBContextStats = &KBContextStats{
			Query:               cfg.GapAnalysis.Query,
			MatchCount:          cfg.GapAnalysis.MatchStats.TotalMatches,
			WasTruncated:        false, // GapAnalysis doesn't track truncation
			ConstraintsCount:    cfg.GapAnalysis.MatchStats.ConstraintCount,
			DecisionsCount:      cfg.GapAnalysis.MatchStats.DecisionCount,
			InvestigationsCount: cfg.GapAnalysis.MatchStats.InvestigationCount,
		}
	}

	return telemetry
}

// LogSpawnTelemetry writes a spawn telemetry event to the events log file.
// This is a standalone function that can be called from WriteContext().
func LogSpawnTelemetry(logPath string, telemetry SpawnTelemetry) error {
	// Ensure directory exists
	dir := filepath.Dir(logPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create the event wrapper
	event := struct {
		Type      string         `json:"type"`
		Timestamp int64          `json:"timestamp"`
		Data      SpawnTelemetry `json:"data"`
	}{
		Type:      EventTypeSpawnTelemetry,
		Timestamp: time.Now().Unix(),
		Data:      telemetry,
	}

	// Encode to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal telemetry event: %w", err)
	}

	// Append to file
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write telemetry event: %w", err)
	}

	return nil
}

// DefaultTelemetryLogPath returns the default path to events.jsonl (~/.orch/events.jsonl).
func DefaultTelemetryLogPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".orch/events.jsonl"
	}
	return filepath.Join(home, ".orch", "events.jsonl")
}

// splitLines splits a string into lines.
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// hasPrefix checks if a string starts with a given prefix (handles multi-byte prefixes like emoji).
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
