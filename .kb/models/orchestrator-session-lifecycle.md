# Model: Orchestrator Session Lifecycle

**Domain:** Orchestrator / Meta-Orchestration / Session Management
**Last Updated:** 2026-01-29
**Synthesized From:** 45+ investigations (Dec 21, 2025 - Jan 29, 2026) on orchestrator session boundaries, completion verification, frame collapse, checkpoint discipline, hierarchical completion model, and session handoff removal

---

## Summary (30 seconds)

Orchestrator sessions operate in a **three-tier hierarchy** (meta-orchestrator → orchestrator → worker) where each level is completed by the level above. Orchestrators produce **SYNTHESIS.md** (same as workers) and **wait** for completion (not /exit). They track via **session registry** (not beads) because orchestrators manage conversations, not work items. Frame collapse occurs when orchestrators drop levels and do work below their station - detected externally, not self-diagnosed. Checkpoint discipline uses duration thresholds (2h/3h/4h) as a proxy for context exhaustion. **Session handoff machinery** (orch session start/end, .orch/session/ directories, auto-resume) was **removed in Jan 2026** in favor of capturing learnings to kb/beads during work.

---

## Core Mechanism

### Three-Tier Hierarchy

Orchestration operates across three distinct levels:

```
┌─────────────────────────┐
│   Meta-Orchestrator     │  ← Dylan (human)
│   Strategic decisions   │     - Decides WHICH focus
│   Spawns orchestrators  │     - Reviews handoffs
└────────────┬────────────┘     - Completes orchestrators
             │ spawns/completes
             ▼
┌─────────────────────────┐
│    Orchestrator         │  ← Claude agent (policy skill)
│    Strategic comprehension │  - COMPREHEND → TRIAGE → SYNTHESIZE
│    SESSION_HANDOFF.md   │     - Spawns workers
└────────────┬────────────┘     - Synthesizes results
             │ spawns/completes
             ▼
┌─────────────────────────┐
│    Worker               │  ← Claude agent (task skill)
│    Implementation       │     - Executes specific task
│    SYNTHESIS.md         │     - Reports Phase: Complete
└─────────────────────────┘
```

**Key invariant:** Each level is completed by the level above. Workers don't self-terminate; orchestrators complete them. Orchestrators don't self-terminate; the meta-orchestrator completes them.

### Session Types and Boundaries

Two distinct session types exist with different completion patterns:

| Session Type | Boundary Trigger | Handoff Mechanism | Artifact | Beads Tracking |
|--------------|------------------|-------------------|----------|----------------|
| **Worker** | `Phase: Complete` + `/exit` | SPAWN_CONTEXT → SYNTHESIS | SYNTHESIS.md | Required |
| **Orchestrator** | SYNTHESIS.md + wait | ORCHESTRATOR_CONTEXT → SYNTHESIS | SYNTHESIS.md | Skipped |

**Worker boundaries:** Protocol-driven. Agent reports completion via beads comment, exits, waits for orchestrator verification.

**Orchestrator boundaries:** State-driven. Agent writes synthesis artifact, waits (doesn't exit), meta-orchestrator reviews and completes.

**Note:** Session handoff machinery (orch session start/end, SESSION_HANDOFF.md for cross-session continuity) was removed in Jan 2026. Context continuity now relies on kb/beads capture during work, not session handoffs.

### Orchestrator Detection

Orchestrators are detected via **skill metadata**, not explicit flags:

| Aspect | Worker | Orchestrator |
|--------|--------|--------------|
| Skill type | `skill-type: worker` | `skill-type: policy` or `orchestrator` |
| Context file | SPAWN_CONTEXT.md | ORCHESTRATOR_CONTEXT.md |
| Completion artifact | SYNTHESIS.md | SYNTHESIS.md |
| Beads tracking | Required | Skipped (uses session registry) |
| Default spawn mode | Headless | Tmux (visible) |
| Completion signal | `Phase: Complete` + `/exit` | SYNTHESIS.md + wait |
| Workspace prefix | `og-work-*`, `og-feat-*` | `og-orch-*` |

**Key insight:** Skills with `skill-type: policy` trigger orchestrator context generation, which sets behavioral mode through framing, not instructions.

### Session Registry vs Beads

**Beads tracks work items.** Issues represent tasks to be completed with dependencies, priorities, status transitions.

**Session registry tracks conversations.** Orchestrators aren't issues being worked on - they're interactive sessions managing work.

**Session registry structure** (`~/.orch/sessions.json`):

| Field | Purpose |
|-------|---------|
| `workspace_name` | Unique identifier (og-orch-goal-05jan-a1b2) |
| `session_id` | OpenCode session ID |
| `spawn_time` | When session started |
| `project_dir` | Project the session is managing |
| `status` | active/completed/abandoned |

**Why separate tracking:**
- Orchestrators manage multiple issues (not themselves an issue)
- Session lifecycle independent of any single work item
- Cross-project operation (orch-knowledge orchestrator managing orch-go work)
- Completion means "conversation ended", not "task finished"

### Checkpoint Discipline

**Purpose:** Prevent quality degradation from context exhaustion.

**Thresholds:**

| Duration | Level | Action |
|----------|-------|--------|
| < 2h | ok | Continue working |
| 2-3h | warning | Consider checkpoint |
| 3-4h | strong | Strongly recommend checkpoint |
| > 4h | exceeded | Session should have checkpointed |

**Why duration as proxy:** Maps roughly to token consumption. Prior evidence showed 5h sessions with partial outcomes, quality degradation, incomplete synthesis.

**Enforcement:** `orch session status` shows checkpoint level with visual indicators. No hard blocks - respects orchestrator judgment while surfacing risk.

### Frame Shift

**What:** The transition from orchestrator to meta-orchestrator is a change in **perspective**, not just hierarchy.

```
Orchestrator frame:  "What agents should I spawn to accomplish this?"
Meta frame:          "What is the orchestrator struggling with?"
```

**Why it matters:**
- Dylan operates from meta frame (sees the orchestration system)
- Claude instances operate as orchestrators within that frame (use the orchestration system)
- Frame collapse = meta-orchestrator dropping into orchestrator tasks (spawning agents directly) or orchestrator tasks (coding)

**Detection challenge:** Orchestrators can't see their own frame collapse. Requires external detection.

---

## Why This Fails

### 1. Frame Collapse (Orchestrator → Worker)

**What happens:** Orchestrator drops into worker-level implementation (editing code, debugging, investigating).

**Root cause:** Vague goals → exploration mode → investigation → debugging. **Framing cues override skill instructions.**

**Why detection is hard:** Orchestrators can't self-diagnose frame collapse. The frame defines what's visible, so from inside the collapsed frame, the behavior feels appropriate.

**Detection signals:**
- Edit tool usage on code files (not orchestration artifacts)
- Time spent >15 minutes on direct fixes
- SYNTHESIS.md shows "Manual fixes" sections
- Post-mortem reveals work that should have been spawned

**NOT the fix:** Adding more ABSOLUTE DELEGATION RULE warnings. The agent already knows. The problem is framing, not awareness.

**Prevention:**
1. Provide specific goals with action verbs, concrete deliverables, success criteria
2. Use WHICH vs HOW test: meta decides WHICH focus, orchestrator decides HOW to execute
3. Frame collapse check in SESSION_HANDOFF.md template
4. Potential: OpenCode plugin tracking Edit usage on code vs artifacts

**Trigger pattern:** Failure-to-implementation. After agents fail, orchestrator tries to "just fix it" instead of trying different spawn strategy.

### 2. Self-Termination Attempts

**What happens:** Spawned orchestrator tries to run `/exit` instead of waiting for completion.

**Root cause:** ORCHESTRATOR_CONTEXT.md template contradicted the hierarchical completion model (told orchestrator to self-terminate).

**Why it's wrong:** Breaks the "completed by level above" invariant. Orchestrator can't verify its own work from meta perspective.

**Fix:** Template updated Jan 2026 to instruct "write SYNTHESIS.md and WAIT".

### 3. Session Registry Drift

**What happens:** `~/.orch/sessions.json` shows status "active" for completed sessions.

**Root cause:** `orch complete` didn't update session registry status (only closed beads issues or removed registry entries).

**Why it matters:** Stale active sessions accumulate, `orch status` shows ghost sessions, registry becomes unreliable.

**Fix:** `orch complete` now updates status to "completed", `orch abandon` updates to "abandoned" - sessions preserved for history, not removed.

---

## Constraints

### Why Orchestrators Skip Beads Tracking?

**Constraint:** Orchestrators are tracked via session registry, not beads issues.

**Implication:** `orch spawn orchestrator` doesn't create beads issue, `orch complete` doesn't check beads for orchestrator tier.

**Workaround:** If you need work tracking for an orchestrator session, create manual beads issue and reference it in workspace metadata.

**This enables:** Separation between conversation tracking (sessions) and work tracking (beads)
**This constrains:** Cannot use `bd` commands to track orchestrator progress

---

### Why Tmux Default for Orchestrators?

**Constraint:** Orchestrators default to tmux mode (visible windows), workers default to headless.

**Implication:** Spawning orchestrators creates visible tmux windows consuming screen space.

**Workaround:** Use `--headless` flag if visibility not needed (rare for orchestrators).

**This enables:** Real-time visibility into orchestrator spawn/complete/synthesize cycles
**This constrains:** Screen space consumed by visible orchestrator windows

---

### Why Orchestrators Don't Report Phase?

**Constraint:** Orchestrator tier skips Phase reporting via beads comments.

**Implication:** Can't track orchestrator progress via `bd show`, no Phase: Planning/Implementing/Complete transitions.

**Workaround:** Read SESSION_HANDOFF.md progress sections, or check orchestrator's active agent count.

**This enables:** Non-linear orchestrator workflow (parallel spawns, iteration-based progress)
**This constrains:** Cannot use Phase-based progress tracking for orchestrators

---

### Why Checkpoint Thresholds, Not Hard Limits?

**Constraint:** Checkpoint discipline uses warnings (2h/3h/4h), not hard blocks.

**Implication:** Orchestrator can ignore warnings and continue past 4h.

**Workaround:** None needed - this is intentional respect for orchestrator judgment.

**This enables:** Orchestrator autonomy for productive flow continuation
**This constrains:** Cannot enforce hard session limits, relies on orchestrator judgment

---

## Evolution

### Phase 1: Workers Only (Dec 2025)

**What existed:** Worker spawning, completion verification, beads tracking.

**Gap:** No infrastructure for spawning orchestrators. Meta-orchestrator (Dylan) operated interactively without session artifacts.

**Trigger:** Wanted to spawn orchestrators for complex multi-agent coordination, but no template/verification infrastructure existed.

### Phase 2: Spawnable Orchestrators (Dec 26-30, 2025)

**What changed:** Added orchestrator tier, ORCHESTRATOR_CONTEXT.md template, SESSION_HANDOFF.md artifact, session registry.

**Investigations:** 12 investigations on orchestrator session boundaries, completion patterns, skill detection.

**Key insight:** Orchestrators ARE structurally spawnable. SESSION_CONTEXT.md ↔ SPAWN_CONTEXT.md, SESSION_HANDOFF.md ↔ SYNTHESIS.md. The "not spawnable" perception was false.

### Phase 3: Frame Collapse Detection (Jan 4-5, 2026)

**What changed:** Recognized that orchestrators can't self-diagnose frame collapse. Added detection mechanisms: time checks in skill, SESSION_HANDOFF.md reflection prompts, potential OpenCode plugin.

**Investigations:** 8 investigations on frame collapse, level confusion, template contradictions.

**Key insight:** Framing is stronger than instructions. ORCHESTRATOR_CONTEXT.md sets behavioral mode, skill warnings don't override it.

### Phase 4: Checkpoint Discipline (Jan 6-7, 2026)

**What changed:** Added session duration tracking, checkpoint warnings in `orch session status`, 2h/3h/4h thresholds.

**Investigations:** 6 investigations on context exhaustion, quality degradation, session boundaries.

**Key insight:** Duration is a practical proxy for context usage. 5h sessions consistently showed partial outcomes. Visibility + judgment beats enforcement.

### Phase 5: Interactive vs Spawned (Jan 2026)

**What changed:** Clarified two orchestrator models - interactive sessions (`orch session start`) vs spawned orchestrators (`orch spawn orchestrator`).

**Investigations:** 4 investigations on workspace structure differences, lightweight vs full workspaces.

**Key insight:** Interactive sessions serve legitimate functions: goal refinement through conversation, real-time frame correction, synthesis of worker results. NOT compensation for daemon gaps.

### Phase 6: Strategic Comprehension Model (Jan 7, 2026)

**What changed:** Redefined orchestrator role from "tactical execution / coordination" to "strategic comprehension / understanding". Orchestrators no longer coordinate (daemon's job) - they comprehend.

**Model shift:**
- Old: "What should we spawn next?" (tactical dispatch)
- New: "What do we need to understand?" (strategic comprehension)

**Impact on hierarchy:** Three-tier hierarchy description updated - orchestrators now do strategic comprehension (COMPREHEND → TRIAGE → SYNTHESIZE), not tactical execution. The line at :31 previously read "Tactical execution" - this is incorrect per Strategic Orchestrator Model.

**Reference:** `.kb/decisions/2026-01-07-strategic-orchestrator-model.md`

### Clarification: Token Usage Constraints (Jan 13, 2026)

**Context added:** Agents cannot observe their own token usage through available APIs. Duration-based thresholds (2h/3h/4h) serve as a practical proxy for context consumption.

**Why this matters:** Checkpoint discipline (Phase 4) uses duration thresholds because direct token observation isn't available to agents. Duration correlates with token usage and is easily measurable.

**Reference:** `.kb/investigations/2026-01-06-inv-orchestrator-sessions-checkpoint-discipline-max.md:75` - "Duration-based thresholds are a practical proxy"

### Clarification: Skill-Type Policy Meaning (Jan 13, 2026)

**Expanded explanation:** Skills with `skill-type: policy` trigger orchestrator context generation, which sets behavioral mode through **framing**, not instructions.

**Key distinction - Framing vs Instructions:**

| Context Type | Example | Mechanism |
|--------------|---------|-----------|
| Worker instructions | "TASK: Implement user authentication" | Directive guidance - what to do |
| Orchestrator framing | "You are an orchestrator. COMPREHEND → TRIAGE → SYNTHESIZE" | Behavioral mode - how to think |

**Why framing is stronger:** Framing shapes perception and available actions. Instructions can be overridden by situational reasoning, but framing defines what's visible. This is why "ABSOLUTE DELEGATION RULE" warnings don't prevent frame collapse - the frame already determines what looks appropriate.

**Reference:** Line 75 of this document explains skill-type:policy detection; this clarifies the mechanism.

### Phase 7: Session Handoff Machinery Removal (Jan 2026)

**What changed:** Removed entire session handoff system (orch session start/end commands, .orch/session/ directories, session-resume plugin, global session store).

**Rationale:** Session handoff was a buffer for un-externalized knowledge that should have been captured in kb/beads during work. System became overengineered and brittle with multiple patches (cross-window scan, staleness detection, active/ cleanup). Reminders fail under cognitive load - orchestrators couldn't maintain session hygiene reliably.

**New model:** Context continuity relies on kb/beads capture during work, not session handoffs. New sessions start fresh from durable state: `kb context` for knowledge, `bd ready` for work state, `orch status` for agent state.

**Orchestrator artifact change:** Spawned orchestrators now produce SYNTHESIS.md (same as workers), not SESSION_HANDOFF.md. Unified completion verification across tiers.

**Key decision:** Pressure Over Compensation - removing the buffer creates pressure to capture learnings properly during work rather than deferring to end-of-session handoff that often wasn't filled.

**Reference:** `.kb/decisions/2026-01-19-remove-session-handoff-machinery.md`

### Status: Resume Protocol Implementation (Jan 13-19, 2026)

**Implementation status:** Resume protocol was implemented Jan 13-15, then **deprecated and removed Jan 19-21** as part of session handoff machinery removal.

**What was removed:**
- `orch session start/end/resume` commands
- `.orch/session/` directory structure (window-scoped active/ directories, latest/ symlinks)
- Session-resume OpenCode plugin (`~/.config/opencode/plugin/session-resume.js`)
- Global session store (`~/.orch/session.json`)

**What remains:**
- Session registry (`~/.orch/sessions.json`) - still used for spawned orchestrator workspace tracking
- `orch resume <id>` command - for resuming paused agents (unrelated to session handoff)

**Reference:** `.kb/guides/session-resume-protocol.md` - Historical documentation (now deprecated)

### Capability: OpenCode Session Inspection (Jan 2026)

**What added:** Ability to inspect and tail non-orch-spawned OpenCode sessions.

**Commands:**
- `orch tail --session <session-id>` - Tail messages from any OpenCode session by ID
- `orch status --all` - Show all sessions including untracked (no beads ID)

**Use case:** Orchestrators can now inspect sessions started outside orch spawn (interactive sessions, manually started agents, cross-project work) without requiring beads integration.

**Key insight:** Gap was UI, not infrastructure - OpenCode API always supported direct session access, CLI just needed flags to expose it.

**Reference:** `.kb/investigations/2026-01-29-inv-orch-cannot-inspect-opencode-sessions.md`

---

## References

**Investigations (Foundational):**
- `.kb/investigations/2025-12-21-inv-orchestrator-session-boundaries.md` - Session boundary patterns
- `.kb/investigations/2026-01-04-design-orchestrator-skill-spawnable-agent-gap.md` - Spawnable infrastructure
- `.kb/investigations/2026-01-04-inv-meta-orchestrator-level-collapse-spawned.md` - Frame collapse analysis
- `.kb/investigations/2026-01-06-inv-checkpoint-discipline-orchestrator-sessions.md` - Checkpoint thresholds

**Investigations (Session Handoff - Jan 2026):**
- `.kb/investigations/2026-01-19-inv-stale-session-handoffs-injected-after.md` - Stale handoff injection issue
- `.kb/investigations/2026-01-21-inv-remove-session-handoff-machinery-remove.md` - Implementation of removal
- `.kb/investigations/2026-01-15-inv-session-handoff-content-injected-into.md` - Plugin bug causing worker contamination
- `.kb/investigations/2026-01-18-inv-documentation-add-session-handoff-location.md` - Active directory documentation

**Investigations (OpenCode Integration - Jan 2026):**
- `.kb/investigations/2026-01-29-inv-orch-cannot-inspect-opencode-sessions.md` - Non-orch-spawned session inspection

**Decisions:**
- `.kb/decisions/2026-01-04-meta-orchestrator-frame-shift.md` - Frame shift concept
- `.kb/decisions/2026-01-04-orchestrator-session-lifecycle.md` - Hierarchical completion model
- `.kb/decisions/2026-01-19-remove-session-handoff-machinery.md` - Session handoff removal rationale

**Guides:**
- `.kb/guides/orchestrator-session-management.md` - Procedural guide (commands, debugging, workflows)
- `.kb/guides/session-resume-protocol.md` - Session resume protocol (deprecated Jan 2026)

**Models:**
- `.kb/models/agent-lifecycle-state-model.md` - Worker lifecycle (related but different tier)
- `.kb/models/spawn-architecture.md` - How spawn determines orchestrator vs worker context

**Source code:**
- `pkg/spawn/orchestrator_context.go` - ORCHESTRATOR_CONTEXT.md generation
- `pkg/session/registry.go` - Session registry management
- `cmd/orch/complete_cmd.go` - Tier-aware completion flow
- `pkg/verify/check.go` - VerifyCompletionWithTier()
