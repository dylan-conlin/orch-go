# Session Resume Protocol

**Purpose:** Single authoritative reference for session handoff and automatic resume system.

**Synthesized from:** Design doc (2026-01-11) + Implementation findings (2026-01-13)

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

**Automatic behavior:** When you start a new session in Claude Code or OpenCode, hooks automatically inject the latest handoff if one exists.

---

## The Problem

**Before session resume:**
- Dylan had to remember session mechanics (create handoff, read handoff manually)
- Fresh Claude instances didn't receive prior context automatically
- Manual SESSION_HANDOFF.md creation bypassed proper tooling
- No parity with worker spawns (workers get SPAWN_CONTEXT.md automatically)

**Dylan's core need:**
> "I want to be able to start any session just by saying 'let's resume' and from there, the orchestrator should have a protocol to determine what comes next. Session handoff mechanics should be handled automatically by the system, freeing me to think about higher level goals."

**Principle:** Pressure Over Compensation - if Dylan has to remember session mechanics, that's system failure.

---

## How It Works

```
Dylan starts new session
         │
         ▼
┌────────────────────────────────────────┐
│  SESSION START                         │
│  (Claude Code or OpenCode)             │
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

## File Structure

Session handoffs are **project-specific**:

```bash
{project}/.orch/session/
├── latest -> 2026-01-13-0827/    # Symlink to most recent
├── 2026-01-11-1935/
│   └── SESSION_HANDOFF.md        # Created at session end
└── 2026-01-13-0827/
    └── SESSION_HANDOFF.md
```

**Examples:**
- `~/Documents/personal/orch-go/.orch/session/latest/SESSION_HANDOFF.md`
- `~/orch-knowledge/.orch/session/latest/SESSION_HANDOFF.md`

**Why symlink:**
- `orch session end` updates symlink to new session
- `orch session resume` always reads from `latest/SESSION_HANDOFF.md`
- No timestamp parsing needed
- Project-specific: each repo maintains its own session history

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

1. **Start from current directory**
2. **Walk up tree** to find `.orch/session/latest` symlink
3. **Read** `{symlink}/SESSION_HANDOFF.md`
4. **Exit code 1** if not found

**Key insight:** Discovery works from any subdirectory within the project. You don't need to be in the project root.

**Example:**
```bash
# Working in subdirectory
cd ~/Documents/personal/orch-go/cmd/orch

# Still finds handoff in project root
orch session resume
# → Reads ~/Documents/personal/orch-go/.orch/session/latest/SESSION_HANDOFF.md
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

### Starting Fresh Session

```bash
# Start new session (no prior handoff)
# Hook runs, finds no handoff, stays silent
# ✅ This is valid - fresh starts are expected
```

**No error:** Fresh sessions (first time in a project, or after long break) are valid scenarios.

---

### Resuming After Break

```bash
# Start new session
# Hook automatically injects latest handoff
# Claude sees context from prior session
```

**Zero cognitive load:** You don't need to remember to resume. It happens automatically.

---

### Manual Review Before Resume

```bash
# Before starting Claude session
orch session resume

# Review handoff content
# Decide if context is still relevant
# Then start session (hook will inject same content)
```

**Use when:** You want to verify the handoff is still relevant before starting work.

---

### Ending Session

```bash
# Before closing Claude session
orch session end

# Fill in reflection prompts
# Handoff created automatically
# Symlink updated to new session
```

**Critical:** Run this BEFORE closing the session, not after. Fresh Claude won't have context to create meaningful handoff.

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

1. **Zero cognitive load:** Dylan doesn't need to remember to resume - hooks handle it automatically
2. **Project-specific:** Each repo maintains its own session history via `.orch/session/`
3. **Graceful degradation:** Fresh starts (no handoff) are valid and don't produce errors
4. **Forcing function:** `orch session end` is part of session close protocol - creates handoff for next session
5. **Cross-environment:** Works in both Claude Code and OpenCode via hooks/plugins
6. **Manual fallback:** `orch session resume` command available if hooks fail or for manual review
