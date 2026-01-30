# Session Resume Protocol

**⚠️ DEPRECATED:** This entire protocol was removed Jan 19-21, 2026. Session handoff machinery (orch session start/end, .orch/session/ directories, auto-resume plugin) is no longer part of the system. This document is preserved for historical reference only.

**Replacement:** Context continuity now relies on kb/beads capture during work, not session handoffs. New sessions start fresh from durable state via `kb context`, `bd ready`, `orch status`.

**See:** `.kb/decisions/2026-01-19-remove-session-handoff-machinery.md` for removal rationale.

---

# Session Resume Protocol (Historical - Removed Jan 2026)

**Purpose:** Single authoritative reference for session handoff and automatic resume system.

**Scope:** This protocol applied ONLY to **interactive orchestrator sessions** (when Dylan starts Claude Code or OpenCode directly). Spawned worker agents use SPAWN_CONTEXT.md instead and did NOT use this resume system.

**Synthesized from:** Design doc (2026-01-11) + Implementation findings (2026-01-13)

**Removed:** Jan 19-21, 2026

---

## Quick Reference

```bash
# Manual resume (see what handoff exists)
orch session resume

# Check if handoff exists (exit code 0 = yes, 1 = no)
orch session resume --check

# For hook integration (bare content, no decoration)
orch session resume --for-injection

# End session and create handoff
orch session end
```

**Automatic behavior:** When you start a new **interactive orchestrator session** in Claude Code or OpenCode, hooks automatically inject the latest handoff if one exists.

**Not for workers:** Spawned agents (via `orch spawn`) receive SPAWN_CONTEXT.md at spawn time and do NOT use session resume.

---

## The Problem

**Context:** This addresses continuity for **interactive orchestrator sessions** specifically. Workers are spawned fresh each time with SPAWN_CONTEXT.md.

**Before session resume:**
- Dylan (orchestrator) had to remember session mechanics (create handoff, read handoff manually)
- Fresh orchestrator instances didn't receive prior context automatically
- Manual SESSION_HANDOFF.md creation bypassed proper tooling
- No parity with worker spawns (workers get SPAWN_CONTEXT.md automatically, orchestrators needed similar mechanism)

**Dylan's core need (for orchestrator sessions):**
> "I want to be able to start any orchestrator session just by saying 'let's resume' and from there, the orchestrator should have a protocol to determine what comes next. Session handoff mechanics should be handled automatically by the system, freeing me to think about higher level goals."

**Principle:** Pressure Over Compensation - if Dylan has to remember session mechanics, that's system failure.

---

## How It Works

**For interactive orchestrator sessions only.** Workers skip this entirely and use SPAWN_CONTEXT.md.

```
Dylan starts new orchestrator session
         │
         ▼
┌────────────────────────────────────────┐
│  INTERACTIVE SESSION START             │
│  (Claude Code or OpenCode - NOT spawn) │
└────────────────────────────────────────┘
         │
         ▼
┌────────────────────────────────────────┐
│  HOOK RUNS AUTOMATICALLY               │
│  ~/.claude/hooks/session-start.sh  OR  │
│  ~/.config/opencode/plugin/            │
│      session-resume.js                 │
└────────────────────────────────────────┘
         │
         ▼
┌────────────────────────────────────────┐
│  orch session resume --for-injection   │
│  Discovers handoff in:                 │
│  {project}/.orch/session/latest/       │
└────────────────────────────────────────┘
         │
         ▼
┌────────────────────────────────────────┐
│  HANDOFF INJECTED                      │
│  Fresh Claude sees context             │
│  (or silent if no handoff - fresh      │
│   start is valid)                      │
└────────────────────────────────────────┘
```

---

## Orchestrator Sessions vs Worker Sessions

| Aspect | Interactive Orchestrator Session | Spawned Worker Session |
|--------|----------------------------------|------------------------|
| **How started** | Dylan runs `claude` or `opencode` | `orch spawn <skill> "task"` |
| **Context source** | SESSION_HANDOFF.md (via hooks) | SPAWN_CONTEXT.md (at spawn) |
| **Resume behavior** | Auto-inject prior session handoff | Fresh start every time |
| **Duration** | Hours to days (multi-session) | 1-4 hours (single task) |
| **Purpose** | Strategic coordination | Task execution |
| **Who uses** | Orchestrators (Dylan as orchestrator) | Worker agents |

**Key insight:** Orchestrators need continuity across sessions. Workers are ephemeral and receive full context at spawn time.

---

## File Structure

Session handoffs are **window-scoped** to prevent concurrent orchestrator sessions from clobbering each other:

```bash
{project}/.orch/session/
├── orchestrator/                  # Window name
│   ├── latest -> 2026-01-13-0827/ # Symlink to most recent for this window
│   ├── 2026-01-11-1935/
│   │   └── SESSION_HANDOFF.md
│   └── 2026-01-13-0827/
│       └── SESSION_HANDOFF.md
└── meta-orchestrator/             # Different window, independent handoffs
    ├── latest -> 2026-01-13-0830/
    └── 2026-01-13-0830/
        └── SESSION_HANDOFF.md
```

**Examples:**
- `~/Documents/personal/orch-go/.orch/session/orchestrator/latest/SESSION_HANDOFF.md`
- `~/orch-knowledge/.orch/session/default/latest/SESSION_HANDOFF.md` (not in tmux)

**Window scoping:**
- Each tmux window gets its own handoff directory
- `orch session end` creates `.orch/session/{window-name}/{timestamp}/`
- `orch session resume` reads from `.orch/session/{window-name}/latest/`
- If not in tmux, uses "default" as window name
- Prevents concurrent sessions in different windows from interfering

**Why this matters:**
- Dylan can run multiple orchestrator sessions in different tmux windows
- Each session maintains independent context
- Session end from one window doesn't clobber another window's handoff
- Enables parallel orchestration workflows

---

## Command Modes

### Interactive Mode (Default)

```bash
orch session resume
```

**Behavior:**
- Discovers handoff by walking up directory tree to find `.orch/session/latest`
- Displays formatted handoff with source path
- Shows "Session Resumed" header with metadata
- Exit code 0 if found, 1 if not

**Use when:** You want to manually review the handoff before starting work.

---

### Injection Mode (For Hooks)

```bash
orch session resume --for-injection
```

**Behavior:**
- Outputs bare content only (no decorations)
- Silent if no handoff exists (fresh start is valid)
- Optimized for token efficiency
- Exit code 0 if found, 1 if not

**Use when:** Called by hooks to auto-inject context.

---

### Check Mode (Silent)

```bash
orch session resume --check
```

**Behavior:**
- Exit code 0 if handoff exists
- Exit code 1 if not found
- No output
- Useful for scripting/conditional logic

**Use when:** You need to test if a handoff exists without reading it.

---

## Creating Handoffs

### Automatic via Session End

```bash
# End current session
orch session end
```

**What happens:**
1. Prompts for reflection (session goal, accomplishments, friction)
2. Creates timestamped directory: `.orch/session/{timestamp}/`
3. Creates `SESSION_HANDOFF.md` from template
4. Updates `latest` symlink to point to new session
5. Prompts for git push (session close protocol)

**Template sections:**
- Session goal and duration
- What was accomplished
- Active agents and pending work
- Blockers and friction encountered
- Next priorities

---

### Manual Fallback

If `orch session end` isn't available or fails:

```bash
# Create directory manually
mkdir -p .orch/session/2026-01-13-0900

# Create handoff file
# (use template from ~/.orch/templates/SESSION_HANDOFF.md)

# Update symlink
cd .orch/session
ln -sf 2026-01-13-0900 latest
```

---

## Hook Integration

### Claude Code Hook

**Location:** `~/.claude/hooks/session-start.sh`

**What it does:**
- Runs `orch session resume --for-injection` before other hooks
- Injects handoff if found, silent if not
- Output appears in Claude's initial context

**Status:** ✅ Implemented (Jan 13, 2026)

---

### OpenCode Plugin

**Location:** `~/.config/opencode/plugin/session-resume.js`

**What it does:**
- Hooks into `on_session_created` event
- Runs `orch session resume --for-injection` from session's working directory
- Sends system message with handoff content if found

**Status:** ✅ Implemented (Jan 13, 2026)

---

## Discovery Logic

When you run `orch session resume`:

1. **Detect tmux window name** (or use "default" if not in tmux)
2. **Start from current directory**
3. **Walk up tree** to find `.orch/session/{window-name}/latest` symlink
4. **Read** `{symlink}/SESSION_HANDOFF.md`
5. **Exit code 1** if not found

**Key insight:** Discovery is window-scoped - you get the handoff for YOUR window, not other windows' handoffs.

**Example:**
```bash
# Working in orchestrator window
cd ~/Documents/personal/orch-go/cmd/orch

# Finds handoff for "orchestrator" window
orch session resume
# → Reads ~/Documents/personal/orch-go/.orch/session/orchestrator/latest/SESSION_HANDOFF.md

# Meanwhile, in meta-orchestrator window, gets different handoff
# → Reads ~/Documents/personal/orch-go/.orch/session/meta-orchestrator/latest/SESSION_HANDOFF.md
```

---

## Multi-Project Support

Each project maintains its own session history:

| Project | Handoff Location |
|---------|------------------|
| orch-go | `~/Documents/personal/orch-go/.orch/session/` |
| orch-knowledge | `~/orch-knowledge/.orch/session/` |
| kb-cli | `~/Documents/personal/kb-cli/.orch/session/` |

**No cross-contamination:** Working in orch-go discovers orch-go handoffs only, even if you have active sessions in other projects.

---

## Common Workflows

**Note:** These workflows apply to interactive orchestrator sessions only. Workers (spawned agents) don't use these commands.

### Starting Fresh Orchestrator Session

```bash
# Start new orchestrator session (no prior handoff)
# Hook runs, finds no handoff, stays silent
# ✅ This is valid - fresh starts are expected
```

**No error:** Fresh orchestrator sessions (first time in a project, or after long break) are valid scenarios.

---

### Resuming Orchestrator Session After Break

```bash
# Start new orchestrator session
# Hook automatically injects latest handoff
# Claude sees context from prior orchestrator session
```

**Zero cognitive load:** You (orchestrator) don't need to remember to resume. It happens automatically.

---

### Manual Review Before Resuming Orchestrator Session

```bash
# Before starting orchestrator session
orch session resume

# Review handoff content
# Decide if context is still relevant
# Then start orchestrator session (hook will inject same content)
```

**Use when:** You want to verify the handoff is still relevant before starting orchestrator work.

---

### Ending Orchestrator Session

```bash
# Before closing orchestrator session
orch session end

# Fill in reflection prompts
# Handoff created automatically
# Symlink updated to new session
```

**Critical:** Run this BEFORE closing the orchestrator session, not after. Fresh Claude won't have context to create meaningful handoff.

**Not for workers:** Worker agents complete via `bd comment <id> "Phase: Complete"` and `/exit`, not `orch session end`.

---

## Edge Cases

### No Handoff Exists (Fresh Start)

**Scenario:** First time in project, or no prior `orch session end` run.

**Behavior:**
- `orch session resume` exits with code 1
- Hooks detect and stay silent
- Claude starts without injected context

**This is valid.** Fresh starts are expected and should not produce errors.

---

### Stale Handoff (>7 Days Old)

**Current behavior:** Still injects without warning.

**Future enhancement:** Add warning like "⚠️ Handoff is 8 days old" to let orchestrator decide relevance.

**Not implemented yet** - tracked as Phase 4 optimization.

---

### Handoff Mid-Edit

**Scenario:** `SESSION_HANDOFF.md` exists but `latest` symlink is broken.

**Behavior:** Graceful failure with error message.

**Fix:** Repair symlink or complete the session end properly.

---

### Cross-Project Work

**Scenario:** Handoff in orch-go, but working in orch-knowledge.

**Behavior:** Only discovers handoff in current project tree. orch-knowledge session won't see orch-go handoff.

**If you need cross-project context:** Manually reference or copy relevant sections.

---

## Troubleshooting

### "No handoff found" but I created one

**Check:**
1. Is `.orch/session/latest` symlink present?
   ```bash
   ls -la .orch/session/latest
   ```
2. Does the symlink point to the right directory?
   ```bash
   readlink .orch/session/latest
   ```
3. Does `SESSION_HANDOFF.md` exist in that directory?
   ```bash
   ls .orch/session/latest/SESSION_HANDOFF.md
   ```

**Fix:**
- If symlink missing: `cd .orch/session && ln -sf {timestamp-dir} latest`
- If pointing wrong: Update symlink to correct directory
- If file missing: Run `orch session end` to create properly

---

### Hook not injecting handoff

**Check Claude Code:**
```bash
# Verify hook exists
ls -la ~/.claude/hooks/session-start.sh

# Test hook manually
~/.claude/hooks/session-start.sh
# Should show handoff if one exists
```

**Check OpenCode:**
```bash
# Verify plugin exists
ls -la ~/.config/opencode/plugin/session-resume.js

# Check OpenCode server logs for plugin loading
# (plugin load messages appear on server startup)
```

**Common causes:**
- Hook/plugin file not executable: `chmod +x ~/.claude/hooks/session-start.sh`
- `orch` command not in PATH for hook environment
- Plugin syntax error (check OpenCode server logs)
- **Hook output not in JSON format** (see below)
- **Hook not registered in settings.json** (see below)

---

### Hook runs but handoff doesn't appear (Claude Code)

**Symptoms:** Hook executes successfully when tested manually, but handoff doesn't appear in Claude Code session.

**Cause 1: Output format**
Hook output must be wrapped in JSON format for Claude Code:
```json
{
  "hookSpecificOutput": {
    "hookEventName": "SessionStart",
    "additionalContext": "content here"
  }
}
```

Plain text output is ignored. Check `~/.claude/hooks/session-start.sh` lines 10-22 for proper JSON wrapping.

**Cause 2: Hook not registered**
Hook must be in `~/.claude/settings.json`, not just `~/.claude/hooks/cdd-hooks.json`.

Check settings.json has:
```json
"SessionStart": [
  {
    "hooks": [
      {
        "type": "command",
        "command": "$HOME/.claude/hooks/session-start.sh",
        "timeout": 10
      }
    ]
  },
  ...
]
```

**Fix:** Commits `867d0af` and `ee14afc` in `~/.claude` repo contain correct configuration.

---

### Wrong project's handoff injected

**Cause:** Working directory confusion - hook runs `orch session resume` from wrong directory.

**Check:**
```bash
pwd
# Are you in the project you think you're in?

orch session resume
# Shows which handoff file it's reading
```

**Fix:** Ensure you're starting Claude session from the correct project directory.

---

## Session Close Protocol Integration

Session resume is part of the larger **Session Close Protocol**:

```
Before saying "done" or "complete":

[ ] 1. git status              (check what changed)
[ ] 2. git add <files>         (stage code changes)
[ ] 3. bd sync                 (commit beads changes)
[ ] 4. git commit -m "..."     (commit code)
[ ] 5. bd sync                 (commit any new beads changes)
[ ] 6. orch session end        (CREATE HANDOFF + reflection)
[ ] 7. git push                (push to remote)
```

**Critical:** `orch session end` (step 6) creates the handoff that enables resume. Skipping this step breaks continuity for the next session.

---

## Implementation Status

| Component | Status | Location |
|-----------|--------|----------|
| `orch session resume` command | ✅ Complete | `cmd/orch/session.go:537-657` |
| Session end symlink update | ✅ Complete | `cmd/orch/session.go:490-499` |
| Claude Code hook | ✅ Complete | `~/.claude/hooks/session-start.sh:6-16` |
| OpenCode plugin | ✅ Complete | `~/.config/opencode/plugin/session-resume.js` |
| Condensed format (Phase 4) | ❌ Not started | Design: lines 246-272 in design doc |

---

## Related Documentation

- `.kb/investigations/2026-01-11-design-session-resume-protocol.md` - Complete design rationale
- `.kb/investigations/2026-01-13-inv-implement-session-resume-protocol-orch.md` - Implementation findings
- `~/.claude/skills/meta/orchestrator/reference/session-reflection.md` - Session end guidance
- `.kb/guides/orchestrator-session-management.md` - Broader session management patterns

---

## Key Takeaways

1. **Scope:** This is for **interactive orchestrator sessions only** - workers use SPAWN_CONTEXT.md instead
2. **Zero cognitive load:** Dylan (orchestrator) doesn't need to remember to resume - hooks handle it automatically
3. **Project-specific:** Each repo maintains its own session history via `.orch/session/`
4. **Graceful degradation:** Fresh starts (no handoff) are valid and don't produce errors
5. **Forcing function:** `orch session end` is part of session close protocol - creates handoff for next orchestrator session
6. **Cross-environment:** Works in both Claude Code and OpenCode via hooks/plugins
7. **Manual fallback:** `orch session resume` command available if hooks fail or for manual review
