# Model: Orchestrator Session Lifecycle

**Domain:** Orchestrator / Meta-Orchestration / Session Management
**Last Updated:** 2026-01-12
**Synthesized From:** 40 investigations (Dec 21, 2025 - Jan 7, 2026) on orchestrator session boundaries, completion verification, frame collapse, checkpoint discipline, and hierarchical completion model

---

## Summary (30 seconds)

Orchestrator sessions operate in a **three-tier hierarchy** (meta-orchestrator → orchestrator → worker) where each level is completed by the level above. Orchestrators produce **SESSION_HANDOFF.md** (not SYNTHESIS.md) and **wait** for completion (not /exit). They track via **session registry** (not beads) because orchestrators manage conversations, not work items. Frame collapse occurs when orchestrators drop levels and do work below their station - detected externally, not self-diagnosed. Checkpoint discipline uses duration thresholds (2h/3h/4h) as a proxy for context exhaustion.

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

Three distinct session types exist with different completion patterns:

| Session Type | Boundary Trigger | Handoff Mechanism | Artifact | Beads Tracking |
|--------------|------------------|-------------------|----------|----------------|
| **Worker** | `Phase: Complete` + `/exit` | SPAWN_CONTEXT → SYNTHESIS | SYNTHESIS.md | Required |
| **Orchestrator** | SESSION_HANDOFF.md + wait | ORCHESTRATOR_CONTEXT → SESSION_HANDOFF | SESSION_HANDOFF.md | Skipped |
| **Cross-session** | End of working day | Manual reflection | SESSION_HANDOFF.md | N/A |

**Worker boundaries:** Protocol-driven. Agent reports completion via beads comment, exits, waits for orchestrator verification.

**Orchestrator boundaries:** State-driven. Agent writes handoff artifact, waits (doesn't exit), meta-orchestrator reviews and completes.

**Cross-session boundaries:** Manual checkpointing when Dylan ends work session.

### Orchestrator Detection

Orchestrators are detected via **skill metadata**, not explicit flags:

| Aspect | Worker | Orchestrator |
|--------|--------|--------------|
| Skill type | `skill-type: worker` | `skill-type: policy` or `orchestrator` |
| Context file | SPAWN_CONTEXT.md | ORCHESTRATOR_CONTEXT.md |
| Completion artifact | SYNTHESIS.md | SESSION_HANDOFF.md |
| Beads tracking | Required | Skipped (uses session registry) |
| Default spawn mode | Headless | Tmux (visible) |
| Completion signal | `Phase: Complete` + `/exit` | SESSION_HANDOFF.md + wait |
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
- SESSION_HANDOFF.md shows "Manual fixes" sections
- Post-mortem reveals work that should have been spawned

**NOT the fix:** Adding more ABSOLUTE DELEGATION RULE warnings. The agent already knows. The problem is framing, not awareness.

**Prevention:**
1. Provide specific goals with action verbs, concrete deliverables, success criteria
2. Use WHICH vs HOW test: meta decides WHICH focus, orchestrator decides HOW to execute
3. Frame collapse check in SESSION_HANDOFF.md template
4. Potential: OpenCode plugin tracking Edit usage on code vs artifacts

**Trigger pattern:** Failure-to-implementation. After agents fail, orchestrator tries to "just fix it" instead of trying different spawn strategy.

### 2. Self-Termination Attempts

**What happens:** Spawned orchestrator tries to run `orch session end` or `/exit` instead of waiting for completion.

**Root cause:** ORCHESTRATOR_CONTEXT.md template contradicted the hierarchical completion model (told orchestrator to self-terminate).

**Why it's wrong:** Breaks the "completed by level above" invariant. Orchestrator can't verify its own work from meta perspective.

**Fix:** Template updated Jan 2026 to instruct "write SESSION_HANDOFF.md and WAIT".

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

**Why this is correct:** Orchestrators manage sessions (conversations), not tasks (work items). Beads tracks "what needs doing", session registry tracks "who's managing work".

---

### Why Tmux Default for Orchestrators?

**Constraint:** Orchestrators default to tmux mode (visible windows), workers default to headless.

**Implication:** Spawning orchestrators creates visible tmux windows consuming screen space.

**Workaround:** Use `--headless` flag if visibility not needed (rare for orchestrators).

**Why this is correct:** Orchestrators need visibility for interactive work. Watching orchestrator spawn/complete/synthesize provides meta-level feedback.

---

### Why Orchestrators Don't Report Phase?

**Constraint:** Orchestrator tier skips Phase reporting via beads comments.

**Implication:** Can't track orchestrator progress via `bd show`, no Phase: Planning/Implementing/Complete transitions.

**Workaround:** Read SESSION_HANDOFF.md progress sections, or check orchestrator's active agent count.

**Why this is correct:** Orchestrators don't have linear phases. They spawn multiple agents, synthesize in parallel, iterate based on results. Phase model doesn't apply.

---

### Why Checkpoint Thresholds, Not Hard Limits?

**Constraint:** Checkpoint discipline uses warnings (2h/3h/4h), not hard blocks.

**Implication:** Orchestrator can ignore warnings and continue past 4h.

**Workaround:** None needed - this is intentional respect for orchestrator judgment.

**Why this is correct:** Only the orchestrator knows if they're near completion vs mid-synthesis. Hard limits would interrupt productive flow. Visibility + judgment beats enforcement.

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

### Status: Resume Protocol Implementation (Jan 13, 2026)

**Current implementation status:** Resume protocol is partially implemented with two distinct commands:

| Command | Purpose | Status |
|---------|---------|--------|
| `orch session resume` | Display SESSION_HANDOFF.md for NEW session context | ✅ Implemented (Jan 13) |
| `orch resume <id>` | Resume PAUSED agent by sending continuation prompt | ✅ Implemented (Dec 2025) |

**What exists:**
- Session registry supports workspace/session lookups
- ORCHESTRATOR_CONTEXT.md template includes resume guidance
- `orch session resume` discovers and displays handoffs (with --for-injection mode for hooks)
- `orch resume` can resume workers (beads ID) or orchestrators (--workspace flag)

**What's pending:**
- Auto-resume on session start (Dylan says "let's resume" without specifying which)
- Smart discovery across multiple potential resume candidates
- Tracked in backlog (specific issue TBD)

**Reference:** `.kb/guides/session-resume-protocol.md` - Complete protocol documentation

---

## References

**Investigations:**
- `.kb/investigations/2025-12-21-inv-orchestrator-session-boundaries.md` - Session boundary patterns
- `.kb/investigations/2026-01-04-design-orchestrator-skill-spawnable-agent-gap.md` - Spawnable infrastructure
- `.kb/investigations/2026-01-04-inv-meta-orchestrator-level-collapse-spawned.md` - Frame collapse analysis
- `.kb/investigations/2026-01-06-inv-checkpoint-discipline-orchestrator-sessions.md` - Checkpoint thresholds

**Decisions:**
- `.kb/decisions/2026-01-04-meta-orchestrator-frame-shift.md` - Frame shift concept
- `.kb/decisions/2026-01-04-orchestrator-session-lifecycle.md` - Hierarchical completion model

**Guides:**
- `.kb/guides/orchestrator-session-management.md` - Procedural guide (commands, debugging, workflows)

**Models:**
- `.kb/models/agent-lifecycle-state-model.md` - Worker lifecycle (related but different tier)
- `.kb/models/spawn-architecture.md` - How spawn determines orchestrator vs worker context

**Source code:**
- `pkg/spawn/orchestrator_context.go` - ORCHESTRATOR_CONTEXT.md generation
- `pkg/session/registry.go` - Session registry management
- `cmd/orch/complete_cmd.go` - Tier-aware completion flow
- `pkg/verify/check.go` - VerifyCompletionWithTier()
