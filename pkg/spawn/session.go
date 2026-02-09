// Package spawn provides spawn configuration and context generation.
package spawn

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/atomicwrite"
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
	if err := atomicwrite.WriteFile(sessionFile, []byte(sessionID+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to write session ID: %w", err)
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

// BeadsIDFilename is the name of the file storing the beads ID in the workspace.
const BeadsIDFilename = ".beads_id"

// ReadBeadsID reads the beads issue ID from the workspace directory.
// Returns empty string if the file doesn't exist or is empty.
func ReadBeadsID(workspacePath string) string {
	beadsFile := filepath.Join(workspacePath, BeadsIDFilename)
	data, err := os.ReadFile(beadsFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// BeadsIDPath returns the path to the beads ID file for a workspace.
func BeadsIDPath(workspacePath string) string {
	return filepath.Join(workspacePath, BeadsIDFilename)
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
	if err := atomicwrite.WriteFile(tierFile, []byte(tier+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to write tier: %w", err)
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
	content := strconv.FormatInt(t.UnixNano(), 10) + "\n"

	if err := atomicwrite.WriteFile(spawnTimeFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write spawn time: %w", err)
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

// AttemptIDFilename is the name of the file storing the spawn attempt UUID in the workspace.
const AttemptIDFilename = ".attempt_id"

// GenerateAttemptID generates a UUIDv4 for identifying a specific spawn attempt.
func GenerateAttemptID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Set version (4) and variant (RFC 4122).
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

// WriteAttemptID writes the attempt UUID to the workspace directory.
// Uses atomic write (temp file + rename) to prevent partial reads.
// The workspace directory must already exist.
func WriteAttemptID(workspacePath, attemptID string) error {
	if attemptID == "" {
		return nil
	}

	attemptFile := filepath.Join(workspacePath, AttemptIDFilename)
	if err := atomicwrite.WriteFile(attemptFile, []byte(attemptID+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to write attempt ID: %w", err)
	}
	return nil
}

// ReadAttemptID reads the attempt UUID from the workspace directory.
// Returns empty string if the file doesn't exist or is empty.
func ReadAttemptID(workspacePath string) string {
	attemptFile := filepath.Join(workspacePath, AttemptIDFilename)
	data, err := os.ReadFile(attemptFile)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// AttemptIDPath returns the path to the attempt ID file for a workspace.
func AttemptIDPath(workspacePath string) string {
	return filepath.Join(workspacePath, AttemptIDFilename)
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

	// SourceProjectDir is the canonical source repository root.
	// This remains stable even when runtime execution moves to a dedicated worktree.
	SourceProjectDir string `json:"source_project_dir,omitempty"`

	// ProjectDir is the absolute path to the project directory.
	// Deprecated compatibility alias for SourceProjectDir.
	ProjectDir string `json:"project_dir"`

	// GitWorktreeDir is the git working directory used by the agent runtime.
	// Defaults to ProjectDir for legacy/monorepo spawns.
	GitWorktreeDir string `json:"git_worktree_dir,omitempty"`

	// GitBranch is the git branch associated with this agent's work.
	GitBranch string `json:"git_branch,omitempty"`

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

	// Model is the model ID used for this agent (e.g., "claude-opus-4-5-20251101", "claude-sonnet-4-5-20250929")
	Model string `json:"model,omitempty"`
}

// WriteAgentManifest writes the agent manifest JSON to the workspace directory.
// The manifest provides a canonical source of agent identity and spawn-time context.
// Uses atomic write (temp file + rename) to prevent partial reads.
// The workspace directory must already exist.
func WriteAgentManifest(workspacePath string, manifest AgentManifest) error {
	manifestFile := filepath.Join(workspacePath, AgentManifestFilename)

	// Marshal to JSON with indentation for human readability
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	// Add trailing newline for POSIX compliance
	data = append(data, '\n')

	if err := atomicwrite.WriteFile(manifestFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
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

// ProcessIDFilename is the name of the file storing the process ID in the workspace.
const ProcessIDFilename = ".process_id"

// WriteProcessID writes the process ID to the workspace directory.
// This enables explicit process termination during cleanup (orch complete, orch abandon,
// daemon session cleanup).
// Uses atomic write (temp file + rename) to prevent partial reads.
// The workspace directory must already exist.
func WriteProcessID(workspacePath string, pid int) error {
	if pid <= 0 {
		return nil // Nothing to write
	}

	processFile := filepath.Join(workspacePath, ProcessIDFilename)
	content := strconv.Itoa(pid) + "\n"

	if err := atomicwrite.WriteFile(processFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write process ID: %w", err)
	}
	return nil
}

// ReadProcessID reads the process ID from the workspace directory.
// Returns 0 if the file doesn't exist or is invalid.
func ReadProcessID(workspacePath string) int {
	processFile := filepath.Join(workspacePath, ProcessIDFilename)
	data, err := os.ReadFile(processFile)
	if err != nil {
		return 0 // Return 0 if file doesn't exist
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0 // Return 0 if parse fails
	}

	return pid
}

// ProcessIDPath returns the path to the process ID file for a workspace.
func ProcessIDPath(workspacePath string) string {
	return filepath.Join(workspacePath, ProcessIDFilename)
}
