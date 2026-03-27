package orient

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanRecentBriefs_Empty(t *testing.T) {
	dir := t.TempDir()
	briefs, unread := ScanRecentBriefs(dir, nil, 5)
	if len(briefs) != 0 {
		t.Errorf("expected 0 briefs, got %d", len(briefs))
	}
	if unread != 0 {
		t.Errorf("expected 0 unread, got %d", unread)
	}
}

func TestScanRecentBriefs_ParsesBriefs(t *testing.T) {
	dir := t.TempDir()

	// Write two brief files
	brief1 := `# Brief: orch-go-abc12

## Frame

Product naming investigation for v1 launch.

## Resolution

Kenning is the strongest candidate.

## Tension

Name obscurity risk needs user testing.
`
	brief2 := `# Brief: orch-go-def34

## Frame

Orient surface has too much operational noise.

## Resolution

Split to five elements.
`
	os.WriteFile(filepath.Join(dir, "orch-go-abc12.md"), []byte(brief1), 0644)
	os.WriteFile(filepath.Join(dir, "orch-go-def34.md"), []byte(brief2), 0644)

	briefs, unread := ScanRecentBriefs(dir, nil, 5)
	if len(briefs) != 2 {
		t.Fatalf("expected 2 briefs, got %d", len(briefs))
	}
	if unread != 2 {
		t.Errorf("expected 2 unread (no read state), got %d", unread)
	}

	// Both should be unread (no read state provided)
	for _, b := range briefs {
		if !b.IsUnread {
			t.Errorf("brief %s should be unread", b.BeadsID)
		}
	}
}

func TestScanRecentBriefs_RespectsReadState(t *testing.T) {
	dir := t.TempDir()

	brief1 := `# Brief: orch-go-abc12

## Frame

Some investigation.

## Resolution

Some finding.
`
	os.WriteFile(filepath.Join(dir, "orch-go-abc12.md"), []byte(brief1), 0644)

	readState := map[string]bool{
		"orch-go-abc12": true,
	}

	briefs, unread := ScanRecentBriefs(dir, readState, 5)
	if len(briefs) != 1 {
		t.Fatalf("expected 1 brief, got %d", len(briefs))
	}
	if unread != 0 {
		t.Errorf("expected 0 unread, got %d", unread)
	}
	if briefs[0].IsUnread {
		t.Error("brief should be marked as read")
	}
}

func TestScanRecentBriefs_DetectsTension(t *testing.T) {
	dir := t.TempDir()

	briefWithTension := `# Brief: orch-go-tens1

## Frame

Some investigation.

## Resolution

Some finding.

## Tension

Open question remains.
`
	briefWithoutTension := `# Brief: orch-go-nots1

## Frame

Some investigation.

## Resolution

Some finding.
`
	os.WriteFile(filepath.Join(dir, "orch-go-tens1.md"), []byte(briefWithTension), 0644)
	os.WriteFile(filepath.Join(dir, "orch-go-nots1.md"), []byte(briefWithoutTension), 0644)

	briefs, _ := ScanRecentBriefs(dir, nil, 5)

	tensionCount := 0
	for _, b := range briefs {
		if b.HasTension {
			tensionCount++
		}
	}
	if tensionCount != 1 {
		t.Errorf("expected 1 brief with tension, got %d", tensionCount)
	}
}

func TestScanRecentBriefs_LimitsCount(t *testing.T) {
	dir := t.TempDir()

	for i := 0; i < 10; i++ {
		content := "# Brief: orch-go-" + string(rune('a'+i)) + "bc\n\n## Frame\n\nTest.\n\n## Resolution\n\nDone.\n"
		os.WriteFile(filepath.Join(dir, "orch-go-"+string(rune('a'+i))+"bc.md"), []byte(content), 0644)
	}

	briefs, _ := ScanRecentBriefs(dir, nil, 3)
	if len(briefs) != 3 {
		t.Errorf("expected 3 briefs (limited), got %d", len(briefs))
	}
}

func TestScanRecentBriefs_ExtractsTitle(t *testing.T) {
	dir := t.TempDir()

	brief := `# Brief: orch-go-titl1

## Frame

Product naming investigation for v1 launch. This explores candidate names across registries.

## Resolution

Kenning selected.
`
	os.WriteFile(filepath.Join(dir, "orch-go-titl1.md"), []byte(brief), 0644)

	briefs, _ := ScanRecentBriefs(dir, nil, 5)
	if len(briefs) != 1 {
		t.Fatalf("expected 1 brief, got %d", len(briefs))
	}
	// Title should be first sentence of Frame
	if briefs[0].Title == "" {
		t.Error("expected non-empty title")
	}
}

func TestFormatRecentBriefs_Empty(t *testing.T) {
	result := FormatRecentBriefs(nil, 0)
	if result != "" {
		t.Errorf("expected empty string for no briefs, got %q", result)
	}
}

func TestFormatRecentBriefs_Renders(t *testing.T) {
	briefs := []RecentBrief{
		{BeadsID: "orch-go-abc12", Title: "Product naming investigation", HasTension: true, IsUnread: true},
		{BeadsID: "orch-go-def34", Title: "Orient surface redesign", HasTension: false, IsUnread: false},
	}

	result := FormatRecentBriefs(briefs, 3)
	if result == "" {
		t.Fatal("expected non-empty output")
	}
	if !contains(result, "Recent briefs") {
		t.Error("expected 'Recent briefs' header")
	}
	if !contains(result, "3 unread") {
		t.Error("expected unread count")
	}
	if !contains(result, "Product naming investigation") {
		t.Error("expected brief title")
	}
}

// contains is defined in debrief_test.go
