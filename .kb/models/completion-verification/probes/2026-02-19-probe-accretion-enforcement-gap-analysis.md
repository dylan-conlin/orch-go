# Probe: Accretion Enforcement 4-Layer Gap Analysis

**Model:** completion-verification
**Date:** 2026-02-19
**Status:** Complete

---

## Question

The Feb 14 investigation (`.kb/investigations/2026-02-14-inv-architect-design-accretion-gravity-enforcement.md`) designed a 4-layer accretion enforcement system. What actually shipped, and are there gaps between design and implementation?

---

## What I Tested

Checked each of the 4 designed layers against actual code and ran tests to verify behavior.

```bash
# Layer 1: Spawn Gates
# Read pkg/spawn/gates/hotspot.go (CheckHotspot function)
# Read pkg/orch/extraction.go:385-390 (RunPreFlightChecks hotspot handling)
# Searched for --force-hotspot flag (not found)

# Layer 2: Completion Gates
go test ./pkg/verify/ -run TestVerifyAccretion -v -count=1
# 7/7 tests pass

# Layer 3: Coaching Plugin
# Read plugins/coaching.ts:1416-1512 (accretion detection)
# Read plugins/coaching.ts:645-673 (warning message templates)

# Layer 4: CLAUDE.md
# Read CLAUDE.md:120-124 (Accretion Boundaries section)
```

---

## What I Observed

### Layer 1: SPAWN GATES — PARTIAL

**What shipped:**
- Hotspot detection refactored into `pkg/spawn/gates/hotspot.go` (52 lines, clean interface)
- `CheckHotspot()` function takes skill name and daemon flag
- Integrated at spawn via `RunPreFlightChecks` in `pkg/orch/extraction.go:354-393`

**What's missing (from design):**
- Still WARNING-ONLY — `pkg/orch/extraction.go:387` literally comments: "Warning shown but spawn proceeds (non-blocking)"
- `CheckHotspot()` prints to stderr and returns result, but `RunPreFlightChecks` ignores the result (line 389 — return value discarded)
- No `--force-hotspot` override flag
- No skill-based exemptions (design specified: exempt architect, investigation, capture-knowledge, codebase-audit)
- No blocking for feature-impl on CRITICAL (>1500 line) files
- Design spec: "Block when: `hotspotResult.HasHotspots && maxBloatScore >= 1500 && !isExemptSkill`"

**Documentation mismatch:** CLAUDE.md:124 claims "Spawn gates block feature-impl on CRITICAL files" but this is aspirational — the code does NOT block.

### Layer 2: COMPLETION GATES — SHIPPED

**What shipped (matches design exactly):**
- `GateAccretion = "accretion"` constant at `pkg/verify/check.go:26`
- `VerifyAccretionForCompletion(workspacePath, projectDir)` at `pkg/verify/accretion.go:53-128`
- Thresholds: warning at 800 lines, error at 1500 lines, delta threshold 50 lines (constants at lines 37-39)
- Net-negative delta auto-pass for extraction work (line 84-89)
- Orchestrator tier skip (check.go:416)
- Integrated into `VerifyCompletionFull()` at check.go:413-426, after git_diff gate
- Helper functions: `getGitDiffWithLineCounts()`, `getFileLineCount()`, `isSourceFile()`
- Source file filter: `.go`, `.ts`, `.tsx`, `.js`, `.jsx`, `.py`, `.rb`, `.java`, `.c`, `.cpp`, `.h`, `.cs`, `.svelte`, `.vue`
- Excludes: vendor/, node_modules/, dist/, build/, generated files

**Test coverage:** 7 test cases all passing (0.742s):
- Small file + small change passes
- Large file + small change passes (below delta threshold)
- File >800 lines + 50 net lines → warning
- File >1500 lines + 50 net lines → error (blocks completion)
- Extraction work (net negative delta) → pass
- Multiple files, mixed results
- Net negative across all files → pass

### Layer 3: COACHING PLUGIN — SHIPPED

**What shipped (matches design exactly):**
- Accretion detection in `tool.execute.after` at coaching.ts:1416-1512
- Triggers on `edit` or `write` tool calls
- Gets file line count via `getFileLineCount()` (line 1423)
- Thresholds: `AccretionWarningThreshold = 800`, `AccretionCriticalThreshold = 1500` (lines 61-62)
- `AccretionState` interface (line 443): `fileEditCounts`, `fileWarningInjected`, `fileStrongWarningInjected` Maps
- Per-session tracking in `workerSessionState.accretion` (line 1453-1457)
- Tiered injection (matching frame collapse pattern):
  - 1st edit to >800 line file → `accretion_warning` (lines 1491-1497)
  - 3+ edits to same file → `accretion_strong` (lines 1498-1505)
- Warning messages at lines 645-673:
  - `accretion_warning`: explains gravity pattern, links to extraction guide and `orch hotspot`
  - `accretion_strong`: "STOP adding", "EXTRACT logic", references completion gates will block
- Metrics written to `~/.orch/coaching-metrics.jsonl` with `metric_type: "accretion_warning"` (line 1475)
- Only fires for worker sessions (inside worker detection block, returns at line 1515)

### Layer 4: CLAUDE.md BOUNDARIES — SHIPPED

**What shipped:**
- "Accretion Boundaries" section at CLAUDE.md:120-124
- Rule: "Files >1,500 lines require extraction before feature additions"
- References: `orch hotspot`, `.kb/guides/code-extraction-patterns.md`
- Enforcement claim: "Spawn gates block feature-impl on CRITICAL files; completion gates warn on additions >50 lines to files >800 lines"

**Issue:** The enforcement claim about spawn gates blocking is inaccurate (Layer 1 is warning-only). Completion gates section is accurate.

---

## Model Impact

- [x] **Confirms** invariant: Completion verification has tiered gates (warning at 800, error at 1500) with net-negative extraction bypass — the GateAccretion implementation exactly matches the design's Fork 1 (Option B: tiered thresholds) and Fork 3 (Option B: net-negative delta passes).
- [x] **Extends** model with: The completion-verification model should track that GateAccretion is now a live gate (joining the existing 8+ gates). It's integrated after git_diff and before build verification. The accretion gate has separate source/test files (`pkg/verify/accretion.go`, `pkg/verify/accretion_test.go`).
- [x] **Contradicts** CLAUDE.md claim: "Spawn gates block feature-impl on CRITICAL files" is aspirational, not implemented. The spawn hotspot check at `pkg/spawn/gates/hotspot.go:40` only prints warnings. `RunPreFlightChecks` at `pkg/orch/extraction.go:389` discards the result.

---

## Gap Summary Table

| Layer | Design Status | Implementation Status | Gap |
|-------|--------------|----------------------|-----|
| 1. Spawn Gates | Block feature-impl on CRITICAL, exempt architects, --force-hotspot | Warning-only, no blocking, no exemptions, no override flag | **PARTIAL — blocking logic not implemented** |
| 2. Completion Gates | GateAccretion, 800/1500 thresholds, net-negative bypass | Fully implemented with tests (7/7 passing) | **SHIPPED** |
| 3. Coaching Plugin | Accretion detection in tool.execute.after, tiered warnings | Fully implemented with thresholds + injection | **SHIPPED** |
| 4. CLAUDE.md | Document accretion boundaries section | Section present with rule + references | **SHIPPED (with inaccurate spawn gate claim)** |

## Remaining Work

1. **Spawn Gate Blocking (Layer 1)**: Convert `CheckHotspot()` from warning-only to conditional blocking:
   - Block when skill is `feature-impl` or `systematic-debugging` AND hotspot is CRITICAL (>1500 lines)
   - Exempt skills: `architect`, `investigation`, `capture-knowledge`, `codebase-audit`
   - Add `--force-hotspot` flag for explicit override
   - Return error from `RunPreFlightChecks` when blocking conditions met

2. **Fix CLAUDE.md claim**: Either implement spawn blocking (making the claim true) or soften the claim to "Spawn gates warn on feature-impl targeting CRITICAL files" (matching current behavior).

---

## Notes

- spawn_cmd.go is now 799 lines (down from 2,332 referenced in the design) — significant extraction already happened, ironically demonstrating the extraction workflow this system enforces.
- The hotspot gate infrastructure was cleanly refactored into `pkg/spawn/gates/hotspot.go` during or after the design, suggesting the gate abstraction was built to support blocking even though blocking wasn't wired up.
- 3 of 4 layers fully shipped is strong progress. The remaining spawn gate work is well-scoped (~30 lines of gate logic in `RunPreFlightChecks`).
