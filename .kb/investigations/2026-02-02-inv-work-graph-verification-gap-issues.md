<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Work completed outside spawn flow (interactive sessions, manual work, `bd close` without `orch complete`) creates orphaned open issues because the completion verification system only triggers via spawn workspace lifecycle.

**Evidence:** Traced verification flow through `orch complete` → requires workspace → workspace only exists for spawned agents. `orch emit` + beads hooks is a partial solution for `bd close`, but interactive/ad-hoc work has no hook. Phase 2 designs (DeliverableChecklist, AttemptHistory) are orthogonal - they track spawn-flow work, not ad-hoc work.

**Knowledge:** The gap is architectural: Work Graph surfaces beads issues, but verification is spawn-centric. Three classes of work bypass spawn: (1) interactive sessions claiming work, (2) manual commits that resolve issues, (3) `bd close` without `orch complete`. Need reconciliation that looks at git history, not just workspace state.

**Next:** Recommend hybrid approach: (1) commit-aware reconciliation for "likely done" surfacing, (2) lightweight verification for ad-hoc closure, (3) Work Graph UI to show "likely done" state. This is Work Graph's responsibility (observability), not a separate system.

**Authority:** architectural - Crosses spawn system, verification system, and Work Graph; requires design synthesis across components

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Authority: implementation - Tactical fix within existing patterns, no architectural impact

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Authority: Classify by who decides - implementation (worker within scope), architectural (orchestrator across boundaries), strategic (Dylan for irreversible/value choices)
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Work Graph Verification Gap for Ad-Hoc Work

**Question:** How should Work Graph surface "likely done" issues (where commits exist but no workspace/spawn), integrate with existing completion verification, and handle lifecycle for ad-hoc/interactive work?

**Started:** 2026-02-02
**Updated:** 2026-02-02
**Owner:** Claude (spawned architect)
**Phase:** Complete
**Next Step:** None - recommendations ready for orchestrator review
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** N/A
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Current verification is spawn-centric

**Evidence:** The completion verification flow requires a workspace:
- `orch complete <beads-id>` → looks up workspace in `.orch/workspace/`
- Workspace contains `.session_id`, `.tier`, `SPAWN_CONTEXT.md`
- Verification gates (Phase, Evidence, Approval) all read from workspace + beads comments
- No verification path exists for work without a workspace

**Source:** 
- `cmd/orch/complete_cmd.go` - workspace lookup required
- `.kb/models/completion-verification.md` - documents three-layer verification
- `.kb/guides/completion.md` - procedural flow assumes spawn

**Significance:** Any work completed outside the spawn flow has no verification path. This includes:
1. Interactive sessions where Dylan claims an issue
2. Manual commits that resolve an issue
3. Direct `bd close` without `orch complete`

---

### Finding 2: orch emit + beads hooks is a partial solution

**Evidence:** `cmd/orch/emit_cmd.go` provides a mechanism to emit `agent.completed` events from outside the spawn flow:

```bash
# .beads/hooks/on_close
orch emit agent.completed --beads-id "$BD_ISSUE_ID" --reason "Closed via bd close"
```

This closes the tracking gap for `bd close`, but:
- Requires hook setup (not automatic)
- Only fires on explicit `bd close`, not on "work is done but issue is still open"
- No verification - just event emission

**Source:**
- `cmd/orch/emit_cmd.go:30-167` - hook integration instructions
- `.kb/guides/completion.md:155-166` - mentions `bd close` hook

**Significance:** The partial solution exists but doesn't address the core question: "How do we know work is done when it wasn't done via spawn flow?"

---

### Finding 3: Phase 2 designs are orthogonal to this gap

**Evidence:** The Phase 2 Work Graph design (DeliverableChecklist, AttemptHistory, IssueSidePanel) tracks:
- Attempts: How many times an issue was worked on via spawn
- Deliverables: Expected vs actual outputs per issue type
- Lifecycle: Agent state transitions

All of this assumes spawn-flow work. The designs don't address:
- Issues with commits but no spawn history
- Issues closed manually without agent involvement
- Interactive session work that isn't tracked

**Source:**
- `.kb/investigations/2026-01-31-design-work-graph-phase2-agent-overlay.md:110-165`
- `.kb/investigations/2026-02-02-inv-audit-work-graph-design-docs.md:89-100`

**Significance:** Phase 2 is about **deep observability into spawn-flow work**. This investigation is about **basic observability into non-spawn-flow work**. Orthogonal concerns.

---

### Finding 4: Three classes of ad-hoc work bypass spawn

**Evidence:** Based on principles and prior investigations:

| Class | Mechanism | Current State | Needed |
|-------|-----------|---------------|--------|
| **Interactive claim** | Orchestrator opens session, works on issue directly | No workspace, no tracking | "Claimed by interactive session" state |
| **Manual commits** | Human/agent commits outside spawn, issue stays open | Commits exist, issue open | Commit-based "likely done" detection |
| **Direct close** | `bd close` without `orch complete` | Hook exists but opt-in | Automatic reconciliation |

**Source:**
- `.kb/guides/orchestrator-session-management.md:89` - "Orchestrators aren't issues being worked on"
- `.kb/models/completion-verification.md:221-227` - Cross-project verification gaps
- `kb context` output - constraint "Session idle ≠ agent complete"

**Significance:** Each class needs different handling. Interactive claims need state tracking. Manual commits need git-based detection. Direct closes need reconciliation.

---

### Finding 5: Git history can surface "likely done" issues

**Evidence:** The verification system already uses git to check for commits:
- `pkg/verify/git_diff.go:214-226` - uses `git log --since` to find commits since spawn
- `pkg/verify/test_evidence.go:226-239` - checks for commits in spawn timeframe

This same approach could detect commits that reference beads IDs outside spawn flow:

```bash
# Find commits mentioning issue ID
git log --all --grep="orch-go-21121" --oneline

# Find commits in .kb/investigations that match issue
git log --all -- '.kb/investigations/*21121*' --oneline
```

**Source:**
- `pkg/verify/git_diff.go` - existing git log usage
- `pkg/verify/test_evidence.go:226-239` - commit detection pattern

**Significance:** The building blocks exist. Need to compose them into a "likely done" detector that runs outside spawn context.

---

## Synthesis

**Key Insights:**

1. **Verification is workspace-scoped by design** - The current system assumes spawn → workspace → verification. This is correct for agent work but creates a gap for human/ad-hoc work. The gap isn't a bug - it's a scope boundary.

2. **"Likely done" is observability, not verification** - We're not asking "is this work complete?" (verification requires gates). We're asking "does evidence suggest this work might be done?" (observability requires detection). Different standards apply.

3. **Git is the source of truth for non-spawn work** - If commits exist that mention an issue ID, reference investigation files, or touch expected deliverable paths, that's strong evidence of work. Spawn history is one signal; git history is another.

4. **Work Graph is the right owner** - This is about surfacing state in the dashboard, not about completion gates. Work Graph already queries beads and shows status. Adding "likely done" is a new status, not a new system.

**Answer to Investigation Question:**

**How should Work Graph surface "likely done" issues?**

Add a new computed status `likely_done` to the graph API response when:
- Issue is open
- Commits exist that mention the issue ID (in commit message or file paths)
- No active workspace exists for this issue
- Last commit was recent (configurable threshold, e.g., 7 days)

**How should it integrate with existing verification?**

It doesn't. "Likely done" is an observability signal, not a verification gate. The flow is:
1. Work Graph shows "likely done" indicator
2. Human reviews and decides
3. Human runs `orch complete --ad-hoc <id>` (new lightweight path) or `bd close <id>` with reason
4. Issue closed

The `--ad-hoc` path would skip spawn-centric gates (no workspace to check) but still log the completion event.

**Is this Work Graph's responsibility or a separate system?**

Work Graph's responsibility. It's about observability of work state. The reconciliation logic (`git log` queries) should live in a new package (`pkg/reconcile`) that Work Graph calls.

---

## Structured Uncertainty

**What's tested:**

- ✅ Current verification requires workspace (verified: read complete_cmd.go, requires workspace lookup)
- ✅ orch emit exists for bd close hooks (verified: read emit_cmd.go)
- ✅ git log can find commits by message content (verified: common git capability)
- ✅ Phase 2 designs focus on spawn-flow work (verified: read design docs)

**What's untested:**

- ⚠️ git log performance for commit-message search across full history
- ⚠️ False positive rate for "likely done" detection (commits may reference issues without resolving them)
- ⚠️ UX for "likely done" vs "in progress" vs "open" status differentiation

**What would change this:**

- If git log commit-message search is prohibitively slow (test with real repo history)
- If Dylan prefers "likely done" to be a separate reconciliation batch job, not real-time API
- If the false positive rate is too high (need heuristics beyond just "commit exists")

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add "likely done" status to Work Graph | architectural | Crosses Work Graph, beads, git history; requires design synthesis |
| Create `pkg/reconcile` for git-based detection | architectural | New package, shared by multiple consumers |
| Add `--ad-hoc` path to orch complete | architectural | New verification path with different gates |

**Authority Levels:**
- **implementation**: Worker decides within scope (reversible, single-scope, clear criteria, no cross-boundary impact)
- **architectural**: Orchestrator decides across boundaries (cross-component, multiple valid approaches, requires synthesis)
- **strategic**: Dylan decides on direction (irreversible, resource commitment, value judgment, premise-level question)

### Recommended Approach ⭐

**Commit-Aware "Likely Done" Surfacing** - Extend Work Graph API to compute and surface "likely done" status for open issues based on git commit evidence.

**Why this approach:**
- Uses existing git capabilities (no new dependencies)
- Non-blocking observability (doesn't gate workflow)
- Integrates naturally with Work Graph's existing status rendering
- Addresses all three ad-hoc work classes with one mechanism

**Trade-offs accepted:**
- Some false positives (commits that mention but don't resolve issues)
- Performance overhead of git log queries (can cache/batch)
- Human still makes final close decision (not fully automated)

**Implementation sequence:**
1. **Create `pkg/reconcile`** - Git-based detection logic, reusable by API and CLI
2. **Add `likely_done` to graph API** - Compute on-demand or cache with TTL
3. **Update Work Graph UI** - Show "likely done" indicator (distinct from "in progress")
4. **Add `orch complete --ad-hoc`** - Lightweight closure path for manual review
5. **Update guides** - Document new workflow

### Alternative Approaches Considered

**Option B: Periodic Reconciliation Job**
- **Pros:** No API latency impact, runs when system is idle
- **Cons:** Stale data between runs, requires daemon integration, another background process
- **When to use instead:** If git queries prove too slow for real-time API

**Option C: Beads-Level "Likely Done" Detection**
- **Pros:** Could integrate with `bd` CLI directly
- **Cons:** Beads doesn't have git access by design (it's issue tracking, not code tracking)
- **When to use instead:** If we want `bd list` to show "likely done" without orch

**Rationale for recommendation:** Option A (API integration) provides immediate value with minimal architectural change. Work Graph already queries issues; adding commit-based status is additive. The reconciliation job (Option B) could come later as an optimization if performance requires it.

---

### Implementation Details

**What to implement first:**
- `pkg/reconcile/likely_done.go` - Core detection logic
- Detection criteria: commits mention issue ID in message OR modify files matching issue ID patterns

**Things to watch out for:**
- ⚠️ Git log queries can be slow on large repos - need caching strategy
- ⚠️ Cross-project issues have commits in different repos - may need multi-repo support
- ⚠️ "Likely done" should NOT trigger automatic closure - human review required

**Areas needing further investigation:**
- Optimal TTL for likely_done cache (balance freshness vs performance)
- Whether to include closed issues that were reopened
- How to handle epic children (epic likely done if all children likely done?)

**Success criteria:**
- ✅ Open issues with relevant commits show "likely done" in Work Graph
- ✅ No false negatives (commits exist but not detected)
- ✅ False positive rate < 20% (most "likely done" issues are actually done)
- ✅ API response time < 500ms for graph endpoint with likely_done

---

## References

**Files Examined:**
- `cmd/orch/complete_cmd.go` - Workspace lookup, verification flow
- `cmd/orch/emit_cmd.go` - Beads hook integration
- `cmd/orch/serve_beads.go:616-810` - Graph API implementation
- `pkg/verify/git_diff.go` - Git log usage patterns
- `pkg/verify/test_evidence.go` - Commit detection

**Commands Run:**
```bash
# Search for git log usage in verification
grep -r "git log" pkg/verify/

# Check emit command structure
cat cmd/orch/emit_cmd.go

# Query kb for completion context
kb context "completion verification lifecycle issue tracking"
```

**External Documentation:**
- Principles: `~/.kb/principles.md` - Observation Infrastructure, Evidence Hierarchy

**Related Artifacts:**
- **Model:** `.kb/models/completion-verification.md` - Three-layer verification architecture
- **Guide:** `.kb/guides/completion.md` - Completion workflow
- **Investigation:** `.kb/investigations/2026-02-02-inv-audit-work-graph-design-docs.md` - Phase 2 gaps
- **Design:** `.kb/investigations/2026-01-31-design-work-graph-phase2-agent-overlay.md` - AttemptHistory design

---

## Investigation History

**2026-02-02 12:30:** Investigation started
- Initial question: How should Work Graph handle "likely done" issues from ad-hoc work?
- Context: Spawned as architect task to design reconciliation approach

**2026-02-02 12:45:** Context gathered
- Read completion verification model, spawn guide, Work Graph designs
- Identified orch emit + hooks as partial solution
- Identified three classes of ad-hoc work

**2026-02-02 13:00:** Decision forks identified
- Fork 1: Work Graph responsibility vs separate system → Work Graph (observability is its domain)
- Fork 2: Real-time API vs batch reconciliation → API first, batch as optimization
- Fork 3: Full verification vs lightweight closure → Lightweight (observability, not gates)
- Fork 4: Automatic vs human-reviewed closure → Human-reviewed (avoid false positives)

**2026-02-02 13:15:** Investigation completed
- Status: Complete
- Key outcome: Recommend commit-aware "likely done" surfacing via Work Graph API, with `orch complete --ad-hoc` for manual closure of detected issues
