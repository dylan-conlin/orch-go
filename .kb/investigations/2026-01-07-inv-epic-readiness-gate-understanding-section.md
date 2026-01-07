<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented epic readiness gate: `bd create --type epic` now requires `--understanding` flag (or `--no-understanding --reason`), and `bd ready` warns on epics missing Understanding section.

**Evidence:** Tests pass: TestCreateSuite/EpicWithUnderstanding, TestCreateSuite/EpicWithoutUnderstanding, TestHasUnderstandingSection (all 4 subtests).

**Knowledge:** Following the bug repro pattern (`--repro` or `--no-repro --reason`), we can gate epic creation on understanding model completeness while allowing opt-out with justification.

**Next:** Push to beads repo, rebuild bd binary.

**Promote to Decision:** recommend-no - This is implementing an existing decision (Strategic Orchestrator Model 2026-01-07), not creating a new architectural choice.

---

# Investigation: Epic Readiness Gate Understanding Section

**Question:** How to implement the epic readiness gate where epics require an Understanding section documenting the problem model?

**Started:** 2026-01-07
**Updated:** 2026-01-07
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Bug repro pattern provides a template

**Evidence:** `--type=bug` requires either `--repro 'steps'` or `--no-repro --reason 'why'`. Same validation pattern works for epics.

**Source:** `beads/cmd/bd/create.go:139-164` - bug reproduction validation logic

**Significance:** Reusing this pattern ensures consistency in CLI UX and maintains the principle of "require by default, allow opt-out with justification."

---

### Finding 2: Description field can hold structured content

**Evidence:** Description is a free-form string. We can prepend `## Understanding\n\n{content}\n\n---\n\n` to make it machine-detectable while human-readable.

**Source:** `beads/internal/types/types.go:21` - Description field definition

**Significance:** No schema changes needed - the Understanding section lives IN the issue description, following "Surfacing Over Browsing" principle.

---

### Finding 3: Ready command has per-issue display logic

**Evidence:** Ready work display loops through issues with optional fields (estimate, assignee). Adding epic understanding warning follows same pattern.

**Source:** `beads/cmd/bd/ready.go:222-233` (direct mode) and `ready.go:151-163` (daemon mode)

**Significance:** Warning is light-touch surfacing - appears inline with epic in ready list, doesn't block anything.

---

## Synthesis

**Key Insights:**

1. **Gate at creation, warn at display** - Creation gate ensures new epics have understanding; display warning surfaces legacy epics that need attention.

2. **Shared --reason flag** - Both `--no-repro` and `--no-understanding` use the same `--reason` flag, reducing flag proliferation.

3. **Detection via marker** - `## Understanding` string detection is simple and robust for both CLI display and potential future automation.

**Answer to Investigation Question:**

Implemented by:
1. Adding `--understanding` and `--no-understanding` flags to `bd create`
2. Requiring one of them when `--type=epic`
3. Prepending Understanding section to description with `## Understanding` marker
4. Adding `hasUnderstandingSection()` helper in ready.go
5. Displaying warning for epics without the marker

---

## Structured Uncertainty

**What's tested:**

- ✅ Epic with understanding section stores correctly (verified: TestCreateSuite/EpicWithUnderstanding)
- ✅ Epic without understanding section detected correctly (verified: TestHasUnderstandingSection all subtests)
- ✅ Non-epic types always return true from hasUnderstandingSection (verified: TestHasUnderstandingSection/NonEpicAlwaysTrue)
- ✅ Code compiles with `go build ./cmd/bd/...`

**What's untested:**

- ⚠️ CLI validation error messages displayed correctly (would need integration test)
- ⚠️ Daemon mode RPC handling (not modified, should work but not tested)
- ⚠️ Warning rendering in terminal colors (manual verification needed)

**What would change this:**

- If Understanding section format changes, marker detection would need updating
- If epics need different fields than free-form text, schema change would be needed

---

## Implementation Recommendations

**Recommended Approach ⭐**

**Gate + Warn Pattern** - Implemented as described above.

**Why this approach:**
- Consistent with existing bug repro pattern
- No schema changes required
- Light-touch warning doesn't block workflows

**Trade-offs accepted:**
- Understanding section format is convention, not enforced structure
- Legacy epics only get warnings, not blocked

---

## References

**Files Modified:**
- `beads/cmd/bd/create.go` - Added understanding flags and validation
- `beads/cmd/bd/create_test.go` - Added epic understanding tests
- `beads/cmd/bd/ready.go` - Added hasUnderstandingSection helper and warning display
- `beads/cmd/bd/ready_test.go` - Added TestHasUnderstandingSection

**Commands Run:**
```bash
# Build verification
cd ~/Documents/personal/beads && go build ./cmd/bd/...

# Test verification
go test ./cmd/bd/... -run "TestCreateSuite|TestReadySuite|TestHasUnderstandingSection" -v
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-07-strategic-orchestrator-model.md` - Defines epic readiness as model completeness

---

## Investigation History

**2026-01-07:** Investigation started
- Initial question: How to implement epic readiness gate?
- Context: Strategic Orchestrator Model defines epic readiness as understanding completeness

**2026-01-07:** Implementation complete
- Status: Complete
- Key outcome: bd create --type epic now requires --understanding or --no-understanding --reason; bd ready warns on missing Understanding section
