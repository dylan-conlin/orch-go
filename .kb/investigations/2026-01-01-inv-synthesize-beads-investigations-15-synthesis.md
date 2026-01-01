## Summary (D.E.K.N.)

**Delta:** 15 beads-related investigations (Dec 19-30, 2025) tell a cohesive story of evolving from CLI subprocess calls to a robust pkg/beads abstraction layer with RPC client, CLI fallback, and deduplication.

**Evidence:** Reviewed all 15 investigations covering: spawning integration (3), pkg/beads client evolution (3), architecture decisions (3), database issues (3), UI/dashboard features (2), and bug fixes (1).

**Knowledge:** The investigations document a complete integration strategy: (1) beads is external OSS tool - stay on upstream only, (2) orch-go now has a complete pkg/beads abstraction layer with RPC client, CLI fallback, and mock for testing, (3) multi-repo hydration works in v0.33.2+, (4) SQLite WAL mode requires freshness checks in daemon mode.

**Next:** Archive this synthesis; these investigations can be considered consolidated - no individual investigations need superseding as each documents a distinct step in the evolution.

---

# Investigation: Synthesis of 15 Beads Investigations

**Question:** What patterns and consolidated knowledge can be extracted from 15 beads-related investigations (Dec 19-30, 2025)?

**Started:** 2026-01-01
**Updated:** 2026-01-01
**Owner:** synthesis agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Supersedes:** None (synthesis consolidates but doesn't replace individual investigations)

---

## Investigation Inventory

### Category 1: Spawning & Issue Lifecycle Integration (3 investigations)

| Date | Investigation | Key Finding | Status |
|------|---------------|-------------|--------|
| 2025-12-19 | inv-set-beads-issue-status-progress | Call `verify.UpdateIssueStatus(beadsID, "in_progress")` after beads ID determination | Complete |
| 2025-12-21 | inv-orch-complete-closes-beads-issue | Registry update must happen BEFORE beads close to maintain consistency | Complete |
| 2025-12-21 | inv-beads-kb-workspace-relationships-how | Three-layer architecture: Beads (WIP) → KB (knowledge) → Workspace (ephemeral) | Complete |

### Category 2: pkg/beads Client Evolution (3 investigations)

| Date | Investigation | Key Finding | Status |
|------|---------------|-------------|--------|
| 2025-12-25 | inv-implement-pkg-beads-go-rpc | Implemented 7-operation RPC client with Unix socket and health checks | Complete |
| 2025-12-26 | inv-implement-pkg-beads-rpc-client | Extended to 13 operations with auto-reconnect and exponential backoff | Complete |
| 2025-12-27 | inv-create-pkg-beads-abstraction-layer | Confirmed abstraction layer is COMPLETE: interface + RPC + CLI + Mock | Complete |

### Category 3: Architecture & Integration Strategy (3 investigations)

| Date | Investigation | Key Finding | Status |
|------|---------------|-------------|--------|
| 2025-12-21 | inv-beads-oss-relationship-fork-vs | Clean slate - stay on upstream beads only; local features weren't used | Complete |
| 2025-12-25 | inv-design-beads-integration-strategy-orch | Recommend native Go RPC client; 7-command CLI surface is tractable | Complete |
| 2025-12-22 | inv-beads-multi-repo-hydration-why | Multi-repo works in v0.33.2+; historical bug was config-DB-YAML disconnect | Complete |

### Category 4: Database & Data Issues (3 investigations)

| Date | Investigation | Key Finding | Status |
|------|---------------|-------------|--------|
| 2025-12-25 | inv-beads-database-pollution-orch-go | Cross-repo `additional` config caused 787 bd-* issues to pollute database | Complete |
| 2025-12-30 | inv-add-deduplication-check-beads-cli | Client-side deduplication prevents duplicate issues; uses FindByTitle | Complete |
| 2025-12-30 | inv-investigate-beads-comments-sync-issue | SQLite WAL mode race condition; fixed in beads commit 2e0ce160 | Complete |

### Category 5: UI & Dashboard Features (2 investigations)

| Date | Investigation | Key Finding | Status |
|------|---------------|-------------|--------|
| 2025-12-20 | inv-scaffold-beads-ui-v2-bun | Created SvelteKit 5 dashboard in web/ with agent cards and SSE placeholder | Complete |
| 2025-12-24 | inv-add-beads-stats-dashboard-stats | `bd stats --json` provides ready/blocked counts; API endpoint added | Complete |

### Category 6: Fallback & Resilience (1 investigation)

| Date | Investigation | Key Finding | Status |
|------|---------------|-------------|--------|
| 2025-12-24 | inv-add-fallback-beads-close-reason | Light-tier agents use beads close_reason as SYNTHESIS.md fallback | Complete |

---

## Findings

### Finding 1: Complete pkg/beads Abstraction Layer Now Exists

**Evidence:** Three investigations (Dec 25-27) document the evolution from CLI subprocess calls to a complete abstraction:
- `pkg/beads/interface.go` - BeadsClient interface with 12+ methods
- `pkg/beads/client.go` - RPC client (799 lines) with auto-reconnect
- `pkg/beads/cli_client.go` - CLI fallback (287 lines)
- `pkg/beads/mock_client.go` - Mock for testing (403 lines)

**Source:** 
- `.kb/investigations/2025-12-25-inv-implement-pkg-beads-go-rpc.md`
- `.kb/investigations/2025-12-26-inv-implement-pkg-beads-rpc-client.md`
- `.kb/investigations/2025-12-27-inv-create-pkg-beads-abstraction-layer.md`

**Significance:** All orch-go consumers (daemon, verify, serve) now use the abstraction. Only one exec.Command("bd", ...) remains outside pkg/beads: `cmd/orch/init.go` for project initialization (appropriate exception).

---

### Finding 2: Beads Integration Strategy is "Clean Slate - Upstream Only"

**Evidence:** Investigation on Dec 21 found that local beads features (ai-help, health, tree, --discovered-from) had zero usage in orch-go. Decision was made to:
1. Abort rebase with conflicts
2. Reset to upstream main
3. Delete local-features branch
4. Stay on vanilla upstream beads

**Source:** `.kb/investigations/2025-12-21-inv-beads-oss-relationship-fork-vs.md`

**Significance:** Reduces maintenance burden. Any future beads improvements should be upstream PRs, not local patches. This decision was documented in `.kb/decisions/2025-12-21-beads-oss-relationship-clean-slate.md`.

---

### Finding 3: Multi-Repo Hydration Works (With Prerequisites)

**Evidence:** Investigation on Dec 22 found that multi-repo was "buggy" in v0.29.0 due to config disconnect (bd repo add wrote to DB, hydration read from YAML). Fixed in commit 634c0b93. Current v0.33.2 works correctly.

Prerequisites for multi-repo:
1. beads >= v0.33.2
2. Healthy database (no orphaned dependencies)
3. `bd repo sync` run after `bd repo add`

**Source:** `.kb/investigations/2025-12-22-inv-beads-multi-repo-hydration-why.md`

**Significance:** Multi-repo can be used for cross-project issue visibility, but requires understanding of how it imports ALL issues from additional repos into the primary database.

---

### Finding 4: SQLite WAL Mode Requires Freshness Checks

**Evidence:** Investigation on Dec 30 found that `bd comments` returned empty when JSONL had data. Root cause: different SQLite connections have different WAL snapshots. The `checkFreshness()` + RLock pattern was present in GetIssue but missing from comment retrieval. Fixed in beads commit 2e0ce160.

**Source:** `.kb/investigations/2025-12-30-inv-investigate-beads-comments-sync-issue.md`

**Significance:** Any new beads storage functions that follow a write must include freshness checking. This is a beads-level concern, not orch-go, but important for debugging future issues.

---

### Finding 5: Three-Layer Artifact Architecture is Documented

**Evidence:** Investigation on Dec 21 documented the relationship model:
- **Beads** (.beads/) - WIP tracking with JSONL persistence
- **KB** (.kb/) - Persistent knowledge artifacts (investigations, decisions)
- **Workspace** (.orch/workspace/) - Ephemeral agent execution context

Linking mechanisms: investigation_path comments, kb link, beads ID in SPAWN_CONTEXT.md.

**Source:** `.kb/investigations/2025-12-21-inv-beads-kb-workspace-relationships-how.md`

**Significance:** This provides a reference architecture for future agents understanding how artifacts connect across the system.

---

## Synthesis

**Key Insights:**

1. **Evolution from CLI to RPC** - The investigations document a 12-day evolution from direct exec.Command("bd", ...) calls to a full abstraction layer with RPC client, CLI fallback, and mock for testing. This was driven by performance needs (dashboard polling 50+ agents every 2-5s) and reliability concerns (daemon race conditions).

2. **OSS Relationship Clarified** - The decision to stay on upstream-only beads resolved ongoing rebase conflicts and maintenance burden. Local features that weren't integrated were creating friction without value.

3. **Database Hygiene Matters** - Multiple investigations (pollution, multi-repo, WAL race) highlight that beads database state requires attention. The .gitignore patterns, freshness checks, and deduplication all address data integrity concerns.

4. **Light-Tier Agents Need Fallbacks** - The close_reason fallback addresses a gap where agents without SYNTHESIS.md still need summary information in the dashboard.

**Consolidated Knowledge for Future Agents:**

| Topic | What You Need to Know | Source |
|-------|----------------------|--------|
| Beads client usage | Use `beads.NewClient()` with `WithAutoReconnect(3)`, falls back to CLI | inv-implement-pkg-beads-rpc-client |
| OSS relationship | Stay on upstream only; don't maintain local patches | inv-beads-oss-relationship-fork-vs |
| Multi-repo | Works in v0.33.2+; requires `bd repo sync` after add | inv-beads-multi-repo-hydration-why |
| Database pollution | Don't use `additional:` config unless you want ALL issues imported | inv-beads-database-pollution-orch-go |
| Comments empty? | Check WAL freshness; fixed in beads 2e0ce160 | inv-investigate-beads-comments-sync-issue |
| Deduplication | Create() checks for existing open issue by title; use Force=true to bypass | inv-add-deduplication-check-beads-cli |
| Lifecycle | Set in_progress on spawn; update registry BEFORE beads close | inv-set-beads-issue-status-progress, inv-orch-complete-closes-beads-issue |

---

## Structured Uncertainty

**What's tested:**

- ✅ pkg/beads abstraction layer works (verified: 71 tests pass, all consumers migrated)
- ✅ Multi-repo hydration works in v0.33.2+ (verified: tested with 323 orch-go issues)
- ✅ Deduplication prevents duplicate issues (verified: 6 unit tests)
- ✅ Comments sync fixed (verified: bd v0.33.2 returns correct data)

**What's untested:**

- ⚠️ RPC client behavior under very high load (50+ concurrent agents) - not stress-tested
- ⚠️ Connection pooling strategy effectiveness - no benchmarks
- ⚠️ Edge cases with daemon restart during operations

**What would change this:**

- If beads upstream makes breaking RPC protocol changes, pkg/beads would need updates
- If new CLI operations are added to beads, the abstraction would need extending
- If light-tier spawns become common, close_reason quality may need enforcement

---

## Implementation Recommendations

### Recommended Approach ⭐

**No implementation needed** - This is a synthesis investigation that consolidates knowledge from 15 completed investigations.

**What this synthesis enables:**
- Future agents can read this single document to understand beads integration evolution
- The investigation inventory provides a navigable index to specific details
- The consolidated knowledge table provides quick answers without reading all 15 investigations

**Maintenance recommendations:**
1. Keep these investigations archived (they document the evolution)
2. Don't supersede them - each documents a distinct step
3. Reference this synthesis for future beads-related questions

---

## References

**Files Examined:**

All 15 investigations in `.kb/investigations/`:
- 2025-12-19-inv-set-beads-issue-status-progress.md
- 2025-12-20-inv-scaffold-beads-ui-v2-bun.md
- 2025-12-21-inv-beads-kb-workspace-relationships-how.md
- 2025-12-21-inv-beads-oss-relationship-fork-vs.md
- 2025-12-21-inv-orch-complete-closes-beads-issue.md
- 2025-12-22-inv-beads-multi-repo-hydration-why.md
- 2025-12-24-inv-add-beads-stats-dashboard-stats.md
- 2025-12-24-inv-add-fallback-beads-close-reason.md
- 2025-12-25-inv-beads-database-pollution-orch-go.md
- 2025-12-25-inv-design-beads-integration-strategy-orch.md
- 2025-12-25-inv-implement-pkg-beads-go-rpc.md
- 2025-12-26-inv-implement-pkg-beads-rpc-client.md
- 2025-12-27-inv-create-pkg-beads-abstraction-layer.md
- 2025-12-30-inv-add-deduplication-check-beads-cli.md
- 2025-12-30-inv-investigate-beads-comments-sync-issue.md

**Related Artifacts:**

- **Decision:** `.kb/decisions/2025-12-21-beads-oss-relationship-clean-slate.md` - OSS strategy decision

---

## Investigation History

**2026-01-01:** Investigation started
- Initial question: Synthesize 15 beads-related investigations
- Context: Accumulation of investigations suggested need for consolidation

**2026-01-01:** All 15 investigations reviewed
- Categorized into 6 themes: spawning, client evolution, architecture, database, UI, fallback
- Identified patterns and consolidated knowledge

**2026-01-01:** Investigation completed
- Status: Complete
- Key outcome: Consolidated knowledge into single reference document; no investigations superseded (each documents a distinct step in evolution)
