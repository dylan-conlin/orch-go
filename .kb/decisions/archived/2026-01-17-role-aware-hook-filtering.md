## Summary (D.E.K.N.)

**Delta:** Claude Code hooks must check CLAUDE_CONTEXT and exit early for worker/orchestrator/meta-orchestrator to prevent duplicate context injection.

**Evidence:** Probe 1 audit found session-start.sh injects 4KB for all sessions including spawned agents who already have SPAWN_CONTEXT.md; testing confirms role-aware filtering works correctly.

**Knowledge:** All three roles (worker, orchestrator, meta-orchestrator) are spawned agents with authoritative SPAWN_CONTEXT.md; session resume is wrong context for them.

**Next:** Apply pattern to other hooks (bd prime next); document as standard hook pattern in hook development guide.

---

# Decision: Role Aware Hook Filtering

**Date:** 2026-01-17
**Status:** Active
**Enforcement:** hook

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** ~/Documents/personal/orch-go/.kb/investigations/2026-01-18-inv-ci-implement-role-aware-injection.md
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Context

Probe 1 audit (Jan 16, 2026) identified that session-start.sh was injecting session resume context (~4KB) into ALL sessions, including spawned agents. This violated the Context Injection Architecture model constraint: "For spawned agents, SPAWN_CONTEXT.md is the source of truth. Hooks must back off to avoid duplication."

Spawned agents (worker, orchestrator, meta-orchestrator) receive their context through SPAWN_CONTEXT.md embedded by the spawn machinery. Session resume context is intended for manual sessions where Dylan resumes work across conversations.

---

## Options Considered

### Option A: Filter all three roles (worker|orchestrator|meta-orchestrator)
- **Pros:** Clean separation; all spawned agents are treated consistently
- **Cons:** None identified; all three roles use SPAWN_CONTEXT.md

### Option B: Filter worker only
- **Pros:** Allows spawned orchestrators to receive session resume
- **Cons:** Duplicates context (SPAWN_CONTEXT.md already has task context); no use case identified

### Option C: Disable session-start.sh entirely for spawned contexts
- **Pros:** Maximum separation
- **Cons:** Loses conditional hooks (errors, usage warnings) that might be useful

---

## Decision

**Chosen:** Option A - Filter all three roles

**Rationale:** All three roles (worker, orchestrator, meta-orchestrator) are spawned via `orch spawn` and receive SPAWN_CONTEXT.md. Session resume is for manual sessions only. Pattern matches load-orchestration-context.py's spawn detection.

**Trade-offs accepted:**
- Spawned orchestrators don't get session resume context (acceptable - SPAWN_CONTEXT.md provides task context)
- Silent skip (exit 0) rather than logging which path taken (acceptable - reduces noise)

---

## Structured Uncertainty

**What's tested:**
- ✅ Role detection works (tested with CLAUDE_CONTEXT=worker/orchestrator/empty)
- ✅ Manual sessions receive full context (verified ~4KB output)
- ✅ Pattern matches load-orchestration-context.py (code review)

**What's untested:**
- ⚠️ meta-orchestrator specific behavior (assumed to work like other roles)
- ⚠️ CLAUDE_CONTEXT reliably set by all spawn paths
- ⚠️ Actual token savings in production

**What would change this:**
- If we introduce a fourth role that needs session resume but not SPAWN_CONTEXT
- If spawned orchestrators require session continuity across spawn boundaries
- If CLAUDE_CONTEXT proves unreliable as role detection mechanism

---

## Consequences

**Positive:**
- Eliminates 4KB duplicate context for spawned agents
- Enforces "Authoritative Spawn Context" constraint from Context Injection Architecture
- Consistent treatment of all spawned roles
- Pattern reusable across all hooks (bd prime is next)

**Risks:**
- If CLAUDE_CONTEXT not set, agent incorrectly receives session resume (mitigation: verify spawn machinery sets it)
- Silent skip makes debugging harder (mitigation: CLAUDE_CONTEXT observable via env)

## Auto-Linked Investigations

- .kb/investigations/archived/2026-01-18-inv-ci-implement-role-aware-injection.md
- .kb/investigations/archived/2026-01-14-inv-duration-aware-session-resume-filtering.md
