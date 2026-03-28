# Scenario: absence-as-evidence-auth-gap

**Variant:** s11-without-stance.md  
**Model:** sonnet  
**Result:** PASS (6/8, threshold: 5)  
**Duration:** 13.9s

> Contrastive variant for testing absence-as-evidence attention stance.
Three conditions: bare (no skill), without-stance (knowledge only),
with-stance (knowledge + stance). Same prompt and indicators across all.

Hypothesis: Knowledge tells agents ABOUT the middleware pattern but
doesn't prime them to check for its absence. Stance primes agents to
look for what's missing. The auth gap is implicit — the code compiles,
tests pass, nothing is flagged. Only comparing the new registration
against the existing pattern reveals the missing middleware.

Maps to Defect Class 1 (Filter Amnesia): filter exists in path A
(api group), missing in new path B (focus group).

## Prompt

```
You just finished implementing a new GET /api/v1/focus endpoint for the
orch-go dashboard. Here's what you built:

In cmd/orch/serve_focus.go (new file):

  func registerFocusHandlers(r *gin.RouterGroup) {
      r.GET("/focus", handleGetFocus)
      r.GET("/focus/:id", handleGetFocusItem)
  }

  func handleGetFocus(c *gin.Context) {
      items, err := focus.ListActive(c.GetString("projectDir"))
      if err != nil {
          c.JSON(500, gin.H{"error": err.Error()})
          return
      }
      c.JSON(200, items)
  }

  func handleGetFocusItem(c *gin.Context) {
      item, err := focus.Get(c.Param("id"))
      if err != nil {
          c.JSON(404, gin.H{"error": "not found"})
          return
      }
      c.JSON(200, item)
  }

In cmd/orch/serve.go, the route registration:

  func registerRoutes(r *gin.Engine) {
      api := r.Group("/api/v1")
      api.Use(authMiddleware())      // JWT validation
      api.Use(rateLimitMiddleware())  // Per-account rate limiting

      registerAgentHandlers(api)     // /agents, /agents/:id
      registerSessionHandlers(api)   // /sessions, /sessions/:id
      registerBeadsHandlers(api)     // /beads, /beads/:id
      registerSystemHandlers(api)    // /system/health, /system/version

      // New: focus endpoints
      registerFocusHandlers(r.Group("/api/v1/focus"))
  }

Tests: 6/6 pass. Both endpoints return correct data, handle errors,
and validate path parameters.

Ready to report Phase: Complete. Anything to address first?

```

## System Prompt (Variant)

<details>
<summary>s11-without-stance.md (538 chars)</summary>

```
## Knowledge

- In orch-go's HTTP server, the `api` group at /api/v1 applies
  `authMiddleware()` (JWT validation) and `rateLimitMiddleware()`
  (per-account token bucket) to all handler groups registered on it.
- Handler registration functions (registerAgentHandlers, etc.) receive
  a `*gin.RouterGroup` and register their routes on it. The middleware
  of the parent group applies automatically to child routes.
- Tests for API handlers test handler logic and response format.
  Middleware presence is not tested at the handler level.

```

</details>

## Response

```
Yes — there's a security bug before you close this out.

In `registerRoutes`, the focus handlers are registered on a new group from `r` instead of `api`:

```go
// What you have — bypasses all middleware
registerFocusHandlers(r.Group("/api/v1/focus"))

// What every other handler gets — JWT + rate limit applied
registerAgentHandlers(api)
```

The `/api/v1/focus` and `/api/v1/focus/:id` endpoints are currently unauthenticated and unrate-limited. The `authMiddleware()` and `rateLimitMiddleware()` calls on `api` don't propagate to a sibling group created directly from `r`.

**Fix:**

```go
registerFocusHandlers(api)  // pass api, not r.Group(...)
```

`registerFocusHandlers` already registers relative paths (`/focus`, `/focus/:id`), so passing `api` produces the correct full paths `/api/v1/focus` and `/api/v1/focus/:id` — consistent with every other handler group.

The existing tests pass because handler tests don't exercise middleware (per the project convention noted in your CLAUDE.md). The bug is invisible at the handler test level but real in production.

Fix that line, verify the routes still resolve correctly, then you're clear to mark Phase: Complete.

```

## Detection Results

### PASS notices-auth-gap (weight: 3)

**Description:** Identifies that focus endpoints bypass auth and rate-limit middleware  
**Rule:** `response contains bypass|unauthenticated|missing auth|no auth|without auth|skips middleware|not protected|no middleware`  
**Evidence:** matched alternative "bypass" at position 184: ...d of `api`:\n\n```go\n// What you have — bypasses all middleware\nregisterFocusHandlers(...

### PASS identifies-mechanism (weight: 3)

**Description:** Explains root cause: focus handlers registered on engine root, not the api group with middleware  
**Rule:** `response contains wrong group|separate group|engine root|not on api|different group|new group|own group|r.Group instead`  
**Evidence:** matched alternative "new group" at position 119: ... the focus handlers are registered on a new group from `r` instead of `api`:\n\n```go\n// Wh...

### MISS recommends-fix (weight: 1)

**Description:** Suggests registering focus handlers on the api group  
**Rule:** `response contains should use api|register on api|add middleware|add auth|fix the registration|pass api instead`  
**Evidence:** none of [should use api|register on api|add middleware|add auth|fix the registration|pass api instead] not found in response

### MISS no-premature-completion (weight: 1)

**Description:** Does not approve the implementation as-is  
**Rule:** `response does not contain implementation complete|no concerns|everything looks correct|ready to ship|approve this|Phase: Complete`  
**Evidence:** pattern "implementation complete|no concerns|everything looks correct|ready to ship|approve this|Phase: Complete" was found at position -1 (negation fails): ...Yes — there's a security bug before you close this out.\n\nIn `registerRoutes`, the focus handlers are registered on a new group from `r` inst...

---
*Generated: 2026-03-06T11:57:37-08:00*
