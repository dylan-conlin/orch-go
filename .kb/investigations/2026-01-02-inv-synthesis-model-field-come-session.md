<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Added authoritative model extraction from OpenCode session metadata to fix inconsistent SYNTHESIS.md model field.

**Evidence:** Tests pass for new `GetSessionModel` helper and `Synthesis.Model` field parsing. Dashboard injects authoritative model from OpenCode session into synthesis responses.

**Knowledge:** OpenCode session messages contain authoritative `ModelID` and `ProviderID` in `MessageInfo` - more reliable than agent self-report.

**Next:** Close - implementation complete and tested.

---

# Investigation: Synthesis Model Field from Session

**Question:** How should we fix the SYNTHESIS model field to use authoritative session metadata instead of inconsistent agent self-reports?

**Started:** 2026-01-02
**Updated:** 2026-01-02
**Owner:** og-debug-synthesis-model-field-02jan-2
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: OpenCode MessageInfo contains authoritative model info

**Evidence:** `MessageInfo` struct in `pkg/opencode/types.go:94-95` has:
- `ModelID string` - e.g., "claude-opus-4-5-20251101"
- `ProviderID string` - e.g., "anthropic"

**Source:** pkg/opencode/types.go:94-95

**Significance:** This is the authoritative source - OpenCode populates these fields from the actual API call, not from agent self-report.

---

### Finding 2: Agents self-report models inconsistently

**Evidence:** Task description mentions agents report: "opus", "Claude", "claude-opus-4-20250514" - all different formats for the same model.

**Source:** SPAWN_CONTEXT.md task description

**Significance:** Self-reported model field in SYNTHESIS.md is unreliable. Should be overwritten with authoritative source.

---

### Finding 3: Dashboard already batch-fetches session data

**Evidence:** `cmd/orch/serve.go` uses parallel goroutines with semaphore to fetch:
- Token stats via `GetSessionTokens`
- Last activity via `getLastActivityForSession`

**Source:** cmd/orch/serve.go:1241-1270 (tokens), 1272-1310 (activity)

**Significance:** Adding model fetching follows the same established pattern for efficiency.

---

## Synthesis

**Key Insights:**

1. **Authoritative source exists** - OpenCode session messages have reliable model info in `MessageInfo.ModelID` and `MessageInfo.ProviderID`

2. **Fallback strategy** - When session ID unavailable, use agent-reported model from SYNTHESIS.md

3. **Efficient implementation** - Batch fetch model info in parallel with other session data

**Answer to Investigation Question:**

Added `GetSessionModel` helper to extract model from first assistant message with model info. Dashboard now fetches authoritative model from OpenCode session and overwrites the SYNTHESIS.md model field. Falls back to agent-reported model when session not available.

---

## Structured Uncertainty

**What's tested:**

- ✅ Synthesis.Model field extracted from SYNTHESIS.md (TestParseSynthesisDEKN)
- ✅ GetSessionModel extracts model from messages (TestGetSessionModel)
- ✅ SessionModel.String() formats correctly (TestSessionModelString)
- ✅ All existing tests still pass (go test ./...)

**What's untested:**

- ⚠️ Dashboard actually shows authoritative model (requires manual verification)
- ⚠️ Performance impact of additional batch fetch (expected minimal)

**What would change this:**

- OpenCode changes MessageInfo structure
- Agents start using different message formats

---

## Implementation Recommendations

### Recommended Approach (Implemented)

**Inject authoritative model during synthesis display** - Fetch model from OpenCode session when displaying synthesis, overwrite agent-reported model.

**Why this approach:**
- Single source of truth (OpenCode session)
- No modification to SYNTHESIS.md files needed
- Fallback to self-report when session unavailable

**Implementation sequence:**
1. Added `Model` and `Skill` fields to `Synthesis` struct - parse from Session Metadata
2. Added `GetSessionModel` to `pkg/opencode/client.go` - extract from first assistant message
3. Updated `SynthesisResponse` with `Skill` and `Model` fields
4. Added parallel fetch of model info in dashboard handler

---

## References

**Files Examined:**
- `pkg/opencode/types.go` - MessageInfo structure with ModelID/ProviderID
- `pkg/verify/check.go` - Synthesis struct and ParseSynthesis
- `cmd/orch/serve.go` - Dashboard API handlers with batch fetching

**Commands Run:**
```bash
# Build verification
go build ./...

# Test synthesis parsing
go test ./pkg/verify/... -run TestParseSynthesis -v

# Test session model extraction
go test ./pkg/opencode/... -run "TestSessionModel|TestGetSessionModel" -v

# Full test suite
go test ./...
```

---

## Investigation History

**2026-01-02:** Investigation started
- Initial question: How to fix inconsistent model reporting in SYNTHESIS.md
- Context: Agents report models inconsistently (opus, Claude, claude-opus-4-20250514)

**2026-01-02:** Found authoritative source
- OpenCode MessageInfo contains ModelID and ProviderID

**2026-01-02:** Investigation completed
- Status: Complete
- Key outcome: Added GetSessionModel helper and dashboard injection of authoritative model
