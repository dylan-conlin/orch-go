package spawn

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const stalenessEventRetentionDays = 30

// StalenessEvent represents a single model staleness detection entry.
// Each line in the JSONL file is one event.
type StalenessEvent struct {
	Timestamp    string   `json:"timestamp"`
	Model        string   `json:"model"`
	ChangedFiles []string `json:"changed_files,omitempty"`
	DeletedFiles []string `json:"deleted_files,omitempty"`
	SpawnID      string   `json:"spawn_id"`
	AgentSkill   string   `json:"agent_skill"`
}

// StalenessEventMeta provides spawn metadata for recording staleness events.
type StalenessEventMeta struct {
	SpawnID    string
	AgentSkill string
}

// DefaultStalenessEventPath returns the default path to the staleness events file.
func DefaultStalenessEventPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".orch/model-staleness-events.jsonl"
	}
	return filepath.Join(home, ".orch", "model-staleness-events.jsonl")
}

// RecordModelStalenessEvent appends a staleness event for a model when stale.
// Applies a retention window to keep only recent events.
func RecordModelStalenessEvent(modelPath string, result *StalenessResult, meta *StalenessEventMeta) error {
	if result == nil || !result.IsStale || meta == nil {
		return nil
	}

	event := StalenessEvent{
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		Model:        modelPath,
		ChangedFiles: result.ChangedFiles,
		DeletedFiles: result.DeletedFiles,
		SpawnID:      meta.SpawnID,
		AgentSkill:   meta.AgentSkill,
	}

	return writeStalenessEvent(event)
}

func writeStalenessEvent(event StalenessEvent) error {
	path := DefaultStalenessEventPath()
	if err := pruneStalenessEvents(path, stalenessEventRetentionDays); err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create staleness events directory: %w", err)
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open staleness events file: %w", err)
	}
	defer file.Close()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to encode staleness event: %w", err)
	}

	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write staleness event: %w", err)
	}

	return nil
}

func pruneStalenessEvents(path string, retentionDays int) error {
	if retentionDays <= 0 {
		return nil
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open staleness events file: %w", err)
	}
	defer file.Close()

	tmpPath := path + ".tmp"
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create staleness events temp file: %w", err)
	}

	writer := bufio.NewWriter(tmpFile)
	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var event StalenessEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			if _, err := writer.WriteString(line + "\n"); err != nil {
				tmpFile.Close()
				return fmt.Errorf("failed to write staleness events temp file: %w", err)
			}
			continue
		}

		timestamp, err := parseStalenessTimestamp(event.Timestamp)
		if err != nil || timestamp.IsZero() {
			if _, err := writer.WriteString(line + "\n"); err != nil {
				tmpFile.Close()
				return fmt.Errorf("failed to write staleness events temp file: %w", err)
			}
			continue
		}

		if timestamp.Before(cutoff) {
			continue
		}

		if _, err := writer.WriteString(line + "\n"); err != nil {
			tmpFile.Close()
			return fmt.Errorf("failed to write staleness events temp file: %w", err)
		}
	}

	if err := scanner.Err(); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to read staleness events file: %w", err)
	}

	if err := writer.Flush(); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to flush staleness events temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close staleness events temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to replace staleness events file: %w", err)
	}

	return nil
}

func parseStalenessTimestamp(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("empty timestamp")
	}

	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed, nil
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed, nil
	}

	if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
		if len(value) > 10 {
			return time.Unix(0, parsed), nil
		}
		return time.Unix(parsed, 0), nil
	}

	return time.Time{}, fmt.Errorf("unrecognized timestamp format")
}
