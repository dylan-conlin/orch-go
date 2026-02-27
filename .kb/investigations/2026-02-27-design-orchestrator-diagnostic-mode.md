# Design: Orchestrator Diagnostic Mode

**Question:** How should orchestrators get time-limited, read-only code access during active debugging without sliding into implementer behavior?

**Started:** 2026-02-27
**Updated:** 2026-02-27
**Owner:** feature-impl (orch-go-cp52)
**Phase:** Complete
**Status:** Complete

---

## TL;DR

Add `orch diagnostic start --duration 10m "tracing shipping data path"` that creates a timestamped flag file. The existing `gate-orchestrator-code-access.py` hook checks the flag and allows Read (never Edit/Write) when diagnostic mode is active and not expired. Anti-slide safeguards: hard time cap, read-only enforcement, purpose declaration, per-read coaching injection, and post-diagnostic summary logged to events.jsonl.

---

## Problem

The frame guard (`gate-orchestrator-code-access.py`) blocks orchestrators from reading code files. This is correct for routine orchestration — reading code causes frame collapse. But during **active debugging**, the orchestrator needs to trace data paths through source files to form accurate diagnoses before spawning agents.

**Evidence (Feb 27, 2026):** Frame guard blocked code reading 4 times across 3 sessions. Each block forced spawning an investigation agent with 2-5 minute delay. In the price-watch post-mortem, the orchestrator couldn't read a `.svelte` file to trace a data path, forcing Dylan to relay the information manually ("User as Message Bus" failure).

**The paradox:** Orchestrators are responsible for accurate diagnosis, but can't access the information needed to diagnose accurately.

---

## Findings

### Finding 1: Current Enforcement is Three Layers Deep

**Evidence:**

| Layer | Mechanism | Scope | Bypass Risk |
|-------|-----------|-------|-------------|
| `--disallowedTools` | CLI flag at spawn | Removes Task, Edit, Write, NotebookEdit entirely | None (tools don't exist) |
| `gate-orchestrator-code-access.py` | PreToolUse hook | Blocks Read on code files (.go, .ts, .css, etc.) | None (fires every call) |
| Skill instructions | Orchestrator SKILL.md | "orchestrators don't read code" | High (soft, overridden by framing) |

The diagnostic mode only needs to modify **Layer 2** (the hook). Layer 1 (`--disallowedTools`) already correctly removes Edit/Write/Task — those should stay blocked even in diagnostic mode. Layer 3 (skill instructions) provides soft guidance that diagnostic mode explicitly overrides.

**Source:** `pkg/spawn/claude.go:93-96`, `~/.orch/hooks/gate-orchestrator-code-access.py`, orchestrator SKILL.md lines 48-49

**Significance:** Single point of modification. Only the hook needs to know about diagnostic mode.

---

### Finding 2: The Slide Pattern Has a Predictable Trajectory

**Evidence:** From the Orchestrator Session Lifecycle model's "Why This Fails" section and the Price-Watch Phase 1 Incident:

```
Read code "to understand" → Form hypothesis → "Just need to check one more thing"
→ Start debugging mentally → "I could fix this faster myself"
→ Frame collapse (implementation mode)
```

The slide has three phases:
1. **Legitimate trace** (minutes 0-5): Reading code to answer a specific diagnostic question
2. **Scope creep** (minutes 5-10): Reading adjacent files, forming implementation ideas
3. **Full collapse** (minutes 10+): Attempting to fix, debug, or implement

**Source:** `.kb/models/orchestrator-session-lifecycle/model.md` (Why This Fails: Frame Collapse section), `.kb/investigations/2026-02-27-postmortem-communication-breakdown-sessions.md`

**Significance:** Time-limiting alone catches the slide at phase 2. Adding purpose-anchoring catches it earlier.

---

### Finding 3: The Hook Already Has All Needed Infrastructure

**Evidence:** `gate-orchestrator-code-access.py` already:
- Detects orchestrator sessions via `CLAUDE_CONTEXT`
- Distinguishes code files from non-code files via extension allowlist
- Returns structured deny responses with coaching messages

Adding a diagnostic mode check requires only:
1. Read a flag file (`~/.orch/diagnostic-mode.json`)
2. Check if timestamp hasn't expired
3. If active: allow Read, inject coaching reminder
4. If expired/absent: deny as current behavior

**Source:** `~/.orch/hooks/gate-orchestrator-code-access.py` lines 108-146

**Significance:** Minimal implementation — no new hooks, no new env vars, no spawn changes.

---

### Finding 4: Coaching Injection Prevents Slide Better Than Time Alone

**Evidence:** The coaching plugin model shows tiered pressure is effective — first warning is soft, repeated violations trigger stronger coaching. The same principle applies here: each code file read in diagnostic mode should inject a brief anchoring message.

The message serves two functions:
1. **Purpose anchor:** Reminds the orchestrator WHY they entered diagnostic mode
2. **Slide detector:** If the purpose no longer matches what they're reading, that's the slide signal

**Source:** `.kb/models/coaching-plugin/model.md`, coaching.ts tiered injection pattern

**Significance:** Per-read coaching is more effective than a single "you have 10 minutes" warning because it creates continuous pressure against scope drift.

---

## Design

### CLI Interface

```bash
# Enter diagnostic mode (required: purpose declaration + duration)
orch diagnostic start --duration 10m "tracing shipping data path through scraper → API → frontend"

# Check status
orch diagnostic status
# → Diagnostic mode: ACTIVE (7m remaining)
# → Purpose: "tracing shipping data path through scraper → API → frontend"
# → Files read: 3 (.rb, .svelte, .ts)

# Exit early
orch diagnostic stop

# Attempt to exceed max duration
orch diagnostic start --duration 30m "..."
# → Error: Maximum diagnostic duration is 15 minutes. Use --duration 15m or less.
```

**Constraints:**
- Default duration: 10 minutes
- Maximum duration: 15 minutes (hard cap, not configurable)
- Purpose string required (cannot be empty)
- Only one diagnostic session at a time

### Flag File Format

`~/.orch/diagnostic-mode.json`:

```json
{
  "started_at": "2026-02-27T15:30:00Z",
  "expires_at": "2026-02-27T15:40:00Z",
  "purpose": "tracing shipping data path through scraper → API → frontend",
  "session_id": "abc123",
  "files_read": []
}
```

When diagnostic mode expires or is stopped, the file is deleted and an event is logged.

### Hook Modification (`gate-orchestrator-code-access.py`)

Current flow:
```
Is orchestrator? → Is code file? → DENY
```

New flow:
```
Is orchestrator? → Is code file? → Is diagnostic mode active?
  → YES + not expired → ALLOW (with coaching injection via permissionDecisionReason)
  → YES + expired → auto-cleanup flag file → DENY (with "diagnostic expired" message)
  → NO → DENY (current behavior)
```

**Critical:** Only `Read` is allowed in diagnostic mode. `Edit` remains blocked by `--disallowedTools` (Layer 1). This is defense-in-depth — even if someone manually creates the flag file, they can only read, never write.

### Per-Read Coaching Injection

When diagnostic mode allows a code file read, the hook returns `"allow"` but also logs the file. The coaching plugin (or a PostToolUse hook) can inject an anchoring message after each read:

```
📋 DIAGNOSTIC MODE (6m remaining): "tracing shipping data path"
   Files read this session: 3
   ⚠️ If your purpose has shifted from tracing to fixing, run: orch diagnostic stop
```

**Implementation choice:** Rather than a PostToolUse hook (which adds complexity), the simplest approach is to update the flag file's `files_read` array on each allowed read. The coaching plugin can then check this file periodically and inject messages when the count grows.

Alternatively, the per-read message can be injected as the `permissionDecisionReason` on an `"allow"` decision — but Claude Code hooks may not surface allow-reasons to the agent. **Needs testing.** If allow-reasons aren't surfaced, use the coaching plugin integration instead.

### Auto-Expiry

The hook itself enforces expiry:

```python
def check_diagnostic_mode() -> bool:
    """Check if diagnostic mode is active and not expired."""
    flag_path = Path.home() / ".orch" / "diagnostic-mode.json"
    if not flag_path.exists():
        return False

    try:
        data = json.loads(flag_path.read_text())
        expires_at = datetime.fromisoformat(data["expires_at"])
        if datetime.now(timezone.utc) > expires_at:
            # Auto-cleanup expired flag
            flag_path.unlink(missing_ok=True)
            log_diagnostic_end(data, reason="expired")
            return False
        return True
    except (json.JSONDecodeError, KeyError):
        flag_path.unlink(missing_ok=True)
        return False
```

No background timer needed. The hook checks on every Read attempt.

### Event Logging

When diagnostic mode ends (expired or manual stop), log to `~/.orch/events.jsonl`:

```json
{
  "type": "diagnostic.ended",
  "timestamp": "2026-02-27T15:40:00Z",
  "duration_seconds": 600,
  "purpose": "tracing shipping data path",
  "files_read": ["pkg/scraper/shipping.rb", "src/routes/api/quotes/+server.ts", "src/lib/components/QuoteTable.svelte"],
  "reason": "expired"
}
```

This provides post-hoc visibility: did the orchestrator use diagnostic mode appropriately? Did file access match the stated purpose?

---

## Anti-Slide Safeguards (Summary)

| Safeguard | What It Prevents | Enforcement |
|-----------|-----------------|-------------|
| **Time cap** (15m max) | Extended code reading → full collapse | Hard limit in CLI + hook expiry check |
| **Read-only** | "While I'm here, let me fix..." | `--disallowedTools` blocks Edit/Write at Layer 1 |
| **Purpose declaration** | Aimless browsing | Required string on `orch diagnostic start` |
| **Per-read anchoring** | Scope drift (reading unrelated files) | Coaching message after each code file read |
| **Post-diagnostic log** | Unreviewed diagnostic sessions | Event logged to events.jsonl |
| **Single-session scope** | Diagnostic mode becoming permanent | Flag file scoped to current session, auto-expires |

### What This Does NOT Prevent

- **Legitimate strategic code reading:** That's the point. Orchestrators sometimes need to read code for diagnosis.
- **Spawning from within diagnostic mode:** Orchestrators can (and should) `orch spawn` based on what they learn.
- **Reading non-code files:** Those were never blocked.

---

## Implementation Plan

**Estimated effort:** ~2 hours, 3 files

### Step 1: `orch diagnostic` CLI command (~45 min)

New file: `cmd/orch/diagnostic_cmd.go`
- Subcommands: `start`, `stop`, `status`
- Validates duration (max 15m), requires purpose string
- Writes/reads/deletes `~/.orch/diagnostic-mode.json`
- Logs events to `~/.orch/events.jsonl`

### Step 2: Modify hook (~30 min)

Edit: `~/.orch/hooks/gate-orchestrator-code-access.py`
- Add `check_diagnostic_mode()` function
- When active: allow Read, append file to `files_read` in flag file
- When expired: auto-cleanup, deny with "diagnostic expired" message

### Step 3: Coaching integration (~30 min)

Edit: `.opencode/plugin/coaching.ts` OR new PostToolUse hook
- After code file read in diagnostic mode: inject anchoring message
- Include remaining time, purpose, file count
- **Alternative:** If `permissionDecisionReason` on allow is surfaced, skip this step entirely

### Step 4: Skill documentation (~15 min)

Edit: `~/.claude/skills/meta/orchestrator/SKILL.md`
- Add diagnostic mode to Mode Declaration Protocol section
- Document when to use it vs when to spawn investigation

---

## Structured Uncertainty

**Tested:**
- ✅ Hook mechanism works (gate-orchestrator-code-access.py is production-proven)
- ✅ Flag file approach is simple and race-free (single writer: CLI; single reader: hook)
- ✅ `--disallowedTools` will continue blocking Edit/Write regardless of diagnostic mode
- ✅ Events logging pattern exists (`~/.orch/events.jsonl`)

**Untested:**
- ⚠️ Whether `permissionDecisionReason` on `"allow"` decisions is surfaced to the agent (if not, need PostToolUse hook for coaching)
- ⚠️ Whether the coaching plugin can read `diagnostic-mode.json` fast enough to inject timely messages
- ⚠️ Whether 15-minute max is the right threshold (may need adjustment after real-world usage)

**What would change this:**
- If per-read coaching proves too noisy → reduce to every 3rd read or time-interval coaching
- If 15m is too short for complex cross-repo tracing → consider 20m with explicit justification flag
- If diagnostic mode is used daily → it's a sign the frame guard is too aggressive, and the real fix is relaxing what counts as "code" (e.g., allow Read on files orchestrators frequently need to trace)

---

## Decision Recommendation

**Implement as designed.** The approach:

1. **Modifies one existing hook** (no new infrastructure)
2. **Adds one CLI command** (consistent with orch patterns)
3. **Preserves Edit/Write blocks** (defense-in-depth via Layer 1)
4. **Self-documents usage** (purpose + events log)
5. **Self-expires** (no cleanup burden)

The key insight is that diagnostic mode doesn't "relax" the frame guard — it creates a **bounded exception** with explicit entry ceremony, continuous pressure, and automatic exit. The orchestrator must deliberately choose to enter diagnostic mode, state why, and accept a time limit. This is fundamentally different from silently allowing code access.

---

## References

- `.kb/investigations/2026-02-27-postmortem-communication-breakdown-sessions.md` — Evidence of frame guard blocking during debugging
- `.kb/investigations/2026-02-24-spike-claude-code-hooks-orchestrator-guard.md` — Hook enforcement mechanisms
- `.kb/investigations/2026-02-24-design-orchestrator-skill-behavioral-compliance.md` — Behavioral compliance analysis
- `.kb/models/orchestrator-session-lifecycle/model.md` — Frame collapse failure mode
- `~/.orch/hooks/gate-orchestrator-code-access.py` — Current enforcement hook
- `pkg/spawn/claude.go` — `--disallowedTools` injection
