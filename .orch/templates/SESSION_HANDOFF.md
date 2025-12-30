<!--
TEMPLATE: Session Handoff
This is a template file used by orch-go for orchestrator session handoffs via `orch session end`.

DO NOT: Confuse with instance files - SESSION_HANDOFF.md in .orch/ root is an instance, not the source.
SOURCE: /Users/dylanconlin/Documents/personal/orch-go/.orch/templates/SESSION_HANDOFF.md
MODIFY: Edit this file directly, then future `orch session end` will use the updated template.
UPDATED: 2025-12-30

Instance files (like .orch/SESSION_HANDOFF.md) can be edited freely - they're specific to a session.
-->

# Session Handoff - {date}

## TLDR

[1-2 sentence summary of what was accomplished and what's in progress]

---

## What Happened This Session

### Work Completed
- **{issue-id}** - {brief description of completed work}

### PRs Submitted
| PR | Status | Description |
|----|--------|-------------|
| #{number} | {Open/Merged/Closed} | {brief description} |

### Code Shipped
- {feature or change} ({repo})
- {feature or change} ({repo})

### Decisions Made
- **{topic}:** {decision and brief rationale}

### Housekeeping
- {cleanup/maintenance task completed}

---

## Agents Still Running

| Agent | Repo | Task |
|-------|------|------|
| **{beads-id}** | {repo} | {task description} |

*(Use `orch status` to check current agent states)*

---

## Blocking Issues

### {Issue Title}

**Problem:** {description of blocking issue}

**Root cause:** {known or suspected cause}

**What we tried:**
1. {attempt 1} {result}
2. {attempt 2} {result}

**Next:** {recommended next steps}

---

## Local State

### Uncommitted Work
```bash
# Check status in each repo
cd ~/{path} && git status
```

### Branches with Local Changes
- **{repo}:** `{branch-name}` - {what's there}

### When to Clean Up
{conditions under which local state can be cleaned}

---

## Cross-Repo Context

| Repo | Status | Notes |
|------|--------|-------|
| {repo-1} | {active/blocked/complete} | {brief notes} |
| {repo-2} | {active/blocked/complete} | {brief notes} |

---

## Next Session Priorities

1. **{priority-1}** - {brief description}
2. **{priority-2}** - {brief description}
3. **{priority-3}** - {brief description}

### Lower Priority
- {deferred task 1}
- {deferred task 2}

---

## Quick Commands

```bash
# Resume context
{command to check status}

# Verify key state
{command to test}

# Common operations
{useful command}
```

---

## Session Metadata

**Agents spawned:** {count}
**Issues closed:** {count}
**PRs:** {submitted/merged counts}
**Focus:** {current focus from orch focus}
