package kbgate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckPublish_MissingFile(t *testing.T) {
	result := CheckPublish("/nonexistent/file.md")
	if result.Pass {
		t.Error("expected failure for missing file")
	}
}

func TestCheckPublish_MissingChallengeRefs(t *testing.T) {
	dir := t.TempDir()
	pub := filepath.Join(dir, "pub.md")
	content := `---
claim_refs:
  - .kb/models/test/claims.md
---

# My Publication

Some content here.
`
	os.WriteFile(pub, []byte(content), 0644)

	result := CheckPublish(pub)
	if result.Pass {
		t.Error("expected failure for missing challenge_refs")
	}
	assertHasVerdict(t, result, "MISSING_CHALLENGE_REFS")
}

func TestCheckPublish_MissingClaimRefs(t *testing.T) {
	dir := t.TempDir()
	pub := filepath.Join(dir, "pub.md")
	content := `---
challenge_refs:
  - .kb/challenges/2026-03-10-test.md
---

# My Publication

Some content here.
`
	os.WriteFile(pub, []byte(content), 0644)

	result := CheckPublish(pub)
	if result.Pass {
		t.Error("expected failure for missing claim_refs")
	}
	assertHasVerdict(t, result, "MISSING_CLAIM_REFS")
}

func TestCheckPublish_ChallengeArtifactNotFound(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	os.MkdirAll(filepath.Join(kbDir, "challenges"), 0755)

	pub := filepath.Join(dir, "pub.md")
	content := `---
challenge_refs:
  - .kb/challenges/2026-03-10-nonexistent.md
claim_refs:
  - .kb/models/test/claims.md
---

# My Publication
`
	os.WriteFile(pub, []byte(content), 0644)

	result := CheckPublish(pub)
	if result.Pass {
		t.Error("expected failure for missing challenge artifact")
	}
	assertHasVerdict(t, result, "CHALLENGE_ARTIFACT_MISSING")
}

func TestCheckPublish_ChallengeArtifactExists(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	challengesDir := filepath.Join(kbDir, "challenges")
	os.MkdirAll(challengesDir, 0755)

	// Create the challenge artifact
	challengeContent := `# Challenge: Test
## Target Artifact
pub.md
## Reviewer Independence
External human reviewer
## Blind Canonicalization
Done
## Prior-Art Mapping
Done
## Evidence Loop Findings
None
## Severity Codes
None
## Publication Verdict
pass
`
	os.WriteFile(filepath.Join(challengesDir, "2026-03-10-test.md"), []byte(challengeContent), 0644)

	pub := filepath.Join(dir, "pub.md")
	content := `---
challenge_refs:
  - .kb/challenges/2026-03-10-test.md
claim_refs:
  - .kb/models/test/claims.md
---

# My Publication

This is a working model for coordination patterns.
`
	os.WriteFile(pub, []byte(content), 0644)

	result := CheckPublish(pub)
	// Should not have CHALLENGE_ARTIFACT_MISSING
	for _, v := range result.Verdicts {
		if v.Code == "CHALLENGE_ARTIFACT_MISSING" {
			t.Error("should not have CHALLENGE_ARTIFACT_MISSING when file exists")
		}
	}
}

func TestCheckPublish_BannedLanguage(t *testing.T) {
	tests := []struct {
		name    string
		content string
		banned  bool
	}{
		{
			name: "uses physics",
			content: `---
challenge_refs:
  - .kb/challenges/2026-03-10-test.md
claim_refs:
  - .kb/models/test/claims.md
---

# Knowledge Physics

This is a new physics of knowledge.
`,
			banned: true,
		},
		{
			name: "uses new framework",
			content: `---
challenge_refs:
  - .kb/challenges/2026-03-10-test.md
claim_refs:
  - .kb/models/test/claims.md
---

# Publication

We present a new framework for understanding coordination.
`,
			banned: true,
		},
		{
			name: "uses validated theory",
			content: `---
challenge_refs:
  - .kb/challenges/2026-03-10-test.md
claim_refs:
  - .kb/models/test/claims.md
---

# Publication

This validated theory shows that agents need governance.
`,
			banned: true,
		},
		{
			name: "uses substrate-independent",
			content: `---
challenge_refs:
  - .kb/challenges/2026-03-10-test.md
claim_refs:
  - .kb/models/test/claims.md
---

# Publication

These are substrate-independent patterns.
`,
			banned: true,
		},
		{
			name: "uses proves",
			content: `---
challenge_refs:
  - .kb/challenges/2026-03-10-test.md
claim_refs:
  - .kb/models/test/claims.md
---

# Publication

This proves that coordination costs scale linearly.
`,
			banned: true,
		},
		{
			name: "uses general law",
			content: `---
challenge_refs:
  - .kb/challenges/2026-03-10-test.md
claim_refs:
  - .kb/models/test/claims.md
---

# Publication

We have discovered a general law of agent coordination.
`,
			banned: true,
		},
		{
			name: "clean language",
			content: `---
challenge_refs:
  - .kb/challenges/2026-03-10-test.md
claim_refs:
  - .kb/models/test/claims.md
---

# Publication

This is a working model for coordination patterns observed in our system.
`,
			banned: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			kbDir := filepath.Join(dir, ".kb")
			challengesDir := filepath.Join(kbDir, "challenges")
			os.MkdirAll(challengesDir, 0755)
			os.WriteFile(filepath.Join(challengesDir, "2026-03-10-test.md"), []byte("# Challenge"), 0644)

			pub := filepath.Join(dir, "pub.md")
			os.WriteFile(pub, []byte(tt.content), 0644)

			result := CheckPublish(pub)
			hasBanned := hasVerdict(result, "BANNED_LANGUAGE")
			if tt.banned && !hasBanned {
				t.Error("expected BANNED_LANGUAGE verdict")
			}
			if !tt.banned && hasBanned {
				t.Error("unexpected BANNED_LANGUAGE verdict")
			}
		})
	}
}

func TestCheckPublish_EndogenousEvidence(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	challengesDir := filepath.Join(kbDir, "challenges")
	modelsDir := filepath.Join(kbDir, "models", "test-model")
	probesDir := filepath.Join(modelsDir, "probes")
	invDir := filepath.Join(kbDir, "investigations")
	os.MkdirAll(challengesDir, 0755)
	os.MkdirAll(probesDir, 0755)
	os.MkdirAll(invDir, 0755)

	// Create artifacts
	os.WriteFile(filepath.Join(challengesDir, "2026-03-10-test.md"), []byte("# Challenge"), 0644)
	os.WriteFile(filepath.Join(modelsDir, "model.md"), []byte("# Model"), 0644)
	os.WriteFile(filepath.Join(probesDir, "2026-03-10-probe.md"), []byte("# Probe"), 0644)
	os.WriteFile(filepath.Join(invDir, "2026-03-10-inv-real-data.md"), []byte("# Investigation"), 0644)

	t.Run("endogenous only - model and probe refs", func(t *testing.T) {
		pub := filepath.Join(dir, "pub.md")
		content := `---
challenge_refs:
  - .kb/challenges/2026-03-10-test.md
claim_refs:
  - C1
claims:
  - claim_id: C1
    claim_text: "Knowledge has physics"
    claim_type: generalization
    novelty_level: novel
    evidence_refs:
      - .kb/models/test-model/model.md
      - .kb/models/test-model/probes/2026-03-10-probe.md
---

# My Publication
`
		os.WriteFile(pub, []byte(content), 0644)

		result := CheckPublish(pub)
		if result.Pass {
			t.Error("expected failure for endogenous evidence")
		}
		assertHasVerdict(t, result, "ENDOGENOUS_EVIDENCE")
	})

	t.Run("exogenous evidence - investigation ref", func(t *testing.T) {
		pub := filepath.Join(dir, "pub.md")
		content := `---
challenge_refs:
  - .kb/challenges/2026-03-10-test.md
claim_refs:
  - C1
claims:
  - claim_id: C1
    claim_text: "Conventions decay under throughput"
    claim_type: generalization
    novelty_level: novel
    evidence_refs:
      - .kb/investigations/2026-03-10-inv-real-data.md
      - .kb/models/test-model/model.md
---

# My Publication
`
		os.WriteFile(pub, []byte(content), 0644)

		result := CheckPublish(pub)
		if hasVerdict(result, "ENDOGENOUS_EVIDENCE") {
			t.Error("should not have ENDOGENOUS_EVIDENCE when investigation ref exists")
		}
	})

	t.Run("observation claims skip lineage check", func(t *testing.T) {
		pub := filepath.Join(dir, "pub.md")
		content := `---
challenge_refs:
  - .kb/challenges/2026-03-10-test.md
claim_refs:
  - C1
claims:
  - claim_id: C1
    claim_text: "We observed files growing"
    claim_type: observation
    novelty_level: restatement
    evidence_refs:
      - .kb/models/test-model/model.md
---

# My Publication
`
		os.WriteFile(pub, []byte(content), 0644)

		result := CheckPublish(pub)
		if hasVerdict(result, "ENDOGENOUS_EVIDENCE") {
			t.Error("observation claims should not trigger endogenous evidence check")
		}
	})
}

func TestCheckPublish_FullPass(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	challengesDir := filepath.Join(kbDir, "challenges")
	invDir := filepath.Join(kbDir, "investigations")
	pubDir := filepath.Join(kbDir, "publications")
	os.MkdirAll(challengesDir, 0755)
	os.MkdirAll(invDir, 0755)
	os.MkdirAll(pubDir, 0755)

	os.WriteFile(filepath.Join(challengesDir, "2026-03-10-test.md"), []byte("# Challenge"), 0644)
	os.WriteFile(filepath.Join(invDir, "2026-03-10-inv-data.md"), []byte("# Investigation"), 0644)
	os.WriteFile(filepath.Join(pubDir, "claim-ledger.yaml"), []byte(`claims:
  - id: C1
    text: "Conventions decay under agent throughput"
    type: mechanism
    scope: local
    evidence: "Direct observation over 60 days"
    strength: strong
`), 0644)

	pub := filepath.Join(dir, "pub.md")
	content := `---
challenge_refs:
  - .kb/challenges/2026-03-10-test.md
claim_refs:
  - C1
ledger_ref: .kb/publications/claim-ledger.yaml
claims:
  - claim_id: C1
    claim_text: "Conventions decay under agent throughput"
    claim_type: mechanism
    novelty_level: synthesis
    evidence_refs:
      - .kb/investigations/2026-03-10-inv-data.md
---

# My Publication

This is a working model describing how conventions decay under agent throughput.
`
	os.WriteFile(pub, []byte(content), 0644)

	result := CheckPublish(pub)
	if !result.Pass {
		t.Errorf("expected pass, got failures: %v", verdictCodes(result))
	}
}

// helpers

func assertHasVerdict(t *testing.T, result GateResult, code string) {
	t.Helper()
	if !hasVerdict(result, code) {
		t.Errorf("expected verdict %s, got: %v", code, verdictCodes(result))
	}
}

func hasVerdict(result GateResult, code string) bool {
	for _, v := range result.Verdicts {
		if v.Code == code {
			return true
		}
	}
	return false
}

func verdictCodes(result GateResult) []string {
	var codes []string
	for _, v := range result.Verdicts {
		codes = append(codes, v.Code+"="+strings.ToLower(v.Status))
	}
	return codes
}

func TestCheckPublish_MissingLedgerRef(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	challengesDir := filepath.Join(kbDir, "challenges")
	invDir := filepath.Join(kbDir, "investigations")
	os.MkdirAll(challengesDir, 0755)
	os.MkdirAll(invDir, 0755)
	os.WriteFile(filepath.Join(challengesDir, "2026-03-10-test.md"), []byte("# Challenge"), 0644)
	os.WriteFile(filepath.Join(invDir, "2026-01-01-data.md"), []byte("# Inv"), 0644)

	pub := filepath.Join(dir, "pub.md")
	content := `---
challenge_refs:
  - .kb/challenges/2026-03-10-test.md
claim_refs:
  - C1
claims:
  - claim_id: C1
    claim_text: "Decay observed"
    claim_type: observation
    novelty_level: restatement
    evidence_refs:
      - .kb/investigations/2026-01-01-data.md
---

# Working Model
`
	os.WriteFile(pub, []byte(content), 0644)

	result := CheckPublish(pub)
	if result.Pass {
		t.Error("expected failure for missing ledger_ref")
	}
	assertHasVerdict(t, result, "MISSING_LEDGER_REF")
}

func TestCheckPublish_LedgerArtifactMissing(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	challengesDir := filepath.Join(kbDir, "challenges")
	invDir := filepath.Join(kbDir, "investigations")
	os.MkdirAll(challengesDir, 0755)
	os.MkdirAll(invDir, 0755)
	os.WriteFile(filepath.Join(challengesDir, "2026-03-10-test.md"), []byte("# Challenge"), 0644)
	os.WriteFile(filepath.Join(invDir, "2026-01-01-data.md"), []byte("# Inv"), 0644)

	pub := filepath.Join(dir, "pub.md")
	content := `---
challenge_refs:
  - .kb/challenges/2026-03-10-test.md
claim_refs:
  - C1
ledger_ref: .kb/publications/nonexistent-ledger.yaml
claims:
  - claim_id: C1
    claim_text: "Decay observed"
    claim_type: observation
    novelty_level: restatement
    evidence_refs:
      - .kb/investigations/2026-01-01-data.md
---

# Working Model
`
	os.WriteFile(pub, []byte(content), 0644)

	result := CheckPublish(pub)
	if result.Pass {
		t.Error("expected failure for missing ledger artifact")
	}
	assertHasVerdict(t, result, "LEDGER_ARTIFACT_MISSING")
}

func TestCheckPublish_LedgerValid(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	challengesDir := filepath.Join(kbDir, "challenges")
	invDir := filepath.Join(kbDir, "investigations")
	pubDir := filepath.Join(kbDir, "publications")
	os.MkdirAll(challengesDir, 0755)
	os.MkdirAll(invDir, 0755)
	os.MkdirAll(pubDir, 0755)
	os.WriteFile(filepath.Join(challengesDir, "2026-03-10-test.md"), []byte("# Challenge"), 0644)
	os.WriteFile(filepath.Join(invDir, "2026-01-01-data.md"), []byte("# Inv"), 0644)

	// Create valid ledger
	ledger := `claims:
  - id: C1
    text: "Decay observed in daemon.go"
    type: observation
    scope: local
    evidence: "Direct measurement of file size"
    strength: strong
`
	os.WriteFile(filepath.Join(pubDir, "claim-ledger.yaml"), []byte(ledger), 0644)

	pub := filepath.Join(dir, "pub.md")
	content := `---
challenge_refs:
  - .kb/challenges/2026-03-10-test.md
claim_refs:
  - C1
ledger_ref: .kb/publications/claim-ledger.yaml
claims:
  - claim_id: C1
    claim_text: "Decay observed"
    claim_type: observation
    novelty_level: restatement
    evidence_refs:
      - .kb/investigations/2026-01-01-data.md
---

# Working Model

This describes coordination patterns as a working model.
`
	os.WriteFile(pub, []byte(content), 0644)

	result := CheckPublish(pub)
	if hasVerdict(result, "MISSING_LEDGER_REF") || hasVerdict(result, "LEDGER_ARTIFACT_MISSING") || hasVerdict(result, "LEDGER_EMPTY") || hasVerdict(result, "LEDGER_INVALID") {
		t.Errorf("unexpected ledger verdicts: %v", verdictCodes(result))
	}
}

func TestCheckPublish_LedgerEmptyClaims(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	pubDir := filepath.Join(kbDir, "publications")
	os.MkdirAll(pubDir, 0755)

	// Create ledger with no claims
	os.WriteFile(filepath.Join(pubDir, "claim-ledger.yaml"), []byte("claims: []\n"), 0644)

	pub := filepath.Join(dir, "pub.md")
	content := `---
challenge_refs:
  - x
claim_refs:
  - x
ledger_ref: .kb/publications/claim-ledger.yaml
---

# Pub
`
	os.WriteFile(pub, []byte(content), 0644)

	result := CheckPublish(pub)
	assertHasVerdict(t, result, "LEDGER_EMPTY")
}

func TestCheckPublish_LedgerInvalidEntries(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	pubDir := filepath.Join(kbDir, "publications")
	os.MkdirAll(pubDir, 0755)

	// Create ledger with invalid fields
	ledger := `claims:
  - id: C1
    text: "Some claim"
    type: invalid_type
    scope: local
    evidence: "some evidence"
    strength: strong
  - id: C2
    text: ""
    type: observation
    scope: cosmic
    evidence: ""
    strength: mega
`
	os.WriteFile(filepath.Join(pubDir, "claim-ledger.yaml"), []byte(ledger), 0644)

	pub := filepath.Join(dir, "pub.md")
	content := `---
challenge_refs:
  - x
claim_refs:
  - x
ledger_ref: .kb/publications/claim-ledger.yaml
---

# Pub
`
	os.WriteFile(pub, []byte(content), 0644)

	result := CheckPublish(pub)
	assertHasVerdict(t, result, "LEDGER_INVALID")
}

func TestCheckPublish_ClaimUpgradeSignals(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	challengesDir := filepath.Join(kbDir, "challenges")
	invDir := filepath.Join(kbDir, "investigations")
	pubDir := filepath.Join(kbDir, "publications")
	modDir := filepath.Join(kbDir, "models", "test-model")
	probeDir := filepath.Join(modDir, "probes")
	os.MkdirAll(challengesDir, 0755)
	os.MkdirAll(invDir, 0755)
	os.MkdirAll(pubDir, 0755)
	os.MkdirAll(probeDir, 0755)

	os.WriteFile(filepath.Join(challengesDir, "2026-03-10-test.md"), []byte("# Challenge"), 0644)
	os.WriteFile(filepath.Join(invDir, "2026-03-10-inv-data.md"), []byte("# Investigation"), 0644)
	os.WriteFile(filepath.Join(pubDir, "claim-ledger.yaml"), []byte("claims:\n  - id: C1\n    text: \"Decay observed\"\n    type: observation\n    scope: local\n    evidence: \"Direct measurement\"\n    strength: strong\n"), 0644)

	// Publication with novelty language
	os.WriteFile(filepath.Join(pubDir, "draft.md"), []byte(`# Draft
This is a novel framework.
`), 0644)

	// Probe with self-validating conclusion
	os.WriteFile(filepath.Join(probeDir, "probe.md"), []byte(`# Probe
## Model Impact
- **Confirms** the model claim.
`), 0644)

	pub := filepath.Join(dir, "pub.md")
	content := `---
challenge_refs:
  - .kb/challenges/2026-03-10-test.md
claim_refs:
  - C1
ledger_ref: .kb/publications/claim-ledger.yaml
claims:
  - claim_id: C1
    claim_text: "Decay observed"
    claim_type: mechanism
    novelty_level: synthesis
    evidence_refs:
      - .kb/investigations/2026-03-10-inv-data.md
---

# Working Model

This is a novel framework for coordination that we discovered through observation.
`
	os.WriteFile(pub, []byte(content), 0644)

	t.Run("fails with claim-upgrade signals", func(t *testing.T) {
		result := CheckPublish(pub)
		if result.Pass {
			t.Error("expected failure for claim-upgrade signals")
		}
		assertHasVerdict(t, result, "CLAIM_UPGRADE_SIGNALS")
	})

	t.Run("passes with acknowledge-claims", func(t *testing.T) {
		result := CheckPublishWithOpts(pub, CheckPublishOpts{AcknowledgeClaims: true})
		if !result.Pass {
			t.Errorf("expected pass with --acknowledge-claims, got: %v", verdictCodes(result))
		}
		// Should still have the verdict but as warn
		found := false
		for _, v := range result.Verdicts {
			if v.Code == "CLAIM_UPGRADE_SIGNALS" {
				found = true
				if v.Status != "warn" {
					t.Errorf("expected status warn, got %s", v.Status)
				}
			}
		}
		if !found {
			t.Error("expected CLAIM_UPGRADE_SIGNALS verdict even with acknowledge-claims")
		}
	})
}
