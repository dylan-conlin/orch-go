# Decision: Reconcile Claude Code Auto-Memory with kb-cli Knowledge Externalization

**Date:** 2026-02-25
**Status:** Proposed
**Issue:** orch-go-1239
**Deciders:** Dylan (pending review)

## Context

Claude Code has an "auto memory" feature: a per-project `MEMORY.md` file at `~/.claude-personal/projects/<project>/memory/MEMORY.md` that the agent reads/writes to persist notes across sessions. The first ~200 lines are injected into every conversation's system context at startup.

We already have kb-cli for knowledge externalization (decisions, constraints, tried/failed, investigations, guides, probes, models, quick entries). Both systems store learned patterns and preferences that persist across sessions. The Knowledge Placement table in `~/.claude/CLAUDE.md` defines where different knowledge types go but doesn't account for auto-memory.

**The overlap is real:** The current orch-go `MEMORY.md` contains items that span both systems — session-tactical state ("git: clean, 30 commits ahead") alongside durable knowledge ("Agent Cognitive Gaps" insights, "Orch Spawn Quirks") that should be kb artifacts.

## Decision

**Option 2: Distinct lanes with clear boundaries.** Neither disable auto-memory nor merge the systems. Each serves a fundamentally different audience and purpose.

### Lane Definitions

| Dimension | Auto-Memory (MEMORY.md) | kb-cli (.kb/) |
|-----------|--------------------------|---------------|
| **Audience** | This specific Claude Code instance (Dylan's personal sessions) | All agents in the system (spawned workers, orchestrators, future sessions) |
| **Purpose** | Session continuity — recover tactical context between conversations | Knowledge base — durable understanding of how things work and why |
| **Lifecycle** | Ephemeral; entries should churn as work progresses | Persistent; artifacts have provenance chains and governance |
| **Injection** | Automatic — first 200 lines loaded at startup | On-demand — `kb context` queries inject relevant subset |
| **Discovery** | Invisible to spawned agents (per-user, per-project) | Discoverable by all agents via `kb context` |
| **Governance** | None — agent self-manages | Structured — creation thresholds, promotion paths, staleness detection |

### What Goes Where

**Auto-Memory (MEMORY.md) — session-tactical context:**
- Current working state: open issues, blocked items, what was just completed
- Active session close state: git status, daemon status, push status
- Tool quirks discovered in-session that are still being validated
- Dylan's interaction preferences for this project (explanation style, launch setup)
- Work-in-progress patterns not yet confirmed across multiple sessions

**kb-cli (.kb/) — durable cross-agent knowledge:**
- Architectural decisions (why we chose X)
- System models (how X works, failure modes)
- Operational guides (how to do X)
- Investigations (point-in-time evidence)
- Quick entries (in-moment capture with promotion path)
- Constraints and principles (rules all agents must follow)

### The Visibility Gap Is a Feature

Auto-memory being invisible to spawned agents is **correct by design**:

1. **Spawned agents get SPAWN_CONTEXT.md** — curated, task-specific context from the orchestrator. They don't need Dylan's session-tactical state.
2. **kb artifacts are shared** — when knowledge matures to the point where other agents need it, it should be promoted to kb, not left in auto-memory.
3. **The gap becomes a bug only when durable knowledge gets trapped in auto-memory.** This is the same promotion problem that exists within kb itself (quick entries that should become decisions).

### Auto-Memory Does Not Compete with CLAUDE.md

The 200-line auto-memory injection and CLAUDE.md serve different functions:

| System | Content Type | Update Frequency | Who Writes |
|--------|-------------|-------------------|------------|
| **CLAUDE.md** (project) | Stable instructions, conventions, architecture | Rarely (project-level changes) | Dylan (human review) |
| **CLAUDE.md** (user) | Cross-project preferences, workflows | Occasionally | Dylan (human review) |
| **MEMORY.md** | Session-tactical working state | Every session | Agent (self-managed) |

They complement each other: CLAUDE.md says "here's how this project works", MEMORY.md says "here's where we left off." The 200-line limit on auto-memory is adequate — if you need more than 200 lines of tactical context, the real problem is that durable knowledge hasn't been promoted to kb.

## Promotion Trigger

**Add to Knowledge Placement table:** If a MEMORY.md entry survives 3+ sessions unchanged, it should be promoted to the appropriate kb artifact:

| MEMORY.md Pattern | Promote To |
|-------------------|------------|
| Tool quirk confirmed across sessions | `kb quick constrain` |
| Architectural insight | `kb quick decide` or `.kb/investigations/` |
| Recurring interaction preference | `~/.claude/CLAUDE.md` (user memory) |
| Open questions that persist | `bd create --type question` |
| Completed work summaries | Delete from MEMORY.md (kb artifacts already captured the knowledge) |

**Anti-pattern:** MEMORY.md as a graveyard of completed work. Once work is done and knowledge is externalized to kb, remove it from MEMORY.md to stay under the 200-line budget.

## Concrete Changes Required

### 1. Update Knowledge Placement Table (`~/.claude/CLAUDE.md`)

Add auto-memory row:

```
| Session-tactical context | Auto-memory (MEMORY.md) | "Where did we leave off?" |
```

Add promotion path:

```
- MEMORY.md entry surviving 3+ sessions → appropriate kb artifact
```

### 2. Clean Current MEMORY.md

Current orch-go MEMORY.md has items that should be promoted or removed:

| Current Entry | Action |
|---------------|--------|
| Two-Lane Agent Discovery (COMPLETE) | **Remove** — work is done, ADR exists at `.kb/decisions/` |
| Coupling Hotspot Design (COMPLETE) | **Remove** — work is done, investigation exists |
| Agent Cognitive Gaps | **Promote** to `kb quick constrain` or investigation — this is durable insight |
| Daemon Config Extraction (COMPLETE) | **Remove** — work is done |
| Dylan's Claude Code Launch Setup | **Keep** — active tactical reference |
| Orch Spawn Quirks | **Promote** to `kb quick constrain` — these are stable constraints |
| Dylan's Explanation Preferences | **Already in CLAUDE.md** — remove duplicate |
| Session Close State | **Keep** — pure session-tactical |

### 3. Add MEMORY.md Hygiene to Session Close Protocol

After the existing close protocol steps, add:
```
[ ] Review MEMORY.md for stale entries (completed work, promoted knowledge)
```

This is lightweight — just a reminder during the existing close flow, not a gate.

## Rejected Alternatives

### Option 1: Disable Auto-Memory Entirely

Rejected because auto-memory solves a real problem that kb-cli doesn't: fast session-tactical context recovery. `kb context` is query-based and returns relevant knowledge, but it doesn't know "you were working on issue 1096 and git has 30 uncommitted commits." That's session state, not knowledge.

### Option 3: Merge Everything into Auto-Memory

Rejected because auto-memory has no governance, no discoverability by other agents, and a 200-line limit. The kb-cli system's provenance chains, promotion paths, and shared visibility are essential for multi-agent orchestration.

## Consequences

- **Positive:** Clear lanes prevent knowledge from getting trapped in the wrong system
- **Positive:** Auto-memory stays lean and tactical (under 200 lines)
- **Positive:** kb-cli remains the authoritative knowledge base for all agents
- **Risk:** Promotion discipline requires manual effort — same risk as kb quick → decision promotion
- **Mitigation:** `kb reflect` could eventually detect stale MEMORY.md entries (future automation)

## References

- Claude Code memory docs: `~/.claude/docs/official/claude-code/memory.md`
- Context Injection Architecture: `.kb/models/context-injection/model.md`
- Knowledge Placement Table: `~/.claude/CLAUDE.md` (section: "Knowledge Placement")
- Current auto-memory: `~/.claude-personal/projects/-Users-dylanconlin-Documents-personal-orch-go/memory/MEMORY.md`
