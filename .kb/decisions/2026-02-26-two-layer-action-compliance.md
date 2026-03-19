# Two-Layer Action Compliance: Infrastructure + Prompt

**Date:** 2026-02-26
**Status:** Accepted
**Context:** Orchestrator behavioral compliance — agents comply with identity but violate action constraints

**Extracted-From:** `.kb/investigations/2026-02-24-design-orchestrator-skill-behavioral-compliance.md`
**Spike:** `.kb/investigations/2026-02-24-spike-claude-code-hooks-orchestrator-guard.md`

**Blocks:** orchestrator skill, action constraints, behavioral compliance, Task tool, orch spawn, bd close

---

## Decision

Enforce orchestrator action constraints through two complementary layers:

1. **Infrastructure enforcement** — `--disallowedTools` at spawn time + PreToolUse hooks at runtime
2. **Prompt restructuring** — Action-identity fusion, affordance replacement, strategic repetition

Neither layer alone is sufficient. Infrastructure without clear instructions creates confusion. Instructions without enforcement are guidelines under pressure.

---

## Problem

Orchestrator agents correctly identify as orchestrators ("I'm a strategic comprehender") but violate action constraints at the moment of action:
- Use Task tool instead of `orch spawn`
- Use `bd close` instead of `orch complete`
- Read/edit code files instead of delegating to workers

**Root cause:** Identity compliance and action compliance are mechanistically different.

| Dimension | Identity | Action |
|-----------|----------|--------|
| Relationship to defaults | Additive (no conflict) | Subtractive (fights system prompt) |
| Signal competition | None | 17:1 disadvantage vs system prompt |
| Instruction hierarchy | User-level = sufficient | User-level < system-level |
| Temporal persistence | Decays but doesn't compete | Decays while system prompt persists |

The Claude Code system prompt has ~500 words actively promoting the Task tool at the system instruction level. The orchestrator skill counters with ~30 words at the user instruction level, positioned at 88% depth in a 640-line document. This is a structural disadvantage that cannot be overcome by prompt quality alone.

---

## Options Considered

### Option A: Two-layer (infrastructure + prompt) — Chosen
- **Pros:** Addresses both salience (prompt) and enforcement (infrastructure). Defense in depth. Aligns with "Infrastructure Over Instruction" principle and field research consensus (AgentSpec ICSE 2026, PCAS).
- **Cons:** Infrastructure layer adds complexity. Maintenance burden for keeping enforcement rules in sync with skill.

### Option B: Prompt-only (skill restructuring without enforcement)
- **Pros:** Simpler, no infrastructure changes
- **Cons:** Research and evidence show prompt-level constraints are insufficient against system-level competing instructions. ~60-70% compliance ceiling.

### Option C: Infrastructure-only (enforcement without skill restructuring)
- **Pros:** Deterministic enforcement
- **Cons:** Agent won't understand WHY tools are blocked. Error messages without context are noise. Hook/plugin API limitations may prevent full enforcement.

### Option D: Modify Claude Code system prompt
- **Pros:** Would solve the instruction hierarchy problem at root
- **Cons:** System prompt is not user-modifiable without forking Claude Code internals. Disproportionate effort.

---

## Architecture

### Layer 1: Prompt Restructuring (salience)

**What prompts CAN do:** Make constraints maximally salient at the moments they matter.

| Technique | Purpose |
|-----------|---------|
| **Action-identity fusion** | Fuse "who you are" with "what tools you use" at top of skill (0% depth, not 88%) |
| **Affordance replacement** | At every spawn decision point: "use `orch spawn`, NOT Task tool" |
| **Strategic repetition** | 3 locations: top of skill, spawn section, completion section |
| **Signal density** | Reduce skill from 640 to <450 lines to improve constraint signal-to-noise ratio |

**Key pattern:** "NOT your tools" framing creates identity-incongruent inhibition — using a prohibited tool feels like an identity violation, not just a rule violation.

### Layer 2: Infrastructure Enforcement (durability)

Two enforcement mechanisms cover different granularities:

| Mechanism | What It Blocks | How |
|-----------|---------------|-----|
| `--disallowedTools` (spawn-time) | Agent, Edit, Write, NotebookEdit | Tools removed from toolset entirely |
| PreToolUse hook (runtime) | `bd close` within Bash | Hook inspects command, denies with redirect to `orch complete` |

**Tools blocked for orchestrators:** Agent, Edit, Write, NotebookEdit
**Tools remaining:** Bash, Read, Glob, Grep, WebFetch, WebSearch

**Session detection:** `CLAUDE_CONTEXT` env var (already implemented — "worker", "orchestrator", "meta-orchestrator")

**Implementation surface:**
- `pkg/spawn/claude.go` — Add `--disallowedTools` flag when `claudeContext == "orchestrator"` (~10 lines)
- `~/.orch/hooks/gate-bd-close.py` — Add orchestrator-specific `bd close` → `orch complete` redirect (~20 lines)

---

## Structured Uncertainty

**What's tested:**
- ✅ Signal ratio analysis: 17:1 disadvantage (system prompt vs skill)
- ✅ Action constraints at 88% depth in 640-line skill
- ✅ `--disallowedTools` flag exists and accepts tool names
- ✅ `CLAUDE_CONTEXT` env var reliably distinguishes session types
- ✅ PreToolUse hooks can deny Bash commands with contextual reasons

**What's untested:**
- ⚠️ Whether removing Task tool causes agent workarounds (e.g., using Bash to run `claude` directly)
- ⚠️ Whether orchestrators need Write tool for VERIFICATION_SPEC.yaml or .kb/ files
- ⚠️ Whether `--disallowedTools` interacts correctly with `--dangerously-skip-permissions`
- ⚠️ Whether prompt restructuring alone achieves >90% compliance (would make Layer 2 optional)
- ⚠️ Exact attention decay rate for skill constraints over session duration

**What would change this:**
- If prompt-only restructuring reliably achieves >90% compliance, Layer 2 becomes optional guard rail
- If orchestrators need Write for .kb/ files, use hook with path-based exceptions instead of blanket `--disallowedTools`
- If `--dangerously-skip-permissions` overrides `--disallowedTools`, fall back to hook-only enforcement

---

## Implementation Sequence

**Phase 1 (Layer 1 — prompt):** Restructure orchestrator skill
- 1a: Action-identity fusion at top of skill (highest impact)
- 1b: Affordance replacement at spawn/completion decision points
- 1c: Reduce skill length (640 → <450 lines)
- 1d: Strategic repetition at 3 decision points

**Phase 2 (Layer 2 — infrastructure):** Add enforcement
- 2a: `--disallowedTools` in `pkg/spawn/claude.go` (~1 hour)
- 2b: PreToolUse hook for `bd close` gating (~30 min)
- 2c: Manual validation (spawn orchestrator, verify tool restrictions)

**Phase 1 first** because it provides immediate partial improvement and validates the skill restructuring before adding infrastructure.

---

## Consequences

**Positive:**
- Orchestrator action compliance moves from ~30% to >90% (estimated)
- "Infrastructure Over Instruction" principle applied to the orchestrator itself
- Defense in depth: prompt handles understanding, infrastructure handles enforcement
- Pattern reusable for any role-specific action constraints (meta-orchestrator, specialized workers)

**Risks:**
- If Write tool needed for .kb/ files, orchestrators will be blocked (mitigation: monitor, fall back to hook with path exceptions)
- Enforcement rules must stay in sync with skill content (mitigation: both live in orch-go/orch-knowledge)
- Over-restriction could push agents toward workarounds (mitigation: graduated rollout, monitor first)

---

## References

- `.kb/investigations/2026-02-24-design-orchestrator-skill-behavioral-compliance.md` — Root cause analysis
- `.kb/investigations/2026-02-24-spike-claude-code-hooks-orchestrator-guard.md` — Infrastructure feasibility spike
- `.kb/models/architectural-enforcement/model.md` — Multi-layer enforcement model (this decision extends it)
- AgentSpec (ICSE 2026), PCAS — Field research confirming infrastructure > instruction for enforcement
- `.kb/decisions/2026-01-08-observation-infrastructure-principle.md` — "Infrastructure Over Instruction" principle

## Auto-Linked Investigations

- .kb/investigations/archived/2025-12-27-design-action-batching-layer-playwright.md
- .kb/investigations/archived/2025-12-26-design-two-infrastructure-failures-revealed-missing.md
