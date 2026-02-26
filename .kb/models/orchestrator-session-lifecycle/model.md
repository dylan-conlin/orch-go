# Model: Orchestrator Session Lifecycle

**Domain:** Orchestrator / Meta-Orchestration / Session Management
**Last Updated:** 2026-02-26
**Synthesized From:** 40 investigations (Dec 21, 2025 - Jan 7, 2026) on orchestrator session boundaries, completion verification, frame collapse, checkpoint discipline, and hierarchical completion model. Updated Feb 2026 per probe findings: removed deleted session registry references, updated hierarchy to strategic comprehender pattern.

---

## Summary (30 seconds)

Orchestrator sessions use a **strategic comprehender** pattern where the orchestrator's role is understanding, not coordination (daemon handles coordination). Orchestrators produce **SESSION_HANDOFF.md** (not SYNTHESIS.md) and **wait** for completion (not /exit). Agent state is derived at query time from **four independent sources** (OpenCode sessions, tmux windows, beads issues, workspaces) — no persistent local state like registries. Frame collapse occurs when orchestrators drop levels and do work below their station - detected externally, not self-diagnosed. Checkpoint discipline uses duration thresholds (2h/3h/4h) as a proxy for context exhaustion.

---

## Core Mechanism

### Strategic Comprehender Pattern

The orchestrator role is **strategic comprehension** — understanding, not coordination (daemon handles coordination). The hierarchy has two operational levels:

```
┌─────────────────────────┐
│   Dylan (human)         │  ← Strategic comprehender
│   + Orchestrator skill  │     - ORIENT → DELEGATE → RECONNECT
│   Understanding         │     - Reviews handoffs, provides direction
└────────────┬────────────┘     - Completes orchestrators
             │ spawns/completes
             ▼
┌─────────────────────────┐
│    Worker               │  ← Claude agent (task skill)
│    Implementation       │     - Executes specific task
│    SYNTHESIS.md         │     - Reports Phase: Complete
└─────────────────────────┘

Daemon (autonomous)        ← Coordinates: polls bd ready, auto-spawns
```

**Key invariant:** Each level is completed by the level above. Workers don't self-terminate; orchestrators complete them. The daemon handles tactical coordination (spawning from triage queue), freeing the orchestrator to focus on comprehension.

**Historical note:** An earlier three-tier hierarchy (meta-orchestrator → orchestrator → worker) was collapsed into this pattern when the "strategic orchestrator" model (Jan 2026) recognized that orchestrators should comprehend, not coordinate.

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
| Beads tracking | Required | Optional (orchestrators manage conversations, not work items) |
| Default spawn mode | Headless | Tmux (visible) |
| Completion signal | `Phase: Complete` + `/exit` | SESSION_HANDOFF.md + wait |
| Workspace prefix | `og-work-*`, `og-feat-*` | `og-orch-*` |

**Key insight:** Skills with `skill-type: policy` trigger orchestrator context generation, which sets behavioral mode through framing, not instructions.

### State Derivation (No Local Agent State)

Agent state is derived at query time from four authoritative sources — no persistent local registries or caches:

| Source | What it provides | Authority |
|--------|-----------------|-----------|
| **Beads issues** | Work status, dependencies, phase comments | Highest (canonical completion) |
| **OpenCode sessions** | Session existence, activity, messages | Infrastructure layer |
| **Tmux windows** | Visual liveness, process existence | UI layer only |
| **Workspace files** | `.tier`, `.session_id`, SYNTHESIS.md | Filesystem record |

**Architectural constraint:** `pkg/registry/`, `pkg/cache/`, and `sessions.json` are forbidden by architecture lint tests (`architecture_lint_test.go`). If queries are slow, fix the authoritative source — do not build projections.

**Historical note:** A session registry (`~/.orch/sessions.json`, `pkg/session/registry.go`) existed Jan 2026 but was removed due to false positive completion detection and drift. The "no local agent state" constraint was established to prevent recurrence.

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

### 3. State Derivation Disagreement

**What happens:** Different state sources disagree about agent status (e.g., beads says "closed" but OpenCode session still shows "busy").

**Root cause:** Four independent state sources have no coordination protocol. Each updates independently.

**Why it matters:** Dashboard can show conflicting status, requiring priority cascade to resolve (beads > Phase comments > SYNTHESIS.md > session status).

**Mitigation:** The agent-lifecycle-state-model defines a Priority Cascade for resolving disagreements. Beads issue status is the highest authority.

**Historical note:** This replaced the earlier "Session Registry Drift" failure mode. The registry itself was removed; the underlying problem (state disagreement) persists in distributed form.

---

## Constraints

### Why Orchestrators Have Optional Beads Tracking?

**Constraint:** Orchestrators manage conversations, not work items. Beads tracking is optional for orchestrator sessions.

**Implication:** Orchestrator state is derived from OpenCode sessions, workspace files, and tmux windows rather than beads issues.

**Workaround:** If you need work tracking for an orchestrator session, create a beads issue and reference it in workspace metadata.

**This enables:** Separation between conversation management and work tracking
**This constrains:** Orchestrator progress not visible via `bd show` unless manual issue created

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

**What changed:** Added orchestrator tier, ORCHESTRATOR_CONTEXT.md template, SESSION_HANDOFF.md artifact, session registry (later removed — see Phase 7).

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

### Phase 7: Registry Removal and No-Local-State Constraint (Feb 2026)

**What changed:** Session registry (`~/.orch/sessions.json`, `pkg/session/registry.go`) was deleted. Architecture lint test added to forbid re-introduction. Agent state is now derived at query time from four authoritative sources (OpenCode sessions, tmux windows, beads issues, workspaces).

**Why removed:** Registry caused false positive completion detection and drift — stale "active" entries accumulated, `orch status` showed ghost sessions. The fundamental problem was maintaining a fifth state source that could disagree with the authoritative four.

**Key insight:** "No local agent state" is an architectural constraint, not a temporary simplification. If queries are slow, fix the authoritative source.

**Reference:** `cmd/orch/architecture_lint_test.go` — enforces the constraint; probe `2026-02-26-probe-decision-staleness-audit-37-decisions.md` — identified the staleness.

### Status: Resume Protocol Implementation (Jan 13, 2026)

**Current implementation status:** Resume protocol is partially implemented with two distinct commands:

| Command | Purpose | Status |
|---------|---------|--------|
| `orch session resume` | Display SESSION_HANDOFF.md for NEW session context | ✅ Implemented (Jan 13) |
| `orch resume <id>` | Resume PAUSED agent by sending continuation prompt | ✅ Implemented (Dec 2025) |

**What exists:**
- Workspace files and OpenCode sessions support lookups
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
- `.kb/models/agent-lifecycle-state-model/model.md` - Worker lifecycle (related but different tier)
- `.kb/models/spawn-architecture/model.md` - How spawn determines orchestrator vs worker context

**Source code:**
- `pkg/spawn/orchestrator_context.go` - ORCHESTRATOR_CONTEXT.md generation
- `pkg/session/session.go` - Session management utilities
- `cmd/orch/complete_cmd.go` - Tier-aware completion flow
- `pkg/verify/check.go` - VerifyCompletionWithTier()
- `cmd/orch/architecture_lint_test.go` - Enforces no-local-state constraint (forbids sessions.json, registry)

**Primary Evidence (Verify These):**
- `pkg/spawn/orchestrator_context.go` - Orchestrator vs worker context generation logic
- `cmd/orch/complete_cmd.go` - Tier-aware completion routing (orchestrator vs worker vs light)
- `pkg/verify/check.go:VerifyCompletionWithTier()` - Three-tier verification logic
- `cmd/orch/session.go` - Session management commands (start, status, resume)
- `cmd/orch/architecture_lint_test.go` - Lint test forbidding sessions.json and registry packages
