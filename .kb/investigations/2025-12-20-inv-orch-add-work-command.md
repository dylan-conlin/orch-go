**TLDR:** Question: Add `work` command to orch-go that infers skill from beads issue type. Answer: Implemented work command that maps bug→systematic-debugging, feature→feature-impl, task→feature-impl, investigation→investigation. High confidence (95%) - tests pass and integrates with existing spawn infrastructure.

---

# Investigation: Add work command to orch-go

**Question:** How to implement a `work` command that starts work on a beads issue with automatic skill inference from issue type?

**Started:** 2025-12-20
**Updated:** 2025-12-20
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%)

---

## Findings

### Finding 1: Beads issues have an issue_type field

**Evidence:** `bd show <id> --json` returns JSON with `issue_type` field containing values like "bug", "feature", "task", "investigation", "epic".

**Source:** `bd show orch-go-zqd --json` output

**Significance:** This field can be used to infer the appropriate skill for the agent.

---

### Finding 2: Issue type to skill mapping is straightforward

**Evidence:** Existing skills map naturally:
- bug → systematic-debugging (debugging workflow)
- feature → feature-impl (TDD feature implementation)
- task → feature-impl (similar to feature but smaller scope)
- investigation → investigation (exploration skill)
- epic → cannot be spawned (epics are decomposed into sub-issues)

**Source:** Skills directory structure at ~/.claude/skills/

**Significance:** Clear 1:1 mapping with one exception (epic).

---

### Finding 3: Existing spawn infrastructure handles complexity

**Evidence:** `runSpawnWithSkill()` already handles:
- Skill loading
- Beads issue creation/tracking
- Tmux or inline spawning
- Context file generation

**Source:** `cmd/orch/main.go:179-248`

**Significance:** Work command can reuse spawn infrastructure by setting `spawnIssue` flag and calling `runSpawnWithSkill()`.

---

## Synthesis

**Key Insights:**

1. **Issue type is reliable metadata** - Beads stores issue type as structured data, not requiring parsing from title or labels.

2. **Skill inference is deterministic** - No ambiguity in mapping issue types to skills, with clear error case for epics.

3. **Reuse over reimplementation** - Work command is thin wrapper around spawn, adding issue lookup and skill inference.

**Answer to Investigation Question:**

Implemented via:
1. `InferSkillFromIssueType()` function with explicit mapping
2. `workCmd` cobra command that fetches issue, infers skill, calls spawn
3. Tests covering all issue type cases including error paths

---

## Confidence Assessment

**Current Confidence:** Very High (95%)

**Why this level?**

All tests pass, implementation integrates cleanly with existing code, and the mapping logic is straightforward with clear semantics.

**What's certain:**

- ✅ Skill inference logic is correct and tested
- ✅ Work command integrates with existing spawn infrastructure
- ✅ Error handling for invalid types (epic, unknown, empty)

**What's uncertain:**

- ⚠️ Not tested with actual spawn (would create real agent session)

---

## Implementation Details

**Files modified:**
- `cmd/orch/main.go` - Added workCmd, InferSkillFromIssueType, runWork
- `cmd/orch/work_test.go` - Tests for skill inference
- `pkg/verify/check.go` - Added IssueType and Description fields to Issue struct

**Usage:**
```bash
orch-go work proj-123           # Start work on issue in tmux
orch-go work proj-123 --inline  # Start work inline (blocking)
```

---

## References

**Files Examined:**
- `cmd/orch/main.go` - Existing command structure and spawn implementation
- `pkg/verify/check.go` - Issue struct and beads integration

**Commands Run:**
```bash
bd show orch-go-zqd --json     # Verified issue_type field structure
go test ./...                  # All tests pass
go build ./cmd/orch           # Build successful
```

---

## Investigation History

**2025-12-20 10:14:** Investigation started
- Initial question: How to implement work command with skill inference

**2025-12-20 10:25:** Implementation complete
- Added work command with tests
- All tests passing
- Build successful
