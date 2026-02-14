// Package spawn provides spawn configuration and context generation.
package spawn

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// SessionIDFilename is the name of the file storing the session ID in the workspace.
const SessionIDFilename = ".session_id"

// WriteSessionID writes the OpenCode session ID to the workspace directory.
// Uses atomic write (temp file + rename) to prevent partial reads.
// The workspace directory must already exist.
func WriteSessionID(workspacePath, sessionID string) error {
	if sessionID == "" {
		return nil // Nothing to write
	}

	sessionFile := filepath.Join(workspacePath, SessionIDFilename)
	tmpFile := sessionFile + ".tmp"

	// Write to temp file first
	if err := os.WriteFile(tmpFile, []byte(sessionID+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to write session ID temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpFile, sessionFile); err != nil {
		os.Remove(tmpFile) // Clean up temp file on error
		return fmt.Errorf("failed to rename session ID file: %w", err)
	}

	return nil
}

// ReadSessionID reads the OpenCode session ID from the workspace directory.
// Returns empty string if the file doesn't exist or is empty.
func ReadSessionID(workspacePath string) string {
	sessionFile := filepath.Join(workspacePath, SessionIDFilename)
	data, err := os.ReadFile(sessionFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// SessionIDPath returns the path to the session ID file for a workspace.
func SessionIDPath(workspacePath string) string {
	return filepath.Join(workspacePath, SessionIDFilename)
}

// TierFilename is the name of the file storing the spawn tier in the workspace.
const TierFilename = ".tier"

// WriteTier writes the spawn tier to the workspace directory.
// Uses atomic write (temp file + rename) to prevent partial reads.
// The workspace directory must already exist.
func WriteTier(workspacePath, tier string) error {
	if tier == "" {
		return nil // Nothing to write
	}

	tierFile := filepath.Join(workspacePath, TierFilename)
	tmpFile := tierFile + ".tmp"

	// Write to temp file first
	if err := os.WriteFile(tmpFile, []byte(tier+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to write tier temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpFile, tierFile); err != nil {
		os.Remove(tmpFile) // Clean up temp file on error
		return fmt.Errorf("failed to rename tier file: %w", err)
	}

	return nil
}

// ReadTier reads the spawn tier from the workspace directory.
// Returns empty string if the file doesn't exist or is empty.
// Returns TierFull as the default if the file is missing (conservative default).
func ReadTier(workspacePath string) string {
	tierFile := filepath.Join(workspacePath, TierFilename)
	data, err := os.ReadFile(tierFile)
	if err != nil {
		// Conservative default: return TierFull for old workspaces without tier file
		return TierFull
	}
	tier := strings.TrimSpace(string(data))
	if tier == "" {
		return TierFull
	}
	return tier
}

// TierPath returns the path to the tier file for a workspace.
func TierPath(workspacePath string) string {
	return filepath.Join(workspacePath, TierFilename)
}

// SpawnTimeFilename is the name of the file storing the spawn timestamp in the workspace.
const SpawnTimeFilename = ".spawn_time"

// WriteSpawnTime writes the spawn timestamp to the workspace directory.
// The timestamp is stored as Unix epoch nanoseconds for precision.
// Uses atomic write (temp file + rename) to prevent partial reads.
// The workspace directory must already exist.
func WriteSpawnTime(workspacePath string, t time.Time) error {
	spawnTimeFile := filepath.Join(workspacePath, SpawnTimeFilename)
	tmpFile := spawnTimeFile + ".tmp"

	// Store as Unix nanoseconds for precision
	content := strconv.FormatInt(t.UnixNano(), 10) + "\n"

	// Write to temp file first
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write spawn time temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpFile, spawnTimeFile); err != nil {
		os.Remove(tmpFile) // Clean up temp file on error
		return fmt.Errorf("failed to rename spawn time file: %w", err)
	}

	return nil
}

// ReadSpawnTime reads the spawn timestamp from the workspace directory.
// Returns zero time if the file doesn't exist or is invalid.
func ReadSpawnTime(workspacePath string) time.Time {
	spawnTimeFile := filepath.Join(workspacePath, SpawnTimeFilename)
	data, err := os.ReadFile(spawnTimeFile)
	if err != nil {
		return time.Time{} // Return zero time if file doesn't exist
	}

	nanos, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return time.Time{} // Return zero time if parse fails
	}

	return time.Unix(0, nanos)
}

// SpawnTimePath returns the path to the spawn time file for a workspace.
func SpawnTimePath(workspacePath string) string {
	return filepath.Join(workspacePath, SpawnTimeFilename)
}

// AgentManifestFilename is the name of the file storing the agent manifest in the workspace.
const AgentManifestFilename = "AGENT_MANIFEST.json"

// AgentManifest contains canonical agent identity and spawn-time metadata.
// This provides a single source of truth for "what did this agent do" to enable
// reliable git-based scoping for verification gates.
//
// See .kb/investigations/2026-01-17-inv-design-synthesize-26-completion-investigations.md
// for the architectural rationale.
type AgentManifest struct {
	// WorkspaceName is the canonical agent identifier (e.g., "og-feat-add-manifest-17jan-abc1")
	WorkspaceName string `json:"workspace_name"`

	// Skill is the skill used to spawn this agent (e.g., "feature-impl", "investigation")
	Skill string `json:"skill"`

	// BeadsID is the beads issue ID for tracking (empty for --no-track spawns)
	BeadsID string `json:"beads_id,omitempty"`

	// ProjectDir is the absolute path to the project directory
	ProjectDir string `json:"project_dir"`

	// GitBaseline is the git commit SHA at spawn time
	// Used for git-based change detection: git diff <baseline>..HEAD
	// Empty if not in a git repository or git command fails
	GitBaseline string `json:"git_baseline,omitempty"`

	// SpawnTime is the ISO 8601 timestamp when the agent was spawned
	SpawnTime string `json:"spawn_time"`

	// Tier is the spawn tier: "light" or "full"
	Tier string `json:"tier"`

	// SpawnMode is the spawn backend: "opencode" or "claude"
	SpawnMode string `json:"spawn_mode,omitempty"`

	// Model is the model spec used for this agent (e.g., "gemini-3-flash-preview", "claude-opus-4-5-20251101")
	Model string `json:"model,omitempty"`
}

// WriteAgentManifest writes the agent manifest JSON to the workspace directory.
// The manifest provides a canonical source of agent identity and spawn-time context.
// Uses atomic write (temp file + rename) to prevent partial reads.
// The workspace directory must already exist.
func WriteAgentManifest(workspacePath string, manifest AgentManifest) error {
	manifestFile := filepath.Join(workspacePath, AgentManifestFilename)
	tmpFile := manifestFile + ".tmp"

	// Marshal to JSON with indentation for human readability
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	// Add trailing newline for POSIX compliance
	data = append(data, '\n')

	// Write to temp file first
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpFile, manifestFile); err != nil {
		os.Remove(tmpFile) // Clean up temp file on error
		return fmt.Errorf("failed to rename manifest file: %w", err)
	}

	return nil
}

// ReadAgentManifest reads the agent manifest from the workspace directory.
// Returns an error if the file doesn't exist or is malformed.
func ReadAgentManifest(workspacePath string) (*AgentManifest, error) {
	manifestFile := filepath.Join(workspacePath, AgentManifestFilename)
	data, err := os.ReadFile(manifestFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest AgentManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal manifest: %w", err)
	}

	return &manifest, nil
}

// AgentManifestPath returns the path to the agent manifest file for a workspace.
func AgentManifestPath(workspacePath string) string {
	return filepath.Join(workspacePath, AgentManifestFilename)
}

// ReadAgentManifestWithFallback reads the agent manifest from the workspace directory.
// If AGENT_MANIFEST.json exists, it is the source of truth. Otherwise, falls back
// to reading individual dotfiles (.beads_id, .tier, .spawn_time, .spawn_mode) for
// backward compatibility with pre-manifest workspaces.
// Always returns a non-nil manifest (fields may be empty if nothing is readable).
func ReadAgentManifestWithFallback(workspacePath string) *AgentManifest {
	manifest, err := ReadAgentManifest(workspacePath)
	if err == nil {
		return manifest
	}
	// Fallback: construct manifest from individual dotfiles
	return readLegacyDotfiles(workspacePath)
}

// readLegacyDotfiles constructs an AgentManifest from individual dotfiles.
// Used as fallback for pre-manifest workspaces.
func readLegacyDotfiles(workspacePath string) *AgentManifest {
	manifest := &AgentManifest{
		WorkspaceName: filepath.Base(workspacePath),
	}

	// Read .beads_id
	if data, err := os.ReadFile(filepath.Join(workspacePath, ".beads_id")); err == nil {
		manifest.BeadsID = strings.TrimSpace(string(data))
	}

	// Read .tier (default to "full" if missing)
	if data, err := os.ReadFile(filepath.Join(workspacePath, TierFilename)); err == nil {
		tier := strings.TrimSpace(string(data))
		if tier != "" {
			manifest.Tier = tier
		} else {
			manifest.Tier = TierFull
		}
	} else {
		manifest.Tier = TierFull
	}

	// Read .spawn_time (Unix nanos in dotfile -> RFC3339 in manifest)
	if data, err := os.ReadFile(filepath.Join(workspacePath, SpawnTimeFilename)); err == nil {
		nanos, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
		if err == nil {
			manifest.SpawnTime = time.Unix(0, nanos).Format(time.RFC3339)
		}
	}

	// Read .spawn_mode
	if data, err := os.ReadFile(filepath.Join(workspacePath, ".spawn_mode")); err == nil {
		manifest.SpawnMode = strings.TrimSpace(string(data))
	}

	return manifest
}

// ParseSpawnTime parses the manifest's SpawnTime string (RFC3339) to time.Time.
// Returns zero time if the field is empty or unparseable.
func (m *AgentManifest) ParseSpawnTime() time.Time {
	if m.SpawnTime == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, m.SpawnTime)
	if err != nil {
		return time.Time{}
	}
	return t
}
