# Completion Gates

**Purpose:** Single authoritative reference for all gates that block `orch complete`. Read this before debugging completion issues or adding new gates.

**Last verified:** Feb 26, 2026

---

## Overview

When you run `orch complete <id>`, the command runs a series of verification gates before closing the beads issue. Gates exist to ensure agent work is actually complete and correct.

```
orch complete <id>
       │
       ▼
┌─────────────────────────────────────────────────────────────────┐
│  VerifyCompletionFull (9 gates)                                 │
│  - Phase: Complete check                                        │
│  - SYNTHESIS.md check (full tier)                               │
│  - Skill constraints                                            │
│  - Phase gates                                                  │
│  - Skill outputs                                                │
│  - Visual verification (feature-impl + web/)                    │
│  - Test evidence (implementation skills + code changes)         │
│  - Git diff (SYNTHESIS claims vs actual)                        │
│  - Build verification (Go projects)                             │
└─────────────────────────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────────┐
│  Liveness Check                                                 │
│  - Warns if tmux/OpenCode session still running                 │
│  - Prompts for confirmation (or blocks in non-TTY)              │
└─────────────────────────────────────────────────────────────────┘
       │
       ▼
    Close beads issue
```

---

## Gate Reference

### 1. Phase: Complete (BLOCKING)

**File:** `pkg/verify/check.go:528-539`

**What it checks:** Agent must have reported "Phase: Complete" via beads comment.

**How to pass:**
```bash
bd comment <id> "Phase: Complete - task finished successfully"
```

**Bypass:** `--force`

---

### 2. SYNTHESIS.md (BLOCKING for full tier)

**File:** `pkg/verify/check.go:547-557`

**What it checks:** SYNTHESIS.md exists and is non-empty in workspace.

**When it applies:** Only for "full" tier spawns. Light tier spawns skip this.

**Tier determination:** Reads `.tier` file from workspace. Defaults to "full" if missing.

**How to pass:** Agent creates SYNTHESIS.md before reporting Phase: Complete.

**Bypass:** `--force` or spawn with light tier

---

### 3. Skill Constraints (BLOCKING)

**File:** `pkg/verify/constraint.go`

**What it checks:** Patterns in `<!-- SKILL-CONSTRAINTS -->` block match actual files.

**Format in SPAWN_CONTEXT.md:**
```markdown
<!-- SKILL-CONSTRAINTS -->
<!-- required: .kb/investigations/{date}-inv-*.md | Investigation artifact -->
<!-- optional: .kb/decisions/*.md | Decision record if recommendation accepted -->
<!-- /SKILL-CONSTRAINTS -->
```

**How to pass:** Create files matching required patterns.

**Spawn-time scoping:** Only files with mtime >= spawn time count as matches.

**Bypass:** `--force`

---

### 4. Phase Gates (BLOCKING)

**File:** `pkg/verify/phase_gates.go`

**What it checks:** Required phases in `<!-- SKILL-PHASES -->` block were reported via beads comments.

**Format in SPAWN_CONTEXT.md:**
```markdown
<!-- SKILL-PHASES -->
<!-- phase: planning | required: true -->
<!-- phase: implementation | required: true -->
<!-- phase: validation | required: false -->
<!-- /SKILL-PHASES -->
```

**How to pass:** Agent reports each required phase:
```bash
bd comment <id> "Phase: Planning - analyzed requirements"
bd comment <id> "Phase: Implementation - wrote code"
bd comment <id> "Phase: Complete - done"
```

**Bypass:** `--force`

---

### 5. Skill Outputs (BLOCKING)

**File:** `pkg/verify/skill_outputs.go`

**What it checks:** Patterns in skill.yaml `outputs.required` section match actual files.

**Format in skill.yaml:**
```yaml
name: investigation
outputs:
  required:
    - pattern: ".kb/investigations/{date}-inv-*.md"
      description: "Investigation artifact"
```

**How to pass:** Create files matching required patterns.

**Spawn-time scoping:** Only files with mtime >= spawn time count.

**Bypass:** `--force`

---

### 6. Visual Verification (BLOCKING for UI work)

**File:** `pkg/verify/visual.go`

**What it checks:** When web/ files are modified by UI-focused skills, requires:
1. Visual verification evidence in beads comments or SYNTHESIS.md
2. Human approval via `--approve` flag or approval comment

**Skills requiring visual verification:**
- `feature-impl`

**Skills excluded (even if they touch web/):**
- `architect`, `investigation`, `systematic-debugging`, `research`, `codebase-audit`, `reliability-testing`, `design-session`, `issue-creation`, `writing-skills`

**Evidence patterns recognized:**
- Screenshot mentions: "screenshot", "captured image"
- Visual verification: "visually verified", "UI verified", "browser verified"
- Browser tools: "playwright", "browser_take_screenshot"

**Approval patterns recognized:**
- `✅ APPROVED`
- `UI APPROVED` / `VISUAL APPROVED`
- `human_approved: true`
- `I approve the UI/visual/changes`

**How to pass:**
```bash
# Option 1: Agent provides evidence, orchestrator approves
bd comment <id> "Visual verification: screenshot captured showing dashboard"
orch complete <id> --approve

# Option 2: Add approval comment manually
bd comment <id> "✅ APPROVED"
orch complete <id>
```

**Bypass:** `--force` or `--approve`

---

### 7. Test Evidence (BLOCKING for code changes)

**File:** `pkg/verify/test_evidence.go`

**What it checks:** When code files are modified by implementation skills, requires actual test output in beads comments (not just "tests pass").

**Skills requiring test evidence:**
- `feature-impl`
- `systematic-debugging`
- `reliability-testing`

**Skills excluded:**
- `investigation`, `architect`, `research`, `design-session`, `codebase-audit`, `issue-creation`, `writing-skills`

**Code file extensions:**
- `.go`, `.py`, `.js`, `.ts`, `.jsx`, `.tsx`, `.rs`, `.rb`, `.java`, `.kt`, `.swift`, `.c`, `.cpp`, `.h`, `.hpp`, `.cs`, `.svelte`, `.vue`

**Evidence patterns recognized:**
- Go: `ok package/name 0.123s`, `--- PASS: TestName`, `(12 tests in 0.8s)`
- npm/yarn/bun: `15 passing, 0 failing`, `Tests: 15 passed`
- pytest: `======= 15 passed`, `15 passed, 0 failed`
- cargo: `test result: ok`, `15 passed; 0 failed`

**False positives rejected:**
- `tests pass` (no count/timing)
- `all tests pass` (no evidence)
- `verified tests pass` (claim without output)

**How to pass:**
```bash
bd comment <id> "Tests: go test ./pkg/... - PASS (12 tests in 0.8s)"
bd comment <id> "Tests: npm test - 15 passing, 0 failing"
```

**Spawn-time scoping:** Only code changes since spawn time trigger this gate.

**Bypass:** `--force`

---

### 8. Git Diff Verification (BLOCKING)

**File:** `pkg/verify/git_diff.go`

**What it checks:** Files claimed in SYNTHESIS.md Delta section actually appear in git diff.

**What it catches:** False positives where agent claims to modify files but didn't.

**Delta parsing:** Extracts file paths from:
- Backtick-quoted: `` `path/to/file.go` ``
- Bold: `**path/to/file.go**`
- Bullet points: `- path/to/file.go`

**How to pass:** Agent accurately reports modified files in SYNTHESIS.md Delta section.

**Spawn-time scoping:** Only considers commits since spawn time.

**Bypass:** `--force`

---

### 9. Build Verification (BLOCKING for Go projects)

**File:** `pkg/verify/build_verification.go`

**What it checks:** `go build ./...` succeeds when Go files are modified.

**Skills requiring build verification:**
- `feature-impl`
- `systematic-debugging`
- `reliability-testing`

**Skills excluded:**
- `investigation`, `architect`, `research`, `design-session`, `codebase-audit`, `issue-creation`, `writing-skills`

**Go project detection:** Checks for `go.mod` or `.go` files in root/cmd/pkg/internal.

**How to pass:** Fix build errors before completing.

**Bypass:** `--force`

---

### 10. Liveness Check (PROMPTS/BLOCKING - conditional)

**File:** `cmd/orch/main.go:890-940`

**What it checks:** Whether agent appears still running (tmux window or OpenCode session exists).

**When it runs:** Only if Phase: Complete was NOT reported. If the agent reported "Phase: Complete", the liveness check is skipped entirely - the agent said it's done, so whether its session is still open is irrelevant.

**Behavior (when it runs):**
- TTY mode: Prompts "Proceed anyway? [y/N]"
- Non-TTY mode: Blocks with error

**Why it exists:** Prevents closing issues for agents that are still working.

**Why Phase: Complete skips it:** OpenCode sessions persist to disk. An idle session doesn't mean the agent is still working - it might just not have been cleaned up. The reliable signal is Phase: Complete in beads comments.

**How to pass:** 
- Agent reports "Phase: Complete" (preferred - gate is skipped)
- Wait for agent to exit
- Confirm the prompt

**Bypass:** `--force`

---

### 11. Repro Verification (DISABLED)

**File:** `cmd/orch/main.go:935-951`

**Status:** DISABLED as of Jan 4, 2026

**What it checked:** For bug-type issues, prompted orchestrator to verify the original reproduction no longer occurs.

**Why disabled:** Created too much friction - agents couldn't complete without manual intervention. The code is commented out but preserved for potential re-enablement.

**Flags (now no-ops):**
- `--skip-repro-check`
- `--skip-repro-reason`

---

## Bypass Summary

| Flag | What it bypasses |
|------|------------------|
| `--force` | All gates (1-10) |
| `--approve` | Gate 6 (visual verification) - adds approval comment |

---

## Skill-Aware Gating

Many gates only apply to specific skills:

| Gate | Skills that trigger it |
|------|----------------------|
| Visual Verification | `feature-impl` only |
| Test Evidence | `feature-impl`, `systematic-debugging`, `reliability-testing` |
| Build Verification | `feature-impl`, `systematic-debugging`, `reliability-testing` |

Non-implementation skills (investigation, architect, research, etc.) are excluded from these gates even if they incidentally modify code or web/ files.

---

## Spawn-Time Scoping

Several gates use the `.spawn_time` file to only consider work done by THIS agent:

- **Skill Constraints:** Only files with mtime >= spawn time match
- **Skill Outputs:** Only files with mtime >= spawn time match
- **Visual Verification:** Only commits since spawn time checked for web/ changes
- **Test Evidence:** Only commits since spawn time checked for code changes
- **Git Diff:** Only commits since spawn time compared to SYNTHESIS claims

This prevents false positives from prior agents' work.

---

## Common Problems

### "Cannot complete - Phase: Complete not found"

**Cause:** Agent didn't report phase via beads comment.

**Fix:** Agent should run:
```bash
bd comment <id> "Phase: Complete - <summary>"
```

**Or:** Use `--force` to bypass.

### "Visual verification required but no evidence found"

**Cause:** Agent modified web/ files without screenshot/verification evidence.

**Fix options:**
1. Agent adds evidence: `bd comment <id> "Visual verification: screenshot captured"`
2. Orchestrator approves: `orch complete <id> --approve`
3. Bypass: `--force`

### "Test evidence required but not found"

**Cause:** Agent modified code without reporting test output.

**Fix:** Agent should run tests and report actual output:
```bash
bd comment <id> "Tests: go test ./pkg/... - ok (15 tests in 2.1s)"
```

**Not accepted:** Vague claims like "tests pass" without counts/timing.

### "Build failed"

**Cause:** `go build ./...` failed in the project.

**Fix:** Fix the build errors before completing.

### "Agent appears still running"

**Cause:** Liveness check found tmux window or OpenCode session.

**Fix options:**
1. Wait for agent to finish
2. Answer "y" to the prompt
3. Use `--force`

### "SYNTHESIS claims files not in git diff"

**Cause:** Agent claimed to modify files in SYNTHESIS.md but those files have no git changes.

**Fix:** Either the agent made false claims (fix SYNTHESIS) or forgot to commit (commit changes).

---

## Adding New Gates

If you need to add a new gate:

1. **Create verification function** in `pkg/verify/` following existing patterns:
   - Return `*SomethingResult` (nil if not applicable)
   - Include `Passed bool`, `Errors []string`, `Warnings []string`

2. **Add to VerifyCompletionFull** in `pkg/verify/check.go`:
   - Call your function
   - Merge errors/warnings into result
   - Set `result.Passed = false` if your check fails

3. **Consider skill-awareness:** Should this gate apply to all skills or just implementation skills?

4. **Consider spawn-time scoping:** Should this gate only consider work done since spawn?

5. **Update this guide** with the new gate.

---

## Human Verification Gate (Post-Completion)

After automated gates pass, `orch complete` has a two-gate human verification model:

**Gate 1 (explain-back):** The orchestrator must articulate what was built and why in their own words before work is marked verified. Reading is not understanding — explain-back is unfakeable verification of comprehension.

**Gate 2 (behavioral, Tier 1 only):** The orchestrator confirms the behavior is verified (e.g., running the feature, seeing the fix).

**Anti-pattern — batch-completing:** Batch-completing Tier 1 features as light tier during high-velocity sessions violates verifiability-first. Velocity pressure causes treating the review queue as something to clear rather than verify. Same root cause as entropy spiral — local correctness assumed without behavioral verification.

**Anti-pattern — closing without verifying deployed artifacts:** Four P1 issues were closed with "Phase: Complete" but: skill never deployed (checksum mismatch), VerificationTracker never wired (dead code), checkpoint file never created. Locally-correct agent work that doesn't compose into a working system. **Lesson:** "Phase: Complete" means agent believes it's done, not that the system-level integration works. Verification must check deployed artifacts, not just local code changes.

---

## History

- **Feb 2026:** Added human verification gate (explain-back) and batch completion anti-pattern
- **Jan 4, 2026:** Created after auditing all gates following the Dec 27 - Jan 2 spiral
- **Jan 4, 2026:** Repro verification gate disabled (was blocking completions)
- **Jan 4, 2026:** Dependency check gate disabled (was blocking completions)
