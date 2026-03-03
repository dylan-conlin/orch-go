# Investigation: Should --no-track Be Removed, Restricted, or Kept?

**Question:** Should the `--no-track` spawn flag be removed, restricted, or kept as-is, given that untracked workers are invisible to orch status/complete/frontier and accumulate as orphans?

**Started:** 2026-03-03
**Updated:** 2026-03-03
**Owner:** Architect agent (og-arch-orientation-frame-dylan-03mar-5630)
**Phase:** Complete
**Status:** Complete
**Confidence:** High (85%)

---

## Problem Framing

**Design Question:** What should happen to the `--no-track` spawn flag?

**Success Criteria:**
- Every spawned agent is visible in at least one operational view
- Cross-project epic pattern continues to work
- No orphan accumulation from invisible agents
- Cleanup commands can handle all agent types

**Constraints:**
- Cross-project epics currently depend on `--no-track` (Option A pattern from 2025-12-21 investigation)
- Two-Lane decision (2026-02-18) intentionally separates tracked/untracked into separate views
- Beads is per-repo — cannot create issues in secondary repos from primary repo spawn
- `--no-track` already requires `--reason` (min 10 chars) as friction

**Scope:**
- IN: Evaluate 4 options, recommend one, identify migration path for cross-repo epics
- OUT: Implement anything, redesign cross-repo beads

---

## Evidence Gathered

### E1: --no-track generates synthetic beads IDs that fail downstream

When `--no-track` is used, `determineBeadsID()` generates `{project}-untracked-{timestamp}` (e.g., `orch-go-untracked-1735947123`). These synthetic IDs:
- **Fail** `bd comment` commands (no beads issue exists) — progress tracking is broken
- **Fail** `verify.GetIssue()` — completion verification breaks
- **Required special handling** in `orch abandon` and `orch complete` (`isUntrackedBeadsID()` guard, investigation 2026-01-04)
- **Excluded** from daemon active count explicitly (active_count.go:18)
- **Excluded** from `orch status` tracked lane — only visible in `orch sessions` untracked lane

**Source:** `pkg/orch/spawn_beads.go:74-75`, `cmd/orch/spawn_cmd.go:860-866`, `pkg/daemon/active_count.go:18`

### E2: Cross-project epics are the primary legitimate use case

The 2025-12-21 investigation established Option A (ad-hoc spawns with `--no-track` in secondary repos, manual `bd close` with commit refs) as the current cross-project epic pattern. Example: orch-go-ivtg epic with 3/5 children requiring kb-cli work.

**Why --no-track:** Beads issues are per-repo. When spawning in a secondary repo, you can't reference the primary repo's beads issue. `--no-track` avoids creating a duplicate issue in the secondary repo.

**Source:** `.kb/investigations/archived/epic-management-deprecated/2025-12-21-inv-cross-project-epic-orchestration-patterns.md`

### E3: Two-Lane decision acknowledges untracked as a pressure point

The Two-Lane ADR (2026-02-18) explicitly calls out:
> `--no-track` agents are invisible
> These are the exact gaps that triggered cache-building in January. Without addressing them, the cycle restarts.

The two-lane split (tracked in `orch status`, untracked in `orch sessions`) was the structural solution. But `orch sessions` requires the OpenCode server — Claude CLI agents (the default backend since Feb 19) don't create OpenCode sessions, so `--no-track` Claude CLI agents are invisible in **both** lanes.

**Source:** `.kb/decisions/2026-02-18-two-lane-agent-discovery.md:28-29`

### E4: --no-track already has friction gates

Current friction: `--no-track` requires `--reason` (min 10 chars) when used with `--bypass-triage`. The daemon never uses `--no-track`. Only manual spawns can use it.

**Source:** `cmd/orch/spawn_cmd.go:519-537`

### E5: Cleanup automation exists but doesn't cover untracked adequately

- `orch clean --orphans` uses LifecycleManager which queries beads `orch:agent` label — untracked agents don't have this label, so they're invisible to orphan GC
- `orch clean --sessions` only cleans stale tmux windows, and only if the beads issue is closed — untracked agents don't have beads issues
- OpenCode session TTL cleanup handles headless untracked sessions, but Claude CLI agents have no sessions to clean

**Source:** `cmd/orch/clean_cmd.go:351`, `cmd/orch/clean_cmd.go:566-583`

---

## Fork Navigation

### Fork 1: Remove --no-track entirely vs. keep some form of it?

**Options:**
- A: Remove entirely — every spawn creates a beads issue
- B: Keep for specific use cases

**Substrate says:**
- Principle "Evolve by distinction": The pain is from conflating "lightweight" with "invisible". These should be separable.
- Principle "Coherence over patches": 5+ special-case handlers (isUntrackedBeadsID guards, daemon exclusion, clean exclusion, abandon/complete guards, two-lane split) suggest the current design needs redesign, not another patch.
- Decision (Two-Lane): Acknowledges --no-track as a pressure point that could restart the cache-building cycle.
- Decision (Cross-project epics): Currently depends on --no-track as the only working pattern.

**RECOMMENDATION:** Option B — Replace `--no-track` with `--lightweight` that auto-creates a beads issue but marks it for auto-close on completion.

**Trade-off accepted:** Slight overhead of beads issue creation for ad-hoc work (milliseconds). But every agent becomes trackable.

**When this would change:** If beads gets cross-repo issue support (Option D from the epic investigation), `--no-track` for cross-repo work becomes unnecessary entirely.

### Fork 2: What replaces --no-track for cross-project epics?

**Options:**
- A: `--workdir` already handles cross-project spawns — combine with `--issue` using a cross-repo beads reference
- B: Auto-create a lightweight local issue in the secondary repo that links back to the primary issue
- C: Keep --no-track but only allow it with `--workdir` (cross-repo restriction)

**Substrate says:**
- Constraint (beads per-repo): Cannot reference primary repo's beads issue from secondary repo
- Prior (2025-12-21 investigation): Option B (mirror issues) was rejected for "bookkeeping overhead" — but auto-created lightweight issues would eliminate that overhead
- Principle "Session amnesia": Auto-created issues survive sessions; --no-track agents don't

**RECOMMENDATION:** Option B — When `--workdir` is set and `--issue` references a cross-repo issue, auto-create a lightweight tracking issue in the secondary repo with a back-reference. This replaces the need for `--no-track` entirely.

**Trade-off accepted:** Creates an extra issue in secondary repo. But it's auto-created and auto-closed, so no bookkeeping.

**When this would change:** If beads implements native cross-repo references.

### Fork 3: What happens to existing `--no-track` usage during migration?

**Options:**
- A: Hard removal — break immediately, force migration
- B: Deprecation period — warn on use, remove in v2
- C: Soft migration — `--no-track` silently becomes `--lightweight` (creates issue anyway)

**Substrate says:**
- Only the orchestrator and Dylan use the system — no large user migration needed
- Principle "Long-term solution": Dylan prefers the proper fix over workarounds

**RECOMMENDATION:** Option C — Redefine `--no-track` to create a lightweight auto-closing beads issue (silently). The flag name becomes misleading but nothing breaks. Then rename to `--lightweight` and deprecate `--no-track` in next release.

**Trade-off accepted:** Brief naming confusion. But zero breakage.

### Fork 4: What "lightweight" semantics should the replacement have?

**Options:**
- A: Auto-create issue, auto-close on tmux window exit (fire-and-forget)
- B: Auto-create issue, require normal `orch complete` but skip verification gates
- C: Auto-create issue with `triage:auto-close` label, daemon auto-closes after idle timeout

**Substrate says:**
- Principle "Pain as signal": Agents should feel completion pressure (Phase: Complete), even lightweight ones
- Decision (Two-Lane): Source of truth should be beads, not idle timeouts
- The whole point of this change is making agents visible to the completion pipeline

**RECOMMENDATION:** Option B — Auto-create issue, but skip non-essential verification gates (no SYNTHESIS.md requirement, no test evidence check). The agent still reports Phase: Complete, and `orch complete` still runs but with reduced verification. This keeps agents visible while reducing ceremony.

**Trade-off accepted:** Lightweight agents still need `orch complete` (or orphan GC catches them). But that's exactly the cleanup mechanism that's currently missing.

**When this would change:** If auto-close on tmux exit proves reliable enough.

---

## Synthesis: Recommendation

### ⭐ RECOMMENDED: Replace --no-track with lightweight tracking

**What:**
1. **Remove `--no-track` as an escape from tracking** — every spawn creates a beads issue
2. **Add `--lightweight` flag** (or auto-detect from skill tier) that:
   - Creates a beads issue automatically
   - Tags with `tier:lightweight` label
   - Skips non-essential verification gates on `orch complete` (no SYNTHESIS.md, no test evidence)
   - Still requires Phase: Complete for clean closure
   - Auto-closed by orphan GC if agent dies without reporting
3. **Cross-repo migration:** When `--workdir` is set without `--issue`, auto-create issue in target project's beads. Include back-reference to source issue in description.
4. **Deprecation:** `--no-track` becomes alias for `--lightweight` (creates issue silently). Deprecation warning for 1 release, then remove.

**Why:**
- **Eliminates orphan accumulation:** Every agent has a beads issue, so orphan GC sees them all
- **Eliminates invisible agents:** All agents visible in `orch status` tracked lane
- **Eliminates special-case code:** Can remove `isUntrackedBeadsID()` guards, daemon exclusion logic, two-lane untracked category
- **Cross-repo works:** Auto-created local issues replace manual --no-track pattern
- **Minimal overhead:** Beads issue creation is milliseconds; lightweight verification is fast

**Substrate trace:**
- Principle "Evolve by distinction" → separates "lightweight" from "invisible"
- Principle "Coherence over patches" → removes 5+ special-case handlers instead of adding more
- Decision (Two-Lane) → all agents in tracked lane, eliminates the pressure point
- Principle "Session amnesia" → lightweight issues survive sessions

**What we're sacrificing:**
- True zero-overhead ad-hoc work (now creates a beads issue). Acceptable because the overhead is negligible and the visibility is critical.
- `orch sessions` untracked lane loses its primary use case. May simplify to showing only orchestrator sessions.

**When this recommendation would change:**
- If beads issue creation becomes slow (>1s) — but currently ~50ms
- If a use case emerges for truly invisible agents (can't think of one)

---

## Implementation Sequence (if accepted)

**Phase 1 (small):** Rename `--no-track` to `--lightweight` in spawn flags. `--no-track` becomes deprecated alias. When either is used, create a real beads issue with `tier:lightweight` label instead of synthetic `{project}-untracked-{timestamp}` ID.

**Phase 2 (small):** Update `orch complete` to detect `tier:lightweight` label and skip SYNTHESIS.md verification, skip test evidence check. Keep Phase: Complete requirement.

**Phase 3 (small):** Update orphan GC to handle lightweight agents (they now have real beads issues, so this may work already). Remove `isUntrackedBeadsID()` guards from abandon/complete.

**Phase 4 (medium):** Cross-repo spawn with `--workdir` auto-creates issue in target project when `--issue` references a different project. Adds back-reference comment.

**Phase 5 (cleanup):** Remove dead code — `isUntrackedBeadsID()`, daemon untracked exclusion, `orch sessions` untracked category (or repurpose for orchestrator-only).

---

## Decision Gate Guidance

**Add blocks: frontmatter when promoting to decision:**
This decision should block:
- Any feature work on `--no-track` behavior
- Any new special-case handling for untracked agents
- Cross-project epic workflow changes

**Suggested blocks keywords:**
- `no-track`
- `untracked agents`
- `lightweight tracking`
- `cross-project spawn`

---

## Blocking Questions

### Q1: Should --lightweight be explicit or automatic?

- **Authority:** architectural
- **Subtype:** judgment
- **What changes based on answer:** If automatic (inferred from skill tier), no new flag needed — investigation/architect already full tier, everything else becomes lightweight by default. If explicit, adds a flag but gives more control.

### Q2: Should lightweight agents still appear in the daemon's active count?

- **Authority:** implementation
- **Subtype:** judgment
- **What changes based on answer:** If yes, lightweight agents consume capacity slots (which they shouldn't if they're ad-hoc). If no, need a label-based exclusion (simple).

---

## References

**Files Examined:**
- `cmd/orch/spawn_cmd.go` — --no-track flag definition and reason requirement
- `pkg/orch/spawn_beads.go` — determineBeadsID() synthetic ID generation
- `cmd/orch/untracked_sessions.go` — untracked session classification
- `cmd/orch/sessions_untracked.go` — orch sessions command
- `cmd/orch/clean_cmd.go` — cleanup gaps for untracked agents
- `pkg/daemon/active_count.go` — daemon exclusion of untracked agents
- `.kb/decisions/2026-02-18-two-lane-agent-discovery.md` — Two-Lane ADR
- `.kb/investigations/archived/epic-management-deprecated/2025-12-21-inv-cross-project-epic-orchestration-patterns.md` — Cross-project epic pattern
- `.kb/investigations/archived/2026-01-04-inv-untracked-agents-cleanup-path-problem.md` — Cleanup fix
- `.kb/investigations/archived/2025-12-26-inv-investigate-untracked-agents-lingering-orch.md` — Lingering investigation

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-02-18-two-lane-agent-discovery.md` — Acknowledges --no-track as pressure point
- **Investigation:** `.kb/investigations/archived/2025-12-21-inv-cross-project-epic-orchestration-patterns.md` — Cross-project epic pattern
