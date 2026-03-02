# Probe: Can Current Gates Prevent Agent Self-Destabilization?

**Model:** entropy-spiral
**Date:** 2026-03-01
**Status:** Complete

---

## Question

The entropy-spiral model claims: "Empirically proven to fail. Agents cannot halt a spiral they are part of." (Failure Mode 2, established across 3 spirals Dec 2025-Feb 2026). This claim was established pre-enforcement — before the current 48-gate, 4-layer defense system was built.

**Test question:** Have the current defenses (spawn gates, verification gates, architecture lint tests, Claude Code hooks, accretion boundaries) closed the mutable control plane gap? Or are they just more mutable defenses?

---

## What I Tested

### 1. Cataloged All Defense Layers

Enumerated every defense mechanism by reading source code and configuration files.

```bash
# Spawn gates (pkg/spawn/gates/)
ls pkg/spawn/gates/*.go
# → 6 gate files: hotspot.go, triage.go, concurrency.go, ratelimit.go, verification.go, agreements.go

# Verification gates (pkg/verify/)
grep -c 'result.Passed = false' pkg/verify/check.go
# → 18 failure paths across V0-V3 levels

# Architecture lint tests
cat cmd/orch/architecture_lint_test.go
# → 4 test functions checking lifecycle state packages, imports, diff, files

# Claude Code hooks (~/.orch/hooks/)
ls ~/.orch/hooks/*.py | wc -l
# → 12 hook scripts

# Hook denials (hard gates in hooks)
grep -rl 'permissionDecision.*deny' ~/.orch/hooks/ | wc -l
# → 5 hook scripts can deny tool use
```

### 2. Assessed Mutability of Each Defense

For each defense, checked: file permissions, Claude Code permission rules, sandbox status.

```bash
# Check Claude Code permissions for editing settings.json
grep 'Edit.*/.claude' ~/.claude/settings.json
# → "Edit(*/.claude/*)" — agents CAN edit the file that defines all hooks

# Check sandbox
grep -A3 '"sandbox"' ~/.claude/settings.json
# → "enabled": false

# Check if hooks have macOS immutability flags
ls -lO ~/.orch/hooks/gate-bd-close.py
# → No extended flags (no uchg, no schg)

# Check permission to delete files
grep 'Bash(rm' ~/.claude/settings.json
# → "Bash(rm:*)" — agents CAN delete any file
```

### 3. Checked Architecture Lint Test Enforcement

```bash
# Is there CI that runs tests?
ls .github/workflows/ 2>/dev/null
# → No GitHub Actions workflows found

# Does pre-commit hook run go test?
cat .git/hooks/pre-commit
# → Only runs "bd hooks run pre-commit" (beads flush, not go test)

# Can an agent skip/delete the test?
# architecture_lint_test.go is a regular Go file in the repo
# Agent has Edit(*/Documents/personal/*) and Bash(rm:*) permissions
```

### 4. Traced 3-Agent Attack Surface

Scenario: 3 agents simultaneously modify `cmd/orch/*.go` infrastructure files.

```
Agent 1: Modifies cmd/orch/spawn_cmd.go (spawn logic)
Agent 2: Modifies pkg/verify/check.go (verification gates)
Agent 3: Modifies pkg/spawn/gates/hotspot.go (spawn blocking)

Which gates fire?
- Pre-commit hook: NO (only flushes beads state)
- Architecture lint tests: NO (nobody runs go test automatically)
- Spawn gates: NO (these gate NEW spawns, not ongoing work)
- Verification gates: ONLY at orch complete (post-hoc, by orchestrator)
- Claude Code hooks: PARTIAL (some hooks check tool use, but...)
  → gate-orchestrator-code-access.py: coaching nudge only (allows, doesn't block)
  → enforce-phase-complete.py: only at session exit
  → gate-bd-close.py: only gates bd close command
  → None gate edits to infrastructure files
```

### 5. Checked Control Plane Circuit Breaker Status

```bash
ls ~/.orch/hooks/control-plane-post-commit.sh.disabled
# → The circuit breaker that enforced daily commit caps, rolling averages,
#   fix:feat ratio monitoring, and unverified velocity checks is DISABLED
```

### 6. Checked Binary vs Source Separation

```bash
ls -la $(which orch)
# → Symlink: ~/bin/orch → ./build/orch (compiled binary)
# Modifying source code doesn't immediately affect running orch binary
# BUT agent can run 'make install' to rebuild and update the binary
```

---

## What I Observed

### Defense Inventory: 4 Layers, All Mutable

| Layer | Mechanism | Count | Hard Gate? | Agent-Modifiable? |
|-------|-----------|-------|------------|-------------------|
| **Spawn gates** | pkg/spawn/gates/*.go | 6 files, ~9 error paths | YES (block spawn) | YES (source code in repo) |
| **Verify gates** | pkg/verify/check.go + level.go | 18 failure paths (V0-V3) | YES (block completion) | YES (source code in repo) |
| **Architecture lint** | cmd/orch/architecture_lint_test.go | 4 test functions | NO (only when go test runs) | YES (can delete or modify) |
| **Claude hooks** | ~/.orch/hooks/*.py | 12 scripts, 5 denials | YES (PreToolUse deny) | YES (Edit, rm allowed) |
| **Accretion boundary** | CLAUDE.md rule + verify gate | 1 soft rule + 1 gate | PARTIAL (gate at verify) | YES (CLAUDE.md is a file) |
| **Control plane circuit breaker** | control-plane-post-commit.sh | 1 script | N/A — **DISABLED** | YES (was a file) |

### Critical Permissions That Enable Modification

From `~/.claude/settings.json` permissions.allow:

1. **`Edit(*/.claude/*)`** — Agents can edit `~/.claude/settings.json` itself, removing all hook definitions
2. **`Bash(rm:*)`** — Agents can `rm ~/.orch/hooks/gate-bd-close.py` or any hook file
3. **`Write(*/Documents/personal/*)`** — Agents can overwrite any source file in orch-go
4. **`sandbox.enabled: false`** — No filesystem sandboxing
5. **`skipDangerousModePermissionPrompt: true`** — No confirmation for dangerous operations

### The Recursive Vulnerability (Unchanged)

The model identifies the recursive problem: "Defenses against the spiral (gates, metrics, coaching plugin) live inside orch-go. If agents modify orch-go, they modify the defenses."

Current state confirms this is still true for ALL layers:

```
settings.json (defines hooks) → agent can Edit(*/.claude/*)
     ↓
hook scripts (enforce rules) → agent can Bash(rm:*) or Edit
     ↓
spawn gates (block spawns) → agent can modify source
     ↓
verify gates (block completion) → agent can modify source
     ↓
architecture lint (detect violations) → agent can delete test file
     ↓
circuit breaker (halt system) → ALREADY DISABLED
```

### The One Partial Defense: Compiled Binary Separation

The `orch` binary at `~/bin/orch → build/orch` is a compiled Go binary. An agent modifying `pkg/verify/check.go` does NOT immediately affect `orch complete` behavior — the binary must be rebuilt with `make install`.

This provides a **temporal buffer** (time between source modification and binary update), not true immutability. An agent can still run `make install` to compile and deploy modified gates.

### What's New Since the Spirals (Dec-Feb)

Claude Code hooks represent a genuinely new defense layer:

- **PreToolUse hooks** intercept agent actions BEFORE execution (gate-bd-close.py, gate-orchestrator-task-tool.py)
- **Stop hooks** enforce Phase: Complete before session exit (enforce-phase-complete.py)
- These operate at the **tool invocation level**, architecturally closer to control plane than source-code gates

But they inherit the same vulnerability: the hook configuration (`settings.json`) and hook implementations (`.py` files) are both agent-writable.

---

## Model Impact

- [x] **Confirms** invariant: "Agent self-stabilization fails" — all current defenses are mutable by the agents they constrain
- [ ] **Contradicts** invariant
- [x] **Extends** model with: Claude Code hooks are a new defense class (tool-invocation-level gating) that didn't exist during the 3 spirals. They represent the closest current approach to control-plane enforcement but remain structurally mutable. The compiled binary provides a temporal buffer (not true immutability).

**Verdict: CONFIRMS with extension**

The mutable control plane gap has NOT been closed. The system has expanded from ~12 defenses to ~48 gates across 4 layers, but every defense remains modifiable by agents. The model's claim that "agents cannot halt a spiral they are part of" holds because:

1. No defense is architecturally immutable (no filesystem flags, no sandbox, no CI enforcement)
2. The permission model grants agents explicit ability to edit the hook configuration (`Edit(*/.claude/*)`) and delete hook scripts (`Bash(rm:*)`)
3. The circuit breaker (the closest thing to an external control) is disabled
4. Architecture lint tests have no automatic enforcement (no CI, no pre-commit integration)

The system has more defenses but the same structural vulnerability. More mutable walls do not equal one immutable wall.

---

## Notes

### Toward Actual Immutability: What Would Close the Gap

Based on this probe, the minimum changes for genuine control-plane immutability:

1. **Make `~/.claude/settings.json` immutable:** `chflags uchg ~/.claude/settings.json` (macOS) — agents get "Operation not permitted" on edit/delete. Human runs `chflags nouchg` to modify.

2. **Make hook scripts immutable:** `chflags uchg ~/.orch/hooks/*.py` — same pattern.

3. **Remove overly broad permissions:** Remove `Edit(*/.claude/*)`, `Bash(rm:*)` from the allow list. Replace with narrower patterns that exclude infrastructure files.

4. **Enable sandbox:** `sandbox.enabled: true` would restrict filesystem access to project boundaries.

5. **Re-enable circuit breaker:** Rename `control-plane-post-commit.sh.disabled` → `.sh` and integrate as git post-commit hook.

6. **Add CI or pre-commit architecture lint:** Integrate `go test -run TestArchitectureLint` into the pre-commit hook so it runs automatically.

### Why This Matters

The system has invested significant engineering effort in gates (48 across 4 layers). But the effort was spent on *coverage* (more gates) rather than *immutability* (protecting existing gates from modification). The model's prediction holds: quantity of mutable defenses does not substitute for quality of immutable ones.
