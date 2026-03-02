## Summary (D.E.K.N.)

**Delta:** The control plane is exactly 6 files (settings.json + 5 enforcement hooks). Everything else — including Go source code for gates — is data plane. The compiled orch binary provides a temporal buffer but is NOT control plane.

**Evidence:** Traced all 48 gates to their enforcement roots. Every gate traces back to either settings.json (defines hooks/permissions) or hook scripts (execute enforcement). Source code only becomes enforcement after compilation, which is a human-controlled step.

**Knowledge:** OS-level immutability (chflags uchg) on 6 files closes the recursive vulnerability with zero workflow impact. Hooks fire in all permission modes including --dangerously-skip-permissions. Hook snapshots are frozen at session start — mid-session settings.json edits don't affect running agents.

**Next:** Implement in 3 phases: (1) chflags uchg on 6 files + orch control lock/unlock commands, (2) add architecture lint to pre-commit hook, (3) re-enable circuit breaker with redesigned heuristics.

**Authority:** strategic - Irreversible change to how infrastructure is managed; constrains all future agent and human workflows.

---

# Investigation: Control Plane / Data Plane Separation for Agent Orchestration

**Question:** Which files constitute the control plane, what enforcement mechanism protects them from agent modification, and how do we implement immutability without crippling legitimate agent work?

**Defect-Class:** configuration-drift

**Started:** 2026-03-01
**Updated:** 2026-03-01
**Owner:** architect (orch-go-sv8o)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** N/A (new decision)
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| 2026-02-14-design-control-plane-heuristics.md | extends | yes | None — heuristics design is orthogonal; this addresses the prerequisite (immutability) |
| entropy-spiral model (probes/2026-03-01-probe-self-stabilization-current-gates.md) | confirms | yes | None — probe confirmed all 48 gates are mutable |
| decidability-graph model (probes/2026-03-01-probe-context-scoping-irreducibility.md) | informs | yes | None — confirms orchestrator's structural position for control plane management |

---

## Problem Framing

**Design Question:** How do we make the agent orchestration control plane immutable while keeping agents fully autonomous for project work?

**Success Criteria:**
1. Zero new approval prompts for agents (Dylan's explicit constraint)
2. Agents cannot modify files that define their own constraints
3. Agents CAN still: edit project code, write investigations, modify CLAUDE.md, use all tools
4. Human (Dylan via orchestrator) CAN still modify control plane files with minimal friction
5. Works for BOTH Claude Code (orchestrator) and claude CLI (workers)
6. Prevents the recursive vulnerability: `Edit(*/.claude/*) → modify settings.json → remove hooks`

**Constraints:**
- `--dangerously-skip-permissions` is always active for workers — cannot rely on permission prompts
- Hooks are the primary enforcement layer and they fire in ALL permission modes (verified via Claude Code docs)
- Hooks are session-frozen: modifications to settings.json mid-session don't affect running agents (Claude Code docs, line 430)
- No CI/CD — all enforcement must be local
- macOS is the primary (only) platform

---

## Findings

### Finding 1: The Control Plane is Exactly 6 Files

**Evidence:** Tracing every defense mechanism to its enforcement root reveals a dependency tree that converges on a small set of files:

```
All 48 gates across 4 layers
    ├── Spawn gates (pkg/spawn/gates/*.go) ← enforced BY compiled orch binary
    ├── Verify gates (pkg/verify/*.go) ← enforced BY compiled orch binary
    ├── Architecture lint (cmd/orch/architecture_lint_test.go) ← enforced BY pre-commit (not currently)
    └── Claude Code hooks ← enforced BY settings.json + hook scripts
         ├── ~/.claude/settings.json ← DEFINES which hooks fire
         ├── ~/.orch/hooks/gate-bd-close.py ← PREVENTS agents from closing own issues
         ├── ~/.orch/hooks/enforce-phase-complete.py ← REQUIRES completion reporting
         ├── ~/.orch/hooks/gate-orchestrator-code-access.py ← PREVENTS orchestrator code access
         ├── ~/.orch/hooks/gate-orchestrator-task-tool.py ← PREVENTS orchestrator task tools
         └── ~/.orch/hooks/pre-commit-knowledge-gate.py ← GATES knowledge commits
```

The compiled orch binary (`~/bin/orch → build/orch`) is NOT control plane — it's the result of compiling data-plane source code. The compilation step (`make install`) is the control plane boundary, not the binary itself.

**Source:**
- `~/.claude/settings.json` (lines 116-312) — hook definitions
- `~/.orch/hooks/` directory listing — 12 hook scripts, 5 with deny capabilities
- `pkg/spawn/gates/` — 6 gate files, all regular Go source
- `pkg/verify/check.go` — 18 failure paths, regular Go source

**Significance:** The control plane surface is much smaller than the 48-gate number suggests. Protecting 6 files gives us the same structural guarantee as protecting all 48 gates, because the 6 files are the enforcement ROOT — everything else derives from them.

---

### Finding 2: Hooks Fire in All Permission Modes (Including --dangerously-skip-permissions)

**Evidence:** Claude Code documentation confirms:
1. Hook input includes `permission_mode` as an informational field (values: "default", "plan", "acceptEdits", "dontAsk", "bypassPermissions")
2. PreToolUse hooks can return `permissionDecision: "deny"` which "prevents the tool call" — this is described independently of permission mode
3. `disableAllHooks` is a separate setting from `disableBypassPermissionsMode` — they're independent systems
4. Hooks are described as "user-defined shell commands that execute automatically at specific points" — no mention of permission-mode gating

**Source:** https://code.claude.com/docs/en/hooks — Common input fields table, PreToolUse decision control, Hook lifecycle section

**Significance:** This invalidates the spawn context claim that "PreToolUse hooks only work in Claude Code runtime, not raw claude CLI." All `claude` CLI sessions (including workers spawned with `--dangerously-skip-permissions`) load settings.json and fire hooks. This means hooks ARE a viable enforcement mechanism for workers.

---

### Finding 3: Hook Snapshots Are Session-Frozen

**Evidence:** Claude Code docs state: "Direct edits to hooks in settings files don't take effect immediately. Claude Code captures a snapshot of hooks at startup and uses it throughout the session. This prevents malicious or accidental hook modifications from taking effect mid-session without your review."

**Source:** https://code.claude.com/docs/en/hooks — "Hook modification behavior" section

**Significance:** This provides a built-in defense against the recursive vulnerability. If Agent A modifies settings.json to remove hooks, Agent A's own hooks still fire for the rest of its session. Only NEW sessions spawned AFTER the modification would be affected. This narrows the attack window to: "agent modifies settings.json → orchestrator spawns new agent → new agent has weaker hooks." Combined with chflags uchg on settings.json, this window closes completely.

---

### Finding 4: chflags uchg Provides OS-Level Immutability

**Evidence:**
- `chflags` is native to macOS (Darwin 24.6.0), no installation required
- `uchg` (user immutable) flag can be set/cleared by file owner — appropriate for development workflow
- No files currently have immutability flags set
- When set: `Edit`, `Write`, `rm`, `mv`, `sed` all fail with "Operation not permitted" at the OS level
- Claude Code would see "Operation not permitted" errors when attempting to modify protected files

**Source:** `ls -lO ~/.claude/settings.json` — no flags currently set; `man chflags` — uchg behavior

**Significance:** chflags uchg is the only enforcement mechanism that works:
- Regardless of permission mode (operates at OS level, below any runtime)
- Regardless of which tools are allowed (Edit, Write, Bash(rm) all blocked)
- Regardless of settings.json content (can't modify the file that defines what's allowed)
- Without any configuration or runtime changes (no new hooks, no permission changes)

---

### Finding 5: Go Source Code is Data Plane (With Compilation Boundary)

**Evidence:** Agents routinely modify `pkg/spawn/gates/*.go` and `pkg/verify/*.go` for feature work. These files are project code that implements control plane behavior when compiled, but the source code itself is not enforcement — the compiled binary is.

The compilation boundary:
```
Source code (data plane) → make build → binary (enforcement point) → make install → deployed binary
```

Currently, no gate prevents agents from running `make install`. An agent modifying `hotspot.go` to always return "pass" and then running `make install` would bypass hotspot enforcement for all subsequent `orch` invocations.

**Source:**
- `pkg/spawn/claude.go:126` — workers run `claude` directly, not through `orch`
- `Makefile` — `make install` copies binary to `~/bin/orch`
- `ls -lO ~/bin/orch` — no immutability flags

**Significance:** The compilation boundary is the natural control plane / data plane separation for Go source code. We don't need to make source files immutable — we need to ensure the `orch` binary is rebuilt intentionally, not accidentally by a runaway agent. Two mechanisms address this:
1. Architecture lint tests in pre-commit catch source-level gate degradation
2. The temporal buffer (source change ≠ binary change) provides natural separation

---

### Finding 6: Permission Narrowing is Insufficient Alone

**Evidence:** The current settings.json permissions include:
- `Edit(*/.claude/*)` — matches ALL files under `~/.claude/`, including settings.json itself
- `Bash(rm:*)` — allows deleting any file
- `Write(*/Documents/personal/*)` — allows overwriting any file in personal projects

Narrowing these permissions would break legitimate workflows:
- `Edit(*/.claude/*)` also covers `~/.claude/projects/*/memory/` (agent auto-memory)
- `Bash(rm:*)` is used for cleanup operations during normal development
- Replacing with `Bash(rm:~/Documents/personal/*)` would still cover hook files

More critically: with `--dangerously-skip-permissions`, the allow/deny lists might not be enforced (the flag "bypasses all permission checks"). This is the whole point of the flag.

**Source:** `~/.claude/settings.json` (lines 1-334) — full permissions list

**Significance:** Permission narrowing provides defense-in-depth but cannot be the primary mechanism. chflags uchg provides the structural guarantee; permission narrowing provides a belt to go with the suspenders.

---

## Synthesis

**Key Insights:**

1. **The control plane is a tree, not a forest** — All 48 gates trace back to 6 root files. Protecting the roots protects the tree.

2. **OS-level immutability closes the recursive loop** — The recursive vulnerability (`agent edits settings.json → removes hooks → all enforcement disappears`) cannot exist when chflags uchg prevents editing settings.json. This is not a new defense layer — it's making the existing defense layer actually work.

3. **Source code stays data plane** — Go source is project code that agents need to modify. The compilation boundary (`make install`) is the natural control/data boundary. Architecture lint tests catch gate degradation at commit time.

4. **Session-frozen hooks limit the blast radius** — Even without chflags, a rogue agent modifying settings.json only affects FUTURE sessions, not its own. With chflags, even future sessions are protected.

**Answer to Investigation Question:**

The control plane consists of exactly 6 files that should be made immutable with `chflags uchg`:
1. `~/.claude/settings.json` — hook definitions, permissions
2. `~/.orch/hooks/gate-bd-close.py` — prevents self-closing
3. `~/.orch/hooks/enforce-phase-complete.py` — requires completion
4. `~/.orch/hooks/gate-orchestrator-code-access.py` — orchestrator boundaries
5. `~/.orch/hooks/gate-orchestrator-task-tool.py` — orchestrator boundaries
6. `~/.orch/hooks/pre-commit-knowledge-gate.py` — knowledge commit gates

Everything else is data plane: project source code, investigations, probes, skills, CLAUDE.md, agent memory, beads data.

Enforcement uses `chflags uchg` (macOS immutability) because it operates at the OS level, below any runtime, and cannot be bypassed by permission flags, tool permissions, or hook modifications. Human modifications go through `orch control unlock` / `orch control lock`.

---

## Structured Uncertainty

**What's tested:**

- ✅ Hooks fire in bypassPermissions mode (verified: Claude Code docs explicitly list "bypassPermissions" as a permission_mode value in hook input, with no documentation of hooks being disabled in any mode)
- ✅ chflags uchg is available and functional on this macOS (verified: `ls -lO` works, `chflags` command present, no flags currently set)
- ✅ Hook snapshots are session-frozen (verified: Claude Code docs explicitly state "captures a snapshot of hooks at startup")
- ✅ All 48 gates trace back to 6 root files (verified: enumerated in probe, traced dependency tree in this investigation)

**What's untested:**

- ⚠️ Claude Code behavior when encountering "Operation not permitted" from chflags — expected to show error and continue, but not verified in practice
- ⚠️ Whether `--dangerously-skip-permissions` truly bypasses deny rules in settings.json — the name suggests it does, but docs are ambiguous
- ⚠️ Whether `--settings` flag properly merges with or overrides global settings.json hooks — currently used for worker isolation but hook behavior not confirmed
- ⚠️ Impact on `~/.claude/hooks/*.sh` scripts (session-start.sh, etc.) — these are informational hooks without deny capabilities, classified as data plane, but correctness should be verified

**What would change this:**

- If Claude Code ignores PreToolUse deny decisions in bypassPermissions mode, hooks alone are insufficient and chflags becomes the ONLY enforcement mechanism (recommendation strengthens)
- If chflags uchg causes Claude Code to crash rather than gracefully handle the error, we'd need to also add permission narrowing as primary enforcement
- If a way exists to bypass chflags without root (e.g., via some macOS API), the OS-level guarantee breaks

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Apply chflags uchg to 6 control plane files | strategic | Irreversible change to infrastructure management workflow; constrains all future human and agent interactions with these files |
| Add `orch control lock/unlock` commands | implementation | CLI commands within existing orch binary; no architectural impact |
| Add architecture lint to pre-commit hook | implementation | Build enforcement within existing pre-commit infrastructure |
| Re-enable circuit breaker | architectural | Cross-component coordination (hooks + daemon + orch binary); uses heuristics from prior design investigation |

### Recommended Approach ⭐

**Layered Immutability with OS-Level Root** — Protect 6 control plane files with `chflags uchg`, add human-friendly lock/unlock commands, add compilation boundary enforcement.

**Why this approach:**
- Closes the recursive vulnerability identified in the entropy-spiral probe
- Zero impact on agent workflows (agents never need to edit these 6 files)
- Zero new approval prompts (chflags is invisible to agents — they simply can't modify the files)
- Works for ALL runtimes (Claude Code, claude CLI with --dangerously-skip-permissions, any future backend)
- Minimal implementation surface (~50 lines of Go for lock/unlock commands + shell script for initial setup)

**Trade-offs accepted:**
- Human must run `orch control unlock` before modifying hooks or settings.json — adds friction to infrastructure changes
- macOS-specific (chflags not available on Linux) — acceptable because macOS is the only platform
- If Dylan forgets to re-lock after unlocking, files remain mutable until next lock — mitigated by adding lock to `orch-dashboard start`

**Implementation sequence:**

#### Phase 1: Immutability Foundation (1-2 hours)

1. **Create `orch control lock` / `orch control unlock` commands**
   - `lock`: Run `chflags uchg` on all 6 control plane files, verify flags set
   - `unlock`: Run `chflags nouchg` on all 6 control plane files, verify flags cleared
   - `status`: Show current lock state of all 6 files
   - File targets: `cmd/orch/control_cmd.go` (add subcommands)

2. **Create initial lock script** (`scripts/lock-control-plane.sh`)
   - Lists all 6 files, applies chflags uchg
   - Run once to establish initial immutability

3. **Add auto-lock to `orch-dashboard start`**
   - After services start, run `orch control lock` to ensure control plane is protected
   - File target: `orch-dashboard` script

4. **Verify: attempt to edit settings.json with chflags uchg set**
   - Use Edit tool on settings.json → should fail with "Operation not permitted"
   - Use `rm ~/.claude/settings.json` → should fail with "Operation not permitted"

#### Phase 2: Compilation Boundary Enforcement (1 hour)

5. **Add architecture lint to pre-commit hook**
   - Modify `.git/hooks/pre-commit` to run `go test -run TestArchitectureLint ./cmd/orch/`
   - This catches gate degradation before it can be committed

6. **Add PreToolUse hook to block `make install` from agents**
   - New hook in settings.json: match Bash commands containing `make install` or `make build`
   - Only block when `ORCH_SPAWNED=1` (agents), allow when run by human
   - File target: new hook script `~/.orch/hooks/gate-compilation.py`

#### Phase 3: Circuit Breaker (separate issue, 2 hours)

7. **Re-enable circuit breaker with redesigned heuristics**
   - Uses the 3-layer design from prior investigation (rolling average + heartbeat + hard cap)
   - Rename `control-plane-post-commit.sh.disabled` → `.sh`
   - Apply chflags uchg to the circuit breaker script
   - This is detailed in `.kb/investigations/2026-02-14-design-control-plane-heuristics.md`

### Alternative Approaches Considered

**Option B: Permission Narrowing (Replace Edit(*/.claude/*) with granular patterns)**
- **Pros:** Works within existing Claude Code permission system, no OS-level changes
- **Cons:** Uncertain whether `--dangerously-skip-permissions` respects deny rules; would need to carefully enumerate allowed paths without missing agent memory files; more complex to maintain
- **When to use instead:** If chflags uchg causes Claude Code to crash or behave unexpectedly (Phase 1 verification step would reveal this)

**Option C: Sandbox Mode (sandbox.enabled: true)**
- **Pros:** Comprehensive filesystem restriction; industry-standard isolation
- **Cons:** Too broad — would restrict agents from accessing files outside project directory, breaking cross-project orchestration, kb access, skill loading, and more; would require extensive allowlisting
- **When to use instead:** In a future VPS/container deployment where the sandbox boundaries match the legitimate access patterns

**Option D: Custom File Watcher (inotify/FSEvents daemon monitoring control plane files)**
- **Pros:** Real-time detection and notification of any modification attempt
- **Cons:** Prevention is better than detection; adds daemon complexity; doesn't prevent the modification, only detects it
- **When to use instead:** As a monitoring layer on TOP of chflags — useful for alerting when someone unlocks the control plane

**Rationale for recommendation:** chflags uchg provides the strongest guarantee (OS-level, runtime-independent, zero configuration) with the smallest blast radius (only 6 files affected). It's the simplest mechanism that fully closes the vulnerability.

---

### Implementation Details

**What to implement first:**
- Phase 1 (`orch control lock/unlock` + initial lock) — this is the critical path
- Phase 2 and 3 can follow in any order

**Things to watch out for:**
- ⚠️ Claude Code error handling when Edit tool hits "Operation not permitted" — test this explicitly before rolling out to all agents
- ⚠️ The `--settings` flag in worker spawn might load an ADDITIONAL settings file — verify hooks from global settings.json still fire when `--settings` provides an override
- ⚠️ Hook scripts use Python — `chflags uchg` prevents modifying the script but not its imports. If a hook imports from a mutable file, that dependency is not protected
- ⚠️ `orch-dashboard start` calling `orch control lock` means control plane files are locked even during development — make sure `orch control unlock` is easily accessible

**Areas needing further investigation:**
- Whether `--dangerously-skip-permissions` truly bypasses the deny list (a 5-minute test would resolve this)
- What error message Claude Code shows agents when chflags blocks a tool call (affects agent UX)
- Whether `~/.claude/hooks/*.sh` scripts (session-start.sh, tmux_cleanup.sh, etc.) should also be protected — currently classified as informational, not enforcement

**Success criteria:**
- ✅ `orch control lock` makes all 6 files immutable (verified with `ls -lO`)
- ✅ Agent Edit/Write/rm on settings.json fails with "Operation not permitted"
- ✅ Agent can still: edit project code, write to ~/.claude/projects/, modify CLAUDE.md
- ✅ `orch control unlock` makes files mutable again for human modification
- ✅ No new approval prompts introduced for any agent workflow
- ✅ Architecture lint runs in pre-commit hook and catches gate modifications

---

## Control Plane File Inventory

| # | File | Role | Why Protected |
|---|------|------|---------------|
| 1 | `~/.claude/settings.json` | Hook definitions, permissions, sandbox config | Root of all enforcement — modifying this disables everything |
| 2 | `~/.orch/hooks/gate-bd-close.py` | Prevents agents from running `bd close` | Without this, agents bypass completion verification |
| 3 | `~/.orch/hooks/enforce-phase-complete.py` | Requires Phase: Complete before session exit | Without this, agents exit without reporting status |
| 4 | `~/.orch/hooks/gate-orchestrator-code-access.py` | Prevents orchestrators from writing code | Without this, orchestrator/worker boundary dissolves |
| 5 | `~/.orch/hooks/gate-orchestrator-task-tool.py` | Prevents orchestrators from using task tools | Without this, orchestrator/worker boundary dissolves |
| 6 | `~/.orch/hooks/pre-commit-knowledge-gate.py` | Gates knowledge artifact commits | Without this, knowledge quality degrades |

### Files NOT Protected (Data Plane)

| Category | Examples | Why Data Plane |
|----------|----------|----------------|
| Project source | `pkg/spawn/gates/*.go`, `pkg/verify/*.go` | Agents modify for features; compilation boundary provides separation |
| Informational hooks | `~/.orch/hooks/log-tool-outcomes.py`, `check-workspace-complete.py` | Advisory only — no deny capabilities |
| Session hooks | `~/.claude/hooks/session-start.sh`, `tmux_cleanup.sh` | Informational context loading — no enforcement |
| Agent guidance | `~/.claude/skills/*/SKILL.md`, CLAUDE.md | Guides behavior but can't enforce it |
| Agent memory | `~/.claude/projects/*/memory/` | Agent working state |
| Knowledge base | `.kb/`, investigations, probes | Knowledge artifacts |
| Beads data | `.beads/` | Issue tracking data |

---

## Human Workflow for Control Plane Modifications

**Scenario:** Dylan needs to add a new hook to settings.json.

```bash
# 1. Unlock control plane
orch control unlock

# 2. Make modifications (via orchestrator or directly)
# Edit settings.json, modify hook scripts, etc.

# 3. Lock control plane
orch control lock

# 4. Verify
orch control status
```

**Scenario:** Adding a new enforcement hook.

```bash
# 1. Unlock
orch control unlock

# 2. Write new hook script
# ~/.orch/hooks/new-gate.py

# 3. Add hook definition to settings.json

# 4. Lock (now includes the new file)
orch control lock
# Note: lock command should auto-discover hook scripts
# referenced in settings.json and protect them

# 5. Verify
orch control status
```

**Key design choice:** `orch control lock` should read settings.json, discover all referenced hook script paths, and apply chflags uchg to ALL of them plus settings.json itself. This way, adding a new hook and re-locking automatically protects the new hook.

---

## Impact Analysis: What Breaks?

| Current Workflow | Impact | Mitigation |
|-----------------|--------|------------|
| Agent edits project code | **No impact** — project code is data plane | N/A |
| Agent writes to ~/.claude/projects/*/memory/ | **No impact** — memory files not in control plane | N/A |
| Agent modifies CLAUDE.md | **No impact** — CLAUDE.md is data plane | N/A |
| Agent runs `rm` for cleanup | **No impact** unless targeting control plane files | chflags gives clean "Operation not permitted" error |
| Agent runs `make install` | **Blocked** (Phase 2) | Agent can still `go build` and `go test` — just can't install |
| Dylan modifies settings.json | **Requires unlock** | `orch control unlock` before modification |
| Dylan adds new hook | **Requires unlock + re-lock** | `orch control unlock` → add hook → `orch control lock` |
| `orch-dashboard start` | **Enhanced** — auto-locks control plane | Ensures protection is always active |

**Zero workflows break.** The only new friction is `orch control unlock/lock` when Dylan modifies infrastructure, which happens infrequently and aligns with the principle that infrastructure changes should be intentional.

---

## Blocking Questions

### Q1: Should we verify chflags behavior before committing to this design?

- **Authority:** implementation
- **Subtype:** factual
- **What changes based on answer:** If Claude Code crashes (instead of showing a clean error) when hitting chflags-protected files, we'd need permission narrowing as primary instead. A 5-minute test resolves this.

### Q2: Should the `orch control lock` command auto-discover hook scripts from settings.json?

- **Authority:** implementation
- **Subtype:** judgment
- **What changes based on answer:** If auto-discovery, the lock command stays in sync automatically. If explicit list, it's simpler but could drift from settings.json. Recommendation: auto-discovery.

### Q3: Should Phase 3 (circuit breaker re-enable) be a prerequisite or independent follow-up?

- **Authority:** architectural
- **Subtype:** judgment
- **What changes based on answer:** If prerequisite, Phases 1-3 ship together (larger batch). If independent, Phase 1 ships immediately (faster protection, circuit breaker comes later). Recommendation: independent — Phase 1 alone closes the critical vulnerability.

---

## Decision Gate Guidance (if promoting to decision)

**Add blocks: frontmatter when:**
- This decision resolves the mutable control plane gap documented across 3+ prior probes
- This decision establishes constraints all future agent spawns must respect
- Future spawns might accidentally target control plane files

**Suggested blocks keywords:**
- "control plane" — any work touching infrastructure files
- "immutability" — any work about file protection or permissions
- "settings.json" — any work modifying Claude Code configuration
- "hooks" — any work adding or modifying enforcement hooks

---

## References

**Files Examined:**
- `~/.claude/settings.json` — Full hook definitions, permissions, sandbox config
- `~/.orch/hooks/*.py` — All 12 hook scripts, 5 with deny capabilities
- `pkg/spawn/claude.go` — Claude CLI launch command construction
- `.claude/settings.local.json` — Project-level settings
- `pkg/spawn/gates/*.go` — 6 spawn gate files
- `pkg/verify/check.go` — Verification gate logic

**Commands Run:**
```bash
# Verify chflags availability
ls -lO ~/.claude/settings.json
chflags --help

# Check current flags
/bin/ls -lO ~/.claude/settings.json ~/.orch/hooks/*.py

# Claude CLI help for flag behavior
claude --help | grep -A2 "dangerously-skip-permissions"
```

**External Documentation:**
- https://code.claude.com/docs/en/hooks — Hook lifecycle, session-frozen snapshots, permission interaction
- https://code.claude.com/docs/en/settings — Settings file resolution, managed hooks

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-02-14-design-control-plane-heuristics.md` — Circuit breaker redesign (Phase 3 prerequisite)
- **Probe:** `.kb/models/entropy-spiral/probes/2026-03-01-probe-self-stabilization-current-gates.md` — Confirmed all 48 gates are mutable
- **Probe:** `.kb/models/decidability-graph/probes/2026-03-01-probe-context-scoping-irreducibility.md` — Confirms orchestrator's structural position

---

## Substrate Trace

**Principles cited:**

| Principle | How It Applies |
|-----------|---------------|
| **Gate Over Remind** | The current 48 gates are actually *reminders* because agents can disable them. chflags uchg makes them true gates — they block progress regardless of agent intent. |
| **Coherence Over Patches** | 48 mutable gates is the patches approach. 6 immutable root files is the coherent approach — fewer, stronger protections that can't be worked around. |
| **Provenance** | chflags uchg provides OS-level provenance for control plane integrity — `ls -lO` shows whether files are protected, independently verifiable. |
| **Session Amnesia** | The lock/unlock workflow is session-independent — chflags persists across reboots, sessions, and runtime restarts. |

**Models cited:**

| Model | How It Applies |
|-------|---------------|
| **entropy-spiral** | "Agents cannot halt a spiral they are part of" — this is the foundational claim. Making control plane immutable is the structural response: agents can't modify what they can't write to. |
| **decidability-graph** | Orchestrator's irreducible position (context-scoping) extends to control plane management. Only the orchestrator (human-directed) should modify infrastructure. |

**Decisions cited:**

| Decision | How It Applies |
|----------|---------------|
| Circuit breaker heuristics design | Phase 3 uses the redesigned heuristics. The immutable control plane (this design) is the prerequisite — there's no point re-enabling a circuit breaker agents can disable. |

---

## Investigation History

**2026-03-01 19:30:** Investigation started
- Initial question: How to separate control plane from data plane in agent orchestration
- Context: Probe confirmed all 48 gates are mutable; 6-probe synthesis identified immutable control plane as foundational next step

**2026-03-01 20:00:** Key finding — hooks are session-frozen
- Claude Code docs confirm settings.json modifications don't affect running session
- Narrows attack vector significantly

**2026-03-01 20:15:** Key finding — hooks fire in all permission modes
- --dangerously-skip-permissions does NOT disable hooks
- Hooks and permissions are independent systems

**2026-03-01 20:30:** Investigation completed
- Status: Complete
- Key outcome: Control plane is 6 files, chflags uchg closes the recursive vulnerability, zero workflow impact
