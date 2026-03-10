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
	os.MkdirAll(challengesDir, 0755)
	os.MkdirAll(invDir, 0755)

	os.WriteFile(filepath.Join(challengesDir, "2026-03-10-test.md"), []byte("# Challenge"), 0644)
	os.WriteFile(filepath.Join(invDir, "2026-03-10-inv-data.md"), []byte("# Investigation"), 0644)

	pub := filepath.Join(dir, "pub.md")
	content := `---
challenge_refs:
  - .kb/challenges/2026-03-10-test.md
claim_refs:
  - C1
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
