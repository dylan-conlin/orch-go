package artifactsync

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClassifyChanges_NewCommand(t *testing.T) {
	files := []string{"cmd/orch/sync_cmd.go"}
	diffs := map[string]string{
		"cmd/orch/sync_cmd.go": `+func init() {
+	rootCmd.AddCommand(syncCmd)
+}
+var syncCmd = &cobra.Command{`,
	}
	scopes := ClassifyChanges(files, diffs)
	assertContains(t, scopes, "new-command")
}

func TestClassifyChanges_NewFlag(t *testing.T) {
	files := []string{"cmd/orch/spawn_cmd.go"}
	diffs := map[string]string{
		"cmd/orch/spawn_cmd.go": `+	spawnCmd.Flags().String("explore", "", "exploration mode")`,
	}
	scopes := ClassifyChanges(files, diffs)
	assertContains(t, scopes, "new-flag")
}

func TestClassifyChanges_NewEvent(t *testing.T) {
	files := []string{"pkg/events/logger.go"}
	diffs := map[string]string{
		"pkg/events/logger.go": `+	EventTypeArtifactDrift = "artifact.drift"`,
	}
	scopes := ClassifyChanges(files, diffs)
	assertContains(t, scopes, "new-event")
}

func TestClassifyChanges_NewSkill(t *testing.T) {
	files := []string{"skills/src/worker/artifact-sync/main.go"}
	diffs := map[string]string{
		"skills/src/worker/artifact-sync/main.go": "+new skill content",
	}
	scopes := ClassifyChanges(files, diffs)
	assertContains(t, scopes, "new-skill")
}

func TestClassifyChanges_NewPackage(t *testing.T) {
	files := []string{"pkg/artifactsync/artifactsync.go"}
	diffs := map[string]string{
		"pkg/artifactsync/artifactsync.go": "+package artifactsync",
	}
	scopes := ClassifyChanges(files, diffs)
	assertContains(t, scopes, "new-package")
}

func TestClassifyChanges_APIChange(t *testing.T) {
	files := []string{"cmd/orch/serve_handlers.go"}
	diffs := map[string]string{
		"cmd/orch/serve_handlers.go": `+func handleArtifactSync(w http.ResponseWriter, r *http.Request) {`,
	}
	scopes := ClassifyChanges(files, diffs)
	assertContains(t, scopes, "api-change")
}

func TestClassifyChanges_ConfigChange(t *testing.T) {
	files := []string{"pkg/spawn/config.go"}
	diffs := map[string]string{
		"pkg/spawn/config.go": `+	ArtifactSync bool   ` + "`yaml:\"artifact_sync\"`",
	}
	scopes := ClassifyChanges(files, diffs)
	assertContains(t, scopes, "config-change")
}

func TestClassifyChanges_NoMatch(t *testing.T) {
	files := []string{"pkg/verify/check.go"}
	diffs := map[string]string{
		"pkg/verify/check.go": `+	// fixed a bug`,
	}
	scopes := ClassifyChanges(files, diffs)
	if len(scopes) != 0 {
		t.Errorf("expected no scopes for bug fix, got %v", scopes)
	}
}

func TestClassifyChanges_MultipleScopes(t *testing.T) {
	files := []string{
		"cmd/orch/sync_cmd.go",
		"pkg/events/logger.go",
	}
	diffs := map[string]string{
		"cmd/orch/sync_cmd.go": `+	rootCmd.AddCommand(syncCmd)
+	syncCmd.Flags().Bool("dry-run", false, "")`,
		"pkg/events/logger.go": `+	EventTypeSyncRun = "sync.run"`,
	}
	scopes := ClassifyChanges(files, diffs)
	assertContains(t, scopes, "new-command")
	assertContains(t, scopes, "new-flag")
	assertContains(t, scopes, "new-event")
}

func TestClassifyChanges_Deduplication(t *testing.T) {
	files := []string{
		"cmd/orch/spawn_cmd.go",
		"cmd/orch/complete_cmd.go",
	}
	diffs := map[string]string{
		"cmd/orch/spawn_cmd.go":   `+	spawnCmd.Flags().String("a", "", "")`,
		"cmd/orch/complete_cmd.go": `+	completeCmd.Flags().String("b", "", "")`,
	}
	scopes := ClassifyChanges(files, diffs)
	count := 0
	for _, s := range scopes {
		if s == "new-flag" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected new-flag once, got %d times in %v", count, scopes)
	}
}

func TestLogDriftEvent(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "artifact-drift.jsonl")

	event := DriftEvent{
		BeadsID:      "orch-go-abc12",
		Skill:        "feature-impl",
		ChangeScopes: []string{"new-command", "new-flag"},
		FilesChanged: []string{"cmd/orch/sync_cmd.go"},
		CommitRange:  "abc123..def456",
	}

	err := LogDriftEvent(logPath, event)
	if err != nil {
		t.Fatalf("LogDriftEvent failed: %v", err)
	}

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log: %v", err)
	}

	s := string(content)
	if !contains(s, "orch-go-abc12") {
		t.Error("expected beads_id in output")
	}
	if !contains(s, "new-command") {
		t.Error("expected new-command scope in output")
	}
	if !contains(s, "artifact.drift") {
		t.Error("expected artifact.drift event type")
	}
}

func TestLogDriftEvent_AppendsMultiple(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "artifact-drift.jsonl")

	for i := 0; i < 3; i++ {
		err := LogDriftEvent(logPath, DriftEvent{
			BeadsID:      "test",
			ChangeScopes: []string{"new-command"},
		})
		if err != nil {
			t.Fatalf("LogDriftEvent %d failed: %v", i, err)
		}
	}

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}

	lines := 0
	for _, line := range splitLines(string(content)) {
		if line != "" {
			lines++
		}
	}
	if lines != 3 {
		t.Errorf("expected 3 lines, got %d", lines)
	}
}

func assertContains(t *testing.T, scopes []string, expected string) {
	t.Helper()
	for _, s := range scopes {
		if s == expected {
			return
		}
	}
	t.Errorf("expected scopes %v to contain %q", scopes, expected)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

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
