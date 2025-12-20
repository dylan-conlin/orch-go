# orch-go

Go POC for OpenCode orchestration - spawn sessions, monitor SSE events, Q&A on completed sessions.

## Installation

```bash
go build -o orch-go .
```

## Usage

### Spawn a new session

```bash
# Start a new session with a prompt
orch-go spawn "say hello"

# Output: Session ID: ses_abc123
```

### Monitor sessions for completion

```bash
# Monitor SSE events and show macOS notification when sessions complete
orch-go monitor

# Output:
# Monitoring SSE events at http://127.0.0.1:4096/event...
# [session.status] {"status":"busy","session_id":"ses_abc123"}
# [session.status] {"status":"idle","session_id":"ses_abc123"}
# Session ses_abc123 completed!
```

### Ask follow-up questions

```bash
# Send a follow-up question to an existing session
orch-go ask ses_abc123 "what did you just do?"

# Output: Q&A complete for session: ses_abc123
```

## Event Logging

All events are logged to `~/.orch/events.jsonl` in JSONL format:

```json
{"type":"session.spawned","session_id":"ses_abc123","timestamp":1703001600,"data":{"prompt":"say hello","title":"orch-go-1703001600"}}
{"type":"session.status","session_id":"ses_abc123","timestamp":1703001601,"data":{"status":"idle"}}
{"type":"session.completed","session_id":"ses_abc123","timestamp":1703001602}
```

## Requirements

- OpenCode running at `http://127.0.0.1:4096` (default)
- macOS for desktop notifications (uses beeep library)

## API Patterns

Based on validated manual testing:

1. **Spawn**: `opencode run --attach http://127.0.0.1:4096 --format json --title "title" "prompt"`
2. **Q&A**: `opencode run --attach http://127.0.0.1:4096 --session ses_xxx --format json "question"`
3. **SSE**: `curl http://127.0.0.1:4096/event` (server.connected, session.status, etc.)
4. **Completion**: Watch for `session.status` changing from `busy` to `idle`
