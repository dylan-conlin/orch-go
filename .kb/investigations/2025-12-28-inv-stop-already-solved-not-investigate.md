---
linked_issues:
  - orch-go-knvj
---
<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** OpenCode has no `/health` endpoint - the 500 "redirected too many times" error is expected behavior for undefined routes.

**Evidence:** Known answer provided in spawn context; valid endpoints are `/session`, `/session/{id}`, `/session/{id}/message`.

**Knowledge:** OpenCode returns 500 redirect errors for ANY invalid route, not just `/health`. Use `/session` to check server status.

**Next:** Close - no action needed. Document this for future reference.

---

# Investigation: OpenCode Health Endpoint Behavior

**Question:** Why does OpenCode return "redirected too many times" error on `/health` endpoint?

**Started:** 2025-12-28
**Updated:** 2025-12-28
**Owner:** spawned-agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: OpenCode Has No /health Endpoint

**Evidence:** OpenCode's valid endpoints are:
- `/session` - list sessions / check server status
- `/session/{id}` - get specific session
- `/session/{id}/message` - send message to session

**Source:** Prior investigation cited in spawn context (SPAWN_CONTEXT.md)

**Significance:** The `/health` endpoint was never implemented. Hitting it triggers OpenCode's fallback error handling.

---

### Finding 2: Redirect Loop is Expected Behavior for Undefined Routes

**Evidence:** OpenCode returns HTTP 500 with "redirected too many times" for ANY undefined route, not just `/health`.

**Source:** Prior investigation cited in spawn context

**Significance:** This is not a bug - it's the server's response to routes it doesn't recognize. The error message is misleading but not incorrect.

---

## Synthesis

**Key Insights:**

1. **No bug exists** - The reported "redirect loop on /health" is expected behavior for an undefined route.

2. **Use /session for health checks** - To verify OpenCode is running, hit `/session` which returns a valid response.

3. **Error message is misleading** - "Redirected too many times" suggests a loop, but it's actually just undefined route handling.

**Answer to Investigation Question:**

OpenCode returns a 500 error on `/health` because that endpoint doesn't exist. This is expected behavior, not a bug. Use `/session` to check if the OpenCode server is running.

---

## Structured Uncertainty

**What's tested:**

- ✅ Valid endpoints documented (from prior investigation)
- ✅ /health returns 500 error (expected behavior)

**What's untested:**

- ⚠️ Not re-verified in this session (intentionally - answer already known)

**What would change this:**

- OpenCode adding a `/health` endpoint in a future version

---

## Implementation Recommendations

**No implementation needed.** This was a knowledge clarification, not a bug.

If health checking is needed in scripts/automation, use:
```bash
curl -s http://127.0.0.1:3359/session | head -1
```

---

## References

**Source of Known Answer:**
- SPAWN_CONTEXT.md for this investigation (provided the authoritative answer)

**Related Prior Investigation:**
- `.kb/investigations/2025-12-26-inv-opencode-health-endpoint-returns-redirected.md` - Documented the same finding with actual curl tests (this investigation confirms rather than supersedes that one)

---

## Self-Review

- [x] Question answered clearly
- [x] Known answer cited properly
- [x] No unnecessary re-investigation performed
- [x] D.E.K.N. summary complete

**Self-Review Status:** PASSED

---

## Investigation History

**2025-12-28:** Investigation created
- Initial question: Why does OpenCode /health return redirect loop error?
- Context: Bug report filed, but answer was already known

**2025-12-28:** Investigation completed immediately
- Status: Complete
- Key outcome: Not a bug - /health doesn't exist, use /session instead
