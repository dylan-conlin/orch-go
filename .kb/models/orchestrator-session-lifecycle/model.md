# Model: Orchestrator Session Lifecycle

**Domain:** Orchestrator / Meta-Orchestration / Session Management
**Last Updated:** 2026-03-27
**Synthesized From:** 40 investigations (Dec 21, 2025 - Jan 7, 2026) on orchestrator session boundaries, completion verification, frame collapse, checkpoint discipline, and hierarchical completion model. Updated Feb 2026 per probe findings: removed deleted session registry references, updated hierarchy to strategic comprehender pattern. Updated Mar 2026 (18 probes merged): skill injection path map, behavioral compliance gap, cross-project injection failure, constraint dilution, orientation preservation dimension. Updated 2026-03-11: expanded failure mode taxonomy from 5→12 modes via 6-investigation cross-reference; added intent displacement, error-correction feedback, staleness, injection failure, MUST fatigue, temporal decay, and knowledge surfacing gap. Updated 2026-03-12: added failure mode #13 (architect design bypass via issue framing) — 5-layer failure chain where issue description framing overrides architect design surfaced only as kb context pointer. Updated 2026-03-12: added evidence quality stratification from 67-claim inventory across 7 investigations and 4 probes — 14 high-confidence multi-source claims identified, replication failure caveat highlighted for dilution curve thresholds.

---

## Summary (30 seconds)

Orchestrator sessions use a **strategic comprehender** pattern where the orchestrator's role is understanding, not coordination (daemon handles coordination). Orchestrators produce **SESSION_HANDOFF.md** (not SYNTHESIS.md) and **wait** for completion (not /exit). Agent state is derived at query time from **four independent sources** (OpenCode sessions, tmux windows, beads issues, workspaces) — no persistent local state like registries. Frame collapse occurs when orchestrators drop levels and do work below their station - detected externally, not self-diagnosed. Checkpoint discipline uses duration thresholds (2h/3h/4h) as a proxy for context exhaustion.

**Key probe findings (2026-02-15 to 2026-03-11):** Skill content reaches orchestrators via five distinct injection paths; the primary interactive path (OpenCode plugin) has an init-time caching bug that causes stale versions to persist until server restart. Behavioral action constraints (don't use Task tool) are aspirational — they describe desired behavior but are implemented as guidelines competing against the system prompt's ~17:1 counter-signal ratio. Cross-project interactive orchestrator sessions receive NO skill content due to a two-bug injection failure. Behavioral constraints in full skill documents (50+ constraints) provide zero measurable compliance above bare — budget for reliable behavioral compliance is hypothesized at ~2-4 co-resident constraints (⚠️ unreplicated: the dilution curve experiment that produced these thresholds failed to replicate under clean isolation per Mar 4 caveat; downstream artifacts including this model treated the thresholds as established before the replication failure was documented).

**Skill evolution status (2026-03-11):** The orchestrator skill completed a major simplification cycle: 27,200→5,995 tokens (82% reduction). 22 of 25 recommendations from 6 investigations (Jan-Mar 2026) are implemented. 6 of 7 enforcement hooks are registered and working, replacing ~350 lines of prohibition text with infrastructure enforcement. The skill now uses knowledge-transfer framing (routing tables, vocabulary) with exactly 4 behavioral norms (at the dilution budget limit). **Open risk:** v4 was deployed without behavioral validation — `skillc test` bare-parity regression has never been run due to CLAUDECODE env var blocking nested sessions from spawned agents. Post-simplification regrowth is already visible (+24% tokens in 7 days).

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

### Skill Injection Paths (Feb 2026)

Five distinct paths inject orchestrator skill content into sessions:

| # | Path | Mechanism | Reads From | Caching | Applies To |
|---|------|-----------|------------|---------|------------|
| 1 | `orchestrator-session.ts` plugin | `experimental.chat.system.transform` | `~/.claude/skills/meta/orchestrator/SKILL.md` | **Init-time cache** | OpenCode sessions (interactive + spawned) |
| 2 | `load-orchestration-context.py` hook | Claude Code SessionStart | `~/.claude/skills/orchestrator/SKILL.md` (wrong path — missing `meta/`) | Fresh read per session | Claude Code sessions only |
| 3 | `orchestrator_context.go` template | `{{.SkillContent}}` in ORCHESTRATOR_CONTEXT.md | `pkg/skills/loader.go` | Fresh at spawn time | `orch spawn`-ed orchestrators |
| 4 | OpenCode Skill tool | Agent manually calls Skill tool | All discovered skill dirs | Per-call discovery | Any OpenCode session (manual) |
| 5 | `orch-hud.ts` plugin | `experimental.chat.system.transform` | N/A | N/A | Does NOT inject skill content |

**Canonical deployment target for orch-go spawn:** `~/.claude/skills/meta/orchestrator/SKILL.md`

**Critical bugs in the injection paths:**

1. **Plugin init-time caching (primary stale-version vector):** `orchestrator-session.ts` reads the skill file once at OpenCode server startup and caches in memory. `skillc deploy` does NOT trigger a server restart. Interactive sessions get the version that was on disk when the server last started.

2. **Wrong hook path:** `load-orchestration-context.py` uses `~/.claude/skills/orchestrator/SKILL.md` (missing `meta/` prefix). Works by accident because a duplicate file exists at that path.

3. **Cross-project injection failure:** When `cc personal` launches Claude Code in non-orch-go projects (e.g., toolshed), the SessionStart hook exits before injecting anything due to two bugs: (a) `is_spawned_agent()` conflates spawned agents with interactive orchestrators via shared `CLAUDE_CONTEXT=orchestrator` env var, and (b) skill loading is gated behind `.orch/` directory existence even though the skill is project-independent. Interactive orchestrator sessions in cross-project contexts receive NO skill content.

4. **Skill version sprawl:** `skillc deploy` creates new copies without cleaning old deployment locations, leaving stale copies discoverable by OpenCode's skill scanner (`~/.opencode/skill/policy/orchestrator/`, `~/.opencode/skill/SKILL.md`, `~/.claude/skills/src/meta/orchestrator/`).

**Fix path (not yet implemented):** Add `ORCH_SPAWNED=1` env var to `BuildClaudeLaunchCommand()`, change `is_spawned_agent()` to check `ORCH_SPAWNED` instead of `CLAUDE_CONTEXT`, restructure hook to load skill before `.orch/` gate.

### Session Types and Boundaries

Four distinct session types exist with different completion patterns:

| Session Type | Boundary Trigger | Handoff Mechanism | Artifact | Beads Tracking |
|--------------|------------------|-------------------|----------|----------------|
| **Worker** | `Phase: Complete` + `/exit` | SPAWN_CONTEXT → SYNTHESIS | SYNTHESIS.md | Required |
| **Orchestrator** | SESSION_HANDOFF.md + wait | ORCHESTRATOR_CONTEXT → SESSION_HANDOFF | SESSION_HANDOFF.md | Skipped |
| **Cross-session** | End of working day | Manual reflection | SESSION_HANDOFF.md | N/A |
| **Frustration** | Signal-driven (behavioral or user text) | Question + failure context | FRUSTRATION_BOUNDARY.md | Via Phase: Boundary |

**Worker boundaries:** Protocol-driven. Agent reports completion via beads comment, exits, waits for orchestrator verification.

**Orchestrator boundaries:** State-driven. Agent writes handoff artifact, waits (doesn't exit), meta-orchestrator reviews and completes.

**Frustration boundaries:** Signal-driven. Detected via compound behavioral signals (headless workers) or user text analysis (interactive sessions). Carries forward the QUESTION, not the CONVERSATION — new session gets a fresh cognitive frame with only the original question and a diagnosis of what didn't work. Designed Mar 2026 after observing that mid-session reframing ("let me stop") doesn't reset attention patterns. See `.kb/investigations/2026-03-27-design-frustration-detection-session-boundary.md`.

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

**Historical note:** A session registry (`~/.orch/sessions.json`, `pkg/registry/`) existed Jan 2026 but was eliminated entirely Feb 2026 due to false positive completion detection and drift. Replaced by single-pass query engine (`cmd/orch/query_tracked.go`) that queries beads → workspace manifests → OpenCode directly. Architecture lint tests (`architecture_lint_test.go`) structurally prevent registry recreation. The "no local agent state" constraint was established to prevent recurrence.

**Liveness gap — Claude-backend agents:** `queryTrackedAgents` only checks OpenCode sessions for liveness. Claude-backend agents (spawned via `runSpawnClaude`) bypass OpenCode entirely — they have no OpenCode session, so manifest has no `session_id`. The query engine marks `missing_session` as dead, causing ~30%+ of claude-mode agents to appear dead even when running in tmux. Phase comments from beads serve as a liveness proxy for these agents, filling the gap without introducing new state layers.

**Session status API:** `GET /session/status` exists in the OpenCode fork and returns `Record<string, idle|busy|retry>`. Use this instead of SSE-only polling for session state — zero OpenCode changes needed.

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

### Orientation Preservation Dimension (Feb 2026)

The COMPREHEND → TRIAGE → SYNTHESIZE frame describes what the orchestrator does but does not address a fourth dimension: **Dylan's orientation state**. An orchestrator can execute this frame perfectly while leaving Dylan disoriented — spawning agents without establishing why Dylan cares, completing them with technically correct explain-back but no frame reconnection, ending sessions with hygiene checkpoints that don't provide "here's where we are" framing.

**Proposed extension:** "Orchestrator effectiveness is measured not by correctly executing COMPREHEND → TRIAGE → SYNTHESIZE, but by whether Dylan is oriented at each transition point."

**Four orientation moments:**

| Moment | Current coverage | Gap |
|--------|-----------------|-----|
| Spawn | Mechanics-heavy (3 scattered sections) | No "why Dylan cares" capture |
| During work | ~30 lines ("run bd ready and orch status") | Minimal monitoring guidance |
| Completion | Best-developed (explain-back gate) | Explain-back assumes Dylan still has spawn-time context |
| Session boundaries | ~40 lines across 3 sections | Scattered, session start protocol buried |

**Interactive vs spawned sessions — debrief gap:** SESSION_HANDOFF.md works for spawned orchestrators. Interactive sessions (the majority pattern — Dylan + orchestrator in Claude Code) produce debriefs conversationally with no durable artifact. The debrief evaporates when the session closes.

**Proposed artifact:** `.kb/sessions/YYYY-MM-DD-debrief.md` — durable comprehension (not facts/tactics) surviving cross-session. Distinct from: `orch orient` (facts), `MEMORY.md` (tactical how-to), SESSION_HANDOFF.md (spawned orchestrators only).

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

**NOT the fix:** Adding more ABSOLUTE DELEGATION RULE warnings. The agent already knows. The problem is framing, not awareness. Industry survey of 8 agent frameworks (2026-03-01) confirms: no framework relies solely on prompt-level constraints in production. Field consensus: "Prompts describe desired behavior; infrastructure enforces it."

**Prevention:**
1. Provide specific goals with action verbs, concrete deliverables, success criteria
2. Use WHICH vs HOW test: meta decides WHICH focus, orchestrator decides HOW to execute
3. Frame collapse check in SESSION_HANDOFF.md template
4. Potential: OpenCode plugin tracking Edit usage on code vs artifacts

**Trigger pattern:** Failure-to-implementation. After agents fail, orchestrator tries to "just fix it" instead of trying different spawn strategy.

**Note on frame guard as liability:** Under cross-project debugging pressure, the frame guard can create **debugging paralysis** — orchestrators cannot trace data paths they need to resolve the user's problem, because the frame guard prevents them from reading code. Frame collapse prevention is correct for routine orchestration but becomes a liability in active debugging scenarios.

### 2. Self-Termination Attempts

**What happens:** Spawned orchestrator tries to run `orch session end` or `/exit` instead of waiting for completion.

**Root cause:** ORCHESTRATOR_CONTEXT.md template contradicted the hierarchical completion model (told orchestrator to self-terminate).

**Why it's wrong:** Breaks the "completed by level above" invariant. Orchestrator can't verify its own work from meta perspective.

**Fix:** Template updated Jan 2026 to instruct "write SESSION_HANDOFF.md and WAIT".

### 3. Competing Instruction Hierarchy (Feb 2026)

**What happens:** Orchestrator maintains correct identity ("I'm an orchestrator") but uses the wrong tool for orchestrator-level work — specifically, uses Claude Code's built-in Task tool instead of `orch spawn`/`bd create`.

**Root cause:** This is structurally distinct from frame collapse. The Claude Code system prompt has platform-level authority and promotes Task tool use with ~500 words of encouragement. The orchestrator skill's constraint ("don't use Task tool") competes with roughly a 17:1 counter-signal ratio.

**Why it's different from frame collapse:** The orchestrator doesn't "drop into worker mode." It maintains orchestrator identity while violating orchestrator action constraints. Identity and action compliance operate on different dimensions.

**Why it's structurally unwinnable at prompt level:** The system prompt occupies a privileged position (platform-level, reinforced every turn). The skill's action constraints are user-level, static, and buried. Under Claude's instruction hierarchy, system > user. No current agent framework has solved the competing-instruction-hierarchy problem.

**Current state:** Action space is *described as restricted* (markdown table saying "don't use Task tool") but not *actually restricted* (Task tool remains fully available). The restriction is aspirational until implemented as a hook that intercepts Task tool calls in orchestrator context.

**Closest implementable fix:** Claude Agent SDK's hooks mechanism. Claude Code already has a hooks system — a specific PreToolUse hook to intercept Task tool calls when `CLAUDE_CONTEXT=orchestrator` is set.

### 4. Behavioral Constraint Dilution (Mar 2026)

**What happens:** Behavioral constraints (delegate, don't use Task tool) fail in full skill documents (50+ constraints) even when expressed in 3 structurally diverse forms.

**Root cause:** Attention budget competition. At ~2-4 co-resident behavioral constraints, 3-form structural diversity achieves ceiling compliance. At 5+ constraints, variance returns. At 10 constraints, behavioral constraints regress to bare parity (zero measurable effect). The production skill (50+ constraints) is far beyond the reliable threshold.

**Key distinction:** Knowledge constraints (factual) have a higher budget than behavioral constraints (action-selection). Behavioral constraints require competing against default affordances; knowledge constraints do not.

**Emphasis language effect:** CRITICAL/MUST/NEVER markers provide measurable lift over neutral language (should/prefer/consider) at high constraint counts. Neutral 10C = bare parity; emphasis 10C sometimes breaks through. However, emphasis language is a partial mitigation, not a solution — combined evidence shows ~33% behavioral compliance at 10C with emphasis vs 0% without. Caution: the dilution curve threshold numbers (behavioral budget ~2-4, degradation starts at 5) did not replicate cleanly under isolation — treat as directional, not established.

**Implication:** Behavioral constraints that matter must be implemented as infrastructure gates (hooks, tool interception), not skill guidelines. Skill guidelines reliably cover knowledge-type constraints only.

### 5. State Derivation Disagreement

**What happens:** Different state sources disagree about agent status (e.g., beads says "closed" but OpenCode session still shows "busy").

**Root cause:** Four independent state sources have no coordination protocol. Each updates independently.

**Why it matters:** Dashboard can show conflicting status, requiring priority cascade to resolve (beads > Phase comments > SYNTHESIS.md > session status).

**Mitigation:** The agent-lifecycle-state-model defines a Priority Cascade for resolving disagreements. Beads issue status is the highest authority.

**Historical note:** This replaced the earlier "Session Registry Drift" failure mode. The registry itself was removed; the underlying problem (state disagreement) persists in distributed form.

### 6. Cascaded Intent Displacement — Intent Spiral (Feb 2026)

**What happens:** Human intent passes through multiple translation layers (human → orchestrator → spawn prompt → skill template → worker) and gets reshaped at each layer by the layer's dominant frame. "Evaluate" (experiential) becomes "audit" (methodology-driven). The output is confident execution of the wrong interpretation.

**Root cause:** (a) Skill routing table lacks paths for experiential/exploratory work — forces into closest match. (b) Heavy compiled skills override weak spawn prompts — as skills grew heavier, spawn prompt influence decreased proportionally. (c) No early verification of agent behavior in first minutes of execution.

**Why it's different from frame collapse:** The orchestrator doesn't drop levels — it stays at orchestrator level but misinterprets the intent. Each layer (orchestrator, spawn prompt, skill, worker) competently optimizes the wrong thing.

**Status:** Open / fundamental. Routing table extended with experiential/production/comparative intent distinction. Core mechanism (frame-dominant variety attenuation across layers) is fundamental to multi-layer delegation.

**Reference:** `.kb/investigations/2026-02-28-investigation-orchestrator-intent-spiral.md`

### 7. Error-Correction Feedback Loop — Defensive Spiraling (Feb 2026)

**What happens:** Each correction from Dylan makes the orchestrator more anxious. Pattern: immediately agree (sycophancy) → over-correct with more elaborate methodology → introduce new complexity → drift further from intent.

**Root cause:** The skill's pre-spawn checklists become amplifiers — an anxious orchestrator leans harder into ceremony, adding process without fixing comprehension. Optimizing for "don't get corrected again" replaces "understand what Dylan wants."

**Status:** Open / fundamental — may be an LLM behavioral pattern. Anti-sycophancy constraint is one of 4 behavioral norms in v4 skill, but baseline testing showed anti-sycophancy at bare parity (3/8).

**Reference:** `.kb/investigations/2026-02-28-investigation-orchestrator-intent-spiral.md` (amplification mechanism section)

### 8. Skill Content Staleness / Deployment Drift (Feb 2026)

**What happens:** Skill references commands, flags, and behaviors that don't exist or have changed. Orchestrators following skill guidance get runtime errors or use wrong defaults.

**Root cause:** CLI evolves faster than skill updates. `skillc deploy` doesn't trigger OpenCode restart (init-time cached version persists). Multiple deployment locations leave stale copies discoverable.

**Status:** Partially mitigated — v4 deployment cleaned 10 orphaned files. 72-commit delta audit (Mar 2026) designed 13 specific edits. Init-time caching bug remains.

**Severity:** 7 harmful references (commands that don't exist), 4 misleading (wrong defaults/syntax). Highest impact: `--opus` flag (doesn't exist, correct: `--model opus`) and `orch frontier` (removed, replacement: `orch status`).

**Reference:** Probe `2026-02-18-orchestrator-skill-cli-staleness-audit.md`, Investigation `2026-03-05-inv-design-orchestrator-skill-update-incorporating.md`

### 9. Skill Injection Failure — Cross-Project (Feb 2026)

**What happens:** Interactive orchestrator sessions in non-orch-go projects receive NO skill content. Claude operates as generic assistant.

**Root cause:** Two-bug chain: (1) `is_spawned_agent()` conflates spawned agents with interactive orchestrators via shared `CLAUDE_CONTEXT=orchestrator` env var. (2) Skill loading gated behind `.orch/` directory existence, but the skill is project-independent.

**Status:** Open — fix designed (add `ORCH_SPAWNED=1`, change detection logic) but not implemented.

**Reference:** Probe `2026-02-25-probe-orchestrator-skill-cross-project-injection-failure.md`

### 10. MUST Fatigue / Constraint Overhead (Mar 2026)

**What happens:** Excessive emphasis language (MUST, NEVER, CRITICAL) creates "cry wolf" effect where all constraints lose salience. Distinct from dilution (#4) — this is about emphasis saturation, not constraint count.

**Root cause:** DSL design anti-pattern: >3 MUST/NEVER/CRITICAL per 100 words triggers warning. Production skill had 20+ NEVER directives.

**Status:** Resolved by simplification — v4 uses knowledge framing ("the system works like Y") instead of prohibition framing ("NEVER do X").

**Reference:** Investigation `2026-03-01-design-infrastructure-systematic-orchestrator-skill.md` (DSL design principles fork)

### 11. Temporal Attention Decay (Feb 2026)

**What happens:** Skill constraints become less salient over session duration while system prompt instructions remain persistent. Longer sessions = higher violation probability.

**Root cause:** Skill injected once at session start. System prompt present every turn. Relative salience of skill instructions decreases over time.

**Status:** Open / fundamental — checkpoint discipline (2h/3h/4h) is an indirect mitigation.

**Reference:** Investigation `2026-02-24-design-orchestrator-skill-behavioral-compliance.md` (Finding 5)

### 12. Knowledge Surfacing Gap — Cross-Session Orientation Loss (Feb-Mar 2026)

**What happens:** Interactive sessions produce debriefs conversationally with no durable artifact. Next session's orchestrator has no comprehension context.

**Root cause:** SESSION_HANDOFF.md works for spawned orchestrators. Interactive sessions (majority pattern) had no equivalent.

**Status:** Partially mitigated — `orch debrief` writes to `.kb/sessions/YYYY-MM-DD-debrief.md`. Comprehension quality depends on orchestrator quality.

**Reference:** Probe `2026-02-28-probe-session-debrief-artifact-design.md`, Probe `2026-02-27-probe-flow-integrated-knowledge-surfacing-architecture.md`

### 13. Architect Design Bypass via Issue Framing (Mar 2026)

**What happens:** A feature-impl agent receives an issue description that contradicts a prior architect design. The issue description's framing (high salience — injected as task context) overrides the architect investigation (low salience — listed as title + path in kb context). The agent builds what the issue describes, not what the architect designed.

**Root cause:** Five-layer failure chain:
1. Issue description framed work as "consolidation" (in-repo), architect designed "extraction" (new repo)
2. KB context surfaced architect investigation as pointer only (title + path), not its conclusions
3. `--architect-ref` flag is access control (hotspot gate), not information injection
4. Feature-impl skill has no checkpoint to verify architect alignment before implementing
5. Agent entered implementation within 3 minutes of planning, insufficient exploration time

**Evidence:** orch-go-68gcy built `pkg/harness/` + `cmd/harness/` inside orch-go when architect orch-go-sb13k had designed `github.com/user/harness` as a separate repo. Required immediate revert (b305e6d35).

**Status:** Unmitigated — no spawn pipeline change yet. Same failure can recur whenever an issue description contradicts a prior architect design.

**Reference:** Probe `2026-03-12-probe-68gcy-architect-design-ignored-spawn-context-analysis.md`

### 14. Cognitive Mode Lock-In — Mid-Session Attention Pattern Resistance (Mar 2026)

**What happens:** An agent or orchestrator recognizes it's in a degraded state ("You're right. Let me stop") but continues producing the same pattern of analysis. The verbal acknowledgment doesn't reset the attention patterns established by the conversation history.

**Root cause:** The conversation's accumulated attention patterns form a cognitive frame (see Frame Shift section). Mid-session instructions ("let me stop", "try something completely different") compete against the frame's gravity. The frame defines what the model sees as relevant — "let me stop" is processed WITHIN the degraded frame, not from outside it. This is structurally identical to frame collapse detection being impossible from inside the collapsed frame (failure mode #1).

**Evidence:** Real session where orchestrator said "You're right. Let me stop" then produced more of the same analysis. Dylan manually closed the session and started fresh — the system should automate this.

**Why prompts don't fix this:** Same mechanism as failure mode #1 (frame collapse): "Framing shapes perception and available actions. Instructions can be overridden by situational reasoning, but framing defines what's visible." A new session = new frame = genuine cognitive reset.

**Status:** Design complete — frustration boundary mechanism proposed (two-track: interactive via UserPromptSubmit hook, headless via compound coaching signals + daemon respawn). Implementation pending.

**Reference:** Probe `2026-03-27-probe-frustration-detection-session-boundary-design.md`, Investigation `2026-03-27-design-frustration-detection-session-boundary.md`

Several failure modes amplify each other:

| Interaction | Mechanism |
|-------------|-----------|
| Dilution (#4) amplifies Competing Hierarchy (#3) | More constraints = weaker signal per constraint = less resistance to system prompt |
| Error-Correction Loop (#7) amplifies Intent Displacement (#6) | Corrections drive deeper into wrong methodology |
| Temporal Decay (#11) amplifies Competing Hierarchy (#3) | System prompt signal constant while skill signal decays |
| Injection Failure (#9) makes prompt failures (#1-4) irrelevant | If skill never reaches agent, all prompt-level failures are moot |
| Architect Design Bypass (#13) amplifies Knowledge Surfacing Gap (#12) | Architect designs committed to .kb/ but surfaced as low-salience pointers; issue framing wins |
| Cognitive Mode Lock-In (#14) is caused by Frame Collapse (#1) | Both stem from "framing is stronger than instructions"; #14 is the specific case where self-correction fails mid-session |
| Temporal Decay (#11) amplifies Cognitive Mode Lock-In (#14) | Longer sessions = deeper frame entrenchment = harder to break out via prompts |
| Error-Correction Loop (#7) can trigger Cognitive Mode Lock-In (#14) | Repeated corrections drive agent into defensive pattern that becomes the locked frame |

### Primary Evolutionary Drivers (Jan→Mar 2026)

Three failure modes drove the majority of the skill's evolution:
1. **Behavioral Constraint Dilution (#4)** — Drove the 2,368→448 line simplification (82% reduction)
2. **Competing Instruction Hierarchy (#3)** — Drove hook-based enforcement strategy (6 hooks)
3. **Cascaded Intent Displacement (#6)** — Drove intent clarification additions and experiential/production distinction

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

### Knowledge Surfacing Gap (Feb-Mar 2026)

**What:** The knowledge system serves agents (via `kb context` at spawn) but not the human directly. Session boundaries are confirmed as the natural integration points for knowledge surfacing, but no command composes models + throughput + freshness into a human-facing session start summary.

**Current state:**
- `orch orient` exists and surfaces facts (throughput, ready issues, relevant models, stale models, focus goal) — but not comprehension ("why are we here?", "what changed yesterday?")
- Completion review has 4 touchpoints (probe verdicts, architectural choices, knowledge maintenance, hotspot advisory) but none answer "how does this completion change your model of X?" across the full model corpus
- `kb context` is agent-facing (80k char budget, full model summaries) — needs adaptation for human-facing use

**Proposed additions (not yet implemented):**
1. `orch orient` comprehension layer — `.kb/sessions/` debrief integration alongside existing fact layer
2. Model-impact enrichment at completion — cross-reference SYNTHESIS.md topics against broader model corpus
3. `.kb/sessions/YYYY-MM-DD-debrief.md` artifact for durable comprehension from interactive sessions

### Decidability Graph Loss (Mar 2026)

**What was lost:** The `decidability-graph` model (~470 lines, high quality) was removed during entropy cleanup. It provided the structural explanation for WHY the orchestrator hierarchy exists — not capability differences but **context-scoping**. It also defined node taxonomy (Work/Question/Gate), edge authority (who can traverse which edges), and graph dynamics (questions that fracture, collapse, or reframe change the planning subgraph).

**Current gap:** This model describes orchestrator engagement patterns but lacks the formal framework for authority routing. The authority hierarchy is documented (orchestrator vs daemon vs Dylan) but not *explained* — the decidability model explains it as: "the hierarchy determines who scopes context for a decision, not who is smarter."

**Recovery recommendation:** Restore `decidability-graph.md` + probes from `entropy-spiral-feb2026` branch (Tier 1 priority — unique framework, high quality, actively relevant).

### Lifecycle Ownership Boundaries (Feb 2026)

**Context:** ~8,800 lines of lifecycle code, ~40% compensating for OpenCode gaps. The four-layer model conflates state (beads, workspace) with infrastructure (sessions, tmux).

**Three-bucket model:**

| Bucket | What it covers | Examples |
|--------|---------------|----------|
| **OWN** | Gates and tracking we control | Verification gates, phase tracking, workspace management, beads integration |
| **ACCEPT** | Infrastructure we use but don't control | Session persistence (OpenCode), SSE completion events, dual backend support |
| **LOBBY** | Gaps we want upstream to fix | Session TTL, metadata API improvements, state endpoint enhancements |

**Reference:** `.kb/decisions/2026-02-13-lifecycle-ownership-boundaries.md`

### Session Identity (Feb 2026)

**Hybrid model:** Derive identity from OpenCode session ID (`ses_xxxxx`), enrich with optional orch session label.

- OpenCode session ID is available with zero friction — use as default correlation key
- Optional `orch session label` command adds human-readable names
- Dashboard timeline shows label if exists, falls back to session ID + time range
- Tool-agnostic: if we switch from OpenCode, just change where session ID comes from

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

### Phase 8: Skill Injection Audit and Behavioral Compliance Analysis (Feb-Mar 2026)

**What changed:** Systematic probe series (18 probes, Feb 15 - Mar 2, 2026) mapping the skill injection infrastructure, identifying behavioral compliance gaps, and quantifying constraint dilution.

**Key findings:**
- Five distinct injection paths documented; primary interactive path has init-time caching bug
- Orchestrator skill has 13 stale CLI references (7 harmful — commands that don't exist, 4 misleading — wrong defaults)
- Action constraints are aspirational (described as restricted, not actually restricted)
- Cross-project injection fails due to two-bug chain in `load-orchestration-context.py`
- Behavioral constraint dilution: reliable compliance only below ~5 co-resident behavioral constraints
- Emphasis language (CRITICAL/MUST) provides partial lift at high constraint counts vs neutral language

**Evidence quality stratification (Mar 12 inventory):** The 7-investigation, 4-probe cluster contains 67 distinct claims. 14 are high-confidence (verified across 2+ investigations), including: knowledge transfers stick while behavioral constraints don't (5 sources), two-layer enforcement needed (5 sources), identity ≠ action compliance (3 sources), 17:1 signal ratio (3 sources). However, the most-cited quantitative finding (behavioral budget ≤4, dilution starts at 5) carries a **replication failure caveat** — the dilution curve did not replicate under clean isolation (orch-go-zola, Mar 4). All behavioral measurements used single-turn `--print` mode; interactive session compliance is untested. Treat specific threshold numbers as directional hypotheses, not established facts.

**Unresolved:** All injection path bugs are documented but not fixed. Decidability graph model needs recovery from `entropy-spiral-feb2026` branch.

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
- `.kb/models/decidability-graph/model.md` - **Structural foundation**: explains WHY the authority hierarchy exists (context-scoping, not capability). This model's authority assumptions depend on that model's premises.
- `.kb/models/agent-lifecycle-state-model/model.md` - Worker lifecycle (related but different tier)
- `.kb/models/spawn-architecture/model.md` - How spawn determines orchestrator vs worker context
- `.kb/models/model-relationships/model.md` - How models relate to each other (structural/mechanistic/taxonomic functions)

**Source code:**
- `pkg/spawn/orchestrator_context.go` - ORCHESTRATOR_CONTEXT.md generation
- `pkg/session/session.go` - Session management utilities
- `cmd/orch/complete_cmd.go` - Tier-aware completion flow
- `pkg/verify/check.go` - VerifyCompletionWithTier()
- `cmd/orch/architecture_lint_test.go` - Enforces no-local-state constraint (forbids sessions.json, registry)

**Merged Probes (2026-03-06):**
| Probe | Disposition | Summary |
|-------|-------------|---------|
| `2026-02-15-orchestrator-skill-deployment-sync.md` | Confirms + Extends | skillc deploy required for behavioral changes; new "deployment drift" failure mode |
| `2026-02-16-orchestrator-skill-orientation-redesign.md` | Confirms + Extends | Adds Dylan's orientation state as fourth dimension beyond COMPREHEND→TRIAGE→SYNTHESIZE |
| `2026-02-17-orchestrator-skill-injection-path-trace.md` | Extends | Maps 5 injection paths; documents init-time caching bug and 7 stale skill copies |
| `2026-02-18-orchestrator-skill-cli-staleness-audit.md` | Contradicts + Extends | 13 stale CLI references (7 harmful); `--opus` flag doesn't exist; default is claude/tmux not headless |
| `2026-02-18-probe-skillc-pipeline-audit.md` | Extends | deploy-root-relative pathing explains misdeploy to `~/.opencode/skill/SKILL.md` |
| `2026-02-24-probe-orchestrator-skill-behavioral-compliance.md` | Confirms + Contradicts + Extends | Action space "restriction" is aspirational; competing instruction hierarchy failure mode (17:1 signal ratio) |
| `2026-02-25-probe-orchestrator-skill-cross-project-injection-failure.md` | Contradicts + Extends | Cross-project interactive sessions receive NO skill content; two-bug injection failure chain |
| `2026-02-26-probe-decision-staleness-audit-37-decisions.md` | Confirms | Registry deleted/forbidden confirmed; model already updated; 37 decisions categorized |
| `2026-02-27-probe-communication-breakdown-postmortem-3-sessions.md` | Confirms + Contradicts + Extends | 7-category in-session failure taxonomy; frame guard creates debugging paralysis under pressure |
| `2026-02-27-probe-flow-integrated-knowledge-surfacing-architecture.md` | Confirms + Extends | Session boundaries confirmed as integration points; knowledge surfacing gap; `orch orient` fact-only |
| `2026-02-28-probe-session-debrief-artifact-design.md` | Confirms + Extends | Interactive sessions have no durable comprehension artifact; fourth artifact type proposed |
| `2026-02-28-probe-context-mode-compression-architecture.md` | Extends | Two context pressure sources (spawn-time vs runtime tool output); Context Mode only works in Claude Code |
| `2026-02-28-probe-playwright-cli-vs-mcp-ux-audit.md` | Confirms + Extends | CLI default for visual verification; MCP only for interactive page exploration |
| `2026-03-01-probe-agent-framework-behavioral-constraints-landscape.md` | Confirms + Extends | Industry confirms "prompts describe, infrastructure enforces"; competing-instruction-hierarchy is universal unsolved problem |
| `2026-03-01-probe-decidability-graph-knowledge-recovery-assessment.md` | Extends | Decidability graph lost from entropy cleanup; WHY hierarchy exists (context-scoping); node taxonomy |
| `2026-03-01-probe-constraint-dilution-threshold.md` | Confirms + Contradicts + Extends | Dilution curve quantified; behavioral budget ~2-4 (directional only — did not replicate cleanly) |
| `2026-03-02-probe-playwright-cli-cdp-endpoint-compatibility.md` | Confirms + Extends | CDP via config/env var not CLI flag; shared config layer is MCP↔CLI interoperability surface |
| `2026-03-02-probe-emphasis-language-constraint-compliance.md` | Extends | Emphasis markers (CRITICAL/MUST) provide lift over neutral language at high constraint counts; directional, N=3 |
| `2026-03-11-probe-orchestrator-skill-failure-mode-taxonomy.md` | Confirms + Extends | Complete 12-mode taxonomy from 6 investigations; 7 new modes added to model; 3 primary evolutionary drivers identified |
| `2026-03-11-probe-orchestrator-skill-current-state-audit.md` | Confirms + Extends | 22/25 recommendations implemented across 6 investigations; 6/7 hooks working; token reduction 82% (27K→6K); behavioral validation gate still open; post-simplification regrowth visible (+24% in 7 days) |
| `2026-03-12-probe-68gcy-architect-design-ignored-spawn-context-analysis.md` | Extends | New failure mode #13: architect design bypass via issue framing. 5-layer failure chain; kb context surfaced investigation but as low-salience pointer; issue description framing overrode architect's separate-repo design |
| `2026-03-12-probe-orchestrator-skill-investigation-cluster-contradiction-analysis.md` | Contradicts + Extends | 4 direct contradictions in skill investigation cluster: constraint budget (≤4) cited as established despite replication failure; skillc test fix claimed but broken; emphasis language as anti-pattern vs compliance tool. Probe-to-downstream propagation failure identified as systemic risk. Constraint budget qualified to "hypothesized" |
| `2026-03-12-probe-orchestrator-skill-model-gap-analysis.md` | Confirms + Extends | Gap analysis: 44% of this model is skill-specific content; 5 probes should migrate to orchestrator-skill model; 5 gaps identified in new model; boundary defined (sessions = state/completion, skill = injection/dilution/enforcement) |
| `2026-03-19-probe-command-invoked-telemetry.md` | Extends | Added `command.invoked` event type for measurement command usage tracking; caller context detection via CLAUDE_CONTEXT/ORCH_SPAWNED env vars; instruments 6 diagnostic commands (harness audit/report/gate-effectiveness, health, doctor, stats) |
| `2026-03-27-probe-frustration-detection-session-boundary-design.md` | Extends | 4th session boundary type (frustration boundary) — signal-driven, carries question not conversation. New failure mode #14 (Cognitive Mode Lock-In). Validates "framing > instructions" as theoretical basis for why session boundaries work and prompts don't. |

**Note (Mar 12, 2026):** A dedicated orchestrator-skill model exists at `.kb/models/orchestrator-skill/model.md`. ~44% of this model's content (skill injection paths, skill failure modes, skill evolution) overlaps. Future skill-specific updates should target the orchestrator-skill model; this model should reference it.

**Primary Evidence (Verify These):**
- `pkg/spawn/orchestrator_context.go` - Orchestrator vs worker context generation logic
- `cmd/orch/complete_cmd.go` - Tier-aware completion routing (orchestrator vs worker vs light)
- `pkg/verify/check.go:VerifyCompletionWithTier()` - Three-tier verification logic
- `cmd/orch/session.go` - Session management commands (start, status, resume)
- `cmd/orch/architecture_lint_test.go` - Lint test forbidding sessions.json and registry packages

## Auto-Linked Investigations

- .kb/investigations/archived/2025-12-22-inv-audit-orchestration-lifecycle-post-registry.md
- .kb/investigations/archived/2025-12-25-inv-investigate-orchestration-lifecycle-end-end.md
