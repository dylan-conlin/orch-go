# orch-go

Go CLI for OpenCode orchestration - spawn sessions, monitor events, query sessions, and manage agent lifecycle.

## Installation

Build the CLI using the Cobra-based implementation (recommended):

```bash
# Build from cmd/orch directory
go build -o orch-go ./cmd/orch

# Or use make
make build
```

The legacy monolithic `main.go` at project root is deprecated and only supports `spawn`, `monitor`, and `ask` commands.

## Usage

All commands support a global `--server` flag to specify the OpenCode server URL (default: `http://localhost:4096`).

### Spawn a new session

Spawn a new OpenCode session with skill context. By default, spawns the agent in a tmux window and returns immediately. Use `--inline` to run in the current terminal (blocking).

```bash
# Basic spawn with a task
orch-go spawn investigation "explore the codebase"

# Spawn with feature-impl phases
orch-go spawn feature-impl "add new spawn command" --phases implementation,validation

# Spawn linked to a beads issue
orch-go spawn --issue proj-123 feature-impl "implement the feature"

# Run inline (blocking)
orch-go spawn --inline investigation "explore codebase"
```

**Flags:**
- `--issue <id>`: Beads issue ID for tracking
- `--phases <list>`: Feature-impl phases (e.g., implementation,validation)
- `--mode <tdd|direct>`: Implementation mode (default: tdd)
- `--validation <none|tests|smoke-test>`: Validation level (default: tests)
- `--inline`: Run inline (blocking) instead of in tmux

### Send a message to an existing session

Send a message to an existing OpenCode session. The session can be running or completed. Response text is streamed to stdout as it's received.

```bash
# Using 'send' command
orch-go send ses_abc123 "what files did you modify?"

# Using 'ask' (alias for send)
orch-go ask ses_xyz789 "can you explain the changes?"
```

### Monitor sessions for completion

Monitor the OpenCode server for session events and send notifications on completion.

```bash
orch-go monitor

# Output:
# Monitoring SSE events at http://localhost:4096/event...
# [session.status] {"status":"busy","session_id":"ses_abc123"}
# [session.status] {"status":"idle","session_id":"ses_abc123"}
# Session ses_abc123 completed!
```

### List active sessions

List all active OpenCode sessions with their status, title, directory, and last update time.

```bash
orch-go status

# Output (example):
# SESSION ID                         TITLE                          DIRECTORY                                 UPDATED
# ------------------------------------------------------------------------------------------------------------------
# ses_abc123                         orch-go-1703001600             /Users/me/project                         2025-12-19 14:30:05
# ses_xyz789                         investigation-explore          /Users/me/other                          2025-12-19 14:25:10
```

### Complete an agent and close beads issue

Complete an agent's work by verifying Phase: Complete and closing the beads issue. Checks that the agent has reported "Phase: Complete" via beads comments before closing.

```bash
# Complete with verification
orch-go complete proj-123

# Complete with custom reason
orch-go complete proj-123 --reason "All tests passing"

# Force complete (skip phase verification)
orch-go complete proj-123 --force
```

**Flags:**
- `--force`, `-f`: Skip phase verification
- `--reason <text>`, `-r`: Reason for closing (default: uses phase summary)

### View ecosystem changelog

Aggregate git commits across all repos in Dylan's orchestration ecosystem, grouped by date and categorized by type (skills, kb, cmd, pkg, etc.).

```bash
# Show last 7 days across all ecosystem repos
orch-go changelog

# Show last 14 days
orch-go changelog --days 14

# Filter to a specific repo
orch-go changelog --project orch-go

# Output as JSON (for scripting)
orch-go changelog --json
```

**Flags:**
- `--days <n>`: Number of days to include (default: 7)
- `--project <name>`: Project to filter, or "all" for all repos (default: all)
- `--json`: Output as JSON instead of human-readable format

**Output includes:**
- Summary statistics (total commits, repo count)
- Commits grouped by date
- Category breakdown (skills, kb, cmd, pkg, web, docs, config)
- Semantic labels for breaking changes, behavioral vs documentation changes
- Blast radius indicators (local, cross-skill, infrastructure)

See [Changelog System Documentation](docs/changelog-system.md) for details on semantic parsing, adding new repos, and API integration.

## Event Logging

All events are logged to `~/.orch/events.jsonl` in JSONL format:

```json
{"type":"session.spawned","session_id":"ses_abc123","timestamp":1703001600,"data":{"prompt":"say hello","title":"orch-go-1703001600"}}
{"type":"session.status","session_id":"ses_abc123","timestamp":1703001601,"data":{"status":"idle"}}
{"type":"session.completed","session_id":"ses_abc123","timestamp":1703001602}
```

## Requirements

- OpenCode running at `http://localhost:4096` (default)
- macOS for desktop notifications (uses beeep library)

## API Patterns

Based on validated manual testing:

1. **Spawn**: `opencode run --attach http://localhost:4096 --format json --title "title" "prompt"`
2. **Q&A**: `opencode run --attach http://localhost:4096 --session ses_xxx --format json "question"`
3. **SSE**: `curl http://localhost:4096/event` (server.connected, session.status, etc.)
4. **Completion**: Watch for `session.status` changing from `busy` to `idle`
