<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** BuildSpawnCommand now passes --model flag to opencode CLI when model is provided, fixing inline spawns to respect user's --model choice.

**Evidence:** Tests pass for both with-model and without-model cases; implementation follows pattern from BuildOpencodeAttachCommand.

**Knowledge:** Inline spawns (via BuildSpawnCommand) were silently ignoring cfg.Model parameter because it wasn't passed to opencode CLI.

**Next:** None - fix is complete and tested.

**Confidence:** High (95%) - Simple fix, well-tested, follows existing patterns.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Confidence: High (85%) - small sample size (5 sessions).

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Fix BuildSpawnCommand to Pass Model Flag

**Question:** How do we fix BuildSpawnCommand to pass the --model flag to opencode CLI so inline spawns respect user's model choice?

**Started:** 2025-12-21
**Updated:** 2025-12-21
**Owner:** Agent og-feat-fix-buildspawncommand-pass-21dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (95%)

---

## Findings

### Finding 1: BuildSpawnCommand signature needed model parameter

**Evidence:** Function signature was `BuildSpawnCommand(prompt, title string)` with no model parameter, even though spawn.Config has Model field.

**Source:** pkg/opencode/client.go:128

**Significance:** Without model parameter, there was no way to pass user's model choice to opencode CLI.

---

### Finding 2: Caller had cfg.Model available but couldn't pass it

**Evidence:** runSpawnInline function in cmd/orch/main.go:765 had access to `cfg.Model` but BuildSpawnCommand didn't accept it.

**Source:** cmd/orch/main.go:765

**Significance:** The model was being resolved correctly in orch-go but lost when spawning opencode.

---

### Finding 3: Pattern exists in BuildOpencodeAttachCommand

**Evidence:** BuildOpencodeAttachCommand in pkg/tmux/tmux.go:99-100 shows correct pattern: conditionally add --model flag only when model is not empty.

**Source:** Investigation file .kb/investigations/2025-12-21-inv-model-handling-conflicts-between-orch.md (Finding 4)

**Significance:** Provided clear implementation pattern to follow - consistency across spawn modes.

---

## Synthesis

**Key Insights:**

1. **Simple fix with clear pattern** - The fix required adding a model parameter to BuildSpawnCommand signature and conditionally appending --model flag when model is not empty (Finding 3 pattern).

2. **TDD workflow validated approach** - Writing failing tests first (RED) revealed the exact change needed, then implementing to make tests pass (GREEN) confirmed correctness.

3. **Consistency across spawn modes** - By following BuildOpencodeAttachCommand pattern, inline spawns now handle --model flag the same way as tmux spawns (consistency).

**Answer to Investigation Question:**

Fixed BuildSpawnCommand by: (1) Adding model parameter to function signature, (2) Conditionally appending --model flag when model is not empty, (3) Updating caller to pass cfg.Model. Tests verify both with-model and without-model cases. Implementation follows existing pattern from BuildOpencodeAttachCommand for consistency.

---

## Confidence Assessment

**Current Confidence:** High (95%)

**Why this level?**

Simple fix with comprehensive tests covering both cases (with and without model). Implementation follows established pattern from BuildOpencodeAttachCommand. All related tests pass.

**What's certain:**

- ✅ BuildSpawnCommand now accepts model parameter and passes --model flag when provided (verified via tests)
- ✅ Tests cover both cases: with model (flag included) and without model (flag omitted)
- ✅ Pattern matches BuildOpencodeAttachCommand for consistency (verified via code review)

**What's uncertain:**

- ⚠️ Runtime behavior not tested (only unit tests, no actual spawn execution verification)
- ⚠️ TestFindRecentSession failure exists but is pre-existing (verified via git checkout HEAD~2)

**What would increase confidence to Very High (95%+):**

- Runtime test: `orch spawn --inline investigation "test" --model sonnet` and verify Sonnet is used
- Integration test: Verify opencode CLI actually receives and uses the --model flag

**Confidence levels guide:**

- **Very High (95%+):** Strong evidence, minimal uncertainty, unlikely to change
- **High (80-94%):** Solid evidence, minor uncertainties, confident to act
- **Medium (60-79%):** Reasonable evidence, notable gaps, validate before major commitment
- **Low (40-59%):** Limited evidence, high uncertainty, proceed with caution
- **Very Low (<40%):** Highly speculative, more investigation needed

---

## Test Performed

**Test 1: Write failing test for --model flag (RED)**

```bash
# Added TestBuildSpawnCommandWithModel and TestBuildSpawnCommandWithoutModel
go test ./pkg/opencode -v -run TestBuildSpawnCommand
```

**Result:** Tests failed with "too many arguments" - confirmed function signature needed model parameter.

**Test 2: Implement fix and verify tests pass (GREEN)**

```bash
# Modified BuildSpawnCommand to accept model param and add --model flag conditionally
# Updated caller to pass cfg.Model
go test ./pkg/opencode -v -run TestBuildSpawnCommand
```

**Result:** All 3 tests pass (TestBuildSpawnCommand, TestBuildSpawnCommandWithModel, TestBuildSpawnCommandWithoutModel).

**Test 3: Regression check - verify no other tests broke**

```bash
go test ./... 2>&1 | tail -20
```

**Result:** No new failures. TestFindRecentSession fails but verified as pre-existing via `git checkout HEAD~2`.

---

## References

**Files Examined:**

- pkg/opencode/client.go:127-137 - BuildSpawnCommand function that needed modification
- pkg/opencode/client_test.go:142-210 - Tests for BuildSpawnCommand
- cmd/orch/main.go:765 - Caller that needed to pass cfg.Model
- pkg/spawn/config.go:40 - Verified Model field exists in spawn.Config

**Commands Run:**

```bash
# Verify tests fail before fix (RED)
go test ./pkg/opencode -v -run TestBuildSpawnCommand

# Verify tests pass after fix (GREEN)
go test ./pkg/opencode -v -run TestBuildSpawnCommand

# Regression check
go test ./...

# Verify pre-existing test failure
git checkout HEAD~2 && go test ./pkg/opencode -v -run TestFindRecentSession
```

**Related Artifacts:**

- **Investigation:** .kb/investigations/2025-12-21-inv-model-handling-conflicts-between-orch.md - Root cause investigation that identified this bug

---

## Investigation History

**2025-12-21 10:00:** Investigation started

- Initial question: How do we fix BuildSpawnCommand to pass --model flag?
- Context: Following TDD approach to fix bug identified in investigation .kb/investigations/2025-12-21-inv-model-handling-conflicts-between-orch.md

**2025-12-21 10:15:** TDD RED phase - Failing tests written

- Added TestBuildSpawnCommandWithModel and TestBuildSpawnCommandWithoutModel
- Tests fail as expected (function doesn't accept model param yet)

**2025-12-21 10:25:** TDD GREEN phase - Implementation complete

- Modified BuildSpawnCommand signature to accept model parameter
- Added conditional --model flag logic
- Updated caller to pass cfg.Model
- All tests pass

**2025-12-21 10:30:** Investigation completed

- Final confidence: High (95%)
- Status: Complete
- Key outcome: BuildSpawnCommand now passes --model flag to opencode CLI, inline spawns respect user's model choice
