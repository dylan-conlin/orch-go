<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** `orch send` had two silent failure modes: (1) empty session ID prefix accepted, (2) non-existent sessions return 204 success from OpenCode API.

**Evidence:** Tested `orch send ses_` - reported success. Tested `orch send ses_nonexistent_id` - also reported success. API returns 204 No Content for invalid/non-existent sessions.

**Knowledge:** OpenCode API's `/session/:id/prompt_async` endpoint silently accepts messages for non-existent sessions. Client-side validation required.

**Next:** Fix implemented and committed - validates session ID format and verifies existence via GetSession before sending.

**Confidence:** Very High (95%) - Smoke-tested both failure modes and verified fix.

---

# Investigation: orch send silent failure modes

**Question:** Why does `orch send` fail silently for tmux-based agents, and what causes the two reported failure modes?

**Started:** 2025-12-22
**Updated:** 2025-12-22
**Owner:** Debugging agent (systematic-debugging skill)
**Phase:** Complete
**Next Step:** None - fix implemented and verified
**Status:** Complete
**Confidence:** Very High (95%)

---

## Findings

### Finding 1: Empty session ID prefix was accepted without validation

**Evidence:** Running `orch send ses_ "test"` reported success:
```
✓ Message sent to session ses_ (via API)
Exit code: 0
```

The `resolveSessionID` function in cmd/orch/main.go blindly trusted any identifier starting with `ses_`:
```go
if strings.HasPrefix(identifier, "ses_") {
    return identifier, nil  // No validation!
}
```

**Source:** cmd/orch/main.go:1336-1340

**Significance:** Users could accidentally pass truncated session IDs (e.g., from copy-paste errors) and receive false success messages.

---

### Finding 2: OpenCode API silently accepts messages for non-existent sessions

**Evidence:** Testing the API directly with curl:
```bash
curl -v -X POST "http://127.0.0.1:4096/session/ses_nonexistent/prompt_async" \
  -H "Content-Type: application/json" \
  -d '{"parts":[{"type":"text","text":"test"}],"agent":"build"}'
```
Returns `HTTP/1.1 204 No Content` - a success status code!

However, the GET endpoint properly returns an error:
```bash
curl "http://127.0.0.1:4096/session/ses_nonexistent"
# Returns: {"name":"NotFoundError","data":{"message":"Resource not found..."}}
```

**Source:** Direct API testing

**Significance:** The async endpoint design allows fire-and-forget messages, but provides no feedback when the target doesn't exist. Client must verify before sending.

---

### Finding 3: The existing tmux fallback works correctly

**Evidence:** Testing `orch send orch-go-kszt "test"` correctly:
1. Fails to find OpenCode session (not captured for tmux spawns)
2. Falls back to tmux window lookup
3. Finds window by beads ID in window name
4. Sends via `tmux send-keys`
5. Reports: `✓ Message sent to orch-go-kszt (via tmux workers-orch-go:6)`

**Source:** Smoke tests, cmd/orch/main.go:1463-1495 (sendViaTmux)

**Significance:** The fallback mechanism added in the prior fix (resolveSessionID + tmux fallback) works correctly. The issue was only in the ses_ prefix shortcut path.

---

## Synthesis

**Key Insights:**

1. **Client validation is essential** - The OpenCode API's async endpoint design sacrifices acknowledgment for performance. The client must verify session existence before sending to catch errors.

2. **Session ID format validation prevents accidents** - Real session IDs have substantial content after `ses_` (e.g., `ses_4bc758a0affevWoGLNGREjeAKM`). Checking minimum length catches truncated IDs.

3. **GetSession provides existence check** - While prompt_async silently succeeds for non-existent sessions, the GET /session/:id endpoint properly returns NotFoundError.

**Answer to Investigation Question:**

`orch send` failed silently due to two root causes:
1. **Format validation gap:** The `resolveSessionID` function accepted any string starting with `ses_` without validating the suffix had meaningful content
2. **API design assumption:** The OpenCode prompt_async endpoint returns 204 No Content for any session ID, existing or not, so there's no server-side error to catch

The fix adds: (1) minimum length validation for the ses_ suffix, and (2) a GetSession call to verify the session exists before sending. If verification fails, the command properly falls through to the tmux fallback path.

---

## Confidence Assessment

**Current Confidence:** Very High (95%)

**Why this level?**

Both failure modes were reproduced, root causes traced, fix implemented, and all test cases verified.

**What's certain:**

- ✅ Empty ses_ prefix now rejected with clear error message
- ✅ Non-existent session IDs now verified via GetSession before sending
- ✅ Valid session IDs still work correctly
- ✅ Beads ID → tmux fallback still works correctly
- ✅ All unit tests pass

**What's uncertain:**

- ⚠️ Additional latency from GetSession call (single HTTP request per send)

**What would increase confidence to 100%:**

- Long-term monitoring in production

---

## Implementation Recommendations

**Purpose:** Document the fix that was implemented.

### Recommended Approach ⭐

**Add session validation before API send** - Validate session ID format and verify existence via GetSession.

**Why this approach:**
- Catches both failure modes (truncated IDs and non-existent sessions)
- Provides clear error messages to users
- Allows fallback to tmux when session doesn't exist in OpenCode

**Implementation sequence:**
1. Check if ses_ suffix has minimum length (8 chars) - catches truncated IDs
2. Call GetSession to verify existence - catches non-existent sessions
3. If validation fails, return error and let runSend try tmux fallback

### Implementation Details

**What was implemented:**

In `resolveSessionID()` (cmd/orch/main.go:1336-1352):
```go
if strings.HasPrefix(identifier, "ses_") {
    // Validate the session ID has content after the prefix
    suffix := strings.TrimPrefix(identifier, "ses_")
    if len(suffix) < 8 { // Session IDs have substantial content after ses_
        return "", fmt.Errorf("invalid session ID format: %s (too short)", identifier)
    }
    // Verify the session exists in OpenCode
    client := opencode.NewClient(serverURL)
    _, err := client.GetSession(identifier)
    if err != nil {
        return "", fmt.Errorf("session not found in OpenCode: %s", identifier)
    }
    return identifier, nil
}
```

**Success criteria:**
- ✅ `orch send ses_` returns error
- ✅ `orch send ses_nonexistent_id` returns error
- ✅ `orch send ses_<valid_id>` works correctly
- ✅ `orch send <beads_id>` works via tmux fallback

---

## References

**Files Examined:**
- cmd/orch/main.go:1329-1421 - resolveSessionID and runSend functions
- pkg/opencode/client.go:156-177 - SendMessageAsync implementation

**Commands Run:**
```bash
# Test empty session ID prefix
./build/orch send --async "ses_" "test"

# Test non-existent session ID  
./build/orch send --async "ses_nonexistent_session_id_12345" "test"

# Test valid session ID
./build/orch send --async ses_4bc758a0affevWoGLNGREjeAKM "test"

# Test beads ID (tmux fallback)
./build/orch send --async "orch-go-kszt" "test"

# Direct API tests
curl -v -X POST "http://127.0.0.1:4096/session/ses_/prompt_async" ...
curl "http://127.0.0.1:4096/session/ses_nonexistent"
```

**Related Artifacts:**
- **Investigation:** .kb/investigations/2025-12-21-debug-orch-send-fails-silently-tmux.md - Prior investigation that added resolveSessionID

---

## Investigation History

**2025-12-22 18:07:** Investigation started
- Initial question: Why does orch send fail silently for tmux-based agents?
- Context: Spawned from issue orch-go-c3uj to continue debugging after prior fix

**2025-12-22 18:08:** Built and tested existing behavior
- Found valid session IDs and beads IDs work correctly
- Discovered empty ses_ prefix accepted (failure mode 1)

**2025-12-22 18:10:** Root cause identified
- OpenCode API returns 204 for non-existent sessions (failure mode 2)
- GetSession endpoint properly returns NotFoundError

**2025-12-22 18:11:** Fix implemented
- Added minimum length validation for ses_ suffix
- Added GetSession verification before sending

**2025-12-22 18:12:** Investigation completed
- Final confidence: Very High (95%)
- Status: Complete
- Key outcome: Both failure modes fixed with proper validation
