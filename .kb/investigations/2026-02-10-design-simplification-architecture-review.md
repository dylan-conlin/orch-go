# Investigation: Simplification Architecture Review

**Question:** Is there a fundamentally simpler architecture for orch-go that avoids problems rather than patching around them? What is essential vs accidental complexity?

**Started:** 2026-02-10
**Updated:** 2026-02-10
**Owner:** architect worker (orch-go-qv699)
**Phase:** Complete
**Status:** Complete
**Defect Class:** configuration-drift

**TLDR:** The current complexity (14 gates, 3 spawn modes, layered zombie defense, etc.) exists primarily to compensate for failures at system boundaries — not from essential requirements. The architecture should shift from "daemon-first autonomous" to "supervised-with-batch-mode" by making two key changes: (1) fix the orch-OpenCode process lifecycle boundary, and (2) reduce verification gates to an essential set while adding the missing commit-evidence gate. The dual-mode (tmux+HTTP) architecture is confirmed correct.

---

## Problem Framing

### The Question

Orch-go has accumulated:
- 14 verification gates
- 3 spawn modes (headless, tmux, docker)
- Worktree isolation
- Process ledgers (never populated for current spawn paths)
- Coaching plugins (disabled)
- Model-specific bypass profiles (caused ghost completions)
- Layered zombie defense (5 layers, all broken)
- 580-line CLAUDE.md

Much of this complexity exists to compensate for failures in adjacent systems:
- OpenCode crashes (5 in recent days)
- GPT behavioral gaps (no commits, missed Phase:Complete)
- Shared working tree conflicts (now fixed by worktrees)
- Process lifecycle gaps (orch owns session, OpenCode owns process, neither terminates)

### Recent Incidents (Evidence of Failure Modes)

| Incident | Root Cause | Complexity Created |
|----------|------------|-------------------|
| 22 ghost completions | No commit gate + GPT model bypass + shared working tree | Model-specific bypass profiles |
| 5 OpenCode crashes | Various (jetsam, storage race, etc.) | Layered restart detection |
| 3 zombie RAM exhaustion | Process ledger empty, orphan pattern stale | 5-layer zombie defense |
| OAuth token loss | No detection of auth.json changes | (not addressed yet) |
| 30-min reap destroyed UI | Careless automation without scope limits | (not addressed yet) |

### Success Criteria

A simpler architecture should:
1. Reduce the number of components that can fail silently
2. Fail visibly when something goes wrong (not 22 issues closed with no commits)
3. Have fewer layers of compensating complexity
4. Make the daemon optional (batch mode) rather than primary

---

## Exploration: Essential vs Accidental Complexity

### Essential Complexity (Cannot Be Removed)

| Component | Why Essential |
|-----------|---------------|
| **Multi-model access** | Different providers require different auth paths |
| **Dual-mode (tmux + HTTP)** | Decision confirmed: each serves irreplaceable needs |
| **Worktree isolation** | Just landed, prevents agent interference |
| **Issue tracking (beads)** | Integral for coordination, cannot remove |
| **Some verification** | Need *some* completion verification |
| **Spawn context** | Agents need context to do their work |
| **Dashboard** | Visibility into system state |

### Accidental Complexity (Can Be Simplified/Removed)

| Component | Why Accidental | Evidence |
|-----------|---------------|----------|
| **14 verification gates** | Many redundant or unused | Phase:Complete bypassed for GPT models |
| **Model-specific bypass profiles** | Compensates for model behavior gaps | GPT bypass enabled ghost completions |
| **Process ledger** | Never populated for current spawn paths | `~/.orch/process-ledger.jsonl` is 0 bytes |
| **Stale orphan detection** | Pattern `"run --attach"` doesn't match current `opencode attach` | Investigation 2026-02-10 |
| **Coaching plugins** | Disabled by decision | Decision 2026-01-28 |
| **580-line CLAUDE.md** | Contains knowledge that should be in guides/models | Symptom of drift |
| **5-layer zombie defense** | All layers broken, none functional | Investigation 2026-02-10 |

### Wrong System Boundaries

| Boundary Issue | Symptom | Impact |
|----------------|---------|--------|
| **orch owns session, OpenCode owns process** | Session deleted, bun process keeps running | Zombie accumulation |
| **No commit verification gate** | Agents can "complete" without committing | 22 ghost completions |
| **OAuth detection** | Tokens disappear, silent fallback to pay-per-token | Unexpected costs |
| **Daemon vs UI scope** | 30-min reaper killed Dylan's primary UI session | User harm |

---

## Decision Forks

### Fork 1: Primary Operating Mode

**Question:** Should the daemon-driven autonomous model be the default, or should supervised manual use be primary?

**Options:**
- A: **Daemon-first** — Keep current architecture, fix verification gaps
- B: **Supervised-first** — Manual/tmux primary, daemon as opt-in batch mode
- C: **Hybrid** — Daemon for workers, supervised for orchestrators

**Substrate says:**
- Principle: "Verification bottleneck principle" — tight verification at completion is cheaper than wide supervision during execution
- Principle: "Escape hatches" — critical paths need independent secondary paths
- Evidence: Dylan's concern — "this system is just too fragile as is"
- Evidence: Ghost completions happened because daemon trusted components that fail silently

**RECOMMENDATION: Option B — Supervised-first with daemon as batch mode**

**Reasoning:**
1. Current incidents show daemon auto-closes work that shouldn't be closed
2. Making daemon opt-in for batch work reduces blast radius
3. Supervised mode (tmux) provides visibility when things go wrong
4. Already have the infrastructure (dual-mode confirmed correct)

**Trade-off accepted:** Less autonomous means more orchestrator involvement for spawning
**When this would change:** If all verification gaps are fixed AND stability reaches 7+ days without intervention

---

### Fork 2: Verification Gate Simplification

**Question:** Which of the 14 verification gates are essential?

**Current gates (from investigation):**
1. phase_complete
2. synthesis
3. handoff_content
4. constraint
5. phase_gate
6. skill_output
7. visual_verification
8. test_evidence
9. model_connection
10. verification_spec
11. git_diff
12. build
13. decision_patch_limit
14. dashboard_health

**Options:**
- A: **Keep all 14** — Maximum safety
- B: **Reduce to core 5** — phase_complete, synthesis, commit_evidence (new), test_evidence, git_diff
- C: **Reduce to core 3** — phase_complete, commit_evidence (new), synthesis

**Substrate says:**
- Principle: "Coherence over patches" — if 5+ fixes hit the same area, redesign
- Evidence: Many gates are unused or bypassed (GPT model bypass for phase_complete)
- Evidence: The missing gate (commit_evidence) caused the worst incident

**RECOMMENDATION: Option B — Core 5 gates with new commit_evidence gate**

The essential gates:
1. **commit_evidence** (NEW, CRITICAL) — Verify actual git commits exist
2. **phase_complete** — Agent reports completion (no more model-specific bypasses)
3. **synthesis** — SYNTHESIS.md exists
4. **test_evidence** — Test output in completion comment
5. **git_diff** — Files claimed match files changed

**Trade-off accepted:** Some edge cases may slip through
**When this would change:** If specific incidents show a removed gate was needed

---

### Fork 3: Process Lifecycle Fix

**Question:** Where should process termination happen — orch side, OpenCode side, or both?

**Options:**
- A: **orch-only** — Kill tmux window, broader orphan detection
- B: **OpenCode-only** — Session.remove() kills process
- C: **Both** — Defense in depth

**Substrate says:**
- Principle: "Escape hatches" — critical paths need independent secondary paths
- Decision: Two-tier cleanup required
- Evidence: Current single-tier (detection only) is completely broken

**RECOMMENDATION: Option C — Both layers**

1. **orch side:** Fix orphan detection pattern (`./src/index.ts` not `run --attach`), add startup sweep
2. **OpenCode side:** Session.remove() signals attached bun processes to terminate

**Trade-off accepted:** Requires OpenCode fork change
**When this would change:** If OpenCode upstream provides session→process lifecycle

---

### Fork 4: CLAUDE.md Simplification

**Question:** How should the 580-line CLAUDE.md be simplified?

**Options:**
- A: **Keep as-is** — It works
- B: **Extract to guides** — Move procedural content to `.kb/guides/`, keep only orientation
- C: **Split by audience** — Separate orchestrator vs worker content

**Substrate says:**
- Principle: "Session amnesia" — persistent artifacts beat in-context instruction
- Evidence: 580 lines is cognitive overload; agents may not read all of it
- Pattern: Guides already exist for most topics

**RECOMMENDATION: Option B — Extract to guides, CLAUDE.md becomes ~100 line orientation**

CLAUDE.md should contain:
1. Project overview (what is orch-go)
2. Forked dependencies (OpenCode, beads)
3. Build/test commands
4. Pointer to guides for details

Everything else moves to:
- `.kb/guides/spawn.md` — spawn details
- `.kb/guides/daemon.md` — daemon details (already exists)
- `.kb/guides/dashboard-architecture.md` — dashboard details (already exists)
- etc.

**Trade-off accepted:** Requires agents to navigate to guides
**When this would change:** If agents consistently fail to find relevant guides

---

## Synthesis: Recommended Simplification Plan

### Phase 0: Remove Dead Code (Immediate, 1 day)

| Action | Why |
|--------|-----|
| Remove coaching plugin code | Already disabled by decision |
| Remove model-specific bypass profiles | Caused ghost completions, not needed |
| Delete empty process ledger references | Never populated, misleading |
| Update orphan detection pattern | `"./src/index.ts"` not `"run --attach"` |

### Phase 1: Fix Critical Boundary (Priority, 1-2 days)

| Action | Why |
|--------|-----|
| Add GateCommitEvidence | Prevent ghost completions |
| Fix orch complete to kill tmux window before session delete | Stop zombie creation |
| Add startup sweep | Clean orphans on orch serve start |

### Phase 2: Simplify Verification (After Phase 1, 1 day)

| Action | Why |
|--------|-----|
| Reduce to core 5 gates | Less cognitive load, clearer failures |
| Remove phase_complete model bypass | All models must report completion |
| Make gate failures block completion (no auto-pass) | Visible failures |

### Phase 3: Simplify CLAUDE.md (When stable, 1 day)

| Action | Why |
|--------|-----|
| Extract spawn details to guides | Already have spawn.md |
| Extract zombie defense to guides | Procedural, not orientation |
| Extract event tracking to guides | Procedural |
| Target: ~100 lines | Orientation only |

### Phase 4: Shift to Supervised-First (After Phase 1-3 proven)

| Action | Why |
|--------|-----|
| Document supervised mode as primary | Set expectations |
| Make daemon opt-in (remove auto-start) | Reduce blast radius |
| Add `orch batch` command for explicit batch runs | Clear intent |
| Daemon only runs when explicitly requested | User control |

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Remove dead code (coaching plugins, bypasses) | implementation | Single-component, reversible |
| Add GateCommitEvidence | architectural | New gate affects daemon, complete, review |
| Fix orphan detection pattern | implementation | Pattern update, no new architecture |
| Add startup sweep | architectural | New lifecycle phase |
| Reduce to core 5 gates | architectural | Changes verification semantics |
| Simplify CLAUDE.md | implementation | Documentation, no code changes |
| Supervised-first shift | strategic | Changes operational model |

### Recommended Approach ⭐

**Incremental simplification with stability validation at each phase.**

1. Phase 0 (immediate) cleans up dead code — low risk, immediate clarity
2. Phase 1 (priority) fixes the critical boundary — addresses root cause of ghost completions and zombies
3. Phase 2 (after Phase 1 proven) simplifies verification — reduces cognitive load
4. Phase 3 (when stable) simplifies docs — reduces onboarding overhead
5. Phase 4 (after stability proven) shifts operational model — reduces blast radius

**Why this sequence:**
- Phase 0 has zero risk (removing dead code)
- Phase 1 addresses the actual failures (22 ghost completions, zombies)
- Each phase validates stability before the next
- Phase 4 (supervised-first) is the most significant change, so it comes last with full validation

### Things to Watch Out For

- ⚠️ Don't remove gates that are actually needed — validate with test runs first
- ⚠️ GateCommitEvidence must handle investigation-only agents (no code changes expected)
- ⚠️ Startup sweep may race with new spawns — use grace period (30s)
- ⚠️ Supervised-first shift may slow throughput — acceptable trade-off for reliability

### Success Criteria

- ✅ Zero ghost completions over 1 week
- ✅ Zero zombie bun processes after 1 week
- ✅ CLAUDE.md reduced to ~100 lines
- ✅ Dashboard shows clear state (no untracked agents, no clogged slots)
- ✅ Dylan reports system feels "solid"

---

## Decision Gate Guidance

**Add `blocks:` frontmatter when promoting to decision:**
- This addresses recurring failures (ghost completions, zombies)
- Establishes constraints future agents might violate (verification gates)
- Future spawns involving daemon or verification should see this

**Suggested blocks keywords:**
- "daemon", "verification", "completion", "ghost"
- "zombie", "process", "cleanup"
- "architecture", "simplification"

---

## References

### Investigations Reviewed

- `.kb/investigations/2026-02-09-inv-post-mortem-daemon-overnight-ghost-completions.md` — 22 ghost completions root cause
- `.kb/investigations/2026-02-10-design-zombie-bun-root-cause-architecture.md` — Zombie process root cause
- `DYLANS_THOUGHTS.org` — Dylan's concerns about system fragility

### Decisions Consulted

- `2026-02-07-unbounded-resource-consumption-constraints.md` — C1-C5 constraints
- `2026-01-14-two-tier-cleanup-pattern.md` — Two-tier cleanup principle
- `2026-01-28-coaching-plugin-disabled.md` — Coaching plugins already disabled
- `2026-01-09-dual-spawn-mode-architecture.md` — Dual-mode confirmed correct

### Models Consulted

- `.kb/models/daemon-autonomous-operation.md` — Daemon poll-spawn-complete cycle
- `.kb/models/system-reliability-feb2026.md` — Unbounded resource pattern
- `.kb/models/completion-verification.md` — Current verification architecture

### Open Issues (Related)

- orch-go-qv699 — This architecture review
- orch-go-w4pj9 — OpenCode boundary fix (process termination)
- orch-go-6v2ta — OpenCode crashes (5 recent)
- orch-go-n1kpb — Careless agent behavior (30-min reap)
- orch-go-cmdfh — Audit ghost completion work loss
