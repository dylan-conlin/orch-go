---
status: active
blocks:
  - keywords:
      - switch to docker desktop
      - use docker desktop
      - replace colima
---

# Decision: Use Colima Instead of Docker Desktop

**Date:** 2026-01-21
**Status:** Accepted
**Context:** Docker backend for orch spawn

## Problem

Docker Desktop on macOS frequently enters a hung state where:
- The application processes run but the daemon won't start
- All docker commands hang or timeout
- Restarts don't reliably fix it
- Manual GUI interaction sometimes required

This undermined the docker backend's value as an "escape hatch" for rate limit bypass.

## Decision

Replace Docker Desktop with Colima for container runtime on Dylan's machine.

## Rationale

- **Reliability:** Colima uses Lima VMs with a simpler architecture, fewer moving parts
- **Lightweight:** No Electron app, no GUI overhead
- **CLI-native:** Starts/stops cleanly via `colima start/stop`
- **Compatible:** Uses standard Docker socket, existing docker commands work unchanged
- **Industry standard:** Widely adopted alternative to Docker Desktop

## Implementation

```bash
# Install
brew install colima docker-credential-helper

# Stop Docker Desktop
osascript -e 'quit app "Docker Desktop"'

# Start Colima (4 CPUs, 12GB RAM)
# 12GB allows 6GB per container (2 concurrent agents)
# Increased from 8GB (4GB/container) after SIGKILL crashes at ~9k tokens
colima start --cpu 4 --memory 12

# Auto-start on boot
brew services start colima

# Rebuild claude-code-mcp image (fresh Colima VM has no images)
cd ~/.claude/docker-workaround && docker build -t claude-code-mcp .
```

## Consequences

- Docker Desktop can be uninstalled (or kept dormant)
- `orch spawn --backend docker` now works reliably
- Need to rebuild images after fresh Colima install
- Colima VM uses ~8GB RAM when running

## References

- Triggered by: Docker Desktop hung state blocking docker backend testing
- Related: `.kb/decisions/2026-01-09-dual-spawn-mode-architecture.md`
