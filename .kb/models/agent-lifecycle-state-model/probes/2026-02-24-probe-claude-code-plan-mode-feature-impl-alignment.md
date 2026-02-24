# Probe: Claude Code Plan Mode vs Feature-Impl Phase Model Alignment

**Date:** 2026-02-24
**Status:** Active
**Model:** agent-lifecycle-state-model
**Triggered by:** Agent 1169 (architect skill) spontaneously used Claude Code plan mode — wrote plan, got human approval, then executed. Evaluated whether this should be integrated into feature-impl's Planning phase.

---

## Question

Does Claude Code's native plan mode align with the feature-impl skill's phase model, and should it replace or augment the current prompt-driven Planning phase?

## What I Tested

### 1. Claude Code Plan Mode Mechanics (web research)

**How plan mode works:**
- Two tools: `EnterPlanMode` (transitions to read-only mode) and `ExitPlanMode` (surfaces plan for user approval)
- Plan is written to `~/.claude/plan/<name>.md` — a dedicated file the agent can edit during plan mode
- **Read-only restriction:** In plan mode, agents CANNOT edit files, run bash commands, or make any changes except to the plan file. This is enforced via system prompt injection.
- **Approval gate:** When `ExitPlanMode` is called, user sees the plan and gets 4 options:
  1. "Yes, clear context and auto-accept edits" (DEFAULT — clears conversation context)
  2. "Yes, auto-accept edits" (preserves context)
  3. "Yes, manually approve edits"
  4. Type feedback to revise plan
- **Headless support:** Partial. Can start with `--permission-mode plan`, but the approval prompt still fires. No clean programmatic bypass exists. Feature requests for `--plan-only` and `--plan-file` exist but are unresolved.

### 2. Feature-Impl Current Planning Phase

**How feature-impl plans currently:**
- **Step 0: Scope Enumeration** — agent reads SPAWN_CONTEXT, lists all requirements, reports via `bd comment`
- **Phase: Planning** — reported via `bd comment <beads-id> "Phase: Planning - ..."`
- **Investigation Phase** (if configured) — creates investigation file, explores codebase, documents findings
- **Design Phase** (if configured) — documents architectural approach, gets orchestrator approval
- Phases are configured at spawn time via `--phases "investigation,design,implementation,validation"`
- No tool-level enforcement of read-only during planning — it's prompt-driven (skill text instructs the agent)

### 3. Daemon Spawn Compatibility

**Tested against:** Daemon spawn path analysis (daemon.go → SpawnWork → `orch work <beadsID>`)

**Key findings:**
- Daemon spawns are headless by default (no tmux, no human present)
- `SpawnWork()` runs `orch work <beadsID>` which creates an OpenCode session or Claude CLI session
- No human is present to approve a plan — the approval gate in plan mode requires interactive input
- Plan mode's approval prompt fires even in headless mode (per web research, no clean bypass exists)
- **Daemon-spawned agents cannot use plan mode** — they would hang at the approval gate indefinitely

### 4. Plan Mode's Context-Clearing Default

**Critical finding:** The DEFAULT plan approval option (option 1) clears conversation context. This means:
- All SPAWN_CONTEXT instructions would be lost after plan approval
- Skill guidance (feature-impl, architect, worker-base) injected via spawn context would vanish
- Beads tracking instructions (`bd comment`) would be gone
- Phase reporting protocol would be lost
- The agent would continue with only the plan file as context

This directly conflicts with the orchestration model where SPAWN_CONTEXT.md carries the full operational instructions.

### 5. Phase Reporting Protocol Interaction

**Current protocol:** Agents report phases via `bd comment <beads-id> "Phase: Planning - ..."` → `"Phase: Implementing - ..."` → `"Phase: Complete - ..."`

**Plan mode interaction:**
- Plan mode would add a new lifecycle state between spawn and first phase
- The plan itself goes to `~/.claude/plan/<name>.md` — NOT to the workspace or beads system
- Plan approval creates a hard gate (human must approve) that's invisible to the orchestrator
- The orchestrator tracks agent progress via beads comments — a plan mode pause would appear as the agent being "stuck" or "unresponsive" (no bd comments during plan mode since bash is blocked)

## What I Observed

### Fork 1: Does Plan Mode Align with the Phase Model?

**No — fundamental misalignment.**

| Aspect | Feature-Impl Phases | Claude Code Plan Mode |
|--------|--------------------|-----------------------|
| Enforcement | Prompt-driven (skill text) | Tool-level (system prompt blocks writes) |
| Approval | Orchestrator via bd comment | Human via interactive prompt |
| Artifact location | Workspace + beads | `~/.claude/plan/` (ephemeral) |
| Context | SPAWN_CONTEXT preserved throughout | Default clears context on approval |
| Observability | bd comments visible to daemon/orchestrator | Invisible — no bd comments possible |
| Headless | Fully compatible | Breaks (hangs at approval gate) |

### Fork 2: Daemon Compatibility

**Incompatible.** Daemon-spawned agents are headless. Plan mode requires interactive human approval. There is no programmatic bypass. Integrating plan mode into feature-impl would make all daemon-spawned feature-impl agents hang at the approval gate.

### Fork 3: Quality Improvement

**Marginal, with significant downsides.**

The quality benefit of plan mode is:
1. Read-only exploration before implementation (prevents premature coding)
2. Human-approved plan ensures alignment before work begins

But feature-impl already achieves (1) through the Investigation and Design phases, which are *better* because:
- They produce durable artifacts (`.kb/investigations/`, `docs/designs/`)
- They're observable via beads comments
- They work headlessly
- They don't clear context

And (2) is handled by the Design Phase's "Get orchestrator approval before implementation" instruction, plus the multi-phase validation level.

**Where plan mode IS better:** Tool-level enforcement of read-only during exploration. The current feature-impl relies on prompt instructions ("explore before coding"), which agents can ignore. Plan mode's system-prompt-level restriction makes premature implementation mechanically impossible.

### Fork 4: Skill Phase Reporting Interaction

**Breaks phase reporting.** During plan mode:
- Bash tool is blocked → cannot run `bd comment` → orchestrator sees no phase transitions
- Agent appears unresponsive to monitoring (daemon, dashboard)
- Plan file location (`~/.claude/plan/`) is outside workspace → not captured in SYNTHESIS.md or agent artifacts
- Phase: Complete reporting would need to happen *after* plan mode exits, creating a gap in the lifecycle

## Model Impact

### Confirms: Agent lifecycle model's assumption of continuous observability

The agent-lifecycle-state-model assumes agents can report phase transitions throughout their lifecycle. Plan mode creates a "dark period" where the agent is active but invisible to the orchestration layer. This confirms the model's design assumption — the phase reporting protocol requires continuous tool access, which plan mode revokes.

### Extends: New lifecycle incompatibility between Claude Code features and orchestrated agents

The model doesn't currently document tool-level feature incompatibilities. Plan mode reveals a class of Claude Code features that assume interactive, single-user operation and break when used in orchestrated, headless environments. Other features in this class may exist (e.g., `--permission-mode plan` as a spawn flag).

### Contradicts nothing in the current model.

The model doesn't make claims about plan mode specifically, so nothing is contradicted.

## Recommendation

**Do NOT integrate Claude Code plan mode into feature-impl's Planning phase.**

### Reasoning (Substrate Trace)

**Principle: Gate Over Remind**
- Plan mode IS a gate (mechanically prevents implementation before planning) — this is good
- But it's a gate that the orchestration layer cannot observe or manage — this violates the spirit
- A gate that makes the agent invisible to its supervisor is worse than a reminder the agent can see

**Principle: Surfacing Over Browsing**
- Plan mode outputs go to `~/.claude/plan/` — the orchestrator must browse to find them
- Feature-impl phases produce artifacts in standard locations (workspace, .kb/) that get surfaced via beads

**Model: Daemon Autonomous Operation**
- Daemon requires fully headless agent operation
- Plan mode requires interactive approval
- These are fundamentally incompatible

**Decision precedent: Dual spawn architecture**
- The system already has escape-hatch (tmux, interactive) vs primary path (headless, autonomous)
- Plan mode only works on the escape-hatch path
- Making feature-impl depend on plan mode would break the primary path

### Alternative: Prompt-Level Planning Enforcement

Instead of Claude Code's plan mode, the better approach is to strengthen the prompt-level planning enforcement in feature-impl:

1. **Add explicit anti-premature-implementation language** to Step 0 and Investigation Phase: "Do NOT create or modify source files until Phase: Implementation begins"
2. **Add a planning checkpoint gate** in the skill: Agent must report `bd comment "Phase: Planning - Scope: ..."` with enumerated requirements BEFORE any file writes
3. **Coaching plugin detection** (existing infrastructure): Detect premature writes during Investigation/Design phases and inject friction

This preserves:
- Headless compatibility (daemon)
- Continuous observability (bd comments)
- Context preservation (no context clearing)
- Artifact durability (workspace, .kb/)

### Where Plan Mode IS Appropriate

Plan mode is appropriate for **interactively spawned agents** (with `--tmux` or `--inline` flags) where a human is actively watching:
- Manual `orch spawn architect -i` sessions (interactive brainstorming)
- Direct Claude Code usage (not through orch spawn)

The architect skill agent that spontaneously used plan mode (Agent 1169) was likely in an interactive session, which is the correct use case. Plan mode is a Claude Code UX feature for human-agent collaboration, not an orchestration primitive.

---

**Status:** Complete
