package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dylan-conlin/orch-go/pkg/kbgate"
)

func TestKBGatePublish_FailsOnMissingContract(t *testing.T) {
	dir := t.TempDir()
	pub := filepath.Join(dir, "pub.md")
	os.WriteFile(pub, []byte("---\n---\n# No contract fields\n"), 0644)

	result := kbgate.CheckPublish(pub)
	if result.Pass {
		t.Error("expected failure for missing contract fields")
	}

	hasChallengeRef := false
	hasClaimRef := false
	for _, v := range result.Verdicts {
		if v.Code == "MISSING_CHALLENGE_REFS" {
			hasChallengeRef = true
		}
		if v.Code == "MISSING_CLAIM_REFS" {
			hasClaimRef = true
		}
	}
	if !hasChallengeRef {
		t.Error("expected MISSING_CHALLENGE_REFS verdict")
	}
	if !hasClaimRef {
		t.Error("expected MISSING_CLAIM_REFS verdict")
	}
}

func TestKBGatePublish_PassesValidPublication(t *testing.T) {
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
	os.WriteFile(filepath.Join(pubDir, "claim-ledger.yaml"), []byte("claims:\n  - id: C1\n    text: \"Decay observed\"\n    type: observation\n    scope: local\n    evidence: \"Direct measurement\"\n    strength: strong\n"), 0644)

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
      - .kb/investigations/2026-01-01-data.md
---

# Working Model

This describes coordination patterns as a working model.
`
	os.WriteFile(pub, []byte(content), 0644)

	result := kbgate.CheckPublish(pub)
	if !result.Pass {
		codes := []string{}
		for _, v := range result.Verdicts {
			codes = append(codes, v.Code+"="+v.Status)
		}
		t.Errorf("expected pass, got failures: %v", codes)
	}
}

func TestKBGatePublish_ClaimUpgradesScopedToTargetFile(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	challengesDir := filepath.Join(kbDir, "challenges")
	invDir := filepath.Join(kbDir, "investigations")
	pubDir := filepath.Join(kbDir, "publications")
	modDir := filepath.Join(kbDir, "models", "test-model", "probes")
	os.MkdirAll(challengesDir, 0755)
	os.MkdirAll(invDir, 0755)
	os.MkdirAll(pubDir, 0755)
	os.MkdirAll(modDir, 0755)
	os.WriteFile(filepath.Join(challengesDir, "2026-03-10-test.md"), []byte("# Challenge"), 0644)
	os.WriteFile(filepath.Join(invDir, "2026-01-01-data.md"), []byte("# Inv"), 0644)
	os.WriteFile(filepath.Join(pubDir, "claim-ledger.yaml"), []byte("claims:\n  - id: C1\n    text: \"Decay observed\"\n    type: observation\n    scope: local\n    evidence: \"Direct measurement\"\n    strength: strong\n"), 0644)

	// OTHER publication with novelty language — should NOT affect our target
	os.WriteFile(filepath.Join(pubDir, "other.md"), []byte("# Other\nThis is a novel discovery.\n"), 0644)

	// Model file with causal language — should NOT affect our target
	os.WriteFile(filepath.Join(kbDir, "models", "test-model", "model.md"),
		[]byte("# Model\n## Summary\nThis always causes degradation.\n"), 0644)

	// Probe with self-validating language — should NOT affect our target
	os.WriteFile(filepath.Join(modDir, "probe.md"),
		[]byte("# Probe\n## Model Impact\n- **Confirms** the model claim.\n"), 0644)

	// Target publication: clean content, no claim-upgrade signals
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
      - .kb/investigations/2026-01-01-data.md
---

# Working Model

This describes coordination patterns as a working model.
`
	os.WriteFile(pub, []byte(content), 0644)

	result := kbgate.CheckPublish(pub)

	// Should NOT have CLAIM_UPGRADE_SIGNALS because the target file is clean,
	// even though other KB files contain signals
	for _, v := range result.Verdicts {
		if v.Code == "CLAIM_UPGRADE_SIGNALS" {
			t.Errorf("publish gate should not report claim-upgrade signals from other KB files, got: %s", v.Note)
		}
	}
	if !result.Pass {
		codes := []string{}
		for _, v := range result.Verdicts {
			codes = append(codes, v.Code+"="+v.Status)
		}
		t.Errorf("expected pass for clean publication, got failures: %v", codes)
	}
}

func TestKBGatePublish_BannedLanguageBlocks(t *testing.T) {
	dir := t.TempDir()
	kbDir := filepath.Join(dir, ".kb")
	os.MkdirAll(filepath.Join(kbDir, "challenges"), 0755)
	os.WriteFile(filepath.Join(kbDir, "challenges", "2026-03-10-test.md"), []byte("# C"), 0644)

	pub := filepath.Join(dir, "pub.md")
	content := `---
challenge_refs:
  - .kb/challenges/2026-03-10-test.md
claim_refs:
  - C1
---

# Publication

This proves a new framework for substrate-independent general law as validated theory in physics.
`
	os.WriteFile(pub, []byte(content), 0644)

	result := kbgate.CheckPublish(pub)
	if result.Pass {
		t.Error("expected failure for banned language")
	}

	hasBanned := false
	for _, v := range result.Verdicts {
		if v.Code == "BANNED_LANGUAGE" {
			hasBanned = true
		}
	}
	if !hasBanned {
		t.Error("expected BANNED_LANGUAGE verdict")
	}
}
