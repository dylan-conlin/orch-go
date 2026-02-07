<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented `orch session validate` command that shows unfilled handoff sections without ending the session.

**Evidence:** Command builds successfully, produces expected human-readable and JSON output, all existing tests pass.

**Knowledge:** The existing `validateHandoff()` function provides all needed validation logic; new command primarily needed window name resolution and output formatting.

**Next:** Close - command is complete and tested.

**Promote to Decision:** recommend-no - Tactical feature addition, not architectural.

---

# Investigation: Orch Session Validate Standalone Command

**Question:** How to add a standalone `orch session validate` command that shows unfilled handoff sections without ending the session?

**Started:** 2026-01-14
**Updated:** 2026-01-14
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: validateHandoff() already provides all validation logic

**Evidence:** The existing `validateHandoff(activeDir string)` function at `cmd/orch/session.go:305-328` reads SESSION_HANDOFF.md and returns a `ValidationResult` containing unfilled sections.

**Source:** `cmd/orch/session.go:305-328`

**Significance:** No new validation logic needed - the new command only needs to expose existing logic without prompting or archiving.

---

### Finding 2: Window name resolution needs both session and fallback paths

**Evidence:** The validate command needs to find the active handoff directory, which depends on window name. Active sessions store the window name (`sess.WindowName`), but if no session is active, we fall back to current tmux window.

**Source:** `cmd/orch/session.go:1548-1562` (new `getWindowNameForValidation()` function)

**Significance:** This dual-path approach matches the pattern used by `session end` (lines 836-844) and ensures the command works with or without an active session.

---

### Finding 3: JSON output needed for scripting use cases

**Evidence:** Task description mentioned debugging use case. Added `--json` flag with structured `ValidationOutput` type that includes:
- `found`: whether active handoff exists
- `unfilled_count`: total unfilled sections
- `required_filled/required_total`: required section status
- `optional_filled/optional_total`: optional section status
- `unfilled_details`: array with section names, placeholders, and prompts

**Source:** `cmd/orch/session.go:1369-1389` (new types)

**Significance:** Enables programmatic checks like `orch session validate --json | jq '.unfilled_count == 0'` for CI/automation.

---

## Synthesis

**Key Insights:**

1. **Reuse over reimplementation** - The existing validation infrastructure (`handoffSections`, `validateHandoff()`, `ValidationResult`) provided everything needed. New command is primarily about exposure and formatting.

2. **Consistent patterns** - Window name resolution, JSON output formatting, and command structure follow existing patterns in session.go (status, resume, end commands).

3. **Clear separation** - The validate command explicitly does NOT prompt or archive, unlike `session end`. This clean separation enables mid-session quality checks.

**Answer to Investigation Question:**

The command was implemented by:
1. Adding `sessionValidateCmd` Cobra command with `--json` flag
2. Creating `runSessionValidate()` that resolves window name, finds active handoff, and calls existing `validateHandoff()`
3. Adding `ValidationOutput` and `ValidationSectionInfo` types for JSON output
4. Formatting human-readable output with required/optional grouping and next-action guidance

---

## Structured Uncertainty

**What's tested:**

- ✅ Command builds successfully (verified: `make build`)
- ✅ Human-readable output shows expected sections (verified: ran command with active handoff)
- ✅ JSON output parses correctly (verified: `./build/orch session validate --json | jq .`)
- ✅ Existing tests pass (verified: `go test ./cmd/orch/...`)

**What's untested:**

- ⚠️ Behavior when no active session AND not in tmux (would need manual verification)
- ⚠️ Behavior with partially filled handoff (handoff would need manual editing to test)

**What would change this:**

- Finding would be wrong if validateHandoff() changes to require additional context
- Finding would be wrong if window name resolution doesn't work in non-tmux environments

---

## Implementation Recommendations

**Purpose:** Document the implementation approach taken.

### Recommended Approach ⭐

**Direct reuse of validateHandoff()** - Call existing validation function and format output.

**Why this approach:**
- Minimal code duplication
- Consistent validation behavior with `session end`
- Straightforward implementation (~200 lines)

**Trade-offs accepted:**
- No additional validation scenarios (only what validateHandoff() checks)
- No editing capability (read-only by design)

**Implementation sequence:**
1. Add flag variable and command registration
2. Implement runSessionValidate() with window resolution
3. Add JSON output types and formatting

---

## References

**Files Examined:**
- `cmd/orch/session.go` - Main implementation file, contains validateHandoff() and related types

**Commands Run:**
```bash
# Build
make build

# Test help
./build/orch session validate --help

# Test execution
./build/orch session validate
./build/orch session validate --json | jq .

# Unit tests
go test ./cmd/orch/...
```

---

## Investigation History

**2026-01-14 22:20:** Investigation started
- Initial question: How to add orch session validate command
- Context: Part of progressive capture decision 2026-01-14

**2026-01-14 22:30:** Investigation completed
- Status: Complete
- Key outcome: Command implemented with human-readable and JSON output, reusing existing validateHandoff() logic
