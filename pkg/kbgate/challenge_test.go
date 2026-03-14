package kbgate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSeverityCodes_AllValid(t *testing.T) {
	// Fixed severity codes from design doc
	expected := []string{
		"ENDOGENOUS_EVIDENCE",
		"VOCABULARY_INFLATION",
		"EXTERNAL_NOVELTY_DELTA",
		"PUBLICATION_LANGUAGE",
	}
	for _, code := range expected {
		if !ValidSeverityCode(code) {
			t.Errorf("expected %q to be a valid severity code", code)
		}
	}
}

func TestSeverityCodes_Invalid(t *testing.T) {
	if ValidSeverityCode("MADE_UP_CODE") {
		t.Error("expected MADE_UP_CODE to be invalid")
	}
}

func TestSeverityEntry_Validation(t *testing.T) {
	tests := []struct {
		name    string
		entry   SeverityEntry
		wantErr bool
	}{
		{
			name: "valid entry",
			entry: SeverityEntry{
				Code:      "ENDOGENOUS_EVIDENCE",
				Status:    "fail",
				AppliesTo: "C2",
				Note:      "Claim cites only model+probe descendants",
			},
		},
		{
			name: "valid pass",
			entry: SeverityEntry{
				Code:      "VOCABULARY_INFLATION",
				Status:    "pass",
				AppliesTo: "C1",
				Note:      "Term has predictive residue",
			},
		},
		{
			name:    "invalid code",
			entry:   SeverityEntry{Code: "BAD", Status: "fail", AppliesTo: "C1", Note: "x"},
			wantErr: true,
		},
		{
			name:    "invalid status",
			entry:   SeverityEntry{Code: "ENDOGENOUS_EVIDENCE", Status: "maybe", AppliesTo: "C1", Note: "x"},
			wantErr: true,
		},
		{
			name:    "missing applies_to",
			entry:   SeverityEntry{Code: "ENDOGENOUS_EVIDENCE", Status: "fail", Note: "x"},
			wantErr: true,
		},
		{
			name:    "missing note",
			entry:   SeverityEntry{Code: "ENDOGENOUS_EVIDENCE", Status: "fail", AppliesTo: "C1"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.entry.Validate()
			if tt.wantErr && err == nil {
				t.Error("expected validation error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestReviewerIndependence_Validation(t *testing.T) {
	tests := []struct {
		name    string
		ri      ReviewerIndependence
		wantErr bool
	}{
		{
			name: "valid - external human",
			ri: ReviewerIndependence{
				ReviewerType:         "human",
				ReviewerID:           "jane@example.com",
				ModelIndependence:    "External human reviewer, not an AI model",
				ContextIndependence:  "Received fixed challenge packet only",
				AuthorityIndependence: "Can block/downgrade but cannot bless",
			},
		},
		{
			name: "valid - different model family",
			ri: ReviewerIndependence{
				ReviewerType:         "model",
				ReviewerID:           "google/gemini-2.5-pro",
				OriginModel:          "anthropic/claude-opus-4-5",
				ModelIndependence:    "Different provider (Google vs Anthropic)",
				ContextIndependence:  "Received blind challenge packet, not full thread",
				AuthorityIndependence: "Structured objections only, no freeform approval",
			},
		},
		{
			name: "fail - same model family",
			ri: ReviewerIndependence{
				ReviewerType:         "model",
				ReviewerID:           "anthropic/claude-sonnet-4-5",
				OriginModel:          "anthropic/claude-opus-4-5",
				ModelIndependence:    "Same provider",
				ContextIndependence:  "Received blind packet",
				AuthorityIndependence: "Structured only",
			},
			wantErr: true,
		},
		{
			name: "fail - missing reviewer type",
			ri: ReviewerIndependence{
				ReviewerID:           "someone",
				ModelIndependence:    "x",
				ContextIndependence:  "x",
				AuthorityIndependence: "x",
			},
			wantErr: true,
		},
		{
			name: "fail - missing context independence",
			ri: ReviewerIndependence{
				ReviewerType:         "human",
				ReviewerID:           "jane",
				ModelIndependence:    "External human",
				AuthorityIndependence: "Can block",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ri.Validate()
			if tt.wantErr && err == nil {
				t.Error("expected validation error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestGenerateChallengeTemplate(t *testing.T) {
	tmpl := GenerateChallengeTemplate("test-artifact", ".kb/models/accretion/model.md")

	// Must contain all required sections
	requiredSections := []string{
		"## Target Artifact",
		"## Reviewer Independence",
		"## Blind Canonicalization",
		"## Prior-Art Mapping",
		"## Evidence Loop Findings",
		"## Severity Codes",
		"## Publication Verdict",
	}
	for _, section := range requiredSections {
		if !strings.Contains(tmpl, section) {
			t.Errorf("template missing required section: %s", section)
		}
	}

	// Must reference target artifact
	if !strings.Contains(tmpl, ".kb/models/accretion/model.md") {
		t.Error("template should reference target artifact path")
	}

	// Must contain severity code table header
	if !strings.Contains(tmpl, "| code | status | applies_to | note |") {
		t.Error("template missing severity code table")
	}
}

func TestGenerateBlindPacket(t *testing.T) {
	dir := t.TempDir()

	// Create a model file with claims
	modelsDir := filepath.Join(dir, ".kb", "models", "test-model")
	os.MkdirAll(modelsDir, 0755)
	modelContent := `# Test Model

## Summary
Files grow because adding is cheaper than cleanup. This pattern
appears in codebases, documentation, and institutional knowledge.

## Core Claims
1. Addition is cheaper than subtraction in shared systems
2. Conventions decay under agent throughput
3. Governance emerges as a response to coordination cost
`
	os.WriteFile(filepath.Join(modelsDir, "model.md"), []byte(modelContent), 0644)

	packet, err := GenerateBlindPacket(filepath.Join(modelsDir, "model.md"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Blind packet should contain the content but strip model-specific vocabulary
	if packet.ArtifactPath != filepath.Join(modelsDir, "model.md") {
		t.Error("packet should reference artifact path")
	}

	// Should have instructions for the blind reviewer
	if !strings.Contains(packet.Instructions, "existing concepts") {
		t.Error("blind packet instructions should ask about existing concepts")
	}
	if !strings.Contains(packet.Instructions, "surprising") {
		t.Error("blind packet instructions should ask what is surprising")
	}
	if !strings.Contains(packet.Instructions, "overclaim") {
		t.Error("blind packet instructions should ask about overclaim")
	}

	// Should include the raw content
	if !strings.Contains(packet.Content, "Addition is cheaper than subtraction") {
		t.Error("blind packet should contain model content")
	}
}

func TestGenerateFramedPacket(t *testing.T) {
	dir := t.TempDir()

	modelsDir := filepath.Join(dir, ".kb", "models", "test-model")
	os.MkdirAll(modelsDir, 0755)
	modelContent := `# Accretion Dynamics Model

## Summary
Knowledge accretion is a substrate-independent process.

## Core Claims
1. Accretion dynamics follow physics-like patterns
2. This is a novel framework for understanding governance
`
	os.WriteFile(filepath.Join(modelsDir, "model.md"), []byte(modelContent), 0644)

	packet, err := GenerateFramedPacket(filepath.Join(modelsDir, "model.md"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Framed packet includes the actual model content
	if !strings.Contains(packet.Content, "Accretion Dynamics Model") {
		t.Error("framed packet should include full model content")
	}

	// Should have instructions for the framed reviewer
	if !strings.Contains(packet.Instructions, "renamed familiar concepts") {
		t.Error("framed packet instructions should ask about renamed concepts")
	}
	if !strings.Contains(packet.Instructions, "endogenous evidence") {
		t.Error("framed packet instructions should ask about endogenous evidence")
	}
	if !strings.Contains(packet.Instructions, "banned") {
		t.Error("framed packet instructions should ask about banned terms")
	}
}

func TestCreateChallengeArtifact(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	challengesDir := filepath.Join(kbDir, "challenges")
	os.MkdirAll(challengesDir, 0755)

	targetArtifact := ".kb/models/accretion/model.md"
	path, err := CreateChallengeArtifact(dir, "accretion-model", targetArtifact)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be in challenges dir with date prefix
	if !strings.HasPrefix(filepath.Base(path), "2026-") {
		t.Error("challenge filename should start with date")
	}
	if !strings.HasSuffix(path, "-accretion-model.md") {
		t.Errorf("challenge filename should end with slug: %s", path)
	}

	// File should exist and contain template
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read created file: %v", err)
	}
	if !strings.Contains(string(content), "## Target Artifact") {
		t.Error("created file should contain challenge template")
	}
}

func TestParseChallengeArtifact(t *testing.T) {
	dir := t.TempDir()
	challengePath := filepath.Join(dir, "challenge.md")

	content := `---
target_artifact: .kb/models/accretion/model.md
reviewer:
  reviewer_type: model
  reviewer_id: google/gemini-2.5-pro
  origin_model: anthropic/claude-opus-4-5
  model_independence: "Different provider (Google vs Anthropic)"
  context_independence: "Received blind challenge packet only"
  authority_independence: "Structured objections only"
severity_codes:
  - code: ENDOGENOUS_EVIDENCE
    status: fail
    applies_to: C2
    note: "Claim cites only model+probe descendants"
  - code: VOCABULARY_INFLATION
    status: fail
    applies_to: C2
    note: "'substrate-independent physics' collapses to governance framing"
  - code: EXTERNAL_NOVELTY_DELTA
    status: pass
    applies_to: C1
    note: "Predictive residue remains after canonicalization"
publication_verdict: fail
---

# Challenge: accretion-model

## Target Artifact
.kb/models/accretion/model.md

## Reviewer Independence
External model review via google/gemini-2.5-pro.

## Blind Canonicalization
Done.

## Prior-Art Mapping
Done.

## Evidence Loop Findings
C2 cites only model+probe descendants.

## Severity Codes
| code | status | applies_to | note |
|------|--------|------------|------|
| ENDOGENOUS_EVIDENCE | fail | C2 | Claim cites only model+probe descendants |
| VOCABULARY_INFLATION | fail | C2 | collapses to governance framing |
| EXTERNAL_NOVELTY_DELTA | pass | C1 | Predictive residue remains |

## Publication Verdict
fail
`
	os.WriteFile(challengePath, []byte(content), 0644)

	challenge, err := ParseChallengeArtifact(challengePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if challenge.TargetArtifact != ".kb/models/accretion/model.md" {
		t.Errorf("wrong target artifact: %s", challenge.TargetArtifact)
	}
	if challenge.Reviewer.ReviewerType != "model" {
		t.Errorf("wrong reviewer type: %s", challenge.Reviewer.ReviewerType)
	}
	if challenge.Reviewer.ReviewerID != "google/gemini-2.5-pro" {
		t.Errorf("wrong reviewer ID: %s", challenge.Reviewer.ReviewerID)
	}
	if len(challenge.SeverityCodes) != 3 {
		t.Errorf("expected 3 severity codes, got %d", len(challenge.SeverityCodes))
	}
	if challenge.PublicationVerdict != "fail" {
		t.Errorf("expected verdict fail, got %s", challenge.PublicationVerdict)
	}
}

func TestValidateChallengeArtifact(t *testing.T) {
	valid := ChallengeArtifact{
		TargetArtifact: ".kb/models/test/model.md",
		Reviewer: ReviewerIndependence{
			ReviewerType:         "human",
			ReviewerID:           "jane@example.com",
			ModelIndependence:    "External human",
			ContextIndependence:  "Fixed packet",
			AuthorityIndependence: "Can block only",
		},
		SeverityCodes: []SeverityEntry{
			{Code: "ENDOGENOUS_EVIDENCE", Status: "pass", AppliesTo: "C1", Note: "Has external refs"},
		},
		PublicationVerdict: "pass",
	}

	if err := valid.Validate(); err != nil {
		t.Errorf("valid challenge should pass validation: %v", err)
	}

	// Missing target
	bad := valid
	bad.TargetArtifact = ""
	if err := bad.Validate(); err == nil {
		t.Error("missing target artifact should fail")
	}

	// Invalid verdict
	bad2 := valid
	bad2.PublicationVerdict = "maybe"
	if err := bad2.Validate(); err == nil {
		t.Error("invalid verdict should fail")
	}
}

func TestChallengeArtifact_YAMLRoundTrip(t *testing.T) {
	original := ChallengeArtifact{
		TargetArtifact: ".kb/models/test/model.md",
		Reviewer: ReviewerIndependence{
			ReviewerType:         "model",
			ReviewerID:           "google/gemini-2.5-pro",
			OriginModel:          "anthropic/claude-opus-4-5",
			ModelIndependence:    "Different provider",
			ContextIndependence:  "Blind packet",
			AuthorityIndependence: "Block only",
		},
		SeverityCodes: []SeverityEntry{
			{Code: "ENDOGENOUS_EVIDENCE", Status: "fail", AppliesTo: "C1", Note: "Only self-refs"},
		},
		PublicationVerdict: "fail",
	}

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var parsed ChallengeArtifact
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if parsed.TargetArtifact != original.TargetArtifact {
		t.Error("round-trip target mismatch")
	}
	if parsed.Reviewer.ReviewerID != original.Reviewer.ReviewerID {
		t.Error("round-trip reviewer mismatch")
	}
	if len(parsed.SeverityCodes) != 1 {
		t.Error("round-trip severity codes mismatch")
	}
}
