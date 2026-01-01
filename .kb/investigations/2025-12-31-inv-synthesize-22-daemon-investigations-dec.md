<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Consolidated 22+ daemon investigations (Dec 20-30) into 3 decision records capturing core daemon principles: excludes untracked agents, skips failing issues per cycle, recomputes state each cycle.

**Evidence:** Analyzed investigations chronologically via `kb chronicle 'daemon'`; identified 5 capacity-counting bug fixes that converged on common design principles; traced evolution from initial implementation to production-ready daemon.

**Knowledge:** The daemon evolved through iterative bug discovery - initial implementation worked in isolation but failed at scale/overnight. Key insight: "When in doubt, recompute. Don't trust yesterday's state."

**Next:** Close - 3 decision records created, 9 investigations identified for archival (obsolete after fixes shipped), narrative captured.

---

# Investigation: Synthesize 22 Daemon Investigations (Dec 2025)

**Question:** What decisions emerged from the daemon investigations, and which investigations are now obsolete?

**Started:** 2025-12-31
**Updated:** 2025-12-31
**Owner:** og-work-synthesize-22-daemon-31dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## The Daemon Evolution Narrative

### Phase 1: Initial Implementation (Dec 20)

The daemon was created in `2025-12-20-inv-orch-add-daemon-command.md` with a clean design:
- `pkg/daemon` package with `NextIssue()`, `Preview()`, `Once()`, `Run()`
- Skill inference from issue type (bug → systematic-debugging, etc.)
- Integration with beads via `bd list --status open --json`
- 16 passing tests

**Key architectural choice:** The daemon was designed to poll beads and spawn work, treating each cycle independently.

### Phase 2: Queue Selection Bugs (Dec 24)

Early operational testing revealed queue selection issues:
- `2025-12-24-inv-daemon-finds-triage-ready-issues.md` - Daemon printed misleading "No spawnable issues" when issues existed but couldn't be spawned
- `2025-12-24-inv-daemon-selects-issues-triage-ready.md` - Fixed message to use `result.Message` from `Once()`
- `2025-12-24-inv-daemon-uses-bd-list-status.md` - Confirmed daemon correctly uses `bd list --status open`

**Pattern:** Early bugs were in messaging/visibility, not core logic.

### Phase 3: Capacity Counting Bugs (Dec 25-26)

The daemon's capacity management required 4+ iterations to get right:

1. **Initial bug:** WorkerPool tracked slots but never reconciled
   - `2025-12-26-inv-daemon-capacity-count-goes-stale.md` - Added `Pool.Reconcile()`

2. **Sessions persist after completion:** OpenCode sessions don't disappear when agents exit
   - `2025-12-26-inv-daemon-capacity-count-stuck-while.md` - Added 30-minute recency filter

3. **Untracked agents inflated count:** `--no-track` spawns counted toward daemon capacity
   - `2025-12-26-inv-daemon-capacity-count-stale-after.md` - Added untracked detection via `-untracked-` pattern

4. **Closed issues still counted:** Sessions with closed beads issues weren't excluded
   - `2025-12-26-debug-daemon-capacity-stale-after-complete.md` - Added beads status check via batch query

**Key insight:** Each bug revealed another dimension of "what does 'active agent' mean?" The final definition:
> Active agent = OpenCode session updated within 30 minutes + tracked (not `--no-track`) + open beads issue

### Phase 4: Blocking/Queue Bugs (Dec 28-30)

As the daemon processed larger queues, new issues emerged:

- `2025-12-28-inv-daemon-ignores-skill-labels-inferring.md` - Added `InferSkillFromLabels()` to respect `skill:*` labels
- `2025-12-30-inv-daemon-blocked-cross-project-failure.md` - Daemon would block on ANY spawn failure; added per-cycle skip tracking

**Pattern:** Scaling revealed queue fairness issues - one problematic issue could block the entire queue.

---

## Findings

### Finding 1: Capacity Counting Required 4 Incremental Fixes

**Evidence:** Four separate investigations each fixed one dimension of the capacity bug:
1. Pool reconciliation (pool internal state vs OpenCode reality)
2. Recency filtering (session persistence)
3. Untracked filtering (`--no-track` agents)
4. Closed issue filtering (completed work)

**Source:** 
- `.kb/investigations/2025-12-26-inv-daemon-capacity-count-goes-stale.md`
- `.kb/investigations/2025-12-26-inv-daemon-capacity-count-stuck-while.md`
- `.kb/investigations/2025-12-26-inv-daemon-capacity-count-stale-after.md`
- `.kb/investigations/2025-12-26-debug-daemon-capacity-stale-after-complete.md`

**Significance:** These investigations are now obsolete as individual artifacts but their findings are consolidated into the decision record `2025-12-31-daemon-recomputes-state-each-cycle.md`.

---

### Finding 2: Queue Blocking Was All-or-Nothing

**Evidence:** When spawn failed for any reason, daemon would `break` from the entire loop:
```go
if !result.Processed {
    break  // Blocked entire queue
}
```

**Source:** `.kb/investigations/2025-12-30-inv-daemon-blocked-cross-project-failure.md`

**Significance:** Required fundamental change from "stop on any failure" to "skip and continue." Documented in decision `2025-12-31-daemon-skips-failing-issues-per-cycle.md`.

---

### Finding 3: Many Investigations Were Point-in-Time Debug Sessions

**Evidence:** Multiple investigations from Dec 22-23 are tests/validations that confirmed fixes worked:
- `2025-12-22-inv-test-headless-mode.md` (archived)
- `2025-12-22-inv-test-headless-spawn-list-files.md` (archived)
- Many `test-*` named investigations are validation, not discovery

**Source:** `.kb/investigations/archived/` already contains 18 such test investigations

**Significance:** Test/validation investigations served their purpose but don't provide ongoing value.

---

## Proposed Actions

### Archive Actions (Obsolete Investigations)

| ID | Target | Reason | Approved |
|----|--------|--------|----------|
| A1 | `2025-12-26-inv-daemon-capacity-count-goes-stale.md` | Findings consolidated into decision record; fix shipped | [ ] |
| A2 | `2025-12-26-inv-daemon-capacity-count-stuck-while.md` | Findings consolidated into decision record; fix shipped | [ ] |
| A3 | `2025-12-26-inv-daemon-capacity-count-stale-after.md` | Findings consolidated into decision record; fix shipped | [ ] |
| A4 | `2025-12-26-debug-daemon-capacity-stale-after-complete.md` | Findings consolidated into decision record; fix shipped | [ ] |
| A5 | `2025-12-24-inv-daemon-finds-triage-ready-issues.md` | Simple message fix, now obsolete | [ ] |
| A6 | `2025-12-24-inv-daemon-selects-issues-triage-ready.md` | Simple message fix, now obsolete | [ ] |
| A7 | `2025-12-24-inv-daemon-uses-bd-list-status.md` | Verification only, no ongoing value | [ ] |
| A8 | `2025-12-25-inv-migrate-daemon-listreadyissues-use-new.md` | Migration complete, now obsolete | [ ] |
| A9 | `2025-12-28-inv-daemon-ignores-skill-labels-inferring.md` | Fix shipped, now obsolete | [ ] |

### Create Actions (Already Done)

| ID | Type | Title | Description | Status |
|----|------|-------|-------------|--------|
| C1 | decision | "Daemon Excludes Untracked Agents" | Consolidated capacity filtering decisions | Done |
| C2 | decision | "Daemon Skips Failing Issues Per Cycle" | Queue fairness decision | Done |
| C3 | decision | "Daemon Recomputes State Each Cycle" | Stateless-per-cycle principle | Done |

**Summary:** 9 archive proposals, 3 decisions created

---

## Synthesis

**Key Insights:**

1. **Iterative bug discovery is the norm** - Complex systems reveal bugs progressively. The daemon was correct in isolation but failed at scale/overnight operation.

2. **"Stateless per cycle" is the right mental model** - Rather than tracking internal state that can drift, the daemon should recompute from authoritative sources (OpenCode, beads) each cycle.

3. **Decisions extracted from investigations provide lasting value** - The 22 investigations captured point-in-time debugging, but the 3 decision records capture the principles that emerged.

**Answer to Investigation Question:**

Three key decisions emerged:
1. **Untracked exclusion** - `--no-track` agents don't count toward daemon capacity
2. **Per-cycle skip tracking** - Failed spawns skip to next issue, retry next cycle
3. **State recomputation** - Trust external sources over internal state

Nine investigations are now obsolete (fixes shipped, findings consolidated) and should be archived.

---

## References

**Daemon Chronicle:**
```bash
kb chronicle 'daemon'  # 186 entries covering Dec 20-30
```

**Decisions Created:**
- `.kb/decisions/2025-12-31-daemon-excludes-untracked-agents-from-capacity.md`
- `.kb/decisions/2025-12-31-daemon-skips-failing-issues-per-cycle.md`
- `.kb/decisions/2025-12-31-daemon-recomputes-state-each-cycle.md`

**Key Implementation Files:**
- `pkg/daemon/daemon.go` - Core daemon logic
- `pkg/daemon/pool.go` - WorkerPool with reconciliation
- `cmd/orch/daemon.go` - CLI daemon loop

---

## Investigation History

**2025-12-31 17:40:** Investigation started
- Initial question: What decisions emerged from 22 daemon investigations?
- Context: Synthesis task from orchestrator

**2025-12-31 17:55:** Chronological analysis complete
- Used `kb chronicle 'daemon'` to see temporal evolution
- Identified 4 phases: initial impl, queue bugs, capacity bugs, blocking bugs

**2025-12-31 18:10:** Decision records created
- Created 3 decision records consolidating findings
- Identified 9 investigations for archival

**2025-12-31 18:15:** Investigation completed
- Status: Complete
- Key outcome: 3 decisions extracted, 9 archives proposed, narrative documented
