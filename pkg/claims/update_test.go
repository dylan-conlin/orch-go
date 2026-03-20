package claims

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExtractClaimRef(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantID  string
	}{
		{"frontmatter claim", "---\nclaim: AE-08\nmodel: arch\n---\n", "AE-08"},
		{"body claim", "# Probe\n\nclaim: MH-05\n\nSome findings.", "MH-05"},
		{"case insensitive", "Claim: SCT-03\n", "SCT-03"},
		{"no claim", "# Just a probe\n\nNo claim reference.", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref := ExtractClaimRef(tt.content)
			if tt.wantID == "" {
				if ref != nil {
					t.Errorf("expected nil, got %+v", ref)
				}
				return
			}
			if ref == nil {
				t.Fatal("expected non-nil ref")
			}
			if ref.ClaimID != tt.wantID {
				t.Errorf("ClaimID = %q, want %q", ref.ClaimID, tt.wantID)
			}
		})
	}
}

func TestApplyProbeVerdict_Confirms(t *testing.T) {
	dir := t.TempDir()
	modelDir := filepath.Join(dir, "test-model")
	os.MkdirAll(modelDir, 0755)

	initial := &File{
		Model:   "test-model",
		Version: 1,
		Claims: []Claim{
			{
				ID:         "TM-01",
				Text:       "Test claim",
				Confidence: Unconfirmed,
				Priority:   PriorityCore,
			},
		},
	}
	SaveFile(filepath.Join(modelDir, "claims.yaml"), initial)

	now := time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC)
	ref := ProbeClaimRef{
		ClaimID:   "TM-01",
		ModelName: "test-model",
		Verdict:   "confirms",
		Source:    "Probe test",
	}

	result, err := ApplyProbeVerdict(dir, ref, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != "confirmed" {
		t.Errorf("Action = %q, want %q", result.Action, "confirmed")
	}

	// Verify file was updated
	updated, _ := LoadFile(filepath.Join(modelDir, "claims.yaml"))
	c := updated.Claims[0]
	if c.Confidence != Confirmed {
		t.Errorf("Confidence = %q, want %q", c.Confidence, Confirmed)
	}
	if c.LastValidated != "2026-03-19" {
		t.Errorf("LastValidated = %q, want %q", c.LastValidated, "2026-03-19")
	}
	if len(c.Evidence) != 1 {
		t.Fatalf("Evidence count = %d, want 1", len(c.Evidence))
	}
}

func TestApplyProbeVerdict_Contradicts(t *testing.T) {
	dir := t.TempDir()
	modelDir := filepath.Join(dir, "test-model")
	os.MkdirAll(modelDir, 0755)

	initial := &File{
		Model:   "test-model",
		Version: 1,
		Claims: []Claim{
			{
				ID:         "TM-01",
				Text:       "Test claim",
				Confidence: Confirmed,
				Priority:   PriorityCore,
			},
		},
	}
	SaveFile(filepath.Join(modelDir, "claims.yaml"), initial)

	now := time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC)
	ref := ProbeClaimRef{
		ClaimID:   "TM-01",
		ModelName: "test-model",
		Verdict:   "contradicts",
		Source:    "Contradicting probe",
	}

	result, err := ApplyProbeVerdict(dir, ref, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != "contested" {
		t.Errorf("Action = %q, want %q", result.Action, "contested")
	}

	updated, _ := LoadFile(filepath.Join(modelDir, "claims.yaml"))
	if updated.Claims[0].Confidence != Contested {
		t.Errorf("Confidence = %q, want %q", updated.Claims[0].Confidence, Contested)
	}
}

func TestApplyProbeVerdict_NotFound(t *testing.T) {
	dir := t.TempDir()
	modelDir := filepath.Join(dir, "test-model")
	os.MkdirAll(modelDir, 0755)

	initial := &File{
		Model:   "test-model",
		Version: 1,
		Claims: []Claim{
			{ID: "TM-01", Text: "Test", Confidence: Confirmed, Priority: PriorityCore},
		},
	}
	SaveFile(filepath.Join(modelDir, "claims.yaml"), initial)

	now := time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC)
	ref := ProbeClaimRef{
		ClaimID:   "TM-99", // doesn't exist
		ModelName: "test-model",
		Verdict:   "confirms",
	}

	result, err := ApplyProbeVerdict(dir, ref, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Action != "not_found" {
		t.Errorf("Action = %q, want %q", result.Action, "not_found")
	}
}
