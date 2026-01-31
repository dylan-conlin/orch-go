# Decision: Bloat Control Enforcement Patterns

**Date:** 2026-01-30
**Status:** Accepted
**Context:** Synthesized from investigations 2026-01-17-inv-design-800-line-bloat-gate.md and 2026-01-23-inv-design-bloat-control-system-800.md

## Summary

Bloat control uses two-layer approach: (1) `orch hotspot` with `bloat-size` type for detection, (2) CI gate + spawn-time context injection for enforcement. Detection belongs in hotspot system (coherence signal, not context gap). Enforcement uses Gate Over Remind principle via CI workflow and spawn-time surfacing.

## The Problem

Current state:
- 42 files over 800 lines (bloat threshold)
- 12 critical files over 1500 lines (spawn_cmd.go: 2630 lines)
- No enforcement mechanism - detection exists but can be ignored
- 800-line gate from principles defines "Context Noise" threshold but lacks teeth

Pain point: Bloat accumulates unchecked. Agents work on bloated files without awareness. No gate prevents further bloat.

## The Decision

### Detection Layer: bloat-size Hotspot Type

Add `bloat-size` as third hotspot type in `orch hotspot`:

**Why `orch hotspot` (not `orch learn`):**
- Bloat is coherence signal (like fix-density), not context gap
- `orch hotspot` already has extensible architecture for detection types
- Existing exclusion patterns (test files, generated code) apply to bloat too

**Implementation:**
- Add `analyzeBloatFiles()` function returning `[]Hotspot` with `Type: "bloat-size"`
- CLI: `orch hotspot --bloat-threshold 800` (default)
- Severity-based recommendations:
  - 800-1500 lines: "Recommend extraction - see .kb/guides/code-extraction-patterns.md"
  - >1500 lines: "CRITICAL: Recommend architect session for structural redesign"
- Include in API /api/hotspot for dashboard visibility

### Enforcement Layer: CI Gate + Spawn-Time Context

Two enforcement mechanisms:

**1. CI Gate (True Enforcement):**
- GitHub Actions workflow `.github/workflows/bloat-check.yml`
- Fails if modified file exceeds 800 lines OR total project bloat increases
- Cannot be bypassed (unlike `--no-verify` for pre-commit hooks)
- Runs outside sandbox (no beads corruption risk)

**2. Spawn-Time Context Injection (Surfacing at Context):**
- Modify `pkg/spawn/context.go` to check bloat via `orch hotspot`
- Inject warning into SPAWN_CONTEXT.md when target file >800 lines
- Shows extraction recommendations before agent starts work
- Surfacing, not gating - agent can proceed but is informed

## Why This Design

### Principle: Coherence Over Patches

From principles.md: "When fixes accumulate in the same area, the problem isn't insufficient fixing - it's a missing coherent model."

Bloat is about code coherence (too much in one place), not missing knowledge (what `orch learn` tracks). This system boundary clarity keeps tools focused.

### Principle: Gate Over Remind

From principles.md: "Reminders fail under cognitive load. Gates make capture unavoidable."

CI gate provides true enforcement. Spawn-time is surfacing (reminder with context), not gating, but surfaces when context exists (agent is starting work on that file).

### Principle: Capture at Context

Spawn-time injection surfaces bloat issues when agent has context to act on it - right as they're starting work on the bloated file.

### Lesson: Beads Corruption from Complex Hooks

Investigation 2026-01-21 (beads corruption) showed hooks that interact with beads in complex ways are risky. Pre-commit hooks would:
- Run in sandbox (chmod fails)
- Interact with beads potentially
- Be bypassable with `--no-verify`

CI runs outside sandbox, doesn't touch beads, can't be bypassed.

## Trade-offs

**Accepted:**
- Spawn-time is surfacing not gating (agent can still work on bloated file)
- CI gate catches after local development (some wasted work possible if PR fails)
- Severity thresholds (800/1500) are heuristics, may need tuning

**Rejected:**
- Pre-commit hooks: Run in sandbox, bypassable, slow for large files
- Issue-level gates: Bad mapping between issues and files, complex beads interaction
- Spawn-time blocking gate: Prevents agents from fixing existing bloat

## Constraints

1. **Never use pre-commit hooks for bloat** - Beads corruption lesson requires minimal hook complexity
2. **Bloat detection in hotspot, not learn** - System boundary: hotspot = code health, learn = context gaps
3. **800-line threshold is gate** - From `.kb/models/extract-patterns.md`, below threshold is encouraged, above is problematic
4. **Severity escalation at 1500 lines** - Above this likely requires architect involvement, not just extraction

## Implementation Notes

**Phase 1: Detection (feat-049)**
- File: `cmd/orch/hotspot.go` - Add `analyzeBloatFiles()` function (~100 lines)
- Reuse `shouldCountFileWithExclusions()` for exclusion logic
- Add `--bloat-threshold` flag (default: 800)
- Include in existing `/api/hotspot` endpoint

**Phase 2: Spawn-Time Injection**
- File: `pkg/spawn/context.go` - Add bloat check before spawn
- Quick line count or cache hotspot results
- Inject warning + link to `.kb/guides/code-extraction-patterns.md`

**Phase 3: CI Gate**
- File: `.github/workflows/bloat-check.yml`
- Fails if modified file exceeds 800 lines
- Distinguish "already bloated" from "this PR made it bloated"
- Handle test files specially (expected to be long)

**Success Criteria:**
- `orch hotspot` shows bloat-size hotspots for files >800 lines
- Spawned agents see bloat warning when target file >800 lines
- PRs that increase file past 800 lines fail CI
- Over 6-month period, total bloated file count decreases

## References

**Investigations:**
- `.kb/investigations/2026-01-17-inv-design-800-line-bloat-gate.md` - Detection design
- `.kb/investigations/2026-01-23-inv-design-bloat-control-system-800.md` - Enforcement design
- `.kb/investigations/2026-01-21-inv-investigate-beads-sqlite-database-corruption.md` - Hook complexity lesson

**Models:**
- `.kb/models/extract-patterns.md` - 800-line gate definition and rationale
- `.kb/guides/code-extraction-patterns.md` - Extraction workflow and benchmarks

**Principles:**
- Coherence Over Patches - `~/.kb/principles.md:422-463`
- Gate Over Remind - `~/.kb/principles.md:162-189`
- Capture at Context - Implicit in spawn-time injection design
