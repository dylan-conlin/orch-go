## Summary (D.E.K.N.)

**Delta:** Dylan's OpenCode fork is 13 commits ahead of upstream with well-isolated fixes for memory management, SSE cleanup, and OAuth; all three proposed lifecycle features (TTL, metadata API, status endpoint) are feasible with the status endpoint already existing.

**Evidence:** Read all session-related source files (session/index.ts, status.ts, instance.ts, server routes), git diff upstream/dev..HEAD, and the fork's own resource audit investigation.

**Knowledge:** Sessions are JSON files on disk with no TTL; status is in-memory only (lost on restart); PATCH endpoint only accepts title and time.archived today; GET /session/status already returns idle/busy/retry map.

**Next:** Create implementation tasks for the three features, prioritized: metadata API (1-2h), orch-go status endpoint integration (1h), session TTL (2-4h).

**Authority:** architectural - Crosses OpenCode fork and orch-go boundaries; the lifecycle ownership decision depends on these findings.

---

# Investigation: Build Model of Dylan's OpenCode Fork

**Question:** What is the architecture of Dylan's OpenCode fork, and how feasible are three proposed features (session TTL, metadata API, status endpoint) that would eliminate ~3,400 lines of orch-go lifecycle code?

**Started:** 2026-02-13
**Updated:** 2026-02-13
**Owner:** worker-agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** .kb/decisions/2026-02-13-lifecycle-ownership-boundaries.md
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| ~/Documents/personal/opencode/.kb/investigations/2026-02-11-inv-opencode-fork-resource-audit-investigate.md | extends | yes | No conflicts — resource audit confirms Instance has TTL but Session does not |
| .kb/models/agent-lifecycle-state-model.md | extends | yes | Model says "Sessions persist indefinitely" — confirmed by code (no TTL, no auto-delete) |

---

## Findings

### Finding 1: Fork is Linear Forward from Upstream with 13 Manageable Custom Commits

**Evidence:** `git merge-base --is-ancestor upstream/dev HEAD` confirms upstream is ancestor. `git log --oneline upstream/dev..HEAD` shows 13 commits. Changes span 94 files with +7,763/-53 lines, but most are workspace/investigation artifacts. Core code changes touch ~8 source files.

**Source:** `git remote -v`, `git log --oneline upstream/dev..HEAD`, `git diff upstream/dev..HEAD --stat`

**Significance:** The fork is maintainable — rebasing 13 well-isolated commits onto upstream is feasible. Any features we add should follow the same pattern: isolated, minimal footprint, backward-compatible.

---

### Finding 2: GET /session/status Already Exists — Zero OpenCode Work Needed

**Evidence:** `packages/opencode/src/server/routes/session.ts:69-91` defines `GET /session/status` (operationId: `session.status`) returning `Record<string, SessionStatus.Info>` where Info is `idle | busy | retry`. This is exactly what orch-go needs for status polling without SSE.

**Source:** `packages/opencode/src/session/status.ts:49-55` — `get()` returns status or defaults to idle. `status.ts:57-59` — `list()` returns the full state map.

**Significance:** This is the highest-value, lowest-effort win. Orch-go can start using this endpoint immediately with zero OpenCode changes. Eliminates SSE-only status polling complexity.

---

### Finding 3: Session Schema Has No Metadata Bag — Easy to Add

**Evidence:** `Session.Info` Zod schema at `packages/opencode/src/session/index.ts:52-93` has fixed fields only: id, slug, projectID, directory, parentID, title, version, time, summary, share, permission, revert. No extensible `metadata` field. PATCH route at `session.ts:258-291` only accepts `title` and `time.archived`.

**Source:** `packages/opencode/src/session/index.ts:206-247` — `createNext()` builds Info from fixed inputs.

**Significance:** Adding `metadata?: Record<string, string>` requires: (1) add field to Zod schema, (2) accept in create input, (3) accept in PATCH validator, (4) include in createNext(). All backward-compatible — old sessions without metadata still valid.

---

### Finding 4: No Session TTL Exists — Instance TTL Pattern is Template

**Evidence:** Sessions persist forever (`Storage.write` → JSON file, never auto-deleted). Instance has TTL at `instance.ts:26` (`IDLE_TTL_MS = 30 * 60 * 1000`). Session.remove() exists and handles recursive cleanup (children, messages, parts, shares). No periodic cleanup job exists for sessions.

**Source:** `packages/opencode/src/project/instance.ts:155-204` — `applyEviction()` runs TTL then LRU. `packages/opencode/src/session/index.ts:353-374` — `remove()` handles full cleanup.

**Significance:** Session TTL would follow the Instance eviction pattern: add TTL field, add periodic scanner, call existing `Session.remove()`. Main new work is the periodic scanner since eviction today only runs reactively (inside `Instance.provide()`).

---

### Finding 5: ORCH_WORKER Metadata Server-Side Code May Have Been Lost

**Evidence:** Commit `2e851f3` added `x-opencode-env-ORCH_WORKER` header forwarding in SDK client. Commit message references "session.metadata.role" being set in session.ts:207-211. But `grep -r "ORCH_WORKER\|metadata\.role" packages/opencode/src/` returns no matches. The server-side handler was likely lost during an upstream rebase.

**Source:** `git show 2e851f308 -- packages/sdk/js/src/v2/client.ts` confirms SDK change exists. Codebase grep confirms server-side handling is missing.

**Significance:** Demonstrates the rebase risk — custom changes can be lost silently. Also demonstrates the need for the metadata API: if a generic metadata bag existed, the ORCH_WORKER role would just be a metadata key rather than a special header→field mapping.

---

## Synthesis

**Key Insights:**

1. **The status endpoint already exists** — `GET /session/status` returns exactly what orch-go needs. This is the single biggest finding: zero OpenCode work, immediate orch-go simplification.

2. **Metadata API is the highest ROI new feature** — 1-2 hours of work in the fork eliminates ~800 lines of workspace cross-reference code in orch-go. Schema addition is backward-compatible and follows existing Zod patterns.

3. **Session TTL is feasible but needs a new pattern** — No periodic job exists today. Instance eviction is reactive (runs inside request handlers). Session TTL needs a proper timer-based scanner. The `Session.remove()` function already handles cleanup correctly.

4. **Fork is well-maintained and rebasing is manageable** — 13 commits, mostly isolated. But the ORCH_WORKER loss proves that custom changes need tests or at least documentation to survive rebases.

**Answer to Investigation Question:**

All three features are feasible. The status endpoint already exists (zero effort). The metadata API requires a small schema addition (1-2 hours). Session TTL requires a new periodic cleanup job modeled on Instance eviction (2-4 hours). Total OpenCode fork work: ~4-6 hours. This would enable eliminating ~3,400 lines of orch-go lifecycle code as identified in the lifecycle ownership boundaries decision.

---

## Structured Uncertainty

**What's tested:**

- ✅ GET /session/status exists and returns idle/busy/retry (verified: read session routes and status.ts source)
- ✅ Session.Info has no metadata field (verified: read Zod schema at session/index.ts:52-93)
- ✅ Sessions have no TTL/auto-expiry (verified: read create/update/list functions, no expiry check anywhere)
- ✅ Fork is linear forward from upstream (verified: `git merge-base --is-ancestor upstream/dev HEAD`)
- ✅ Instance eviction exists with LRU+TTL (verified: read instance.ts, matched fork resource audit findings)

**What's untested:**

- ⚠️ Whether adding metadata field causes any downstream TypeScript errors (SDK types, app components)
- ⚠️ Whether periodic session cleanup timer interacts badly with Instance eviction timer
- ⚠️ How many sessions currently exist in Dylan's storage directory (affects cleanup job performance)
- ⚠️ Whether SDK generated types auto-update when Zod schema changes

**What would change this:**

- If Zod schema changes require SDK regeneration and that process is broken, metadata API effort increases significantly
- If upstream adds its own metadata/TTL features, we'd want to adopt those instead of custom fork changes
- If session count is very high (>10,000), periodic cleanup needs pagination/batching

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Use GET /session/status in orch-go | implementation | Change stays within orch-go scope |
| Add metadata field to Session.Info | architectural | Crosses OpenCode fork and orch-go boundaries |
| Add session TTL with periodic cleanup | architectural | New capability affecting both systems |

### Recommended Approach: Incremental Feature Addition

**Metadata first, then TTL** — start with the lowest-risk, highest-value change.

**Why this approach:**
- Metadata API is backward-compatible and immediately usable
- Status endpoint is already available — just needs orch-go client code
- TTL can be implemented after metadata proves the fork-change pattern works

**Implementation sequence:**
1. **orch-go: integrate GET /session/status** — immediate, zero OpenCode changes
2. **Fork: add metadata field to Session.Info** — small, backward-compatible schema change
3. **orch-go: store beads_id/workspace/tier in session metadata** — eliminates cross-ref files
4. **Fork: add session TTL with periodic cleanup** — most complex, builds on metadata pattern

---

## References

**Files Examined:**
- `~/Documents/personal/opencode/packages/opencode/src/session/index.ts` — Session schema, CRUD operations
- `~/Documents/personal/opencode/packages/opencode/src/session/status.ts` — In-memory status tracking
- `~/Documents/personal/opencode/packages/opencode/src/project/instance.ts` — Instance LRU/TTL eviction
- `~/Documents/personal/opencode/packages/opencode/src/server/routes/session.ts` — REST API routes
- `~/Documents/personal/opencode/packages/opencode/src/server/routes/global.ts` — SSE event stream
- `~/Documents/personal/opencode/packages/opencode/src/server/server.ts` — Server setup, SSE cleanup
- `~/Documents/personal/opencode/packages/opencode/src/storage/storage.ts` — File-based JSON storage layer
- `~/Documents/personal/opencode/.kb/investigations/2026-02-11-inv-opencode-fork-resource-audit-investigate.md` — Prior resource audit

**Commands Run:**
```bash
git remote -v
git log --oneline upstream/dev..HEAD
git diff upstream/dev..HEAD --stat
git show 2e851f308 -- packages/sdk/js/src/v2/client.ts
git diff upstream/dev..HEAD -- packages/opencode/src/project/instance.ts
git merge-base --is-ancestor upstream/dev HEAD
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-02-13-lifecycle-ownership-boundaries.md` — Parent decision identifying these three features
- **Model:** `.kb/models/opencode-fork.md` — Model created by this investigation
- **Model:** `.kb/models/agent-lifecycle-state-model.md` — Lifecycle model that depends on session behavior

---

## Investigation History

**2026-02-13:** Investigation started
- Initial question: What is the OpenCode fork's architecture and how feasible are three proposed lifecycle features?
- Context: Lifecycle ownership boundaries decision identified ~3,400 lines eliminable if OpenCode fork gains these features

**2026-02-13:** Parallel exploration of git history, package structure, and session management
- Three subagents explored fork divergence, architecture, and session code simultaneously
- Key discovery: GET /session/status already exists

**2026-02-13:** Investigation completed
- Status: Complete
- Key outcome: All three features are feasible; status endpoint already exists; metadata API is 1-2 hours; TTL is 2-4 hours
- Deliverable: Model created at `.kb/models/opencode-fork.md`
