package episodic

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const defaultConfidenceThreshold = 0.7

var codeFencePattern = regexp.MustCompile("(?s)```.*?```")

// Scope defines where an episode is being reused.
type Scope struct {
	Project    string
	Workspace  string
	SessionID  string
	BeadsID    string
	ProjectDir string
}

// ValidateOptions controls validation behavior.
type ValidateOptions struct {
	Now           time.Time
	Scope         Scope
	AutoInjection bool
	MinConfidence float64
}

// ValidatedEpisode captures gate results for a single episode.
type ValidatedEpisode struct {
	Episode  Episode         `json:"episode"`
	State    ValidationState `json:"state"`
	Reasons  []string        `json:"reasons,omitempty"`
	Summary  string          `json:"summary,omitempty"`
	Rejected bool            `json:"rejected"`
	Degraded bool            `json:"degraded"`
}

// ValidateEpisodesForReuse applies the validation gate to many episodes.
func ValidateEpisodesForReuse(entries []Episode, options ValidateOptions) []ValidatedEpisode {
	results := make([]ValidatedEpisode, 0, len(entries))
	for _, ep := range entries {
		results = append(results, ValidateForReuse(ep, options))
	}

	sort.SliceStable(results, func(i, j int) bool {
		return episodeTime(results[i].Episode).After(episodeTime(results[j].Episode))
	})

	return results
}

// ValidateForReuse enforces the validation-before-reuse gate.
func ValidateForReuse(ep Episode, options ValidateOptions) ValidatedEpisode {
	now := options.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	minConfidence := options.MinConfidence
	if minConfidence <= 0 {
		minConfidence = defaultConfidenceThreshold
	}

	reasons := []string{}
	rejected := false
	degraded := false

	reject := func(reason string) {
		rejected = true
		reasons = append(reasons, reason)
	}

	degrade := func(reason string) {
		if rejected {
			reasons = append(reasons, reason)
			return
		}
		degraded = true
		reasons = append(reasons, reason)
	}

	if ep.ExpiresAt.IsZero() || !now.Before(ep.ExpiresAt) {
		reject("expired")
	}

	if strings.TrimSpace(ep.Evidence.Pointer) == "" {
		reject("missing_evidence")
	}

	item := ep.Evidence
	if !trustedProvenance(item) {
		reject("untrusted_provenance")
	}

	resolved, err := resolvePointer(item.Pointer, options.Scope.ProjectDir)
	if err != nil {
		reject("evidence_unresolvable")
	} else {
		hash, hashErr := fileHash(resolved)
		if hashErr != nil {
			reject("evidence_read_failed")
		} else if !hashMatches(item.Hash, hash) {
			reject("evidence_hash_mismatch")
		}
	}

	if mutableEvidence(ep, item) {
		reason := freshnessReason(item, now)
		if reason != "" {
			degrade(reason)
		}
	}

	if !scopeMatches(ep.Project, options.Scope.Project) {
		reject("scope_project_mismatch")
	}
	if !scopeMatches(ep.Workspace, options.Scope.Workspace) {
		reject("scope_workspace_mismatch")
	}
	if !scopeMatches(ep.SessionID, options.Scope.SessionID) {
		reject("scope_session_mismatch")
	}
	if !scopeMatches(ep.BeadsID, options.Scope.BeadsID) {
		reject("scope_beads_mismatch")
	}

	clean := SanitizeSummary(ep.Outcome.Summary)
	if clean == "" {
		reject("sanitized_summary_empty")
	} else if clean != strings.TrimSpace(ep.Outcome.Summary) {
		degrade("summary_sanitized")
	}

	if options.AutoInjection && ep.Confidence < minConfidence {
		reject("confidence_below_threshold")
	}

	state := ValidationStateAccepted
	if rejected {
		state = ValidationStateRejected
	} else if degraded {
		state = ValidationStateDegraded
	}

	return ValidatedEpisode{
		Episode:  ep,
		State:    state,
		Reasons:  uniqueReasons(reasons),
		Summary:  clean,
		Rejected: rejected,
		Degraded: degraded,
	}
}

// SanitizeSummary strips directive-like and injection-like content.
func SanitizeSummary(summary string) string {
	trimmed := strings.TrimSpace(summary)
	if trimmed == "" {
		return ""
	}

	withoutBlocks := codeFencePattern.ReplaceAllString(trimmed, " ")
	filtered := []string{}
	for _, line := range strings.Split(withoutBlocks, "\n") {
		candidate := strings.TrimSpace(line)
		if candidate == "" {
			continue
		}

		lower := strings.ToLower(candidate)
		if strings.Contains(lower, "ignore previous") ||
			strings.Contains(lower, "ignore all previous") ||
			strings.Contains(lower, "follow these instructions") ||
			strings.Contains(lower, "act as ") ||
			strings.Contains(lower, "you are chatgpt") ||
			strings.Contains(lower, "you are claude") ||
			strings.Contains(lower, "<script") ||
			strings.HasPrefix(lower, "system:") ||
			strings.HasPrefix(lower, "assistant:") ||
			strings.HasPrefix(lower, "developer:") ||
			strings.HasPrefix(lower, "user:") {
			continue
		}

		filtered = append(filtered, candidate)
	}

	return strings.Join(strings.Fields(strings.Join(filtered, " ")), " ")
}

func trustedProvenance(e Evidence) bool {
	base := pointerBase(e.Pointer)
	kind := strings.ToLower(strings.TrimSpace(e.Kind))

	allowedKind := kind == "" ||
		kind == "events_jsonl" ||
		kind == "activity_json" ||
		kind == "events.jsonl" ||
		kind == "activity.json"

	allowedBase := base == "events.jsonl" || base == "activity.json"

	return allowedKind && allowedBase
}

func resolvePointer(pointer, projectDir string) (string, error) {
	path := strings.TrimSpace(pointer)
	if path == "" {
		return "", fmt.Errorf("empty pointer")
	}

	if idx := strings.Index(path, "#"); idx >= 0 {
		path = path[:idx]
	}

	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[2:])
	}

	if filepath.IsAbs(path) {
		return path, nil
	}

	if strings.TrimSpace(projectDir) == "" {
		abs, err := filepath.Abs(path)
		if err != nil {
			return "", err
		}
		return abs, nil
	}

	return filepath.Join(projectDir, path), nil
}

func fileHash(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:]), nil
}

func hashMatches(expected, actual string) bool {
	normalized := strings.ToLower(strings.TrimSpace(expected))
	normalized = strings.TrimPrefix(normalized, "sha256:")
	if normalized == "" {
		return false
	}
	return normalized == strings.ToLower(strings.TrimSpace(actual))
}

func mutableEvidence(ep Episode, e Evidence) bool {
	if ep.Mutable || e.Mutable {
		return true
	}

	kind := strings.ToLower(strings.TrimSpace(e.Kind))
	return kind == "state_db_projection" ||
		kind == "state_db" ||
		kind == "beads_issue" ||
		kind == "gate_state" ||
		kind == "verification_gate"
}

func freshnessReason(e Evidence, now time.Time) string {
	if e.Timestamp == 0 {
		return "mutable_state_missing_timestamp"
	}

	ts := time.Unix(e.Timestamp, 0)

	if now.Sub(ts) > 30*time.Minute {
		return "mutable_state_stale"
	}

	return ""
}

func scopeMatches(value, expected string) bool {
	trimmedExpected := strings.TrimSpace(expected)
	if trimmedExpected == "" {
		return true
	}

	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return false
	}

	if strings.EqualFold(trimmedValue, trimmedExpected) {
		return true
	}

	return strings.EqualFold(filepath.Base(trimmedValue), filepath.Base(trimmedExpected))
}

func pointerBase(pointer string) string {
	path := strings.TrimSpace(pointer)
	if idx := strings.Index(path, "#"); idx >= 0 {
		path = path[:idx]
	}
	path = strings.TrimPrefix(path, "~/")
	return strings.ToLower(filepath.Base(path))
}

func uniqueReasons(reasons []string) []string {
	if len(reasons) == 0 {
		return nil
	}

	seen := map[string]bool{}
	out := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		if seen[reason] {
			continue
		}
		seen[reason] = true
		out = append(out, reason)
	}

	return out
}

func episodeTime(ep Episode) time.Time {
	if !ep.CreatedAt.IsZero() {
		return ep.CreatedAt
	}

	if ep.Evidence.Timestamp > 0 {
		return time.Unix(ep.Evidence.Timestamp, 0)
	}

	return time.Time{}
}
