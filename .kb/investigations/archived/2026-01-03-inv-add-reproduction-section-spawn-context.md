## Summary (D.E.K.N.)

**Delta:** Added Reproduction section to SPAWN_CONTEXT.md template that is conditionally included for bug issues, providing repro steps and verification requirements.

**Evidence:** All tests pass including new tests for bug reproduction section generation (TestGenerateContext_BugReproduction).

**Knowledge:** The existing pkg/verify/repro.go already has ExtractReproFromIssue and GetReproForCompletion functions that extract repro steps from beads issues.

**Next:** Close - implementation complete with tests passing.

---

# Investigation: Add Reproduction Section Spawn Context

**Question:** How to add reproduction steps to SPAWN_CONTEXT.md for bug issues so agents know how to verify their fix?

**Started:** 2026-01-03
**Updated:** 2026-01-03
**Owner:** agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Existing repro extraction infrastructure

**Evidence:** pkg/verify/repro.go already contains:
- `ExtractReproFromIssue()` - parses reproduction steps from issue descriptions
- `GetReproForCompletion()` - returns repro info for beads issue completion
- Pattern matching for various repro formats (markdown sections, bold markers, code blocks)

**Source:** pkg/verify/repro.go:40-113

**Significance:** We can reuse existing extraction logic rather than implementing new parsing.

---

### Finding 2: Config and template data flow

**Evidence:** spawn.Config flows to contextData struct which populates SpawnContextTemplate:
- Config fields are defined in pkg/spawn/config.go
- contextData mirrors fields for template execution in pkg/spawn/context.go
- Template is Go text/template with conditionals

**Source:** pkg/spawn/config.go:66-123, pkg/spawn/context.go:363-380

**Significance:** Adding new fields requires updates to both Config and contextData, plus template.

---

### Finding 3: spawn_cmd.go fetches issue metadata

**Evidence:** spawn_cmd.go already calls verify.GetIssue() to check issue status before spawning.
Similar call can be made to GetReproForCompletion() to get repro info.

**Source:** cmd/orch/spawn_cmd.go:590-618

**Significance:** Integration point exists - just need to add the call and pass data to Config.

---

## Implementation

### Changes Made

1. **pkg/spawn/config.go** - Added `IsBug` and `ReproSteps` fields to Config struct
2. **pkg/spawn/context.go** - Added fields to contextData, updated GenerateContext to pass them
3. **pkg/spawn/context.go** - Added REPRODUCTION section to SpawnContextTemplate (conditionally included when IsBug=true)
4. **cmd/orch/spawn_cmd.go** - Added call to verify.GetReproForCompletion() to fetch repro info during spawn
5. **pkg/spawn/context_test.go** - Added TestGenerateContext_BugReproduction test cases

### Template Section Added

When spawning for a bug issue, SPAWN_CONTEXT.md now includes:

```markdown
## REPRODUCTION (BUG FIX)

🐛 **This is a bug fix issue.** The fix is verified when the reproduction steps no longer produce the bug.

**Original Reproduction:**
[repro steps from issue]

**Verification Requirement:**
Before marking Phase: Complete, you MUST:
1. Attempt to reproduce the original bug using the steps above
2. Confirm the bug NO LONGER reproduces after your fix
3. Report verification via: `bd comment <beads-id> "Reproduction verified: [describe test performed]"`

⚠️ A bug fix is only complete when the original reproduction steps pass.
```

---

## Structured Uncertainty

**What's tested:**

- ✅ Bug issues include REPRODUCTION section (verified: TestGenerateContext_BugReproduction/includes_reproduction_section_for_bug_issues)
- ✅ Non-bug issues exclude REPRODUCTION section (verified: TestGenerateContext_BugReproduction/excludes_reproduction_section_for_non-bug_issues)
- ✅ Empty repro steps still shows section header (verified: TestGenerateContext_BugReproduction/handles_empty_repro_steps_for_bug_issue)
- ✅ All spawn package tests pass (verified: go test ./pkg/spawn/...)
- ✅ Code compiles (verified: go build ./...)

**What's untested:**

- ⚠️ End-to-end spawn with real bug issue (not tested - requires live beads issue)
- ⚠️ Agent behavior when reading reproduction section (not tested - agent interpretation)

---

## References

**Files Modified:**
- pkg/spawn/config.go:122-128 - Added IsBug and ReproSteps fields
- pkg/spawn/context.go:363-382 - Added fields to contextData
- pkg/spawn/context.go:405-420 - Updated GenerateContext
- pkg/spawn/context.go:39-55 - Added REPRODUCTION section to template
- cmd/orch/spawn_cmd.go:687-697 - Added repro extraction call
- pkg/spawn/context_test.go:767-860 - Added reproduction tests

**Commands Run:**
```bash
# Run reproduction tests
go test ./pkg/spawn/... -v -run "BugReproduction"

# Run all spawn tests
go test ./pkg/spawn/...

# Build verification
go build ./...
```

---

## Investigation History

**2026-01-03 14:00:** Investigation started
- Initial question: How to add reproduction steps to SPAWN_CONTEXT.md?
- Context: Issue orch-go-jjd4.5 requesting Reproduction section for bug issues

**2026-01-03 14:15:** Implementation completed
- Status: Complete
- Key outcome: Added REPRODUCTION section to template, integrated with existing repro extraction
