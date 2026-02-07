# Session Synthesis

**Agent:** og-feat-orch-status-surface-07jan-1737
**Issue:** orch-go-9u685
**Duration:** ~35 minutes
**Outcome:** success

---

## TLDR

Added synthesis opportunity surfacing to `orch status` - when 3+ investigations exist on a topic without a corresponding Guide or Decision, the status output now displays a "SYNTHESIS OPPORTUNITIES" section to alert orchestrators about knowledge that could benefit from consolidation.

---

## Delta (What Changed)

### Files Created
- `pkg/verify/synthesis_opportunities.go` - Core detection logic: scans .kb/investigations/, extracts topics from filenames, groups by keywords, checks for existing guides/decisions
- `pkg/verify/synthesis_opportunities_test.go` - 8 comprehensive tests covering edge cases (empty dirs, below threshold, existing guides, multiple topics, various investigation types, simple subdirectory)

### Files Modified
- `cmd/orch/status_cmd.go` - Added SynthesisOpportunities field to StatusOutput, calls DetectSynthesisOpportunities(), added printSynthesisOpportunities() function

### Commits
- `1f77a843` - feat(status): add synthesis opportunity surfacing

---

## Evidence (What Was Observed)

- Investigation filenames follow pattern: `YYYY-MM-DD-{type}-{topic}.md` where type is inv-, design-, audit-, debug-, research-, reliability-
- Project has 600+ investigations in `.kb/investigations/`
- Existing guides (18) and decisions (13) were checked to avoid false positives
- Real-world output shows actionable opportunities like "30 investigations on 'investigation' without synthesis"

### Tests Run
```bash
go test ./pkg/verify/... -run SynthesisOpportunities -v
# PASS: all 8 tests passing

go test ./cmd/orch/... -run Status -v  
# PASS: all status tests passing

go test ./...
# PASS: all project tests passing
```

### Validation
```bash
go run ./cmd/orch status
# Output includes:
# SYNTHESIS OPPORTUNITIES
#   30 investigations on 'investigation' without synthesis
#   22 investigations on 'complete' without synthesis
#   17 investigations on 'context' without synthesis
#   ...
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-07-inv-orch-status-surface-synthesis-opportunities.md` - Investigation file (created but not filled beyond template)

### Decisions Made
- Decision 1: Used topic keywords as the grouping mechanism (not file prefixes) because topics are domain-meaningful
- Decision 2: Set threshold at 3 investigations (from Coherence Over Patches principle)
- Decision 3: Check both guides AND decisions as synthesis indicators (either counts as synthesized)
- Decision 4: Extract topics from investigation filenames rather than parsing file contents (simpler, faster)

### Constraints Discovered
- Guide filenames don't follow date prefix pattern (just `topic.md`), so topic extraction differs from investigations
- Decision filenames do have date prefix, must strip it to extract topic keywords
- Simple subdirectory investigations (`simple/`) need separate handling

### Externalized via `kn`
- N/A - straightforward implementation, no novel constraints discovered

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing
- [x] SYNTHESIS.md created
- [x] Ready for `orch complete orch-go-9u685`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should there be a flag to suppress synthesis opportunities (e.g., `--no-synthesis` for CI pipelines)?
- Should the synthesis opportunities be included in the JSON output even when empty (currently omitted via `omitempty`)?
- Could the topic extraction be smarter (e.g., NLP-based topic modeling instead of keyword matching)?

**Areas worth exploring further:**
- Adding synthesis opportunity detection to the dashboard UI
- Creating an `orch synthesize <topic>` command that opens all investigations for a topic

**What remains unclear:**
- Optimal threshold value (3 was chosen from principle, but may need tuning based on usage)
- Whether some topics should be excluded from synthesis tracking (e.g., "investigation" itself had 30 matches - is that actionable?)

---

## Session Metadata

**Skill:** feature-impl
**Model:** Opus
**Workspace:** `.orch/workspace/og-feat-orch-status-surface-07jan-1737/`
**Investigation:** `.kb/investigations/2026-01-07-inv-orch-status-surface-synthesis-opportunities.md`
**Beads:** `bd show orch-go-9u685`
