<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Headless spawn failed because SendMessageAsync passed model as a string, but OpenCode's prompt_async API expects model as an object with providerID and modelID fields.

**Evidence:** curl tests confirmed 400 error with `{"error":[{"expected":"object","code":"invalid_type","path":["model"]}]}` when model passed as string; 204 success when passed as `{"providerID": "google", "modelID": "gemini-2.5-flash"}`.

**Knowledge:** OpenCode API contract for prompt_async requires model to be structured object, not string; the error message ("redirect loop") in the original report was misleading - actual error was 400 Bad Request with validation failure.

**Next:** Fix implemented and verified - parseModelSpec helper function now converts "provider/modelID" string to proper object format.

**Confidence:** Very High (95%) - Fix verified with both unit tests and end-to-end smoke test.

---

# Investigation: Headless Spawn Model Format Bug

**Question:** Why does headless spawn fail with "redirect loop" error when sessions are created successfully but message API returns errors?

**Started:** 2025-12-23
**Updated:** 2025-12-23
**Owner:** og-debug-headless-spawn-fails-23dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%)

---

## Findings

### Finding 1: Actual error was 400 Bad Request, not redirect loop

**Evidence:** When reproducing the error via `orch spawn --no-track --model flash investigation "test"`, the actual error was:
```
Error: failed to send prompt: unexpected status code: 400
```

The original "redirect loop" description was either a different manifestation or misreported.

**Source:** Direct reproduction via CLI

**Significance:** Pointed investigation toward API contract mismatch rather than HTTP client redirect handling.

---

### Finding 2: OpenCode prompt_async expects model as structured object

**Evidence:** Testing with curl showed:
- String format (what we sent): `"model": "google/gemini-2.5-flash"` → 400 error with `invalid_type: expected object, received string`
- Object format (what works): `"model": {"providerID": "google", "modelID": "gemini-2.5-flash"}` → 204 success

**Source:** `pkg/opencode/client.go:158-182` (SendMessageAsync function)

**Significance:** The model package already parses provider/modelID into separate fields, but SendMessageAsync was passing the raw string instead of structured object.

---

### Finding 3: Fix required minimal code change

**Evidence:** Added `parseModelSpec()` helper function to convert "provider/modelID" string format to the expected object format:
```go
func parseModelSpec(model string) map[string]string {
    idx := strings.Index(model, "/")
    if idx <= 0 || idx >= len(model)-1 {
        return nil
    }
    return map[string]string{
        "providerID": model[:idx],
        "modelID":    model[idx+1:],
    }
}
```

**Source:** `pkg/opencode/client.go:189-203`

**Significance:** Simple fix, backwards compatible (empty model still omits the field), tested with both unit tests and smoke test.

---

## Synthesis

**Key Insights:**

1. **API contract documentation gap** - The OpenCode API expects model as an object, but this wasn't documented in our codebase. The fix now handles the conversion.

2. **Error message quality** - The original error response from the server was detailed (`expected object, received string`), but our client was just reporting `unexpected status code: 400`. Improved error message to include response body.

3. **Existing model infrastructure underused** - The `pkg/model` package already has `ModelSpec` with Provider and ModelID fields. The fix leverages the same format parsing logic.

**Answer to Investigation Question:**

Headless spawn failed because `SendMessageAsync` passed the model parameter as a raw string (e.g., `"google/gemini-2.5-flash"`), but OpenCode's `/session/{id}/prompt_async` endpoint expects model to be a JSON object with `providerID` and `modelID` fields. The fix parses the provider/modelID string and constructs the proper object format.

---

## Confidence Assessment

**Current Confidence:** Very High (95%)

**Why this level?**

The fix was verified both through unit tests (including edge cases for parseModelSpec) and through end-to-end smoke testing with an actual headless spawn that succeeded.

**What's certain:**

- ✅ Root cause identified: model format mismatch between orch-go and OpenCode API
- ✅ Fix works: smoke test succeeded with spawned session ses_4b24561dfffes4Ojb2iQqCCurN
- ✅ All existing tests pass: no regression introduced

**What's uncertain:**

- ⚠️ Whether the original "redirect loop" report was a different issue or misreported (resolved to 400 error in reproduction)

**What would increase confidence to 100%:**

- Confirmation from other users that their headless spawns now work
- Longer soak testing with various model combinations

---

## Implementation Recommendations

**Purpose:** Document the implemented fix for future reference.

### Implemented Approach ⭐

**Parse model string to object** - Convert "provider/modelID" format to `{"providerID": ..., "modelID": ...}` object before sending to API.

**Why this approach:**
- Minimal code change in one location
- Backwards compatible (empty model still works)
- Matches existing model parsing patterns in codebase

**Implementation completed:**
1. Added `parseModelSpec()` helper function in `pkg/opencode/client.go`
2. Updated `SendMessageAsync()` to use helper when model is provided
3. Improved error message to include response body for debugging
4. Added unit tests for parseModelSpec edge cases
5. Updated existing SendMessageAsyncWithModel test to verify object format

---

## References

**Files Modified:**
- `pkg/opencode/client.go` - Added parseModelSpec, updated SendMessageAsync
- `pkg/opencode/client_test.go` - Added TestParseModelSpec, updated TestSendMessageAsyncWithModel

**Commands Run:**
```bash
# Test API contract with curl
curl -X POST http://127.0.0.1:4096/session/{id}/prompt_async \
  -H "Content-Type: application/json" \
  -d '{"parts": [{"type": "text", "text": "hello"}], "agent": "build", "model": {"providerID": "google", "modelID": "gemini-2.5-flash"}}'

# Smoke test after fix
./orch-test spawn --no-track --model flash investigation "test spawn after fix"
```

---

## Investigation History

**2025-12-23 00:00:** Investigation started
- Initial question: Why do headless spawns fail with redirect loop?
- Context: Sessions created but message API returns errors

**2025-12-23 00:15:** Root cause identified
- Reproduced as 400 error, not redirect loop
- API expects model as object, not string

**2025-12-23 00:30:** Fix implemented and verified
- Added parseModelSpec helper function
- All tests passing
- Smoke test successful
- Final confidence: Very High (95%)
- Status: Complete
- Key outcome: Model format converted from string to object per OpenCode API contract
