package skill

import (
	"strings"
	"testing"
)

// --- Rule 1: MUST-density ---

func TestMustDensity_HighDensity(t *testing.T) {
	// 20 words with 2 MUSTs = 10/100 words → above threshold
	content := "You MUST do X. You MUST do Y. Some filler words here to pad."
	results := LintContent(content, nil)
	found := findWarning(results, RuleMustDensity)
	if found == nil {
		t.Fatal("expected MUST-density warning for high density content")
	}
}

func TestMustDensity_LowDensity(t *testing.T) {
	// Lots of words, few MUSTs
	words := strings.Repeat("word ", 100)
	content := "You MUST do this. " + words
	results := LintContent(content, nil)
	found := findWarning(results, RuleMustDensity)
	if found != nil {
		t.Fatalf("unexpected MUST-density warning: %s", found.Message)
	}
}

func TestMustDensity_CountsAllKeywords(t *testing.T) {
	// Should count MUST, NEVER, CRITICAL, ALWAYS
	content := "MUST NEVER CRITICAL ALWAYS word word word word word word"
	results := LintContent(content, nil)
	found := findWarning(results, RuleMustDensity)
	if found == nil {
		t.Fatal("expected MUST-density warning counting all keyword types")
	}
}

// --- Rule 2: Cosmetic redundancy ---

func TestCosmeticRedundancy_Detected(t *testing.T) {
	// Three lines that all extract to the same constraint phrase
	content := `You MUST report phase transitions.
ALWAYS report phase transitions.
You MUST report phase transitions.`
	results := LintContent(content, nil)
	found := findWarning(results, RuleCosmeticRedundancy)
	if found == nil {
		t.Fatal("expected cosmetic redundancy warning for repeated phrase")
	}
}

func TestCosmeticRedundancy_NoRepeats(t *testing.T) {
	content := `You MUST report phase transitions.
You MUST commit your work.
You MUST run tests.`
	results := LintContent(content, nil)
	found := findWarning(results, RuleCosmeticRedundancy)
	if found != nil {
		t.Fatalf("unexpected cosmetic redundancy warning: %s", found.Message)
	}
}

// --- Rule 3: Section sprawl ---

func TestSectionSprawl_TooMany(t *testing.T) {
	var lines []string
	for i := 0; i < 35; i++ {
		lines = append(lines, "- You MUST do something specific "+string(rune('A'+i%26)))
	}
	content := strings.Join(lines, "\n")
	results := LintContent(content, nil)
	found := findWarning(results, RuleSectionSprawl)
	if found == nil {
		t.Fatal("expected section sprawl warning for >30 constraints")
	}
}

func TestSectionSprawl_Acceptable(t *testing.T) {
	content := `- MUST do A
- NEVER do B
- ALWAYS do C`
	results := LintContent(content, nil)
	found := findWarning(results, RuleSectionSprawl)
	if found != nil {
		t.Fatalf("unexpected section sprawl warning: %s", found.Message)
	}
}

// --- Rule 4: Signal imbalance ---

func TestSignalImbalance_Detected(t *testing.T) {
	// Same behavior reinforced 4+ times → imbalance
	content := `You MUST use bd comment for phase transitions.
ALWAYS use bd comment for phase reporting.
NEVER skip bd comment updates.
CRITICAL: bd comment must be called at every phase.`
	results := LintContent(content, nil)
	found := findWarning(results, RuleSignalImbalance)
	if found == nil {
		t.Fatal("expected signal imbalance warning for repeated reinforcement")
	}
}

func TestSignalImbalance_Balanced(t *testing.T) {
	content := `You MUST use bd comment.
You MUST commit your work.
You MUST run tests.
You MUST report phase.`
	results := LintContent(content, nil)
	found := findWarning(results, RuleSignalImbalance)
	if found != nil {
		t.Fatalf("unexpected signal imbalance warning: %s", found.Message)
	}
}

// --- Rule 5: Dead constraint ---

func TestDeadConstraint_NoTests(t *testing.T) {
	content := `You MUST validate input before processing.
You MUST log all errors.`
	// No test files provided → dead constraint info
	results := LintContent(content, nil)
	found := findWarning(results, RuleDeadConstraint)
	if found == nil {
		t.Fatal("expected dead constraint info when no test coverage provided")
	}
	if found.Severity != SeverityInfo {
		t.Fatalf("expected info severity for dead constraint, got %s", found.Severity)
	}
}

func TestDeadConstraint_WithTests(t *testing.T) {
	content := `You MUST validate input before processing.`
	tests := []string{"validate input"}
	results := LintContent(content, tests)
	found := findWarning(results, RuleDeadConstraint)
	if found != nil {
		t.Fatalf("unexpected dead constraint warning when test coverage exists: %s", found.Message)
	}
}

// --- Integration: real orchestrator skill content ---

func TestLintContent_RealishSkill(t *testing.T) {
	// Simulate a skill with known anti-patterns
	content := `---
name: orchestrator
skill-type: policy
---

# Orchestrator

## Identity
You MUST act as orchestrator. NEVER act as worker.
You MUST ALWAYS delegate. CRITICAL: delegation is required.
ALWAYS spawn, NEVER do work yourself.

## Per-Turn Gate
You MUST check beads first. ALWAYS run bd ready.
You MUST NEVER skip the gate. CRITICAL gate.
ALWAYS check. MUST check. NEVER skip.

## Spawning
- MUST use correct skill
- MUST set tier
- MUST set phases
- MUST set mode
- MUST track issue
- MUST check hotspots
- MUST verify spawn
- MUST confirm context
- MUST include kb context
- MUST validate model
- NEVER spawn without issue
- NEVER skip triage
- ALWAYS use bypass-triage
- CRITICAL: spawn correctly
- MUST not investigate yourself
- MUST delegate reading code
- NEVER read more than 2 files
- ALWAYS spawn fast
- CRITICAL: speed matters
- MUST check accounts
- MUST verify capacity
- ALWAYS switch when low
- NEVER ignore rate limits
- CRITICAL: account management
- MUST log events
- MUST track lifecycle
- ALWAYS verify completion
- NEVER skip verification
- CRITICAL: verification required
- MUST clean up
- MUST archive workspaces
- ALWAYS maintain registry
`
	results := LintContent(content, nil)

	if len(results) == 0 {
		t.Fatal("expected warnings for skill with known anti-patterns")
	}

	// Should detect MUST-density
	if findWarning(results, RuleMustDensity) == nil {
		t.Error("expected MUST-density warning")
	}

	// Should detect section sprawl (>30 constraints)
	if findWarning(results, RuleSectionSprawl) == nil {
		t.Error("expected section sprawl warning")
	}
}

// --- Helpers ---

func findWarning(results []LintResult, rule string) *LintResult {
	for i := range results {
		if results[i].Rule == rule {
			return &results[i]
		}
	}
	return nil
}
