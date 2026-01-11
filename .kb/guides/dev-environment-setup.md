# Development Environment Setup

**Last Updated:** 2026-01-10
**Applies To:** Mac development environment

## Quick Start

```bash
# Start all services
overmind start -D

# Check status
overmind status

# View logs
overmind echo

# Stop all
overmind quit
```

## Architecture

```
overmind (process supervisor)
├── api: orch serve (port 3348)
├── web: bun run dev (port 5188)
└── opencode: opencode serve --port 4096
```

### Procfile

```procfile
api: orch serve
web: cd web && bun run dev
opencode: ~/.bun/bin/opencode serve --port 4096
```

## Service Ports

| Service | Port | Purpose |
|---------|------|---------|
| OpenCode | 4096 | Claude/Gemini API sessions |
| orch serve | 3348 | Dashboard backend API |
| Web UI | 5188 | Dashboard frontend (Vite dev server) |

## Management Commands

### Starting Services

```bash
# Start in daemon mode (background)
overmind start -D

# Start in foreground (see logs in terminal)
overmind start

# Start specific service
overmind start api
```

### Checking Status

```bash
# Show all services
overmind status

# Check health endpoints
curl localhost:3348/health  # API
curl localhost:4096/health  # OpenCode
```

### Restarting Services

```bash
# Restart single service
overmind restart api
overmind restart web
overmind restart opencode

# Restart all
overmind restart
```

### Viewing Logs

```bash
# Unified logs with color coding
overmind echo

# Logs for specific service
overmind echo api
overmind echo web
overmind echo opencode

# Follow logs
overmind echo | tail -f
```

### Stopping Services

```bash
# Graceful shutdown
overmind quit

# Stop specific service
overmind stop api
```

## Troubleshooting

### Services Won't Start

**Check for port conflicts:**
```bash
lsof -i:4096  # OpenCode
lsof -i:3348  # API
lsof -i:5188  # Web
```

**Kill orphaned processes:**
```bash
lsof -ti:4096 | xargs kill -9  # OpenCode
lsof -ti:3348 | xargs kill -9  # API
lsof -ti:5188 | xargs kill -9  # Web
```

**Start fresh:**
```bash
overmind quit  # First try graceful
# Kill orphans if needed (above)
overmind start -D
```

### Services Crash Frequently

**View logs to diagnose:**
```bash
overmind echo

# Or check individual service logs
overmind echo api
```

**Restart problematic service:**
```bash
overmind restart api
```

### Can't Access Dashboard

**Verify all services running:**
```bash
overmind status
```

**Check health endpoints:**
```bash
curl localhost:3348/health
curl localhost:4096/health
```

**Open in browser:**
```bash
open http://localhost:5188
```

### Overmind Not Found

**Install via Homebrew:**
```bash
brew install tmux overmind
```

**Verify installation:**
```bash
which overmind  # Should show /opt/homebrew/bin/overmind
overmind version
```

## Development Workflow

### Typical Session

```bash
# 1. Start services
overmind start -D

# 2. Work on code
# ... make changes ...

# 3. Rebuild if needed
make install

# 4. Restart affected service
overmind restart api  # If Go code changed
overmind restart web  # If web code changed (hot reload should work)

# 5. Check logs if issues
overmind echo

# 6. Stop when done
overmind quit
```

### Hot Reload

**Web UI (Vite):** Changes to web/ automatically reload in browser

**API:** Requires `make install && overmind restart api`

**OpenCode:** Rarely needs restart (only if server itself updated)

## Why Overmind (Not launchd)

**Context:** Initially tried launchd plists for production-style reliability, but Mac is a dev environment.

**Overmind benefits for dev:**
- ✅ Simple Procfile configuration (3 lines)
- ✅ Unified logs with color coding
- ✅ Easy restart of individual services
- ✅ No PATH propagation issues
- ✅ Standard tool (same as other projects)

**launchd drawbacks for dev:**
- ❌ XML plists (verbose, hard to edit)
- ❌ tmux PATH issues with overmind supervision
- ❌ Requires unload/load cycle for changes
- ❌ Logs scattered across system
- ❌ Overkill for dev environment

**See:** `.kb/post-mortems/2026-01-09-launchd-recommendation-failure.md` for full context.

## Production Deployment

**Mac is dev only.** Production will be VPS with systemd.

**See:** `.kb/decisions/2026-01-10-dev-vs-prod-architecture.md`
