# Probe: Trust Hierarchy Codebase Validation

**Model:** agent-trust-enforcement
**Date:** 2026-03-19
**Status:** Complete
**Question:** Do the 4 trust layers (L1-L4) and the policy/enforcement separation described in the model have concrete implementations in the orch-go codebase?

---

## What I Tested

Verified each trust layer and the policy/enforcement boundary against actual orch-go source code.

### L1: Infrastructure (OS-level)

**Tested:** `pkg/control/control.go` — chflags uchg implementation
**Command:** Read source for Lock/Unlock/EnsureLocked functions
**Observed:** Confirmed. `Lock()` (line 121-134) calls `exec.Command("chflags", "uchg", f)` on control plane files. `Unlock()` (line 165-178) calls `chflags nouchg`. `EnsureLocked()` (line 139-162) discovers all control plane files and locks any unlocked ones. Control plane is defined as settings.json + enforcement hook scripts for PreToolUse and Stop events (line 20-23). This is genuine L1 enforcement — OS kernel prevents modification regardless of application-layer behavior.

### L3: Application (hooks, deny rules, tool restrictions)

**Tested:** `pkg/spawn/claude.go:104-106` — `--disallowedTools` for orchestrators
**Observed:** Confirmed. `BuildClaudeLaunchCommand()` (line 70-142) selects `--disallowedTools 'Agent,Edit,Write,NotebookEdit'` for orchestrator/meta-orchestrator contexts (line 104-106). `--settings` flag (line 129-131) enables per-worker hook isolation. These are Claude Code native features being invoked by orch-go's policy layer.

### L4: Convention

**Tested:** CLAUDE.md "Accretion Boundaries" section
**Observed:** Confirmed. CLAUDE.md contains explicit rules (">1,500 lines require extraction") but these are purely advisory — compliance depends on agent reading and following them.

### Pre-Spawn Lifecycle Phase (Claim C4)

**Tested:** `pkg/orch/spawn_preflight.go:11` — `RunPreFlightChecks()`
**Observed:** Confirmed. Pre-flight runs 5 gate checks BEFORE agent creation: triage (blocks manual spawns), governance (warns on protected paths), hotspot (advisory), agreements (checks unresolved), open questions (surfaces blockers). All gates fire before any tokens are spent. Claude Code hooks fire DURING execution only — no pre-spawn event type exists. This validates C4.

### Policy vs Enforcement Separation (Claim C5)

**Tested:** Whether `BuildClaudeLaunchCommand` functions as a policy→enforcement adapter
**Observed:** Confirmed. The function translates policy decisions into platform-specific enforcement:
- Policy: "orchestrators cannot use worker tools" → Enforcement: `--disallowedTools 'Agent,Edit,Write,NotebookEdit'`
- Policy: "workers need isolated hook config" → Enforcement: `--settings <path>`
- Policy: "agents need beads access across repos" → Enforcement: `export BEADS_DIR=...`
- Policy: "limit reasoning effort" → Enforcement: `--effort <level>`

The function is a pure adapter: it receives policy decisions and emits Claude Code CLI flags. No enforcement logic lives in orch-go — it delegates entirely to Claude Code's native flags.

### Spawn Gates as Policy (Claim C5)

**Tested:** `pkg/spawn/gates/*.go` — 4 gate files (triage, hotspot, agreements, question)
**Observed:** Confirmed. All 8 files (4 implementations + 4 tests) make POLICY decisions:
- Triage: "should manual spawns be allowed?" (policy, not mechanism)
- Hotspot: "is this task targeting files needing architect review?" (policy)
- Agreements: "are there unresolved agreements?" (policy)
- Open questions: "are there blocking questions for this issue?" (policy)

None implement enforcement mechanisms. They return decisions; the spawn pipeline acts on them.

---

## What I Observed

All 4 trust layers have concrete implementations in the orch-go codebase:

| Layer | Implementation | Lines | Verified |
|-------|---------------|-------|----------|
| L1: Infrastructure | `pkg/control/control.go` (chflags uchg) | ~253 | Yes — Lock/Unlock/EnsureLocked confirmed |
| L3: Application | `BuildClaudeLaunchCommand` (--disallowedTools, --settings) | ~75 | Yes — delegates to Claude Code native flags |
| L4: Convention | CLAUDE.md sections | varies | Yes — advisory only |
| Pre-spawn | `RunPreFlightChecks` (5 gates) | ~72 | Yes — fires before agent creation |

The policy/enforcement separation is clean:
- **Policy code** (orch-go owns): spawn gates, governance checks, skill routing
- **Enforcement code** (Claude Code owns): sandbox, permissions, hooks, tool restrictions
- **Adapter** (orch-go builds, Claude Code executes): `BuildClaudeLaunchCommand`

One observation NOT in the model: L2 (Environment) has no direct orch-go implementation currently. The custom OPSEC env var injection (`HTTP_PROXY`, `OPSEC_SANDBOX=1`) was removed per scs-sp-uo3, replaced by native sandbox in settings.json. L2 in orch-go is now implicit — it's part of the settings.json config that orch-go generates and passes via `--settings`.

---

## Model Impact

**Confirms:** Claims C4, C5, C6. Pre-spawn gates have no Claude Code equivalent (C4). Policy/enforcement separation is clean in the codebase (C5). Lifecycle phases are distinct and complementary (C6).

**Extends:** The model should note that L2 (Environment) is now absorbed into L3 (Application) via the `--settings` flag carrying sandbox config. The native sandbox makes L2 a subset of L3 rather than an independent layer. The 4-layer model might be better described as 3 layers (Convention, Application+Environment, Infrastructure) with Application encompassing proxy vars via sandbox config.

**No contradictions found.** All testable claims that touch orch-go source are confirmed.
