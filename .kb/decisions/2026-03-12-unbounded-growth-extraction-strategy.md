# Decision: Unbounded-Growth Extraction Strategy

**Status:** proposed
**Date:** 2026-03-12
**Authority:** architectural
**Blocks:** unbounded-growth, extraction, hotspot

---

## Context

The `unbounded-growth` defect class generated 4 investigations in 30 days (Feb 14 – Mar 3):

1. **2026-02-14:** CLAUDE.md accretion boundaries documentation
2. **2026-02-27:** Daemon code health audit → 3-phase extraction plan
3. **2026-02-28:** complete_cmd.go extraction → 4-phase extraction plan
4. **2026-03-01:** doctor.go extraction → 5-file extraction plan

All three extraction plans were executed successfully:
- `complete_cmd.go`: 2,267 → 349 lines (extracted to ~15 files)
- `doctor.go`: 1,736 → 269 lines (extracted to ~10 files)
- `daemon.go`: 1,180 → 197 lines

**Current state:** Zero CRITICAL files (>1,500 lines). The enforcement layers work. But 6 files are in the MODERATE zone (800–1,500), two approaching CRITICAL.

## Decision

### 1. Proactive Extraction Triggers (Not Just Reactive)

**Trigger extraction design at 1,200 lines** (not 1,500). This gives ~300 lines of headroom for the design→execute latency. The current 1,500-line gate blocks implementation work but by then the extraction is urgent and happens under pressure.

| Threshold | Action |
|-----------|--------|
| 800 lines | MODERATE advisory in spawn context (existing) |
| **1,200 lines** | **Daemon creates architect extraction issue automatically** |
| 1,500 lines | CRITICAL — spawn gate blocks feature-impl (existing) |

### 2. Priority Extraction Queue (Current Hotspots)

Based on distance-to-CRITICAL and growth velocity:

| Priority | File | Lines | Gap | Velocity | Action |
|----------|------|-------|-----|----------|--------|
| **P1** | `cmd/orch/harness_init.go` | 1,342 | 158 | 7 commits/30d | **Extract now** — will hit CRITICAL in ~1 month |
| **P2** | `cmd/orch/stats_cmd.go` | 1,256 | 244 | 11 commits/30d | **Design extraction** — fastest growth rate, will hit 1,200 trigger imminently |
| P3 | `pkg/opencode/client.go` | 1,040 | 460 | 10 commits/30d | Monitor — partially extracted recently, growth may slow |
| P4 | `pkg/events/logger.go` | 916 | 584 | 13 commits/30d | Monitor — highest velocity but large gap |
| P5 | `pkg/spawn/learning.go` | 979 | 521 | 1 commit/30d | No action — stable |
| P6 | `cmd/orch/handoff.go` | 898 | 602 | 2 commits/30d | No action — stable |

### 3. Extraction Pattern (Confirmed by 3 Successful Extractions)

The three completed extractions confirmed a consistent pattern:

1. **Architect investigation** identifies responsibility clusters and dependency graph
2. **Shared utilities extracted first** (types, helpers used across clusters)
3. **Domain files extracted in parallel** (one file per feature/mode/responsibility)
4. **Residual orchestrator** stays thin (100–350 lines)
5. **Tests split to match** (one test file per extraction file)

This pattern works. No changes needed to the extraction methodology.

### 4. Daemon Integration for Automated Detection

The defect-class pipeline (investigation 2026-02-26) is being activated. Once `kb reflect --type defect-class` flows through the daemon, the system will automatically create architect issues when 3+ investigations share the `unbounded-growth` tag within 30 days. This closes the manual detection loop.

## Consequences

- **Positive:** P1 and P2 extractions prevent emergency extraction under pressure
- **Positive:** 1,200-line trigger catches files before they block spawns
- **Risk:** Extraction work competes with feature work for agent capacity
- **Mitigation:** Extraction is typically 1-2 agent sessions (proven by the 3 completed extractions)

## Alternatives Considered

**A: Reactive-only (extract at 1,500 threshold):**
Rejected — the 4-investigation pattern shows this creates pressure. All three extractions were designed under time pressure (files already past CRITICAL).

**B: Lower the CRITICAL threshold to 1,200:**
Rejected — too aggressive. Many useful files are 800-1,200 lines without being problematic. The issue is uncontrolled growth, not absolute size.

**C: Automated extraction (agent generates split without architect review):**
Rejected — extraction requires understanding responsibility clusters and dependency graphs. The architect step is where value is created. The execution step is mechanical.

## References

- `.kb/investigations/2026-02-14-inv-add-claude-md-accretion-boundaries.md`
- `.kb/investigations/2026-02-27-audit-daemon-code-health-complexity.md`
- `.kb/investigations/2026-02-28-design-extraction-complete-cmd-go.md`
- `.kb/investigations/2026-03-01-design-doctor-go-extraction-plan.md`
- `.kb/investigations/2026-02-26-design-defect-class-pipeline-activation.md`
- `.kb/guides/code-extraction-patterns.md`
