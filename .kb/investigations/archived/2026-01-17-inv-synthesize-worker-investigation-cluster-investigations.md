<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Created worker-patterns.md guide synthesizing 11 investigations into authoritative reference covering worker identity, authority, progress tracking, completion protocol, server separation, and metrics.

**Evidence:** Read all 11 investigations in cluster, extracted patterns, produced `.kb/guides/worker-patterns.md` with 10 sections covering the complete worker lifecycle.

**Knowledge:** Workers are architecturally distinct from orchestrators (different process, different metrics, different authority) and plugins run in server process making env-var detection impossible at init time.

**Next:** Close - guide created and committed, investigations successfully synthesized.

**Promote to Decision:** recommend-no - This is synthesis of existing knowledge into a guide, not a new architectural decision.

---

# Investigation: Synthesize Worker Investigation Cluster

**Question:** What patterns emerge from the 11 worker-related investigations that should be codified into an authoritative guide?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** og-arch-synthesize-worker-investigation-17jan-3c0e
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Worker identity requires multi-signal detection

**Evidence:** Three detection signals emerged across investigations:
1. `ORCH_WORKER=1` environment variable (set at spawn in all paths)
2. `SPAWN_CONTEXT.md` file existence in workspace
3. Path containing `.orch/workspace/` in tool arguments

**Source:**
- `2025-12-23-inv-set-orch-worker-environment-variable.md` - ORCH_WORKER env var
- `2026-01-10-inv-add-worker-filtering-coaching-ts.md` - Three-signal detection
- `2026-01-10-inv-debug-worker-filtering-coaching-ts.md` - Per-session detection necessity

**Significance:** Single-signal detection is fragile. Multiple signals provide redundancy while maintaining accuracy.

---

### Finding 2: Plugin architecture constraint - server process vs per-agent

**Evidence:** OpenCode plugins run in the server process, not per-agent. Workers spawn in separate processes with ORCH_WORKER=1 set in their environment only. Server process never sees ORCH_WORKER=1.

This means:
- Cannot detect workers via `process.env.ORCH_WORKER` at plugin init
- Detection must happen per-session in tool hooks
- Must use observable signals (workdir paths, file reads) not env vars

**Source:** `2026-01-10-inv-debug-worker-filtering-coaching-ts.md`

**Significance:** This is a fundamental architectural constraint affecting all plugins that need worker vs orchestrator distinction. Documented as critical in guide.

---

### Finding 3: Light-tier spawns bypass SPAWN_CONTEXT.md detection

**Evidence:** Light-tier spawns inject context directly into the prompt rather than having agents read SPAWN_CONTEXT.md. This bypasses one of the three detection signals. Zero worker-specific metrics appeared in testing because detection never triggered.

**Source:** `2026-01-17-inv-verify-worker-metrics-perform-10.md`

**Significance:** Detection gap that needs addressing. Either force SPAWN_CONTEXT.md reads or rely on path-based detection as primary signal.

---

### Finding 4: Clear project vs infrastructure server separation

**Evidence:** Two distinct "orch servers" concepts exist:
1. **Project servers** (`orch servers start/stop`) - tmuxinator-managed, worker-appropriate
2. **Infrastructure** (`orch serve`, daemon) - launchd-managed, orchestrator-only

Workers correctly use project server commands for UI work. They should NEVER touch infrastructure.

**Source:** `2026-01-07-inv-workers-attempting-restart-orch-servers.md`

**Significance:** Common confusion point. Clear separation documented in guide with explicit "never do this" list.

---

### Finding 5: Workers need different metrics than orchestrators

**Evidence:** Orchestrator metrics (action_ratio, frame_collapse, compensation_pattern) don't apply to workers. Workers need:
- `tool_failure_rate` - consecutive failures
- `context_usage` - token budget tracking
- `time_in_phase` - stall detection
- `commit_gap` - checkpoint reminders

**Source:** `2026-01-17-inv-add-worker-specific-metrics-plugins.md`

**Significance:** Originally workers skipped all metrics. Now workers get health metrics tailored to their execution patterns.

---

### Finding 6: worker-base skill with runtime dependency resolution

**Evidence:** worker-base skill created with common patterns (authority, beads tracking, phase reporting, completion). skillc doesn't support cross-directory dependencies at compile time, so resolution happens at runtime via `LoadSkillWithDependencies()` in orch-go.

**Source:** `2025-12-25-inv-create-worker-base-skill-shared.md`

**Significance:** Enables skill composition through dependency declaration in frontmatter. Workers can inherit common patterns without duplicating content.

---

## Synthesis

**Key Insights:**

1. **Workers are architecturally distinct from orchestrators** - Different processes (ORCH_WORKER env), different metrics (health vs frame detection), different authority (implementation vs strategic). This isn't just behavioral guidance but fundamental architecture.

2. **Plugin detection requires per-session observation** - Due to server process architecture, plugins cannot detect workers at init time. Detection must use tool hook patterns checking workdir and file paths, with per-sessionID caching.

3. **The first 3 actions rule enables orchestrator monitoring** - Workers must report Phase: Planning within first 3 tool calls or be flagged as unresponsive. This creates reliable visibility into agent state.

4. **Completion protocol order matters** - Report Phase: Complete before committing ensures visibility even if agent dies during commit. The precise sequence (report → synthesis → commit → exit) is critical.

**Answer to Investigation Question:**

The 11 worker investigations revealed 10 distinct pattern areas that needed codification:

1. Worker Identity & Detection (multi-signal, per-session)
2. Authority Delegation (implementation vs escalation)
3. Progress Tracking Protocol (bd comment, first 3 actions, test evidence)
4. Session Complete Protocol (order, what never to do)
5. Workspace & Naming Conventions
6. Server Separation (project vs infrastructure)
7. Worker Metrics (distinct from orchestrator)
8. Knowledge Externalization (Leave it Better)
9. worker-base Skill (runtime dependencies)
10. Tmux Interaction Patterns (headless vs tmux spawns)

These are now documented in `.kb/guides/worker-patterns.md` as the authoritative reference for worker behavior.

---

## Structured Uncertainty

**What's tested:**

- ✅ All 11 investigations read and analyzed (verified: read each file)
- ✅ Guide created with 10 sections (verified: wrote worker-patterns.md)
- ✅ Patterns accurately reflect investigation findings (verified: cross-referenced sources)

**What's untested:**

- ⚠️ Whether guide will be discoverable by future agents (not tested with actual spawn)
- ⚠️ Whether guide resolves recurring worker confusion (no longitudinal tracking)
- ⚠️ Light-tier spawn detection gap fix (documented, not implemented)

**What would change this:**

- Finding additional investigations that weren't in the kb reflect cluster
- Discovery that guide format doesn't match other guides (checked: consistent with decision-authority.md)
- Feedback that certain sections are unclear or incomplete

---

## Implementation Recommendations

### Recommended Approach ⭐

**Guide as primary deliverable** - Create comprehensive worker-patterns.md that serves as single authoritative reference.

**Why this approach:**
- Consolidates 11 scattered investigations into one reference
- Follows established pattern of other guides (decision-authority.md, daemon.md, etc.)
- Enables kb context discovery for future agents

**Trade-offs accepted:**
- Guide may become stale if investigations update (address via provenance section)
- Long document (10 sections) may be overwhelming (mitigate with quick reference table at top)

### Alternative Approaches Considered

**Option B: Update worker-base skill only**
- **Pros:** Skill already loaded at spawn time
- **Cons:** Skill is for inheritance, not reference. Can't include debug/troubleshooting info.
- **When to use instead:** For patterns that should be baked into all worker spawns

**Option C: Create multiple small guides**
- **Pros:** More granular, easier to maintain individual sections
- **Cons:** Harder to discover, loses cross-cutting insights
- **When to use instead:** If sections grow too large independently

---

## References

**Investigations Synthesized:**
- `2025-12-20-inv-update-all-worker-skills-include.md` - Leave it Better phase
- `2025-12-23-inv-set-orch-worker-environment-variable.md` - ORCH_WORKER env var
- `2025-12-25-inv-create-worker-base-skill-shared.md` - worker-base skill
- `2026-01-07-inv-workers-attempting-restart-orch-servers.md` - Server separation
- `2026-01-08-inv-bug-worker-agents-cause-tmux.md` - Tmux interaction
- `2026-01-10-inv-add-worker-filtering-coaching-ts.md` - Three-signal detection
- `2026-01-10-inv-debug-worker-filtering-coaching-ts.md` - Plugin architecture constraint
- `2026-01-13-inv-test-worker-naming.md` - Workspace naming
- `2026-01-17-inv-add-worker-specific-metrics-plugins.md` - Worker metrics
- `2026-01-17-inv-verify-worker-metrics-perform-10.md` - Light-tier detection gap
- `2026-01-17-inv-synthesize-worker-investigation-cluster-9.md` - Prior incomplete attempt

**Deliverable Created:**
- `.kb/guides/worker-patterns.md` - Authoritative worker reference guide

---

## Investigation History

**2026-01-17 15:00:** Investigation started
- Initial question: Synthesize worker investigation cluster into guide
- Context: kb reflect identified 11 worker investigations needing synthesis

**2026-01-17 15:10:** Identified all investigations in cluster
- Read all 11 investigations
- Extracted key patterns and findings
- Identified architectural constraints (plugin process separation)

**2026-01-17 15:15:** Created worker-patterns.md guide
- 10 sections covering complete worker lifecycle
- Provenance section linking to source investigations
- Quick reference table at top

**2026-01-17 15:20:** Investigation completed
- Status: Complete
- Key outcome: Created authoritative worker-patterns.md guide synthesizing 11 investigations
