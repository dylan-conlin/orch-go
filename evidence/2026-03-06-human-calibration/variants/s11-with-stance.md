## Knowledge

- In orch-go's HTTP server, the `api` group at /api/v1 applies
  `authMiddleware()` (JWT validation) and `rateLimitMiddleware()`
  (per-account token bucket) to all handler groups registered on it.
- Handler registration functions (registerAgentHandlers, etc.) receive
  a `*gin.RouterGroup` and register their routes on it. The middleware
  of the parent group applies automatically to child routes.
- Tests for API handlers test handler logic and response format.
  Middleware presence is not tested at the handler level.

## Stance

Absence is evidence. When reviewing code, check what existing peers
do that the new code doesn't. A missing pattern is as significant as
a present bug. If every other X has Y, the one without Y is the finding.
