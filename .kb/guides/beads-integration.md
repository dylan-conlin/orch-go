# Beads Integration

**Purpose:** Single authoritative reference for how orch-go integrates with beads for issue tracking. Read this before debugging beads-related issues.

**Last verified:** Jan 6, 2026
**Synthesized from:** 17 investigations (Dec 19, 2025 - Jan 5, 2026)

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

## Architecture: RPC Client with CLI Fallback

**Always use `pkg/beads`** - never shell out directly with `exec.Command("bd", ...)`.

The integration evolved through three phases:
1. **Dec 2025:** Simple CLI subprocess calls
2. **Late Dec 2025:** Dashboard polling exposed performance issues (~10x increase in bd calls)
3. **Dec 25-26, 2025:** Native Go RPC client implemented with CLI fallback

**Current pattern:**

```go
// pkg/beads provides the canonical interface
import "orch-go/pkg/beads"

// RPC-first with automatic CLI fallback
client := beads.NewClient()
issue, err := client.Show("orch-go-abc1")

// Fallback functions for when client unavailable
issues, err := beads.FallbackReady(10)
```

| Method | When Used | Advantage |
|--------|-----------|-----------|
| **RPC** (default) | Beads daemon running | Faster, no process spawn |
| **CLI fallback** | Daemon unavailable | Always works |

**Reference:** `2025-12-25-inv-design-beads-integration-strategy-orch.md`

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

**Short ID Resolution:**
- Short IDs (`abc1`) must be resolved at **spawn time**, not agent time
- `pkg/beads.ResolveID()` converts short → full ID
- SPAWN_CONTEXT.md must contain full ID for agents to use

**Reference:** `2026-01-03-inv-fix-short-beads-id-resolution.md`

---

## Phase Reporting

Agents report progress via beads comments:

```bash
bd comments add {beads-id} "Phase: Planning - analyzing requirements"
bd comments add {beads-id} "Phase: Implementation - writing code"
bd comments add {beads-id} "Phase: Complete - task finished, tests pass"
```

**Phase: Complete is critical.** This is how `orch complete` knows the agent finished successfully.

---

## Three-Layer Artifact Architecture

```
BEADS (.beads/)
├── Purpose: Track work in progress (issues, dependencies, status)
├── Data: issues.jsonl with structured JSON per issue
├── Links: Comments contain investigation_path, phase transitions
└── Discovery: bd show, bd ready, bd list

KB (.kb/)
├── Purpose: Persist knowledge artifacts (investigations, decisions)
├── Data: Markdown files with structured frontmatter
├── Links: kb link creates bidirectional issue↔artifact links
└── Discovery: kb context, kb search

WORKSPACE (.orch/workspace/)
├── Purpose: Ephemeral agent execution context
├── Data: SPAWN_CONTEXT.md (input), SYNTHESIS.md (output)
├── Links: References beads ID, creates kb investigations
└── Discovery: Direct file access, orch review command
```

**Linking mechanisms:**
- Beads → KB: `investigation_path:` comments link to kb files
- KB → Beads: `kb link artifact.md --issue beads-id`
- Workspace → Both: SPAWN_CONTEXT.md contains beads ID, agents create kb investigations

**Reference:** `2025-12-21-inv-beads-kb-workspace-relationships-how.md`

---

## JSON Schema (Important!)

Beads JSON uses snake_case field names:

| Display | JSON Field |
|---------|------------|
| Type | `issue_type` |
| Close Reason | `close_reason` |
| Status | `status` |
| Priority | `priority` |

**Common mistake:** Using `.type` in jq queries returns `null` because the field is actually `issue_type`.

```bash
# Wrong
bd list --json | jq '.[0].type'      # Returns null

# Correct
bd list --json | jq '.[0].issue_type' # Returns "task"
```

**Reference:** `2026-01-05-inv-fix-beads-type-field-showing.md`

---

## Multi-Repo Configuration (Danger!)

**Default to single-repo mode.** Multi-repo hydration imports ALL issues from referenced repos.

```yaml
# DANGEROUS - this imports all issues from beads repo into your database!
repos:
  primary: "."
  additional: ["/path/to/beads"]
```

**Signs of pollution:**
- Issues with foreign prefixes (e.g., `bd-*` in orch-go)
- Nested `.beads/.beads/` directories
- Issue count unexpectedly high

**Cleanup procedure:**
1. Filter issues.jsonl: `jq -c 'select(.id | startswith("orch-go-"))' issues.jsonl > clean.jsonl`
2. Remove nested dirs: `rm -rf .beads/.beads/`
3. Fix config.yaml: Remove `additional` key
4. Reinitialize: `rm .beads/beads.db* && bd init --prefix orch-go`

**Reference:** 
- `2025-12-22-inv-beads-multi-repo-hydration-why.md`
- `2025-12-25-inv-beads-database-pollution-orch-go.md`

---

## Deduplication

`BeadsClient.Create()` automatically prevents duplicate issues:

```go
// Returns existing issue if title matches open/in_progress issue
issue, err := client.Create(beads.CreateArgs{
    Title: "My task",
    // Will return existing issue if one exists with same title
})

// Force creation even if duplicate exists
issue, err := client.Create(beads.CreateArgs{
    Title: "My task",
    Force: true,  // Bypass deduplication
})
```

**Reference:** `2026-01-03-inv-recover-priority-beads-deduplication-abstraction.md`

---

## Order of Operations

**Registry updates MUST happen before beads close:**

```go
// Correct order in orch complete
1. reg.Complete(agent.ID)      // Update registry first
2. reg.Save()                  // Persist registry
3. bd.CloseIssue(id, reason)   // Then close beads issue
```

**Why this matters:** Three silent failure modes could leave registry in inconsistent state if beads closes first.

**Reference:** `2025-12-21-inv-orch-complete-closes-beads-issue.md`

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

**Solutions:**
1. Use `--no-track` for cross-repo work, track manually
2. Create issue in target repo first, use `--issue`

### "Issue shows open but agent is done"

**Cause:** `orch complete` wasn't run.

**Fix:** Run `orch complete <id>`

### "bd ready shows nothing but there's work"

**Possible causes:**

1. **Issues are blocked** - Have unresolved dependencies
   - Check: `bd list --status blocked`
   
2. **Issues lack triage:ready label** - Daemon only spawns labeled issues
   - Fix: `bd label <id> triage:ready`

3. **Wrong directory** - Looking in wrong repo
   - Check: `pwd` and verify `.beads/` exists

### "Registry shows active but beads shows closed"

**Historical cause:** Order of operations bug (now fixed). 

**If seen now:** Indicates manual `bd close` bypassing `orch complete`.

---

## Directory Context

Beads operations are directory-sensitive:

```bash
# These use CURRENT directory's .beads/
bd list
bd show abc1
bd comments add abc1 "message"

# To operate on different repo:
cd /path/to/other/repo && bd list
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

## Key Decisions (from investigations)

| Decision | Rationale |
|----------|-----------|
| RPC-first with CLI fallback | Performance when daemon running, compatibility when not |
| Registry before beads close | Prevents inconsistent state |
| Short ID resolution at spawn time | Agents can't resolve at runtime |
| Single-repo by default | Multi-repo imports all issues (dangerous) |
| Deduplication by default | Prevents duplicate issue accidents |
| `pkg/beads` is canonical interface | Never use raw exec.Command |

---

## Debugging Checklist

Before spawning an investigation about beads issues:

1. **Check kb:** `kb context "beads"`
2. **Check this guide:** You're reading it
3. **Check issue exists:** `bd show <id>`
4. **Check correct directory:** `pwd` and `ls .beads/`
5. **Check daemon:** `orch doctor` (includes beads daemon check)
6. **Check JSON field names:** `issue_type` not `type`
7. **Check for pollution:** `bd list | wc -l` - unexpectedly high?

If those don't answer your question, then investigate. But **update this guide** with what you learn.

---

## Related Investigations

For historical evidence and deep-dives, see:

| Topic | Investigation |
|-------|---------------|
| RPC Client Design | `2025-12-25-inv-design-beads-integration-strategy-orch.md` |
| Multi-Repo Hydration | `2025-12-22-inv-beads-multi-repo-hydration-why.md` |
| Database Pollution | `2025-12-25-inv-beads-database-pollution-orch-go.md` |
| Short ID Resolution | `2026-01-03-inv-fix-short-beads-id-resolution.md` |
| Three-Layer Architecture | `2025-12-21-inv-beads-kb-workspace-relationships-how.md` |
| Registry/Beads Ordering | `2025-12-21-inv-orch-complete-closes-beads-issue.md` |
| JSON Field Names | `2026-01-05-inv-fix-beads-type-field-showing.md` |
| Deduplication | `2026-01-03-inv-recover-priority-beads-deduplication-abstraction.md` |
