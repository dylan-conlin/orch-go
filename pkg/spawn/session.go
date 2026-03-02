// Package spawn provides spawn configuration and context generation.
package spawn

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/opencode"
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

	// SessionID is the OpenCode session ID (empty for claude backend which has no OpenCode session)
	SessionID string `json:"session_id,omitempty"`

	// VerifyLevel is the verification level declared at spawn time (V0-V3).
	// Determines which gates fire during completion verification.
	// Empty for pre-V0-V3 workspaces (falls back to inference from skill).
	VerifyLevel string `json:"verify_level,omitempty"`

	// ReviewTier is the orchestrator review tier declared at spawn time (auto/scan/review/deep).
	// Determines how thoroughly the orchestrator reviews completion.
	// Empty for pre-review-tier workspaces (falls back to inference from skill).
	ReviewTier string `json:"review_tier,omitempty"`
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
// Tries multiple sources in order:
// 1. AGENT_MANIFEST.json (authoritative — written atomically at spawn time)
// 2. OpenCode session metadata (fallback for pre-manifest workspaces)
// 3. Individual dotfiles (.beads_id, .tier, .spawn_time, .spawn_mode) for backward compatibility
// Always returns a non-nil manifest (fields may be empty if nothing is readable).
//
// AGENT_MANIFEST.json is the primary source because it is written during spawn Phase 1
// from the definitive cfg.BeadsID and cannot be corrupted by session ID discovery races.
// OpenCode session metadata can have incorrect beads_id when FindRecentSessionWithRetry
// in the tmux backend matches the wrong session during concurrent spawns.
func ReadAgentManifestWithFallback(workspacePath string) *AgentManifest {
	// Primary: AGENT_MANIFEST.json (written at spawn time, authoritative source)
	manifest, err := ReadAgentManifest(workspacePath)
	if err == nil {
		return manifest
	}

	// Fallback 1: OpenCode session metadata (for pre-manifest workspaces)
	if manifest := readFromOpenCodeMetadata(workspacePath); manifest != nil {
		return manifest
	}

	// Fallback 2: construct manifest from individual dotfiles
	return readLegacyDotfiles(workspacePath)
}

// readFromOpenCodeMetadata tries to read agent manifest from OpenCode session metadata.
// Returns nil if session_id doesn't exist, OpenCode is unreachable, or session doesn't have metadata.
func readFromOpenCodeMetadata(workspacePath string) *AgentManifest {
	// Read session ID from workspace
	sessionID := ReadSessionID(workspacePath)
	if sessionID == "" {
		return nil // No session ID, can't query OpenCode
	}

	// Try to get session from OpenCode
	client := opencode.NewClient(opencode.DefaultServerURL)
	session, err := client.GetSession(sessionID)
	if err != nil {
		return nil // OpenCode unreachable or session not found
	}

	// If session has no metadata, return nil to fall back to other sources
	if session.Metadata == nil || len(session.Metadata) == 0 {
		return nil
	}

	// Construct manifest from session metadata
	manifest := &AgentManifest{
		WorkspaceName: filepath.Base(workspacePath),
	}

	if beadsID, ok := session.Metadata["beads_id"]; ok {
		manifest.BeadsID = beadsID
	}

	if tier, ok := session.Metadata["tier"]; ok {
		manifest.Tier = tier
	} else {
		manifest.Tier = TierFull // Default to "full" if not specified
	}

	if spawnMode, ok := session.Metadata["spawn_mode"]; ok {
		manifest.SpawnMode = spawnMode
	}

	// Use session creation time as spawn time
	if session.Time.Created > 0 {
		manifest.SpawnTime = time.Unix(session.Time.Created, 0).Format(time.RFC3339)
	}

	return manifest
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

// LookupManifestsByBeadsIDs scans workspace directories and returns manifests
// indexed by beads_id. Used by queryTrackedAgents for batch binding lookup.
// Only returns manifests whose BeadsID matches one of the provided IDs.
func LookupManifestsByBeadsIDs(projectDir string, beadsIDs []string) (map[string]*AgentManifest, error) {
	if len(beadsIDs) == 0 {
		return nil, nil
	}

	// Build lookup set for O(1) matching
	idSet := make(map[string]bool, len(beadsIDs))
	for _, id := range beadsIDs {
		idSet[id] = true
	}

	workspaceRoot := filepath.Join(projectDir, ".orch", "workspace")
	entries, err := os.ReadDir(workspaceRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read workspace directory: %w", err)
	}

	result := make(map[string]*AgentManifest)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// Skip archived directory - only scan active workspaces
		if entry.Name() == "archived" {
			continue
		}
		workspacePath := filepath.Join(workspaceRoot, entry.Name())
		manifest, err := ReadAgentManifest(workspacePath)
		if err != nil {
			continue // Skip unreadable manifests
		}
		if manifest.BeadsID != "" && idSet[manifest.BeadsID] {
			result[manifest.BeadsID] = manifest
		}
	}

	return result, nil
}

// HasLandedArtifacts checks whether an agent workspace has committed git changes
// since its spawn baseline. This detects the "crashed with work" scenario:
// agent committed deliverables but died before reporting Phase: Complete.
//
// Returns true if there are git commits between the manifest's GitBaseline and HEAD.
// Returns false if no manifest, no baseline, or no commits found.
func HasLandedArtifacts(workspacePath, projectDir string) (bool, error) {
	manifest, err := ReadAgentManifest(workspacePath)
	if err != nil {
		return false, fmt.Errorf("no manifest: %w", err)
	}

	if manifest.GitBaseline == "" {
		return false, nil
	}

	// Check for commits between baseline and HEAD
	cmd := exec.Command("git", "log", "--oneline", manifest.GitBaseline+"..HEAD")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("git log failed: %w", err)
	}

	lines := strings.TrimSpace(string(output))
	return lines != "", nil
}
