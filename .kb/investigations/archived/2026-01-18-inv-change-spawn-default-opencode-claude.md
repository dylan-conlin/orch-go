## Summary (D.E.K.N.)

**Delta:** Changed spawn default backend from opencode to claude to align with Claude Max subscription economics.

**Evidence:** Tests pass (TestModelAutoSelection - 5/5 cases). Code changes in spawn_cmd.go:1136 and spawn_cmd_test.go:116-119, 132.

**Knowledge:** Default backend now uses claude CLI (Max subscription, unlimited Opus) instead of opencode API (pay-per-token). Explicit --model sonnet still uses opencode for API access.

**Next:** Close - implementation complete.

**Promote to Decision:** recommend-no (executes existing decision "Opus default, Gemini escape hatch")

---

# Investigation: Change Spawn Default Opencode Claude

**Question:** How to change the spawn command default backend from opencode to claude?

**Started:** 2026-01-18
**Updated:** 2026-01-18
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Default backend is set in single location

**Evidence:** Line 1136 in spawn_cmd.go: `spawnBackend := "opencode"`

**Source:** cmd/orch/spawn_cmd.go:1136

**Significance:** Simple change - only one location needs modification for core behavior.

---

### Finding 2: Help text documents default in Backend Modes section

**Evidence:** Help text states: `opencode: Uses OpenCode HTTP API (default)`

**Source:** cmd/orch/spawn_cmd.go:88

**Significance:** User-facing documentation must be updated to match new default.

---

### Finding 3: Test mirrors the production logic

**Evidence:** Test case "no flags defaults to opencode" with `backend := "opencode"` as initial value simulates production auto-selection logic.

**Source:** cmd/orch/spawn_cmd_test.go:116-120, 132

**Significance:** Test validates the default behavior and must be updated alongside code.

---

### Finding 4: Model auto-selection overrides remain unchanged

**Evidence:** Explicit --model sonnet still routes to opencode (pay-per-token API). Opus routes to claude. No model flag now uses claude default.

**Source:** cmd/orch/spawn_cmd.go:1170-1181

**Significance:** Escape hatch via explicit model flags preserved - users can still access opencode backend when needed.

---

## Synthesis

**Key Insights:**

1. **Single point of change** - The default is set in one place and flows through the priority system, making this change surgical.

2. **Economics align with subscription** - Claude Max subscription provides unlimited usage via Claude CLI, making it the cost-effective default. OpenCode remains available for explicit sonnet requests.

3. **Existing tests cover behavior** - The TestModelAutoSelection test directly validates the default case, confirming the change works.

**Answer to Investigation Question:**

Changed default backend by:
1. Updated `spawnBackend := "opencode"` to `spawnBackend := "claude"` (line 1136)
2. Updated help text to show claude as default (line 87-88)
3. Updated comment about default (line 1135)
4. Updated comment about other models (line 1181)
5. Updated test case to expect claude default (lines 116-119, 132)

---

## Structured Uncertainty

**What's tested:**

- ✅ TestModelAutoSelection passes with new default (ran: go test ./cmd/orch/... -run TestModelAutoSelection)
- ✅ Full test suite passes for cmd/orch package
- ✅ Code compiles with changes

**What's untested:**

- ⚠️ Actual spawn behavior with claude backend (not spawned a real agent)
- ⚠️ Daemon behavior with new default (not tested daemon workflow)

**What would change this:**

- Finding would be incomplete if there are other code paths that assume opencode default

---

## References

**Files Examined:**
- cmd/orch/spawn_cmd.go - Main spawn command logic
- cmd/orch/spawn_cmd_test.go - Test for model auto-selection
- pkg/model/model.go - Default model definition (Sonnet)

**Commands Run:**
```bash
# Run model auto-selection tests
go test ./cmd/orch/... -run TestModelAutoSelection -v

# Run full test suite
go test ./...
```

---

## Investigation History

**2026-01-18:** Investigation started
- Initial question: How to change spawn default from opencode to claude
- Context: Aligns with Claude Max subscription economics

**2026-01-18:** Implementation complete
- Status: Complete
- Key outcome: Default backend changed from opencode to claude with tests passing
