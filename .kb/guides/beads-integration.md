# Beads Integration

**Purpose:** Single authoritative reference for how orch-go integrates with beads for issue tracking. Read this before debugging beads-related issues.

**Last verified:** Jan 4, 2026

---

## What Beads Does

Beads (`bd`) is the issue tracking system. In the orchestration flow:

| Stage | Beads Role |
|-------|------------|
| **Spawn** | Create issue, track work |
| **Work** | Agent reports phases via comments |
| **Complete** | Close issue with reason |
| **Query** | `bd ready`, `bd list` for work discovery |

---

## The Integration Points

```
orch spawn                           Agent                           orch complete
    │                                  │                                  │
    ▼                                  ▼                                  ▼
bd create "{task}"              bd comment {id}                    bd close {id}
    │                           "Phase: Planning"                   --reason "{summary}"
    │                                  │
    ▼                                  ▼
Returns beads ID               Updates issue with
(orch-go-abc1)                 progress/status
```

---

## Beads ID Format

```
{project}-{4-char-hash}
   │          │
   │          └── Unique identifier (e.g., abc1, xyz9)
   └── Project prefix from .beads/ location (e.g., orch-go)
```

**Examples:**
- `orch-go-abc1` - Issue in orch-go project
- `kb-cli-def2` - Issue in kb-cli project
- `orch-go-untracked-1767548133` - Untracked spawn (placeholder, not in DB)

---

## Phase Reporting

Agents report progress via beads comments:

```bash
bd comment {beads-id} "Phase: Planning - analyzing requirements"
bd comment {beads-id} "Phase: Implementation - writing code"
bd comment {beads-id} "Phase: Complete - task finished, tests pass"
```

**Phase: Complete is critical.** This is how `orch complete` knows the agent finished successfully.

---

## RPC vs CLI

orch-go uses two methods to talk to beads:

| Method | When Used | Advantage |
|--------|-----------|-----------|
| **RPC** (default) | Beads daemon running | Faster, no process spawn |
| **CLI fallback** | Daemon unavailable | Always works |

```go
// Pattern in pkg/beads
func CloseIssue(id, reason string) error {
    // Try RPC first
    if client := getRPCClient(); client != nil {
        if err := client.CloseIssue(id, reason); err == nil {
            return nil
        }
    }
    // Fallback to CLI
    return FallbackClose(id, reason)
}
```

---

## Common Problems

### "bd comment fails with 'issue not found'"

**Possible causes:**

1. **Untracked spawn** - `--no-track` creates placeholder IDs that don't exist in DB
   - Expected behavior, not a bug
   
2. **Wrong directory** - Running `bd` from different repo than where issue exists
   - Fix: Use `--workdir` or `cd` to correct repo

3. **Short ID not resolved** - Using `abc1` instead of `orch-go-abc1`
   - Fix: Use full ID, or orch-go resolves automatically

### "Cross-project agent can't update beads"

**Cause:** Agent spawned with `--workdir /other/repo` but beads issue is in orchestrator's repo.

**The pattern:**
- Beads issue created in orchestrator's current directory
- Agent runs in `--workdir` directory
- `bd comment` from agent looks in wrong place

**Solutions:**
1. Use `--no-track` for cross-repo work, track manually
2. Create issue in target repo first, use `--issue`

### "Issue shows open but agent is done"

**Cause:** `orch complete` wasn't run.

**Why this happens:**
- Agent finished and reported Phase: Complete
- But orchestrator didn't run `orch complete`
- Beads issue stays open until explicitly closed

**Fix:** Run `orch complete <id>`

### "bd ready shows nothing but there's work"

**Possible causes:**

1. **Issues are blocked** - Have unresolved dependencies
   - Check: `bd list --status blocked`
   
2. **Issues lack triage:ready label** - Daemon only spawns labeled issues
   - Fix: `bd label <id> triage:ready`

3. **Wrong directory** - Looking in wrong repo
   - Check: `pwd` and verify `.beads/` exists

---

## Directory Context

Beads operations are directory-sensitive:

```bash
# These use CURRENT directory's .beads/
bd list
bd show abc1
bd comment abc1 "message"

# To operate on different repo:
cd /path/to/other/repo && bd list
# Or set BEADS_DIR (if supported)
```

**Key insight:** When orchestrator is in orch-go but agent runs in kb-cli, their `bd` commands hit different databases.

---

## Lifecycle States

| State | Meaning | Transitions To |
|-------|---------|----------------|
| `open` | Work not started | `in_progress` |
| `in_progress` | Agent working on it | `closed` |
| `closed` | Work complete | - |
| `blocked` | Has unresolved dependencies | `open` when unblocked |

**orch spawn** sets issue to `in_progress`.
**orch complete** sets issue to `closed`.
**orch abandon** sets issue to `closed` (with abandonment reason).

---

## Key Decisions (from kn)

- **Beads is source of truth** - not OpenCode sessions, not workspaces
- **Phase: Complete is the signal** - only reliable indicator of agent completion
- **RPC-first with CLI fallback** - performance when daemon running, compatibility when not
- **Registry updates before beads close** - prevents inconsistent state

---

## Debugging Checklist

Before spawning an investigation about beads issues:

1. **Check kb:** `kb context "beads"`
2. **Check this doc:** You're reading it
3. **Check issue exists:** `bd show <id>`
4. **Check correct directory:** `pwd` and `ls .beads/`
5. **Check daemon:** `orch doctor` (includes beads daemon check)

If those don't answer your question, then investigate. But update this doc with what you learn.
