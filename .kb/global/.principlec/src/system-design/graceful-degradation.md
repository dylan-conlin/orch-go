### Graceful Degradation

Core functionality works without optional layers. When a component is degrading, the system should know and adapt.

**Why:** The system shouldn't be fragile to missing pieces. Workspace state persists even if tmux window closes. Commands work without optional dependencies.

**Pattern:** Try preferred path first, fall through to fallback on any error.

**Applied examples:**

- **RPC with CLI fallback:** Try RPC client first for performance; fall through to CLI on any error (socket not found, connect failed, operation failed). Graceful degradation when daemon unavailable.
- **Backend independence:** Claude CLI agents in tmux survive OpenCode server crashes. Critical paths need independent secondary mechanisms.

**Degradation visibility:** Systems should know when they're degrading. Agents flying blind without context token warnings can't prioritize deliverables over nice-to-haves or decide to wrap up before hitting limits.

**The test:** If this dependency disappears, does the core operation still complete?
