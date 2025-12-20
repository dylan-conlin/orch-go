# orch-go

Go rewrite of orch-cli - AI agent orchestration via OpenCode API.

## What This Is

Single Go binary for agent lifecycle management:
- **spawn** - Create OpenCode sessions with skill context
- **status** - List active agents (via SSE)
- **complete** - Verify deliverables, close beads issues
- **send** - Q&A on existing sessions
- **monitor** - Real-time SSE event watching with desktop notifications

## Architecture

```
orch-go (this binary)
    │
    ├── opencode run --attach    ← spawn/Q&A (wraps CLI)
    ├── GET /event               ← SSE monitoring (direct HTTP)
    └── bd CLI                   ← beads integration (shells out)
    │
    ▼
OpenCode Server (http://127.0.0.1:4096)
    │
    ▼
Claude API
```

**Key decision:** Use `opencode run --attach` not raw HTTP POST. Session ID is at top level of JSON events.

## Development

```bash
# Build
go build -o orch-go .
# or after scaffolding:
make build

# Test
go test ./...

# Install
make install  # copies to ~/bin/orch-go
```

## Project Structure (target)

```
orch-go/
├── cmd/orch/           # CLI commands (cobra)
│   ├── main.go
│   ├── spawn.go
│   ├── status.go
│   └── ...
├── pkg/
│   ├── opencode/       # OpenCode client + SSE
│   ├── events/         # Event logging (~/.orch/events.jsonl)
│   ├── notify/         # Desktop notifications
│   └── skills/         # Skill loader
├── internal/
├── Makefile
└── go.mod
```

## OpenCode API Patterns

**Spawn session:**
```bash
opencode run --attach http://127.0.0.1:4096 --format json --title "name" "prompt"
# Returns JSON events with sessionID at top level
```

**Q&A on session:**
```bash
opencode run --attach http://127.0.0.1:4096 --session ses_xxx --format json "question"
# Session context preserved - agent remembers conversation
```

**SSE events (GET /event):**
```
session.created  - new session
session.status   - {busy|idle} for completion detection
message.updated  - message progress
step_finish      - cost/tokens
```

## Related

- **Decision:** `orch-cli/.kb/decisions/2025-12-18-sdk-based-agent-management.md`
- **POC validation:** This repo started as POC, being refactored into proper structure
- **Python orch-cli:** `~/Documents/personal/orch-cli` (being replaced)
- **Beads:** Issue tracking via `bd` CLI
