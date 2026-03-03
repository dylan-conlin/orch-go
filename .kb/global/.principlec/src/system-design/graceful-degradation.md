### Graceful Degradation

Core functionality works without optional layers.

**Why:** Workspace state persists even if tmux window closes. Commands work without optional dependencies. The system shouldn't be fragile to missing pieces.
