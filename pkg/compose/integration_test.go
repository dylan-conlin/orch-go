package compose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dylan-conlin/orch-go/pkg/orient"
)

// Integration tests verifying between-session composition claims (CC-1 through CC-4).
// These tests exercise the cross-package composition pipeline end-to-end.

// TestCC1_MaintenanceBriefsComposeIntoDigests verifies CC-1:
// Briefs with maintenance-like content (bug fixes, test fixes, infra) compose
// into digests alongside knowledge briefs. The compose pipeline does not filter
// by category — all briefs participate in clustering.
func TestCC1_MaintenanceBriefsComposeIntoDigests(t *testing.T) {
	dir := t.TempDir()
	briefsDir := filepath.Join(dir, "briefs")
	threadsDir := filepath.Join(dir, "threads")
	digestsDir := filepath.Join(dir, "digests")
	os.MkdirAll(briefsDir, 0755)
	os.MkdirAll(threadsDir, 0755)

	// Create maintenance-like briefs (bug fixes, infra work)
	maintenanceBriefs := []struct {
		id, frame, resolution, tension string
	}{
		{
			id:         "orch-go-maint1",
			frame:      "The daemon comprehension throttle was bypassed when WriteVerificationSignal reset the counter from non-human completion paths.",
			resolution: "Fixed the daemon comprehension counter to only reset on human-triggered verification, not automated headless completion paths.",
			tension:    "Other non-human paths may exist that similarly bypass the comprehension throttle counter.",
		},
		{
			id:         "orch-go-maint2",
			frame:      "The daemon auto-completion pipeline had a race condition where comprehension labels were stripped before the verification signal was written.",
			resolution: "Reordered the daemon completion pipeline to write the verification signal before stripping comprehension labels, ensuring the counter captures all completions.",
			tension:    "The ordering dependency between verification and comprehension is implicit — should it be enforced structurally?",
		},
		{
			id:         "orch-go-maint3",
			frame:      "The daemon periodic task scheduler crashed when comprehension queue count returned an error from the beads CLI.",
			resolution: "Added error handling to the daemon scheduler comprehension check, defaulting to zero count on error to prevent crash cascades.",
			tension:    "Defaulting to zero on error means the daemon could over-spawn during beads outages.",
		},
	}

	// Create knowledge briefs (investigations, architecture)
	knowledgeBriefs := []struct {
		id, frame, resolution, tension string
	}{
		{
			id:         "orch-go-know1",
			frame:      "The orchestrator session lifecycle has no mechanism to detect Dylan's absence, leading to comprehension queue buildup.",
			resolution: "Designed between-session composition: accumulation threshold triggers clustering, maintenance bypass reduces queue pressure.",
			tension:    "The accumulation threshold may be too low (noisy) or too high (delayed) — needs empirical calibration.",
		},
	}

	for _, b := range maintenanceBriefs {
		content := "# Brief: " + b.id + "\n\n## Frame\n\n" + b.frame + "\n\n## Resolution\n\n" + b.resolution + "\n\n## Tension\n\n" + b.tension + "\n"
		os.WriteFile(filepath.Join(briefsDir, b.id+".md"), []byte(content), 0644)
	}
	for _, b := range knowledgeBriefs {
		content := "# Brief: " + b.id + "\n\n## Frame\n\n" + b.frame + "\n\n## Resolution\n\n" + b.resolution + "\n\n## Tension\n\n" + b.tension + "\n"
		os.WriteFile(filepath.Join(briefsDir, b.id+".md"), []byte(content), 0644)
	}

	// Run composition
	digest, err := Compose(briefsDir, threadsDir)
	if err != nil {
		t.Fatalf("Compose failed: %v", err)
	}

	// CC-1 verification: ALL briefs participate (maintenance + knowledge)
	if digest.BriefsComposed != 4 {
		t.Errorf("CC-1: BriefsComposed = %d, want 4 (3 maintenance + 1 knowledge)", digest.BriefsComposed)
	}

	// Verify maintenance briefs appear in clusters (they share daemon/comprehension keywords)
	allClusteredIDs := make(map[string]bool)
	for _, dc := range digest.Clusters {
		for _, b := range dc.Briefs {
			allClusteredIDs[b.ID] = true
		}
	}

	maintenanceInClusters := 0
	for _, mb := range maintenanceBriefs {
		if allClusteredIDs[mb.id] {
			maintenanceInClusters++
		}
	}

	// At minimum, the 3 maintenance briefs should cluster together (they share daemon/comprehension vocabulary)
	if maintenanceInClusters < 2 {
		t.Errorf("CC-1: only %d/3 maintenance briefs in clusters — maintenance work not composing into digest", maintenanceInClusters)
	}

	// Write digest and verify maintenance briefs appear
	path, err := WriteDigest(digest, digestsDir)
	if err != nil {
		t.Fatalf("WriteDigest failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Reading digest: %v", err)
	}
	content := string(data)

	// At least one maintenance brief ID should appear in the digest
	foundMaintenance := false
	for _, mb := range maintenanceBriefs {
		if strings.Contains(content, mb.id) {
			foundMaintenance = true
			break
		}
	}
	if !foundMaintenance {
		t.Error("CC-1: no maintenance brief IDs found in written digest")
	}
}

// TestCC2_MaintenanceBypassReducesQueuePressure verifies CC-2:
// When maintenance classification is wired (Phase 2a), briefs with
// `category: maintenance` in frontmatter should be detectable by the orient
// subsystem via CountMaintenanceBriefs. This test verifies the detection
// mechanism works — the actual comprehension queue bypass is in the daemon
// completion pipeline (orch-go-ke9h0, not yet implemented).
func TestCC2_MaintenanceBypassDetection(t *testing.T) {
	dir := t.TempDir()

	// Create briefs WITH category frontmatter (as Phase 2a will produce)
	maintenanceBrief := `---
beads_id: orch-go-maint1
category: maintenance
---

# Brief: orch-go-maint1

## Frame

Fixed a bug in the daemon throttle.

## Resolution

Patched the counter reset path.

## Tension

Other paths may exist.
`
	knowledgeBrief := `---
beads_id: orch-go-know1
category: knowledge
---

# Brief: orch-go-know1

## Frame

Designed between-session composition.

## Resolution

Accumulation threshold triggers clustering.

## Tension

Threshold needs calibration.
`
	uncategorizedBrief := `---
beads_id: orch-go-old1
---

# Brief: orch-go-old1

## Frame

An older brief without category metadata.

## Resolution

Legacy content.

## Tension

Should old briefs get retroactively classified?
`

	os.WriteFile(filepath.Join(dir, "orch-go-maint1.md"), []byte(maintenanceBrief), 0644)
	os.WriteFile(filepath.Join(dir, "orch-go-know1.md"), []byte(knowledgeBrief), 0644)
	os.WriteFile(filepath.Join(dir, "orch-go-old1.md"), []byte(uncategorizedBrief), 0644)

	// CC-2 verification: CountMaintenanceBriefs detects category: maintenance
	count := orient.CountMaintenanceBriefs(dir, time.Time{})
	if count != 1 {
		t.Errorf("CC-2: CountMaintenanceBriefs = %d, want 1 (only the maintenance-categorized brief)", count)
	}

	// Verify knowledge briefs are NOT counted as maintenance
	// (uncategorized briefs should also not count)
}

// TestCC3_DaemonComposePrerequisites verifies CC-3 prerequisites:
// The compose pipeline can run without human interaction — it reads files,
// clusters, and writes a digest. This is the core behavior the daemon periodic
// task will invoke. The daemon wiring itself (orch-go-884rx) is not yet
// implemented, but the underlying Compose() function is fully autonomous.
func TestCC3_ComposeRunsWithoutHumanTrigger(t *testing.T) {
	dir := t.TempDir()
	briefsDir := filepath.Join(dir, "briefs")
	threadsDir := filepath.Join(dir, "threads")
	digestsDir := filepath.Join(dir, "digests")
	os.MkdirAll(briefsDir, 0755)
	os.MkdirAll(threadsDir, 0755)

	// Simulate a batch of agent completions (what the daemon sees between sessions)
	topics := []struct {
		id, domain, frame, resolution string
	}{
		// Group A: Authentication work (3 briefs)
		{"orch-go-auth1", "auth", "OAuth token expiry during agent orchestration sessions caused authentication middleware failures.", "Added pre-expiry token refresh to authentication middleware with configurable grace period for OAuth lifecycle management."},
		{"orch-go-auth2", "auth", "Concurrent agent sessions caused OAuth token rotation races in the authentication layer.", "Implemented mutex-based token refresh in authentication middleware to serialize OAuth token rotation across agent sessions."},
		{"orch-go-auth3", "auth", "Stale authentication cache entries caused OAuth token validation failures in the middleware.", "Added cache invalidation to authentication middleware with proper OAuth token lifecycle TTL management."},
		// Group B: Daemon work (3 briefs)
		{"orch-go-dmn1", "daemon", "The daemon scheduler periodic tasks ran in an unpredictable order causing spawn throttle miscalculation.", "Sorted daemon scheduler periodic tasks by priority, ensuring comprehension throttle check runs before spawn evaluation."},
		{"orch-go-dmn2", "daemon", "Daemon periodic task scheduler crashed on comprehension queue timeout during beads CLI latency spikes.", "Added daemon scheduler timeout handling for comprehension queue queries, preventing cascade failures in periodic task execution."},
		{"orch-go-dmn3", "daemon", "The daemon comprehension threshold was hardcoded, preventing tuning of spawn throttle sensitivity.", "Extracted daemon comprehension threshold to scheduler configuration with sensible defaults and per-project override."},
		// Group C: Unrelated brief
		{"orch-go-ui1", "ui", "The dashboard rendered slowly when displaying large agent log output.", "Implemented virtual scrolling for dashboard agent log display, reducing DOM node count from O(n) to O(visible)."},
	}

	for _, b := range topics {
		content := "# Brief: " + b.id + "\n\n## Frame\n\n" + b.frame + "\n\n## Resolution\n\n" + b.resolution + "\n\n## Tension\n\nOpen question about " + b.domain + " patterns.\n"
		os.WriteFile(filepath.Join(briefsDir, b.id+".md"), []byte(content), 0644)
	}

	// CC-3 verification: Compose runs completely autonomously (no human input)
	digest, err := Compose(briefsDir, threadsDir)
	if err != nil {
		t.Fatalf("CC-3: Compose failed (should run without human trigger): %v", err)
	}

	if digest.BriefsComposed != 7 {
		t.Errorf("CC-3: BriefsComposed = %d, want 7", digest.BriefsComposed)
	}

	if digest.ClustersFound < 2 {
		t.Errorf("CC-3: ClustersFound = %d, want >= 2 (auth + daemon groups)", digest.ClustersFound)
	}

	// Verify digest can be written (daemon would do this)
	path, err := WriteDigest(digest, digestsDir)
	if err != nil {
		t.Fatalf("CC-3: WriteDigest failed: %v", err)
	}

	// Verify the file exists and has valid content
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("CC-3: Reading digest: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "epistemic_status: unverified-clustering") {
		t.Error("CC-3: Digest missing epistemic status — safety label required")
	}
	if !strings.Contains(content, "## Epistemic Status") {
		t.Error("CC-3: Digest missing Epistemic Status section")
	}
}

// TestCC4_OrientSurfacesDigestOnReturn verifies CC-4:
// Orient detects digests newer than the previous session and renders
// a summary line in the thinking surface.
func TestCC4_OrientSurfacesDigestOnReturn(t *testing.T) {
	dir := t.TempDir()
	briefsDir := filepath.Join(dir, "briefs")
	threadsDir := filepath.Join(dir, "threads")
	digestsDir := filepath.Join(dir, "digests")
	os.MkdirAll(briefsDir, 0755)
	os.MkdirAll(threadsDir, 0755)

	// Step 1: Create briefs and compose a digest (simulating daemon work)
	for i := 0; i < 5; i++ {
		domain := "authentication"
		if i >= 3 {
			domain = "deployment"
		}
		content := "# Brief: orch-go-ret" + string(rune('a'+i)) + "\n\n" +
			"## Frame\n\nThe " + domain + " system had issues with token management and agent lifecycle.\n\n" +
			"## Resolution\n\nFixed " + domain + " token handling in the middleware.\n\n" +
			"## Tension\n\nShould " + domain + " tokens be centrally managed?\n"
		os.WriteFile(filepath.Join(briefsDir, "orch-go-ret"+string(rune('a'+i))+".md"), []byte(content), 0644)
	}

	digest, err := Compose(briefsDir, threadsDir)
	if err != nil {
		t.Fatalf("CC-4: Compose failed: %v", err)
	}

	_, err = WriteDigest(digest, digestsDir)
	if err != nil {
		t.Fatalf("CC-4: WriteDigest failed: %v", err)
	}

	// Step 2: Simulate Dylan returning — orient scans for digests newer than previous session
	prevSessionDate, _ := time.Parse("2006-01-02", "2026-03-25")
	summary := orient.ScanRecentDigests(digestsDir, prevSessionDate)

	// CC-4 verification: orient detects the digest
	if summary == nil {
		t.Fatal("CC-4: orient.ScanRecentDigests returned nil — digest not surfaced on return")
	}
	if summary.BriefsComposed != digest.BriefsComposed {
		t.Errorf("CC-4: orient shows %d briefs, digest has %d", summary.BriefsComposed, digest.BriefsComposed)
	}
	if summary.ClustersFound != digest.ClustersFound {
		t.Errorf("CC-4: orient shows %d clusters, digest has %d", summary.ClustersFound, digest.ClustersFound)
	}

	// Step 3: Verify the formatted output for the thinking surface
	formatted := orient.FormatDigestSummary(summary)
	if formatted == "" {
		t.Fatal("CC-4: FormatDigestSummary returned empty — digest not rendered")
	}
	if !strings.Contains(formatted, "Between sessions:") {
		t.Error("CC-4: formatted output missing 'Between sessions:' prefix")
	}
	if !strings.Contains(formatted, "briefs") {
		t.Error("CC-4: formatted output missing brief count")
	}
	if !strings.Contains(formatted, "themes") {
		t.Error("CC-4: formatted output missing cluster/theme count")
	}

	// Step 4: Verify older digests are NOT surfaced
	futureDate, _ := time.Parse("2006-01-02", "2026-03-28")
	futureSummary := orient.ScanRecentDigests(digestsDir, futureDate)
	if futureSummary != nil {
		t.Error("CC-4: orient surfaced digest from before the session — should only show newer digests")
	}
}

// TestCC4_OrientMaintenanceCountRendering verifies the maintenance count
// appears in the orient output when maintenance-categorized briefs exist.
func TestCC4_OrientMaintenanceCountRendering(t *testing.T) {
	summary := &orient.DigestSummary{
		DigestCount:      1,
		BriefsComposed:   40,
		ClustersFound:    5,
		MaintenanceCount: 15,
	}

	formatted := orient.FormatDigestSummary(summary)

	if !strings.Contains(formatted, "15 maintenance") {
		t.Errorf("CC-4: maintenance count not rendered, got: %q", formatted)
	}
	if !strings.Contains(formatted, "40 briefs") {
		t.Errorf("CC-4: brief count not rendered, got: %q", formatted)
	}
}

// TestCompositionClaimsSafetyInvariants verifies cross-cutting safety properties
// from the design doc that must hold across all claims.
func TestCompositionClaimsSafetyInvariants(t *testing.T) {
	dir := t.TempDir()
	briefsDir := filepath.Join(dir, "briefs")
	threadsDir := filepath.Join(dir, "threads")
	digestsDir := filepath.Join(dir, "digests")
	os.MkdirAll(briefsDir, 0755)
	os.MkdirAll(threadsDir, 0755)

	// Create a thread to verify composition doesn't modify it
	threadContent := `---
title: "Test thread — should not be modified"
status: forming
---

# Test thread — should not be modified

Original thread content that must not change.
`
	threadPath := filepath.Join(threadsDir, "2026-03-27-test-thread.md")
	os.WriteFile(threadPath, []byte(threadContent), 0644)

	// Create briefs
	for i := 0; i < 4; i++ {
		content := "# Brief: orch-go-safe" + string(rune('a'+i)) + "\n\n" +
			"## Frame\n\nThe testing system verification framework had issues.\n\n" +
			"## Resolution\n\nFixed testing verification in the framework.\n\n" +
			"## Tension\n\nShould testing verification be automated?\n"
		os.WriteFile(filepath.Join(briefsDir, "orch-go-safe"+string(rune('a'+i))+".md"), []byte(content), 0644)
	}

	digest, err := Compose(briefsDir, threadsDir)
	if err != nil {
		t.Fatalf("Compose failed: %v", err)
	}

	_, err = WriteDigest(digest, digestsDir)
	if err != nil {
		t.Fatalf("WriteDigest failed: %v", err)
	}

	// Safety invariant 1: Thread files are NOT modified by composition
	threadData, err := os.ReadFile(threadPath)
	if err != nil {
		t.Fatalf("Reading thread: %v", err)
	}
	if string(threadData) != threadContent {
		t.Error("SAFETY: Composition modified a thread file — must only propose, never edit")
	}

	// Safety invariant 2: Digest includes epistemic status label
	digestEntries, _ := os.ReadDir(digestsDir)
	for _, e := range digestEntries {
		data, _ := os.ReadFile(filepath.Join(digestsDir, e.Name()))
		if !strings.Contains(string(data), "NOT been verified by a human") {
			t.Errorf("SAFETY: Digest %s missing epistemic status warning", e.Name())
		}
	}

	// Safety invariant 3: Brief files are NOT modified by composition
	briefEntries, _ := os.ReadDir(briefsDir)
	for _, e := range briefEntries {
		data, _ := os.ReadFile(filepath.Join(briefsDir, e.Name()))
		content := string(data)
		if strings.Contains(content, "composed") || strings.Contains(content, "clustered") {
			t.Errorf("SAFETY: Brief %s was modified by composition — briefs must be immutable", e.Name())
		}
	}

	// Safety invariant 4: Every cluster traces to specific brief IDs (provenance)
	for i, dc := range digest.Clusters {
		if len(dc.Briefs) == 0 {
			t.Errorf("SAFETY: Cluster %d has no brief provenance", i)
		}
		for _, b := range dc.Briefs {
			if b.ID == "" {
				t.Errorf("SAFETY: Cluster %d contains brief with empty ID — provenance broken", i)
			}
		}
	}
}
