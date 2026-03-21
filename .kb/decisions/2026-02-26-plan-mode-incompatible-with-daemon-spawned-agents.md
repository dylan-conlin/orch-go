# Decision: Plan Mode Incompatible with Daemon-Spawned Agents — Do Not Integrate into Feature-Impl

**Date:** 2026-02-26
**Status:** Accepted
**Enforcement:** context-only
**Deciders:** Dylan
**Blocks:** plan mode, claude code plan, feature-impl planning, planning enforcement, EnterPlanMode, ExitPlanMode

## Context

Agent 1169 (architect skill) spontaneously used Claude Code's native `EnterPlanMode` tool — wrote a plan to `~/.claude/plan/`, got human approval via the interactive prompt, then executed. This raised the question: should plan mode be integrated into feature-impl's Planning phase?

Investigation found three fundamental incompatibilities between plan mode and the orchestrated agent model.

## Decision

**Do NOT integrate Claude Code plan mode into feature-impl or any daemon-spawned skill.**

Plan mode is appropriate only for interactive sessions (direct Claude Code usage, `orch spawn architect -i`). It must not be used in the primary daemon spawn path.

## Rationale

### 1. Daemon incompatibility is disqualifying

Daemon is the primary spawn path. Plan mode's `ExitPlanMode` fires an interactive approval prompt with no programmatic bypass. A daemon-spawned agent entering plan mode hangs indefinitely at the approval gate — no work gets done, and the agent consumes a concurrency slot.

No clean workaround exists. Claude Code feature requests for `--plan-only` and `--plan-file` flags (#16571, #13395) remain unresolved.

### 2. Context clearing destroys operational instructions

Plan mode's DEFAULT approval option (option 1: "Yes, clear context and auto-accept edits") clears conversation context. This would destroy:
- SPAWN_CONTEXT.md (operational instructions, deliverables, skill config)
- Worker-base protocols (phase reporting, authority delegation, hard limits)
- Beads tracking instructions (`bd comment` lifecycle)
- All skill guidance injected at spawn time

The agent would continue with only the plan file, stripped of everything that makes orchestrated execution work.

### 3. Observability gap breaks orchestration

During plan mode, the bash tool is blocked. This means:
- `bd comment` cannot run — no phase reporting to orchestrator
- Agent appears unresponsive to monitoring (daemon health checks, dashboard)
- Plan artifacts go to `~/.claude/plan/` — outside workspace, invisible to beads and SYNTHESIS
- Orchestrator cannot distinguish "agent is planning" from "agent is stuck"

### Feature-impl already has superior planning

The existing feature-impl phase model covers planning needs without these costs:

| Aspect | Feature-Impl Phases | Claude Code Plan Mode |
|--------|--------------------|-----------------------|
| Enforcement | Prompt-driven (skill text) | Tool-level (system prompt blocks writes) |
| Approval | Orchestrator via bd comment | Human via interactive prompt |
| Artifact location | Workspace + .kb/ (durable) | `~/.claude/plan/` (ephemeral) |
| Context | SPAWN_CONTEXT preserved | DEFAULT clears context |
| Observability | bd comments → orchestrator | Invisible (bash blocked) |
| Headless | Fully compatible | Breaks (hangs at approval) |

## What This Accepts

**Accepted gap:** Agents CAN ignore prompt-level planning instructions and jump to implementation. Plan mode's tool-level enforcement of read-only during planning is genuinely better at preventing premature coding.

**Mitigations for the gap:**
1. Step 0: Scope Enumeration forces explicit requirement listing before implementation
2. Investigation and Design phases produce durable artifacts that evidence planning occurred
3. Coaching plugin infrastructure can detect premature source file writes and inject friction

## When This Should Be Revisited

All three conditions would need to be true simultaneously:
- Claude Code adds programmatic plan mode (no interactive approval — e.g., `--plan-only` / `--plan-file` flags)
- Plan mode preserves conversation context by default (no context clearing)
- Plan mode allows `bd comment` execution (bash unblocked for specific commands)

## Where Plan Mode IS Appropriate

- **Interactive architect sessions** — `orch spawn architect -i` with human present in tmux
- **Direct Claude Code usage** — not through orch spawn system
- **Ad-hoc exploration** — not tracked via beads, no phase reporting needed

## Consequences

**Positive:**
- Feature-impl works identically for headless and interactive spawns — no bifurcated code paths
- Phase reporting remains continuous — no "dark periods" invisible to orchestration
- SPAWN_CONTEXT preserved throughout agent lifecycle
- No dependency on Claude Code features that may change

**Negative:**
- Agents can skip planning and jump to implementation (prompt-level enforcement only)
- No tool-level guarantee that agents explore before coding

## References

- Investigation: `.kb/investigations/2026-02-24-design-evaluate-plan-mode-feature-impl-integration.md`
- Probe: `.kb/models/agent-lifecycle-state-model/probes/2026-02-24-probe-claude-code-plan-mode-feature-impl-alignment.md`
- Model: `.kb/models/agent-lifecycle-state-model/` (continuous observability assumption confirmed)
- Prior decisions: dual spawn architecture, daemon unified config construction

## Auto-Linked Investigations

- .kb/investigations/archived/2026-01-04-inv-design-analyze-pkg-daemon-daemon.md
- .kb/investigations/archived/2026-01-06-inv-daemon-auto-complete-agents-report.md
- .kb/investigations/archived/2026-01-03-inv-test-spawned-agents-complete-work.md
- .kb/investigations/archived/2025-12-26-inv-dashboard-active-section-not-showing.md
- .kb/investigations/archived/2026-01-06-inv-daemon-spawns-duplicate-agents-same.md
