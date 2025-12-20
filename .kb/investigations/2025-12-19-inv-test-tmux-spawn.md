**TLDR:** Question: Does tmux spawning correctly exclude --format json flag? Answer: Yes, tmux.BuildSpawnCommand excludes --format json and runSpawnInTmux uses it correctly. However, no integration test exists for the full tmux spawn flow. Medium confidence (65%) - unit tests pass but integration gap remains.

<!--
Example TLDR:
"Question: Why aren't worker agents running tests? Answer: Agents follow documentation literally but test-running guidance isn't in spawn prompts or CLAUDE.md, only buried in separate docs. High confidence (85%) - validated across 5 agent sessions but small sample size."

Guidelines:
- Keep to 2-3 sentences maximum
- Answer: What question? What's the answer? How confident?
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Test tmux spawn - command flag consistency

**Question:** Does the tmux spawning feature correctly exclude the --format json flag when spawning agents in tmux windows, and does the actual spawn command use the correct builder?

**Started:** 2025-12-19
**Updated:** 2025-12-19
**Owner:** worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Medium

---

## Findings

### Finding 1: Tmux spawn correctly excludes --format json flag

**Evidence:** tmux.BuildSpawnCommand does not include --format json in its arguments, while opencode client's BuildSpawnCommand does. The runSpawnInTmux function uses tmux.BuildSpawnCommand, ensuring TUI display. Existing unit test TestBuildSpawnCommand validates this behavior.

**Source:** pkg/tmux/tmux.go:75-88, cmd/orch/main.go:262-269, pkg/tmux/tmux_test.go:75-104

**Significance:** Ensures tmux windows show human-readable TUI instead of JSON output, matching design expectations. This confirms the tmux spawning feature uses the correct command builder.

---

### Finding 2: No integration test for full tmux spawn flow

**Evidence:** Existing tests only cover unit-level functionality (tmux command building, session management). There is no integration test that verifies the full spawn flow: creating tmux window, sending command, and logging event. The runSpawnInTmux function is not directly tested.

**Source:** pkg/tmux/tmux_test.go, cmd/orch/main.go (no test file in cmd/orch), go test ./... output showing no failures.

**Significance:** Lack of integration test increases risk of regression for the tmux spawning feature. A broken change could go undetected until manual testing.

---

## Synthesis

**Key Insights:**

1. **Tmux spawn command consistency** - The tmux spawning feature correctly uses tmux.BuildSpawnCommand which excludes --format json, ensuring TUI display in tmux windows. This matches the design expectation.

2. **Integration test gap** - While unit tests exist for tmux components, there is no integration test for the full tmux spawn flow, increasing regression risk.

3. **Test coverage adequacy** - Existing unit tests pass, indicating basic functionality works, but end-to-end behavior remains untested automatically.

**Answer to Investigation Question:**

Yes, the tmux spawning feature correctly excludes the --format json flag when spawning agents in tmux windows. The actual spawn command uses tmux.BuildSpawnCommand which deliberately omits --format json, while inline spawn uses opencode client's BuildSpawnCommand which includes it. This is verified by code inspection and existing unit tests.

However, there is no integration test for the full tmux spawn flow, leaving a gap in automated validation. The current unit tests provide confidence in component behavior but not in end-to-end functionality.

---

## Confidence Assessment

**Current Confidence:** Medium (65%)

**Why this level?**

Confidence is based on code inspection and passing unit tests, but limited by absence of integration test for full tmux spawn flow. The core behavior (exclusion of --format json) is verified by unit test and code review. However, end-to-end functionality (tmux window creation, command sending, event logging) is not automatically tested.

**What's certain:**

- ✅ tmux.BuildSpawnCommand excludes --format json (verified by unit test)
- ✅ runSpawnInTmux uses tmux.BuildSpawnCommand (code inspection)
- ✅ Inline spawn uses opencode client's BuildSpawnCommand which includes --format json (code inspection)

**What's uncertain:**

- ⚠️ Full integration flow (tmux session/window creation, command execution) works correctly in real environment
- ⚠️ Edge cases (tmux not available, command failures) are handled appropriately
- ⚠️ Event logging and spawn summary output correctness

**What would increase confidence to High (80%+):**

- Add integration test mocking tmux and opencode to verify full spawn flow
- Test fallback behavior when tmux is not available
- Validate event logging data matches expected structure

**Confidence levels guide:**
- **Very High (95%+):** Strong evidence, minimal uncertainty, unlikely to change
- **High (80-94%):** Solid evidence, minor uncertainties, confident to act
- **Medium (60-79%):** Reasonable evidence, notable gaps, validate before major commitment
- **Low (40-59%):** Limited evidence, high uncertainty, proceed with caution
- **Very Low (<40%):** Highly speculative, more investigation needed

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Add integration test for tmux spawn flow** - Create a test that mocks tmux and opencode dependencies to verify the full tmux spawn process works correctly.

**Why this approach:**
- Directly addresses the integration test gap identified in Finding 2
- Provides automated regression detection for tmux spawning feature
- Increases confidence in end-to-end functionality without requiring manual testing

**Trade-offs accepted:**
- Additional test code maintenance overhead
- Mock complexity for tmux and opencode interactions
- Not testing actual tmux/openCode integration (requires real environment)

**Implementation sequence:**
1. Create test helpers for mocking tmux package functions (e.g., EnsureWorkersSession, CreateWindow, SendKeysLiteral)
2. Write integration test in cmd/orch package that uses mocks to verify command string excludes --format json and event logging occurs
3. Extend test to cover edge cases (tmux not available, command failures)

### Alternative Approaches Considered

**Option B: End-to-end test with real tmux and opencode**
- **Pros:** Highest fidelity, tests actual integration
- **Cons:** Heavy, requires tmux and opencode server, flaky, hard to run in CI
- **When to use instead:** When fidelity is critical and environment is controlled (e.g., manual validation)

**Option C: Rely solely on existing unit tests**
- **Pros:** No additional test code, maintains current coverage
- **Cons:** Leaves integration gap, risk of regression undetected
- **When to use instead:** If tmux spawning is considered low-risk or rarely changed

**Rationale for recommendation:** Option A strikes the right balance between increased confidence and practical maintainability. Mock-based integration test provides automated validation without heavy dependencies, addressing the identified gap while being feasible to implement and run in CI.

---

### Implementation Details

**What to implement first:**
- Create mock implementations for tmux package functions (EnsureWorkersSession, CreateWindow, SendKeysLiteral, SendEnter) that record calls
- Write a test in cmd/orch_test.go that uses these mocks and verifies command string excludes --format json
- Ensure test passes with current implementation (validation)

**Things to watch out for:**
- ⚠️ Mocking must not break existing unit tests (use build tags or test helpers)
- ⚠️ Ensure mocked functions match real signatures and behavior
- ⚠️ Test should be skipped when tmux not available (use build tags or skip logic)

**Areas needing further investigation:**
- How to best mock tmux package (interface vs. function variables)
- Integration test patterns in Go for CLI commands
- Performance impact of additional tests

**Success criteria:**
- ✅ Integration test passes with current code
- ✅ Test fails if --format json is incorrectly added to tmux spawn command
- ✅ Test coverage increases for cmd/orch package

---

## References

**Files Examined:**
- pkg/tmux/tmux.go - tmux spawning command builder
- pkg/tmux/tmux_test.go - unit tests for tmux package
- cmd/orch/main.go - spawn command implementation (runSpawnInTmux)
- pkg/opencode/client.go - opencode client command builder
- main_test.go - existing unit tests for main package

**Commands Run:**
```bash
# Run tmux package tests
go test ./pkg/tmux -v

# Run all tests
go test ./...
```

**External Documentation:**
- OpenCode CLI documentation (implicit) - reference for --format json flag

**Related Artifacts:**
- **Decision:** (none)
- **Investigation:** (none)
- **Workspace:** /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-inv-test-tmux-spawn-19dec/SPAWN_CONTEXT.md - spawn context for this investigation

---

## Investigation History

**[2025-12-19 21:30]:** Investigation started
- Initial question: Does tmux spawning correctly exclude --format json flag?
- Context: Spawned from beads issue orch-go-qyd to test tmux spawn functionality

**[2025-12-19 21:45]:** Code exploration completed
- Discovered tmux.BuildSpawnCommand excludes --format json, runSpawnInTmux uses it correctly
- Identified lack of integration test for full tmux spawn flow

**[2025-12-19 22:00]:** Investigation completed
- Final confidence: Medium (65%)
- Status: Complete
- Key outcome: Tmux spawn command consistency confirmed, integration test gap identified and recommended for implementation

## Self-Review

- [x] Real test performed (unit tests executed)
- [x] Conclusion from evidence (code inspection and test results)
- [x] Question answered
- [x] File complete

**Self-Review Status:** PASSED
