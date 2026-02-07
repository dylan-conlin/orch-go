<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Two different skill inference functions had conflicting mappings: daemon used `systematic-debugging` for bugs, but `orch work` used `architect` for bugs.

**Evidence:** `pkg/daemon/skill_inference.go:31` maps bug→`systematic-debugging`; `cmd/orch/spawn_skill_inference.go:18` mapped bug→`architect`. Daemon prints one skill, but workspace name (and frontier display) shows another.

**Knowledge:** Skill inference must be consistent across the entire spawn pipeline. The decision 2026-01-23 established direct action for bugs (systematic-debugging), but one file wasn't updated.

**Next:** Fixed by aligning `InferSkillFromIssueType()` to match daemon's `InferSkill()`. Both now map bug→systematic-debugging.

**Promote to Decision:** recommend-no (this is a bug fix aligning with existing decision 2026-01-23)

---

# Investigation: Orch Frontier Disagrees Orch Status

**Question:** Why does `orch frontier` show a different skill than what `orch daemon` spawned?

**Started:** 2026-01-29
**Updated:** 2026-01-29
**Owner:** architect worker
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Two Separate Skill Inference Functions with Conflicting Mappings

**Evidence:**

In `pkg/daemon/skill_inference.go:26-31`:
```go
func InferSkill(issueType string) (string, error) {
    switch issueType {
    case "bug":
        return "systematic-debugging", nil  // ← daemon uses this
```

In `cmd/orch/spawn_skill_inference.go:11-18` (BEFORE fix):
```go
func InferSkillFromIssueType(issueType string) (string, error) {
    switch issueType {
    case "bug":
        return "architect", nil  // ← orch work used this
```

**Source:**
- `pkg/daemon/skill_inference.go:26-31`
- `cmd/orch/spawn_skill_inference.go:11-18`

**Significance:** The daemon uses `InferSkillFromIssue()` (which calls `InferSkill()`) to print what skill is being spawned. But `orch work` uses `InferSkillFromIssueType()` to actually spawn. These had different mappings for `bug` type, causing the discrepancy.

---

### Finding 2: Workspace Name Reflects Actual Skill Used

**Evidence:**

Workspace names are generated with skill prefixes in `pkg/spawn/config.go:286-302`:
```go
prefixes := map[string]string{
    "systematic-debugging": "debug",
    "architect":            "arch",
    ...
}
```

When daemon says `(systematic-debugging)` but `orch work` actually spawns with `architect`, the workspace name becomes `og-arch-*` not `og-debug-*`.

**Source:** `pkg/spawn/config.go:286-302`

**Significance:** The workspace name is the "ground truth" for what skill was actually used. Frontier extracts skill from workspace name, so it shows the actual skill, not what daemon thought it spawned.

---

### Finding 3: Decision 2026-01-23 Establishes Correct Mapping

**Evidence:**

The decision `.kb/decisions/2026-01-23-investigation-overhead-firefighting-mode.md` establishes:
> "Bug blocking me right now: Commit message only, Knowledge is cheap, speed matters"

This supports direct action (systematic-debugging) over investigative approach (architect) for bugs.

The daemon's `skill_inference.go` correctly references this decision, but `spawn_skill_inference.go` was never updated.

**Source:** `.kb/decisions/2026-01-23-investigation-overhead-firefighting-mode.md`

**Significance:** The daemon's mapping was correct. The spawn command's mapping was stale and needed to be aligned.

---

## Synthesis

**Key Insights:**

1. **Two separate code paths** - Daemon skill inference (`pkg/daemon/`) and spawn skill inference (`cmd/orch/`) evolved independently without staying in sync.

2. **Workspace name is truth** - The actual skill used is reflected in workspace name prefix. Any display (frontier, status) that extracts skill from workspace will show the truth, while components using their own inference may show stale/incorrect values.

3. **Decision drift** - When decision 2026-01-23 was made, only the daemon's inference was updated. The spawn command still had the old mapping.

**Answer to Investigation Question:**

The discrepancy between `orch frontier` and `orch daemon` skill display was caused by two different skill inference functions with conflicting mappings for `bug` type. The daemon's `InferSkill()` mapped bug→systematic-debugging, while `InferSkillFromIssueType()` (used by `orch work`) mapped bug→architect. Since workspace names are generated based on the actual spawned skill, frontier (which extracts skill from workspace name) showed the actual skill used, while daemon printed its own (incorrect) inference.

---

## Structured Uncertainty

**What's tested:**

- ✅ Both inference functions now return `systematic-debugging` for bug type (verified: code inspection)
- ✅ Build passes after fix (verified: `go build ./...`)
- ✅ All skill-related tests pass (verified: `go test ./cmd/orch/... ./pkg/daemon/... -run "Skill|Infer"`)

**What's untested:**

- ⚠️ End-to-end reproduction with actual daemon spawn (not tested - would require running daemon)
- ⚠️ Active agent count discrepancy (separate issue identified but not addressed)

**What would change this:**

- Finding would be wrong if there's another code path that infers skill differently
- Fix would be incomplete if there are other skill inference functions in codebase

---

## Implementation Recommendations

**Purpose:** The fix has been implemented. This section documents what was changed.

### Implemented Fix ⭐

**Aligned `InferSkillFromIssueType()` with daemon's `InferSkill()`**

**Changes made:**
- `cmd/orch/spawn_skill_inference.go:11-18`: Changed `bug` mapping from `architect` to `systematic-debugging`
- `cmd/orch/spawn_cmd.go:218`: Updated help text to reflect new mapping
- Added reference to decision 2026-01-23 in comment

**Trade-offs accepted:**
- None - this aligns with the established decision

### Alternative Issue: Active Agent Count Discrepancy (NOT FIXED)

**Evidence:** Different time windows for "active" agents:
- `frontier.go:163` uses `maxAge = 3 * time.Hour`
- `status_cmd.go:195` uses `maxIdleTime = 30 * time.Minute`

**Recommendation:** This is a separate issue and should be tracked separately. The skill mismatch was the primary bug reported.

---

## References

**Files Examined:**
- `cmd/orch/frontier.go` - Skill extraction from workspace name
- `cmd/orch/status_cmd.go` - Agent discovery and time windows
- `cmd/orch/spawn_cmd.go` - Work command definition
- `cmd/orch/work_cmd.go` - Work command implementation
- `cmd/orch/spawn_skill_inference.go` - Skill inference for spawn (MODIFIED)
- `pkg/daemon/daemon.go` - Daemon spawn flow
- `pkg/daemon/skill_inference.go` - Daemon skill inference
- `pkg/spawn/config.go` - Workspace name generation

**Commands Run:**
```bash
# Build verification
go build ./...

# Test verification
go test ./cmd/orch/... -run "Skill" -v
go test ./pkg/daemon/... -run "Skill|Infer" -v
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-23-investigation-overhead-firefighting-mode.md` - Establishes bug→systematic-debugging mapping

---

## Investigation History

**2026-01-29 19:35:** Investigation started
- Initial question: Why does orch frontier show different skill than daemon spawned?
- Context: Reproduction showed daemon prints `systematic-debugging` but frontier shows `architect`

**2026-01-29 19:45:** Root cause identified
- Found two separate skill inference functions with conflicting mappings
- Daemon uses `pkg/daemon/skill_inference.go`, spawn uses `cmd/orch/spawn_skill_inference.go`

**2026-01-29 19:50:** Fix implemented
- Aligned `InferSkillFromIssueType()` to match daemon's `InferSkill()`
- Updated help text to reflect change

**2026-01-29 19:55:** Investigation completed
- Status: Complete
- Key outcome: Fixed skill inference inconsistency by aligning spawn command with daemon
