# Model: Claude Code Agent Configuration

**Domain:** How CLI flags, CLAUDE.md, settings, and hooks compose to shape spawned agent behavior
**Last Updated:** 2026-03-12
**Synthesized From:** 3 investigations (Feb 14 - Feb 28, 2026) + 1 hook audit probe (Mar 12) + ongoing spawn infrastructure evolution

---

## Summary (30 seconds)

The agent execution environment has four configuration layers — CLAUDE.md (project instructions), CLI flags (spawn-time capabilities), settings.json (hooks and permissions), and SPAWN_CONTEXT.md (task-specific context). Each layer has different update frequency, blast radius, and failure modes. CLAUDE.md is the highest-leverage surface (loaded into every session) but also the most sensitive to bloat. CLI flags control agent capabilities at spawn time and are the primary mechanism for enforcing skill-appropriate restrictions. The system under-uses available CLI features: a Feb 28 audit of Claude Code v2.1.63 identified 4 "adopt now" features (`--effort`, `--max-turns`, `--settings`, `Stop` hook) and 6 "evaluate" features (`--worktree`, `--tools`, HTTP hooks, `--permission-mode plan`).

---

## Core Mechanism

### Four Configuration Layers

Every spawned agent's behavior is shaped by four layers, applied in order:

| Layer | What It Does | Update Frequency | Blast Radius | Failure Mode |
|---|---|---|---|---|
| **CLAUDE.md** | Project instructions, architecture overview, key references | Rarely (project-level) | Every agent in project | Bloat → context dilution; drift → misleading architecture claims |
| **CLI flags** | Spawn-time capability control (`--effort`, `--max-turns`, `--permission-mode`) | Per-spawn (in `BuildClaudeLaunchCommand`) | Single agent | Wrong flags → wrong capabilities; missing flags → under-restricted agents |
| **settings.json** | Hooks (SessionStart, PreToolUse, Stop) and permission rules. 11 hooks in `~/.orch/hooks/`, 12 registrations (1 duplicate). 9 fire on every Bash call (in parallel). | Per-deployment (global) | All agents using that settings file | Hook errors → agent blocked; dual authority (hook + skill text) → confusion; **zero observability** (no invocation logging) |
| **SPAWN_CONTEXT.md** | Task description, skill content, beads context, server context | Per-spawn (generated) | Single agent | Missing context → agent works blind; excess context → dilution |

### CLAUDE.md as Agent Context Surface

CLAUDE.md is loaded into every agent session as system-level context. This makes it the highest-leverage configuration point and the most dangerous one to bloat.

**Key constraint: progressive disclosure.** CLAUDE.md content must be minimal — state the rule, point to tooling, link to detailed docs. A guarded-file reminder caught an initial 20-line accretion boundaries section and enforced reduction to 4 lines.

**Documentation drift is the primary failure mode.** When code structure changes (directories removed, commands renamed, file counts change), CLAUDE.md references become misleading. An audit found: references to deleted `pkg/registry/`, a `cmd/orch/` listing showing 4 files when 100+ exist, and model section content duplicated 3 times.

**Mitigation:** Periodic CLAUDE.md audits against actual codebase structure. The `remediate-configuration-drift-defect-class` decision (Mar 5) addresses this systemically with `skillc lint` for command reference validation.

### CLI Flag Configuration Space

A comprehensive audit of Claude Code v2.1.63 (Feb 28) mapped all available CLI features against the spawn infrastructure:

**Currently Used:**
- `--dangerously-skip-permissions` — full bypass for autonomous agents
- `--mcp-config` — MCP preset injection (e.g., playwright)
- `--disallowedTools` — tool restriction for orchestrator contexts
- Environment vars: `CLAUDE_CONFIG_DIR`, `BEADS_DIR`, `ORCH_SPAWNED`, `CLAUDE_CONTEXT`

**Adopt Now (high value, low risk):**

| Feature | Value | Effort |
|---|---|---|
| `--effort` (low/medium/high) | Cost/speed optimization per skill tier | ~15 lines |
| `--max-turns` (150 default, 30 light) | Runaway prevention — hard cap on agent turns | ~5 lines |
| `--settings` per-spawn | Worker-specific hooks without inheriting orchestrator hooks | ~10 lines |
| `Stop` hook | Enforce Phase: Complete before agent exits | ~50 lines |

**Evaluate Next (medium value, needs design):**

| Feature | Opportunity | Blocking Question |
|---|---|---|
| `--worktree` | Native git isolation per agent | Merge-back problem, bd sync interaction |
| `--tools` allowlist | Fail-closed capability restriction | Needs skill→toolset mapping |
| HTTP hooks → orch serve | Real-time event visibility without tmux parsing | Needs webhook endpoint design |
| `--permission-mode plan` | Read-only mode for investigation/architect | Needs skill updates |
| `--json-schema` | Structured daemon triage output | Print mode only |
| `--fallback-model` | Graceful model degradation | Print mode only |

**Not Applicable:** `--chrome` (use Playwright MCP), `--from-pr` (no PR workflow), `--fork-session` (no conversation branching need), sandbox mode (agents need full access).

---

## Constraints

### Why CLAUDE.md must stay minimal
**Constraint:** Every line competes for agent attention. At current project size, CLAUDE.md is ~350 lines — approaching the boundary where agents skim rather than internalize.

**Implication:** New sections need justification. Progressive disclosure (rule + pointer) over detailed explanation.

### Why hooks and skill text must not overlap
**Constraint:** When both a hook AND skill text prohibit the same action, agents receive conflicting signal types. This creates ambiguity about enforcement level (see `skill-content-transfer` model, Failure Mode 3: Dual Authority).

**Implication:** If a behavior is hook-enforced, remove it from skill text. One authority per behavior.

### Why CLI flags are under-adopted
**Constraint:** The spawn infrastructure (`BuildClaudeLaunchCommand` in `pkg/spawn/claude.go`) was built when fewer flags existed. New flags require explicit integration.

**Implication:** Each "Adopt Now" feature is ~5-15 lines of code but requires coordination (flag mapping, tier logic, testing).

---

## Why This Fails

### Failure Mode 1: CLAUDE.md Drift
Code changes without corresponding CLAUDE.md updates. References to deleted directories, renamed commands, or changed file counts persist. Agents read misleading architecture descriptions and make wrong assumptions.

### Failure Mode 2: Configuration Drift Across Layers
Settings.json hooks evolve independently of skill content. SPAWN_CONTEXT.md template changes don't always sync with CLAUDE.md. The four layers drift apart because they're maintained by different mechanisms (manual edits, code generation, templates).

**Confirmed example (Mar 12 audit):** `pre-commit-knowledge-gate.py` guards the `kn` CLI system which has been dead since Dec 25, 2025 (`kn` binary not on PATH, last entry 2.5 months old). The hook still fires on every `git commit`, consuming ~49ms per invocation to guard a system that no longer exists. Additionally, `gate-worker-git-add-all.py` is registered twice in settings.json (indices 5 and 10).

### Failure Mode 3: Under-Restriction
Agents get more capabilities than needed because CLI flag adoption lags. Investigation agents have full write access when `--permission-mode plan` would be more appropriate. All agents get the same reasoning depth when `--effort` could differentiate.

---

## Evolution

**2026-02-14:** Two CLAUDE.md maintenance investigations revealed the progressive disclosure constraint (4 lines vs 20) and documentation drift pattern (3 types of stale references).

**2026-02-28:** Feature audit of Claude Code v2.1.63 mapped the full CLI configuration space. Identified 4 "adopt now" features and 6 "evaluate" features. Current spawn invocation uses only 4 of 15+ available flags.

**2026-03-12:** Hook infrastructure audit. 11 unique hooks, 12 registrations (1 duplicate). All denial hooks fire correctly. Zero observability — no invocation logging exists. One hook (`pre-commit-knowledge-gate.py`) guards dead `kn` system. Stop hook confirmed in production with escape hatch. Cost: ~50ms wall-clock per Bash call (9 hooks in parallel), ~805 Python processes per worker session.

---

## Open Questions

1. **Should `--effort` map to skill or tier?** Skills are more granular (investigation=medium, architect=high) but tiers are simpler (light=low, full=medium, deep=high). Decision needed before implementation.

2. **~~Is the Stop hook safe for production?~~** **Answered (Mar 12 audit):** The Stop hook (`enforce-phase-complete.py`) IS in production. It uses `{"decision": "block", "reason": "..."}` output format and has a `stop_hook_active` escape hatch — on second attempt (after first block), it allows exit with a stderr warning. Output format differs from PreToolUse hooks and is not formally documented.

3. **When should we adopt `--tools` allowlist?** The fail-closed approach (only allow listed tools) is more secure than `--disallowedTools` (deny specific tools). But it requires maintaining per-skill tool allowlists. Worth it for investigation/architect agents that shouldn't modify code?

---

## Actionable Implications

### For spawn infrastructure
- Implement `--effort` and `--max-turns` first (lowest risk, immediate value)
- Design per-spawn `--settings` for worker hook isolation
- ~~Prototype `Stop` hook for completion enforcement with timeout escape hatch~~ Done — `enforce-phase-complete.py` in production with `stop_hook_active` escape hatch

### For CLAUDE.md maintenance
- Audit CLAUDE.md after major refactors (directory restructuring, command renaming)
- Enforce progressive disclosure — detail in `.kb/guides/`, pointers in CLAUDE.md
- Consider automated drift detection (compare CLAUDE.md references against filesystem)

---

## References

**Investigations:**
- `.kb/investigations/2026-02-14-inv-add-claude-md-accretion-boundaries.md` — Progressive disclosure for CLAUDE.md content
- `.kb/investigations/2026-02-14-inv-fix-claude-md-remove-deleted.md` — Documentation drift cleanup
- `.kb/investigations/2026-02-28-design-claude-code-2163-feature-audit.md` — Full CLI feature audit

**Related decisions:**
- `.kb/decisions/2026-03-05-remediate-configuration-drift-defect-class.md` — Systemic drift remediation
- `.kb/decisions/2026-02-26-phase-based-liveness-over-tmux-as-state.md` — Phase comments as agent heartbeat

**Probes:**
- 2026-03-12: Hook Infrastructure Audit — 11 hooks audited, zero observability, 1 dead-system guard, 1 duplicate registration, Stop hook confirmed in production

**Related models:**
- `.kb/models/skill-content-transfer/` — How skill content transfers to agents (three content types)
- `.kb/models/spawn-architecture/` — Spawn infrastructure architecture

## Auto-Linked Investigations

- .kb/investigations/2026-03-08-design-portable-harness-tooling.md
- .kb/investigations/2026-02-27-inv-claude-code-worktree-agent-isolation.md
- .kb/investigations/archived/2026-01-16-inv-audit-sessionstart-hooks-claude-code.md
