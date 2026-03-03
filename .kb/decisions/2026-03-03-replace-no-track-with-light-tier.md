---
status: accepted
blocks:
  - keywords:
      - no-track
      - untracked agents
      - untracked spawn
      - lightweight tracking
      - cross-project spawn
    patterns:
      - "**/spawn_cmd.go"
      - "**/spawn_beads.go"
      - "**/untracked_sessions.go"
---

## Summary (D.E.K.N.)

**Delta:** Replace `--no-track` with lightweight tracking via the existing `--light` spawn tier. Every spawn creates a beads issue. Lightweight agents get `tier:light` label, skip non-essential verification (no SYNTHESIS.md, no test evidence), but remain visible to all operational tooling. Cross-repo spawns auto-create a local issue in the secondary repo with a back-reference.

**Evidence:** Architect investigation (`orch-go-owuk`) traced `--no-track` through the full lifecycle and found complete invisibility for Claude CLI agents in both lanes of the two-lane architecture. 5 distinct `isUntrackedBeadsID()` guards accumulated as symptoms. Probe `.kb/models/spawn-architecture/probes/2026-03-03-probe-no-track-invisible-agent-operational-cost.md` confirmed Claude CLI `--no-track` agents are invisible to `orch status`, `orch sessions`, `orch clean --orphans`, and daemon active count.

**Knowledge:** The pain from `--no-track` comes from conflating "lightweight verification" with "invisible to system." These are independent concerns. Separating them — lightweight verification via tier, universal visibility via beads — eliminates the orphan accumulation problem without adding ceremony to ad-hoc work. The existing `--light` tier already provides reduced verification; it just needs to replace the tracking escape hatch.

**Next:** Implement in 5 phases (see Implementation Sequence). Phase 1 is the critical change: `--no-track` becomes an alias for `--light` tier that creates a real beads issue. Monitor beads issue creation overhead (currently ~50ms, acceptable).

---

# Decision: Replace --no-track with Lightweight Tracking via --light Tier

**Date:** 2026-03-03
**Status:** Accepted
**Deciders:** Dylan (via orchestrator), architect agent (orch-go-owuk)
**Context Issues:** orch-go-owuk (architect), orch-go-7tyx (promotion)
**Source Investigation:** `.kb/investigations/2026-03-03-design-no-track-flag-removal-restriction-decision.md`
**Supersedes:** Quick entry on `--no-track` requiring `--reason` friction gate

---

## Context

`--no-track` was designed for ad-hoc exploratory work where beads tracking felt like overhead. In practice, it creates a class of agents that are operationally invisible:

- **`orch status`** — untracked agents don't have `orch:agent` label, invisible in tracked lane
- **`orch sessions`** — only works for OpenCode-backed agents; Claude CLI agents (default since Feb 19) invisible in untracked lane too
- **`orch clean --orphans`** — queries beads `orch:agent` label, misses untracked agents entirely
- **Daemon active count** — explicitly excludes untracked agents

Since Claude CLI became the default backend, `--no-track` agents are invisible in **both lanes** of the two-lane architecture (decision: 2026-02-18). They accumulate as orphans with no cleanup path.

The system accumulated 5 `isUntrackedBeadsID()` guards to prevent crashes when beads operations encounter synthetic `{project}-untracked-{timestamp}` IDs. Each guard is a symptom of the root cause: synthetic IDs that look like beads IDs but aren't.

**Cross-repo dependency:** The only legitimate use case for `--no-track` is cross-project epics where spawning in a secondary repo can't reference the primary repo's beads issue (beads is per-repo). This must be handled before removing `--no-track`.

---

## Decision

### 1. Every spawn creates a beads issue

No escape from tracking. The `--no-track` flag is deprecated and becomes an alias for `--light` tier during a transition period, then removed.

### 2. Collapse into existing --light tier

The `--light` tier already exists and provides reduced verification (no SYNTHESIS.md required). Extend it to be the replacement for `--no-track`:

| Concern | Old (`--no-track`) | New (`--light` tier) |
|---------|-------------------|---------------------|
| Beads issue | Synthetic placeholder ID | Real issue, `tier:light` label |
| `orch status` visibility | Invisible | Visible |
| `orch clean` | Missed by orphan GC | Handled normally |
| Daemon active count | Excluded | Counted (with per-tier concurrency caps) |
| Verification on complete | Broken (`bd comment` fails) | Reduced (skip SYNTHESIS.md, skip test evidence) |
| Phase: Complete | Not enforceable | Required for clean closure |

### 3. Concurrency caps per tier

All tiers get concurrency caps in daemon config:

| Tier | Default cap | Rationale |
|------|-------------|-----------|
| `light` | 3 | Quick tasks, low resource |
| `standard` | 2 | Normal work |
| `deep` | 1 | Long-running, resource-intensive |

This replaces the current flat `--no-track` exclusion with nuanced capacity management.

### 4. Cross-repo auto-issue creation

When `--workdir` targets a different project and `--issue` references the source project:
- Auto-create a `tier:light` issue in the target project's beads
- Include back-reference to source issue in description
- Auto-close on `orch complete`

This eliminates the need for `--no-track` in cross-project epics.

### 5. Deprecation path

- **Phase 1:** `--no-track` silently creates a real beads issue with `tier:light`. Deprecation warning emitted.
- **Phase 2:** After one release cycle, remove `--no-track` flag entirely.

---

## Alternatives Considered

### A: Remove --no-track entirely (hard break)

Rejected. Zero migration period is unnecessary friction when the system has a single user. Soft migration via alias achieves the same end state without breakage.

### B: Keep --no-track but add cleanup automation

Rejected. This adds a 6th special-case handler to the existing 5. Principle "Coherence over patches" — the design needs redesign, not another patch.

### C: Restrict --no-track to cross-repo only

Rejected. Still creates invisible agents, just fewer of them. Doesn't solve the root cause.

### D: Auto-close lightweight agents on tmux window exit

Rejected for Phase 1. Beads should be source of truth for lifecycle (Two-Lane decision), not tmux events. May revisit if orphan GC proves insufficient.

---

## Implementation Sequence

**Phase 1 (small):** `--no-track` becomes deprecated alias for `--light` tier. When either is used, create a real beads issue with `tier:light` label instead of synthetic ID. Remove `--reason` requirement (real issue replaces the friction gate).

**Phase 2 (small):** Update `orch complete` to detect `tier:light` label and skip SYNTHESIS.md verification, skip test evidence check. Keep Phase: Complete requirement.

**Phase 3 (small):** Update orphan GC — lightweight agents now have real beads issues, so existing GC should work. Remove `isUntrackedBeadsID()` guards from abandon/complete. Remove daemon untracked exclusion (replace with per-tier concurrency caps).

**Phase 4 (medium):** Cross-repo spawn: when `--workdir` is set without `--issue`, auto-create issue in target project's beads with back-reference.

**Phase 5 (cleanup):** Remove `--no-track` flag, `isUntrackedBeadsID()`, `untracked_sessions.go` untracked category, daemon exclusion logic. Simplify `orch sessions` to show orchestrator sessions only.

---

## Blocking Questions Resolved

**Q: Should --light be explicit or automatic?**
Automatic — inferred from skill tier in spawn config. The `--light` flag exists for manual spawns, but daemon-spawned agents inherit tier from skill metadata. No new flag proliferation.

**Q: Should lightweight agents count in daemon active count?**
Yes, with per-tier caps. Excluding them (current `--no-track` behavior) is the source of the invisibility problem. Per-tier caps prevent lightweight agents from consuming all capacity.

---

## Principles Applied

- **Evolve by distinction** (`.kb/principles.md`) — separates "lightweight verification" from "invisible to system"
- **Coherence over patches** — removes 5+ special-case handlers instead of adding more
- **Session amnesia** — lightweight issues survive sessions; `--no-track` agents don't
- **Gate over remind** — beads issue creation is a gate; `--reason` friction was a reminder

---

## References

- **Architect investigation:** `.kb/investigations/2026-03-03-design-no-track-flag-removal-restriction-decision.md`
- **Probe:** `.kb/models/spawn-architecture/probes/2026-03-03-probe-no-track-invisible-agent-operational-cost.md`
- **Two-Lane decision:** `.kb/decisions/2026-02-18-two-lane-agent-discovery.md` (acknowledges --no-track as pressure point)
- **Cross-project epic investigation:** `.kb/investigations/archived/epic-management-deprecated/2025-12-21-inv-cross-project-epic-orchestration-patterns.md`
- **Untracked cleanup investigation:** `.kb/investigations/archived/2026-01-04-inv-untracked-agents-cleanup-path-problem.md`
