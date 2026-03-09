# Minimum Viable Harness (MVH) — Day-One Governance for Agent-Heavy Projects

**Purpose:** Checklist for setting up governance infrastructure in a new project that will use autonomous AI agents (orch spawn, daemon, multi-agent workflows).

**When to use:** Before the first `orch spawn` in a new project. The checklist is sequential — each tier builds on the previous.

**Key principle:** Enforcement must exist before agents operate. You cannot deploy governance through the system it governs (bootstrap paradox — see `.kb/global/models/control-plane-bootstrap.md`).

---

## Tier 0: Structural Scaffold (Day 0 — ~30 min)

**Automated via `orch init`:**

```bash
cd /path/to/new-project
orch init                          # Or: orch init --type go-cli
```

**Creates:**

| Item | Path | Purpose |
|------|------|---------|
| Workspace dir | `.orch/workspace/` | Agent session storage |
| Templates | `.orch/templates/` | SYNTHESIS.md, PROBE.md templates |
| Project config | `.orch/config.yaml` | Ports, spawn mode, model defaults |
| Knowledge base | `.kb/` | Guides, decisions, investigations, models |
| Issue tracking | `.beads/` | bd-powered issue lifecycle |
| Project charter | `CLAUDE.md` | Conventions, architecture, gotchas |
| Workers session | `~/.tmuxinator/workers-{project}.yml` | Tmux layout for agent windows |

**Verify:** `ls .orch/ .kb/ .beads/ CLAUDE.md` — all exist.

**Not yet safe for autonomous agents.** Proceed to Tier 1.

---

## Tier 1: Behavioral Enforcement (Day 1 — ~2-4h)

**Manual steps — must complete before first `orch spawn`.**

### 1. Add Deny Rules to settings.json

Prevent agents from modifying the files that define their constraints.

```bash
# Check current deny rules
orch control deny

# If missing, add to ~/.claude/settings.json → permissions.deny:
```

Required deny rules:
```json
{
  "permissions": {
    "deny": [
      "Edit(~/.claude/settings.json)",
      "Write(~/.claude/settings.json)",
      "Edit(~/.claude/settings.local.json)",
      "Write(~/.claude/settings.local.json)",
      "Edit(~/.orch/hooks/**)",
      "Write(~/.orch/hooks/**)"
    ]
  }
}
```

**Why essential:** Without deny rules, agents can modify the file that defines their own restrictions — the recursive vulnerability (SPAWN_CONTEXT constraint).

### 2. Install Gate: Prevent Agent Self-Close

Agents must report `Phase: Complete` — only orchestrators close issues via `orch complete`.

```bash
# Copy gate-bd-close.py to ~/.orch/hooks/ (if not already present)
# Register in ~/.claude/settings.json → hooks.PreToolUse
```

Hook registration in settings.json:
```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "python3 ~/.orch/hooks/gate-bd-close.py"
          }
        ]
      }
    ]
  }
}
```

**Why essential:** Without this gate, agents bypass the verification pipeline by closing their own issues.

### 3. Install Gate: Prevent `git add -A`

```bash
# Copy gate-worker-git-add-all.py to ~/.orch/hooks/
# Register in same PreToolUse hook chain
```

**Why essential:** `git add -A` stages secrets, unrelated changes, and build artifacts. Workers must stage files by name.

### 4. Create Beads Close Hook

```bash
# Create .beads/hooks/on_close (executable)
chmod +x .beads/hooks/on_close
```

Minimal on_close hook:
```bash
#!/bin/bash
# Emit completion event when issues are closed
ISSUE_ID="$1"
orch emit agent.completed --beads-id "$ISSUE_ID" --reason "Closed via bd close" 2>/dev/null || true
```

**Why essential:** Without event emission, agent completions are invisible to metrics and monitoring.

### 5. Wire Pre-Commit Growth Gate

Add to `.git/hooks/pre-commit`:
```bash
#!/bin/bash
# Accretion warning gate
orch precommit accretion 2>/dev/null || true
```

**Why essential:** This is the one hard gate — warns when files grow past thresholds. The harness-engineering model's invariant: "Every convention without a gate will eventually be violated."

### 6. Add Governance Sections to CLAUDE.md

`orch init` creates a structural CLAUDE.md. Add governance sections:

- **Accretion Boundaries:** "Files >1,500 lines require extraction before feature additions."
- **Authority Delegation:** Who can decide what (implementation vs architectural vs strategic)
- **Spawn Conventions:** Default skill for tasks, model routing

### 7. Lock Control Plane

```bash
orch harness lock
```

**Verify:**
```bash
orch harness status    # All files show LOCKED
orch harness verify    # Exits 0
```

**Why essential:** Mutable hard harness is soft harness with extra steps. OS-level immutability is the final seal.

---

## Tier 2: Verification & Observability (Week 1 — ~4-8h)

### 8. Configure Completion Verification

Set up `orch complete` with at minimum:
- Phase: Complete check (agent reported completion)
- Test evidence check (agent ran tests and reported results)

### 9. Verify Event Logging

```bash
# Spawn a test agent
orch spawn investigation "verify harness setup" --issue test-001

# After completion, check events
tail -5 ~/.orch/events.jsonl
```

Expect `session.spawned` and `agent.completed` events.

### 10. Human Behavioral Verification (CRITICAL)

Run one full agent lifecycle with human observation:

1. `orch spawn investigation "test task"` — observe spawn context generated
2. Watch agent work in tmux — observe hooks firing on tool use
3. Agent reports `Phase: Complete` — observe completion gate
4. `orch complete <id>` — observe verification pipeline
5. Confirm: deny rules blocked control plane edits? bd close was gated? Pre-commit warned on growth?

**Why essential:** The bootstrap model's core claim: "The first deployment of enforcement must be human-verified end-to-end." Code that exists but has never been observed working is enforcement theater.

### 11. Configure Hotspot Gate (if codebase >10K lines)

```bash
orch hotspot    # Check current file sizes
```

If files already exceed 800 lines, configure spawn gates before enabling feature-impl agents.

### 12. Configure Circuit Breaker (if daemon enabled)

Before `orch daemon run`:
- Set rolling average thresholds (warn at 50 commits/3-day, halt at 70)
- Set hard cap (150 commits/day)
- Set verification recency check (halt if heartbeat stale >2 days AND >15 daily commits)

---

## What's NOT in MVH (Accrete Later)

These are valuable but not day-one essential:

| Component | When to Add | Signal That It's Needed |
|-----------|-------------|------------------------|
| Architecture lint tests | When codebase exceeds 20K lines | Repeated import boundary violations |
| Spawn rate limiter | When running 5+ concurrent agents | Agents stepping on each other's work |
| Coaching plugin | When using OpenCode (not Claude CLI) | Agents looping without self-correction |
| Entropy agent | After 1+ month of agent operations | Hotspot count increasing week-over-week |
| Knowledge base guides | Emerge from operations | Same question investigated 3+ times |
| Custom skill system | When worker-base defaults are insufficient | Agents consistently misrouting or misunderstanding tasks |
| Dashboard/web UI | When monitoring >3 concurrent agents | Can't track agent status via CLI alone |

---

## Quick Reference: Verification Commands

```bash
# Check structural scaffold
ls .orch/ .kb/ .beads/ CLAUDE.md

# Check deny rules
orch control deny

# Check control plane lock state
orch harness status

# Verify lock (pre-commit integration)
orch harness verify

# Check file hotspots
orch hotspot

# Check events are flowing
tail -5 ~/.orch/events.jsonl
```

---

## See Also

- `.kb/models/harness-engineering/model.md` — Why hard harness > soft harness
- `.kb/global/models/control-plane-bootstrap.md` — Why order matters for enforcement deployment
- `.kb/decisions/2026-02-26-three-layer-hotspot-enforcement.md` — Hotspot gate design
- `.kb/investigations/2026-03-08-inv-define-minimum-viable-harness-agent.md` — This guide's source investigation
