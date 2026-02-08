# Investigation: Session Resume Protocol Design

**Date:** 2026-01-11
**Type:** Design
**Status:** Design complete, ready for implementation
**Priority:** P2 (Important, not urgent)

---

## Problem Statement

**Current pain:**
- Dylan has to remember session mechanics (start session, create handoff, read handoff)
- Fresh Claude instances don't automatically receive prior context
- Manual SESSION_HANDOFF.md creation bypasses proper tooling
- No parity with worker spawns (workers get SPAWN_CONTEXT.md automatically)

**Core need (Dylan's framing):**
> "I want to be able to start any session just by saying 'let's resume' and from there, the orchestrator should have a protocol to determine what comes next. Session handoff mechanics should be handled automatically by the system, freeing me to think about higher level goals."

**Principle:** Pressure Over Compensation - if Dylan has to remember session mechanics, that's system failure.

---

## Requirements

**R1: Zero cognitive load for Dylan**
- Dylan shouldn't decide when to start sessions
- Dylan shouldn't remember session mechanics
- Dylan shouldn't locate/read handoff files manually

**R2: Automatic context recovery**
- Fresh Claude instance should find prior context
- Handoff should be discovered and applied automatically
- State should be surfaced (not just loaded silently)

**R3: Forcing function for handoff creation**
- If no handoff exists, failure should be visible
- Previous orchestrator must create handoff via proper protocol
- Manual workarounds should be harder than correct path

**R4: Parity with worker spawns**
- Workers get SPAWN_CONTEXT.md automatically
- Orchestrators should get SESSION_HANDOFF.md automatically
- Same amnesia-resilient design pattern

**R5: Context window awareness**
- Sessions tied to Claude instance lifecycle (amnesia boundaries)
- Handoff creation triggers on context exhaustion
- Each fresh Claude = new session start

**R6: Cross-environment support**
- Must work in both OpenCode and Claude Code
- Dylan switches between environments frequently
- Implementation shouldn't favor one over the other

**R7: Project-specific handoffs**
- orch-go sessions separate from orch-knowledge sessions
- No cross-contamination between projects
- Handoff discovery scoped to current project

---

## Design Options Considered

### Option A: Dual Hook System (Automated)
- OpenCode plugin (`session.created` event)
- Claude Code hook (SessionStart)
- Fully automated, zero Dylan intervention

**Pros:** Zero cognitive load
**Cons:** Two separate implementations, harder to maintain

### Option B: CLI Command (`orch resume`)
- Manual command Dylan runs before session
- Outputs formatted handoff for copy/paste

**Pros:** Simple, works everywhere
**Cons:** Manual step, defeats zero-load goal

### Option C: Shell Wrapper Function
- `orch-session` wrapper in ~/.zshrc
- Calls `orch resume` then starts session

**Pros:** Zero load once wrapper used
**Cons:** Dylan has to remember wrapper, conflicts with existing workflow

### Option D: Hybrid (Automated + Fallback) ✅ CHOSEN
- Primary: Hooks auto-inject (both environments)
- Fallback: `orch resume` command for manual use
- Single source of truth: `orch resume` consumed by both hooks

**Pros:**
- Solves zero-load goal when hooks work
- Provides fallback for edge cases
- Single implementation, consumed by hooks
- Can iterate: start manual, add hooks later

**Cons:**
- Most complex implementation

**Decision rationale:** Best of both worlds, graceful degradation, single source of truth.

---

## Detailed Design

### 1. File Structure (Project-Specific)

```bash
{project}/.orch/session/
├── latest -> 2026-01-11-2015/    # Symlink to most recent
├── 2026-01-11-1935/
│   ├── SESSION_CONTEXT.md        # Created at start
│   └── SESSION_HANDOFF.md        # Created at end
└── 2026-01-11-2015/
    ├── SESSION_CONTEXT.md
    └── SESSION_HANDOFF.md
```

**Examples:**
- `~/Documents/personal/orch-go/.orch/session/latest/SESSION_HANDOFF.md`
- `~/orch-knowledge/.orch/session/latest/SESSION_HANDOFF.md`

**Why symlink:**
- `orch session end` updates symlink to new session
- `orch resume` always reads from `latest/SESSION_HANDOFF.md`
- No timestamp parsing needed

---

### 2. Core CLI Command: `orch resume`

**Usage modes:**

```bash
# Interactive (for Dylan manual use)
orch resume
# Outputs formatted handoff with metadata

# For injection (used by hooks)
orch resume --for-injection
# Outputs condensed content only (no decorations)

# Check only (exit code 0 if exists, 1 if not)
orch resume --check
```

**Discovery logic:**
1. Start from current directory
2. Walk up tree to find `.orch/session/latest` symlink
3. Read `{symlink}/SESSION_HANDOFF.md`
4. If not found, exit code 1

**Project-specificity:**
- Running in `orch-go/` finds orch-go handoffs
- Running in `orch-knowledge/` finds orch-knowledge handoffs
- Working directory determines project scope

---

### 3. OpenCode Plugin (Automated)

**Location:** `~/.config/opencode/plugin/session-resume.js` (global)

```javascript
export default {
  name: 'session-resume',
  version: '1.0.0',

  on_session_created: async (context) => {
    const { cwd, sendSystemMessage } = context;

    // Run orch resume from session's working directory
    const result = await exec('orch resume --for-injection', { cwd });

    if (result.exitCode === 0 && result.stdout) {
      await sendSystemMessage({
        content: `📋 **Session Resumed**\n\n${result.stdout}`,
        role: 'system'
      });
    }
    // Silent if no handoff (fresh start is valid)
  }
};
```

**Behavior:**
- Runs automatically on every new OpenCode session
- Injects if handoff found, silent if not
- Zero Dylan intervention

---

### 4. Claude Code Hook (Automated)

**Location:** `~/.claude/hooks/SessionStart` (existing hook)

```bash
#!/bin/bash

# Existing hook content (beads prime, etc.)
# ...

# Add session resume
if command -v orch >/dev/null 2>&1; then
  handoff=$(orch resume --for-injection 2>/dev/null)
  if [ $? -eq 0 ] && [ -n "$handoff" ]; then
    echo ""
    echo "📋 Session Resumed"
    echo ""
    echo "$handoff"
  fi
fi
```

**Behavior:**
- Runs on SessionStart event
- Output appears in Claude's initial context
- Graceful if `orch` not available
- Silent if no handoff

---

### 5. Session End Updates

**`orch session end` must:**
1. Prompt for reflection (existing)
2. Create SESSION_HANDOFF.md in timestamped directory (existing)
3. **NEW:** Update `latest` symlink to new session

```bash
# After creating session directory
session_dir=".orch/session/2026-01-11-2015"
ln -sf "$(basename $session_dir)" .orch/session/latest
```

---

### 6. Handoff Format for Injection

**Problem:** Current handoffs are verbose (362 lines). Injecting full content wastes tokens.

**Solution:** Condensed format for `--for-injection` mode:

```markdown
# Session Resume: {Project} - {Date}

**Last session:** {duration} ago ({date})
**Focus:** {goal from last session}

## Top Priorities

1. {priority 1}
2. {priority 2}
3. {priority 3}

## System State

- Active agents: {count}
- Ready work: {count} issues
- Completed awaiting review: {count}

## Key Context

{condensed accomplishments + what's TODO}

---

**Full handoff:** `.orch/session/latest/SESSION_HANDOFF.md`
```

**Rationale:**
- Surface most important bits (priorities, state, blockers)
- Link to full handoff for deep dive
- Token-efficient

---

## Implementation Sequence

### Phase 1: Core Command (Foundation)
1. Implement `orch resume` command in Go
2. Add handoff discovery logic (walk up to `.orch/session/latest`)
3. Add formatting modes (`--for-injection`, `--check`)
4. Test: manual use works

### Phase 2: Session End Integration
5. Update `orch session end` to create/update latest symlink
6. Test: symlink updates correctly after session end

### Phase 3: Hook Integration
7. Add to Claude Code SessionStart hook (`~/.claude/hooks/SessionStart`)
8. Create OpenCode plugin (`~/.config/opencode/plugin/session-resume.js`)
9. Test: both environments auto-inject

### Phase 4: Handoff Format Optimization
10. Design condensed format for injection
11. Keep full format in SESSION_HANDOFF.md
12. Test: token usage reduced, essential info preserved

---

## Edge Cases

**No handoff exists (fresh start):**
- `orch resume` exits with code 1
- Hooks detect and stay silent
- Claude starts without injected context
- **Valid scenario** (new project, first session)

**Multiple projects:**
- Each has own `.orch/session/` directory
- Working directory determines scope
- No cross-contamination

**Stale handoff (>7 days):**
- Still inject, but add warning: "⚠️ Handoff is 8 days old"
- Let orchestrator decide if still relevant

**Handoff mid-edit:**
- If SESSION_HANDOFF.md exists but latest symlink broken
- Fail gracefully with error message

**Cross-repo work:**
- Handoff in orch-go, but working in orch-knowledge
- Only discovers handoff in current project tree
- Dylan manually references if needed

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────┐
│  Dylan says "let's resume" or starts session    │
└─────────────────┬───────────────────────────────┘
                  │
        ┌─────────┴──────────┐
        │                    │
    Automated           Manual Fallback
    (Hooks)             (CLI Command)
        │                    │
        └─────────┬──────────┘
                  │
          orch resume --for-injection
                  │
          Discovers handoff in:
          {project}/.orch/session/latest/
                  │
          Returns formatted content
                  │
        Fresh Claude sees context
```

---

## Success Criteria

**After implementation, these should be true:**

1. **Dylan says "let's resume" in OpenCode → handoff auto-injected**
2. **Dylan says "let's resume" in Claude Code → handoff auto-injected**
3. **Fresh project with no handoff → silent, no errors**
4. **Multi-project work → each project shows correct handoff**
5. **Manual fallback works → `orch resume` shows handoff interactively**
6. **Symlink always current → after `orch session end`, latest points to new session**

---

## Open Questions

1. **Handoff compression strategy:**
   - Full format now, optimize later? Or design condensed format upfront?
   - Recommendation: Ship full injection first (simpler), optimize when token usage becomes issue

2. **Hook priority:**
   - Implement Claude Code hook first (Dylan using now)?
   - Or OpenCode plugin (more common)?
   - Recommendation: Claude Code hook first (dogfooding)

3. **Stale handoff threshold:**
   - When to warn about old handoffs? 7 days? 14 days?
   - Recommendation: 7 days (aligns with weekly hygiene rhythm)

---

## Related Work

**Existing session mechanics:**
- `orch session start` - Creates SESSION_CONTEXT.md
- `orch session end` - Prompts for reflection, creates SESSION_HANDOFF.md
- `orch session status` - Shows current session state

**Reference:** `~/.claude/skills/meta/orchestrator/reference/session-reflection.md`

**Prior context:**
- SESSION_HANDOFF.md from 2026-01-11 session identified this gap
- Manual handoff creation bypassed tooling
- No protocol for fresh Claude to discover context

---

## Next Steps

1. Create beads issue for implementation (P2)
2. Prioritize after P1 infrastructure work (session cleanup, infrastructure detection)
3. Implementation order: Phase 1 → 2 → 3 → 4
4. Test with real sessions before declaring complete

---

## Implementation Notes

**Code locations:**
- Command: `cmd/orch/resume.go` (new)
- Session end update: `cmd/orch/session.go` (modify)
- OpenCode plugin: `~/.config/opencode/plugin/session-resume.js` (new)
- Claude Code hook: `~/.claude/hooks/SessionStart` (modify existing)

**Dependencies:**
- Requires `orch session end` to create handoffs properly
- Requires `.orch/session/` directory structure
- Requires project root detection (walk up tree)

**Testing strategy:**
- Unit tests: handoff discovery logic
- Integration tests: full workflow (start → end → resume)
- Manual tests: both OpenCode and Claude Code environments

---

## Conclusion

This design solves Dylan's core need: **zero cognitive load for session resumption**. By implementing a hybrid approach (automated hooks + manual fallback), we achieve:

- ✅ Automatic context recovery
- ✅ Cross-environment support (OpenCode + Claude Code)
- ✅ Project-specific handoffs
- ✅ Forcing function for proper handoff creation
- ✅ Parity with worker spawn pattern

The implementation is phased to allow iteration and validation at each step.
