## TLDR

Recurring backlog hygiene should use a hybrid model: enforce issue quality at creation time, then run recurring detection to catch drift from external/manual paths. I implemented creation-time fixes in `orch spawn` and discovered-work follow-up creation, and recorded the architecture in a decision.

## Summary (D.E.K.N.)

**Delta:** The root quality defects came from orch-managed creation paths that generated prefixed/truncated titles and omitted descriptions.

**Evidence:** `createBeadsIssue` previously used `[project] skill: truncate(task, 50)` with no description, and `processDiscoveredWork` created follow-up issues with empty descriptions.

**Knowledge:** Prevention must be enforced in creation paths, while recurring checks should be added as a separate detection layer.

**Next:** Land hybrid architecture: keep creation-time gates in code now and add periodic backlog quality checks as follow-up work.

**Authority:** architectural - the solution spans spawn behavior, completion pipeline behavior, and recurring operations posture.

---

# Investigation: Design Recurring Backlog Hygiene Solution

**Question:** What architecture should prevent recurring low-quality backlog items (truncated titles, missing descriptions), and where should quality be enforced vs audited?

**Started:** 2026-02-07
**Updated:** 2026-02-07
**Owner:** OpenCode worker (architect spawn)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** `.kb/decisions/2026-02-07-recurring-backlog-hygiene-issue-quality-gates.md`
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A | - | - | - |

---

## Findings

### Finding 1: Spawn-created issue format directly caused title degradation and missing descriptions

**Evidence:** `createBeadsIssue` used `"[%s] %s: %s"` title format with truncated task text and no description in both RPC and CLI fallback creation.

**Source:** `cmd/orch/spawn_beads.go` (pre-fix), verified by malformed examples `bd show orch-go-21288` and `bd show orch-go-21451`.

**Significance:** This confirms defects were not just prompt-level behavior; they were codified in an orch creation path.

---

### Finding 2: Completion discovered-work filing also created description-less issues

**Evidence:** `processDiscoveredWork` created follow-up issues via `beads.FallbackCreate(title, "", ...)`, always omitting description.

**Source:** `cmd/orch/complete_gates.go:319` to `cmd/orch/complete_gates.go:349`.

**Significance:** Even with perfect agent behavior, orchestration-side auto-filing could still generate low-quality backlog items.

---

### Finding 3: Recurring hygiene requires prevention + detection, not either/or

**Evidence:** Existing malformed issues persisted until manual audit cleanup, while code-level generation paths had no hard quality contract.

**Source:** `bd show orch-go-21288`, `bd show orch-go-21451`, and code paths above.

**Significance:** A single mechanism (prompts only, audits only, daemon only) is insufficient; quality must be guaranteed at creation and monitored over time.

---

## Synthesis

**Key Insights:**

1. **Root cause is structural, not stylistic** - orch creation code generated malformed issues by design in multiple paths.
2. **Gate-over-remind applies here** - enforce required descriptions at creation time instead of relying on reminders.
3. **Hybrid architecture is necessary** - creation gates stop new defects; recurring checks catch drift from external/manual paths.

**Answer to Investigation Question:**

Use a hybrid approach. Enforce issue quality in all orch-managed creation paths now (title intent first, mandatory description, metadata moved out of title), and add recurring backlog quality checks as a follow-up operational layer. This matches the failure mode: creation defects are immediate and recurring drift remains possible through non-orch pathways.

---

## Structured Uncertainty

**What's tested:**

- ✅ Spawn issue content now includes structured description and removes prefix-heavy title style (`go test ./cmd/orch -run TestBuildSpawnIssueContent -count=1`).
- ✅ Full `cmd/orch` tests pass after creation-path changes (`go test ./cmd/orch -count=1`).
- ✅ Historical malformed issue pattern confirmed from primary source (`bd show orch-go-21288`, `bd show orch-go-21451`).

**What's untested:**

- ⚠️ No new periodic doctor/audit command is implemented yet (design choice documented, deferred follow-up).
- ⚠️ No live end-to-end manual spawn was run in this session to create a fresh beads issue artifact.

**What would change this:**

- If a remaining orch path still creates issues with empty descriptions, this finding is incomplete and additional gates are required.
- If backlog drift still grows despite creation-path fixes, recurring checks must be elevated from follow-up to required gate.

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Hybrid backlog hygiene (creation gates now + recurring checks follow-up) | architectural | Spans spawn, completion, and ongoing operational checks |

### Recommended Approach ⭐

**Hybrid Quality Contract + Recurring Audit** - Enforce quality at issue creation points and add periodic backlog quality detection.

**Why this approach:**
- Stops defect creation at source (strongest leverage).
- Preserves amnesia-resilience with required descriptions.
- Provides operational safety net for non-orch/manual issue creation.

**Trade-offs accepted:**
- Slightly more verbose auto-generated descriptions.
- Requires a second implementation step for recurring checks.

**Implementation sequence:**
1. Fix orch-managed creation paths (spawn + discovered-work filing). ✅
2. Add regression tests for title/description contract. ✅
3. Add recurring backlog quality detector (doctor/audit) as follow-up. ⏭️

### Alternative Approaches Considered

**Option B: Skill/prompt-only enforcement**
- **Pros:** Fast.
- **Cons:** Non-binding; misses code-managed issue creation.
- **When to use instead:** Temporary stopgap before code changes.

**Option C: Audit-only recurring check**
- **Pros:** Centralized reporting.
- **Cons:** Post-facto; allows bad issues to accumulate between runs.
- **When to use instead:** Secondary layer after creation gates exist.

**Rationale for recommendation:** Hybrid is the only option that addresses both immediate root cause and recurrence dynamics.

---

### Implementation Details

**What to implement first:**
- Spawn issue creation contract (`cmd/orch/spawn_beads.go`).
- Discovered-work auto-filing contract (`cmd/orch/complete_gates.go`).
- Unit tests for spawn issue content behavior (`cmd/orch/spawn_beads_test.go`).

**Things to watch out for:**
- ⚠️ Avoid overloading issue titles with metadata that belongs in labels/description.
- ⚠️ Ensure future issue creation helpers also require non-empty descriptions.
- ⚠️ Verify area-label suggestion uses meaningful description context.

**Areas needing further investigation:**
- Periodic backlog quality check interface (`orch doctor` extension vs dedicated `orch audit backlog-quality`).
- Whether to add hard blocking in beads CLI for empty descriptions by default.

**Success criteria:**
- ✅ New orch-created issues contain descriptions by default.
- ✅ Spawn-created titles prioritize task intent and avoid prefix truncation anti-pattern.
- ✅ Decision record exists with clear rationale and follow-up plan.

---

## References

**Files Examined:**
- `cmd/orch/spawn_beads.go` - Spawn issue creation formatting and fields.
- `cmd/orch/complete_gates.go` - Discovered-work follow-up issue creation.
- `cmd/orch/main_test.go` - Existing spawn/beads-related tests and coverage shape.

**Commands Run:**
```bash
# Verify malformed issue examples from backlog
bd show orch-go-21288
bd show orch-go-21451

# Validate implementation changes
go test ./cmd/orch -run TestBuildSpawnIssueContent -count=1
go test ./cmd/orch -count=1
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-02-07-recurring-backlog-hygiene-issue-quality-gates.md` - Chosen architecture and alternatives.
- **Workspace:** `.orch/workspace/og-arch-design-recurring-backlog-07feb-a734/` - Spawn workspace context.

---

## Investigation History

**2026-02-07 09:54:** Investigation started
- Initial question: Design recurring backlog hygiene architecture and address low-quality issue creation.
- Context: Orch backlog audit found agent-created truncated titles and missing descriptions.

**2026-02-07 10:03:** Root causes verified in code
- Identified spawn and completion creation paths omitting descriptions / degrading titles.

**2026-02-07 10:18:** Investigation completed
- Status: Complete
- Key outcome: Hybrid architecture selected; creation-time fixes implemented; decision recorded.
