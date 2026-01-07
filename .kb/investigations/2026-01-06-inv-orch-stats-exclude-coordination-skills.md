<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added skill category classification (task vs coordination) to `orch stats` - coordination skills now excluded from completion rate warning.

**Evidence:** Implementation verified with all tests passing; `orch stats` now shows Task Skills 68.2% vs Coordination Skills 10.3% separately.

**Knowledge:** Coordination skills (orchestrator, meta-orchestrator) are interactive sessions designed to run until context exhaustion - including them in completion rate was a category error.

**Next:** Close - implementation complete and committed.

**Promote to Decision:** recommend-no (tactical fix implementing prior investigation's recommendation)

---

# Investigation: Orch Stats Exclude Coordination Skills

**Question:** How should `orch stats` handle coordination skills in the completion rate calculation and warning?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** og-feat-orch-stats-exclude-06jan-5e4d
**Phase:** Complete
**Next Step:** None - implementation complete
**Status:** Complete

---

## Findings

### Finding 1: Prior investigation identified coordination skills as category error

**Evidence:** Investigation 2026-01-06-inv-diagnose-overall-66-completion-rate.md found meta-orchestrator (0%) and orchestrator (17.4%) drag down completion rate, but they're interactive sessions not completable tasks.

**Source:** `.kb/investigations/2026-01-06-inv-diagnose-overall-66-completion-rate.md`

**Significance:** Including coordination skills in completion rate is fundamentally incorrect - their low rate is a feature (interactive until context exhaustion), not a failure.

---

### Finding 2: Simple categorization is sufficient

**Evidence:** Only two skills need coordination classification: `orchestrator` and `meta-orchestrator`. All other skills are task-oriented and should be tracked normally.

**Source:** Analysis of skills in events.jsonl and skill definitions

**Significance:** A simple map-based categorization is appropriate - no need for complex interface patterns or extensive configuration.

---

### Finding 3: Implementation verified with real data

**Evidence:** After implementation, `orch stats` shows:
- Task Skills: 236/346 spawns (68.2%)
- Coordination Skills: 4/39 spawns (10.3%)
- Warning correctly triggers on task skill rate

**Source:** Running `go run ./cmd/orch stats` with production events.jsonl

**Significance:** The fix correctly separates the metrics and the warning threshold now applies only to task work.

---

## Synthesis

**Key Insights:**

1. **Category separation fixes the metric** - By tracking task skills separately, the 80% threshold becomes meaningful again (task skills are at 68%, which correctly triggers the warning).

2. **Simple solution works** - A map of coordination skills with Category field on SkillStatsSummary provides clean separation without over-engineering.

3. **Backwards compatible** - Overall CompletionRate is preserved in output for any consumers, while new fields provide the refined metrics.

**Answer to Investigation Question:**

Coordination skills should be:
1. Classified with a Category field (task vs coordination)
2. Tracked separately from task skills in the summary
3. Excluded from the completion rate warning threshold check
4. Visually marked in the skill breakdown table with (C) indicator

---

## Structured Uncertainty

**What's tested:**

- ✅ Skill category classification works (verified: TestGetSkillCategory passes)
- ✅ Separate rate calculations work (verified: TestAggregateStatsCategoryBreakdown passes)
- ✅ Warning uses TaskCompletionRate (verified: TestAggregateStatsCoordinationExcludedFromOverallRate passes)
- ✅ Real data produces expected output (verified: ran orch stats with production events)

**What's untested:**

- ⚠️ Whether 68% task rate is "normal" or indicates real issues (needs historical comparison)
- ⚠️ Whether other skills should be categorized differently (only looked at orchestrator/meta-orchestrator)

**What would change this:**

- If new skills are added that are coordination-oriented (would need to update coordinationSkills map)
- If orchestrator sessions become completable (would reclassify them as task skills)

---

## Implementation Recommendations

Implementation is complete. See SYNTHESIS.md for full details.

---

## References

**Files Examined:**
- `cmd/orch/stats_cmd.go` - Implementation target
- `cmd/orch/stats_test.go` - Test file
- `.kb/investigations/2026-01-06-inv-diagnose-overall-66-completion-rate.md` - Prior investigation

**Commands Run:**
```bash
# Run tests
go test -v ./cmd/orch/... -run "TestAggregateStats|TestGetSkillCategory"

# Build verification
go build -o /dev/null ./cmd/orch

# Test with real data
go run ./cmd/orch stats
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-06-inv-diagnose-overall-66-completion-rate.md` - Original investigation that identified the issue
- **Workspace:** `.orch/workspace/og-feat-orch-stats-exclude-06jan-5e4d/` - Implementation workspace

---

## Investigation History

**2026-01-06 18:52:** Investigation started
- Initial question: How to exclude coordination skills from completion rate warning
- Context: Follow-up from diagnose-overall-66-completion-rate investigation

**2026-01-06 18:56:** Investigation completed
- Status: Complete
- Key outcome: Added skill categories, separate rate calculations, and fixed warning threshold
