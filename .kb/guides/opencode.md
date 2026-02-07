# OpenCode Integration Guide

**Purpose:** Single authoritative reference for orch-go's OpenCode integration. Read this before debugging OpenCode issues.

**Last verified:** 2026-01-08

**Synthesized from:** 24 investigations (2025-12-19 to 2026-01-08)

**Related guide:** `.kb/guides/opencode-plugins.md` - Comprehensive plugin system reference

---

## Overview

OpenCode is the agent execution layer for orch-go. Unlike beads (an external tool), OpenCode is **runtime infrastructure** that orch-go cannot function without. Every agent spawn, every session, and every SSE event flows through OpenCode.

This guide covers:
- HTTP API usage and endpoints
- Session management and lifecycle
- SSE monitoring and completion detection
- Common problems and their root causes
- Authentication and token handling

---

## Architecture

```
┌─────────────────┐     HTTP API      ┌─────────────────┐
│   orch-go CLI   │◄────────────────►│  OpenCode Server │
│ pkg/opencode/   │   (port 4096)     │  (local daemon)  │
└────────┬────────┘                   └────────┬─────────┘
         │                                     │
         │  SSE /event                         │
         ├─────────────────────────────────────┤
         │                                     │
         ▼                                     ▼
┌─────────────────┐                   ┌─────────────────┐
│ Monitor/Service │                   │ Anthropic Claude│
│ completion.go   │                   │    API          │
└─────────────────┘                   └─────────────────┘
```

**Key relationships:**
- **pkg/opencode/client.go** (728 lines) - HTTP REST client for session operations
- **pkg/opencode/sse.go** (159 lines) - SSE stream parsing for real-time events
- **pkg/opencode/monitor.go** (221 lines) - Session completion detection
- **Total integration:** ~3,600+ LoC, 12+ HTTP API endpoints

---

## How It Works

### Session Lifecycle

**What:** OpenCode sessions persist across restarts and are stored on disk.

**Key insight:** There are TWO types of session storage - in-memory (running) and disk (historical). The `x-opencode-directory` header controls which you query.

| Query Type | Header | Returns |
|------------|--------|---------|
| In-memory | None | Currently running sessions only (2-3) |
| Disk | `x-opencode-directory: /path/to/project` | All historical sessions (238+) |

### Spawn Modes

orch-go supports three spawn modes:

| Mode | Command | Use Case |
|------|---------|----------|
| **Headless** (default) | `orch spawn` | Automation, daemon, batch work |
| **Tmux** | `orch spawn --tmux` | Interactive monitoring, visual debugging |
| **Inline** | `orch spawn --inline` | Quick tasks, blocking execution |

**Headless is preferred** for:
- CI/CD automation
- Daemon-driven work (`orch daemon run`)
- Parallel agent spawning
- No TUI overhead

### Completion Detection

**What:** SSE events signal when agents finish work.

**Key insight:** Completion is detected via `session.status` event transitioning from `busy` to `idle`, NOT by session existence.

```
SSE Stream: /event
──────────────────────────────────────────
session.status { status: "busy" }    ← Agent working
message.part.updated                 ← Content streaming
session.status { status: "idle" }    ← Agent finished
```

---

## Key Concepts

| Concept | Definition | Why It Matters |
|---------|------------|----------------|
| **Disk session** | Session persisted to `~/.local/share/opencode/storage/` | Sessions never auto-delete; cleanup via `orch clean --verify-opencode` |
| **Session ID** | `ses_xxx` unique identifier | Required for API calls; stored in `.session_id` file in workspace |
| **x-opencode-directory** | HTTP header specifying project path | Controls session scoping - WITH header = disk, WITHOUT = memory |
| **SSE events** | Server-sent events from `/event` endpoint | Real-time monitoring without polling |
| **Plugin system** | TypeScript plugins for extensibility | Used for context injection, auth handling |

---

## HTTP API Quick Reference

### Working Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/session` | GET | List sessions |
| `/session` | POST | Create new session |
| `/session/{id}` | GET | Get session details |
| `/session/{id}` | DELETE | Delete session |
| `/session/{id}/message` | GET | Get message history |
| `/session/{id}/prompt_async` | POST | Send message (async) |
| `/event` | GET (SSE) | Real-time event stream |

### Invalid/Proxied Endpoints

These routes get proxied to `desktop.opencode.ai` and fail:

| Endpoint | Result |
|----------|--------|
| `/sessions` (plural) | ❌ "redirected too many times" |
| `/health` | ❌ "redirected too many times" |
| `/prompt_async` (root) | ❌ Wrong path |

**Critical:** Always use `/session` (singular), never `/sessions` (plural).

---

## Common Problems

### "The response redirected too many times"

**Cause:** Hitting an invalid endpoint that gets proxied to desktop.opencode.ai (a web app, not API).

**Fix:** Use correct endpoints:
- ✅ `/session` (singular)
- ❌ `/sessions` (plural) 
- ❌ `/health`

**NOT the fix:** This is NOT an authentication issue. Token refresh is handled automatically by OpenCode.

### Cross-project sessions show wrong directory

**Cause:** Sessions created via `orch spawn --workdir ~/other-project` have the orchestrator's directory (orch-go) instead of the target project directory.

**Impact:** Sessions unfindable via directory-based queries (`x-opencode-directory` header filtering).

**Root cause:** `cmd.Dir` is set but `BuildSpawnCommand` doesn't explicitly pass directory to OpenCode session creation.

**Status:** Bug identified (2026-01-06), fix pending. Workaround: Query all sessions and filter client-side.

### "No user message found in conversation"

**Cause:** OpenCode's `/prompt_async` endpoint doesn't await the prompt call, leading to race conditions.

**Fix options:**
1. Defensive error handling in orch-go (implemented - checks `session.error` events)
2. OpenCode fix needed: Add `await` to prompt call in server.ts:1240

**This is an OpenCode bug, not orch-go.**

### Session accumulation / memory leaks

**Cause:** Two issues in monitor.go:
1. `m.sessions` map never cleaned after completion
2. `reconnect()` spawns orphaned goroutines

**Fix:** 
1. Delete session from map after completion handler fires
2. Rewrite reconnect with proper channel lifecycle management

### Plugin crashes with exit code 133 (SIGTRAP)

**Cause:** Missing `@opencode-ai/plugin` dependency in project's `.opencode/` directory.

**Fix:**
```bash
cd /path/to/project/.opencode
bun add @opencode-ai/plugin
```

**Why:** `opencode-dev` (local development) skips auto-install. Production installs should auto-install.

### `orch clean --verify-opencode` deletes active session

**Cause:** Early implementation only checked workspace `.session_id` files, missing orchestrator/interactive sessions that don't have workspaces.

**Fix:** Two-tier active session detection:
1. Check session update timestamp (< 5 min = potentially active)
2. Call `IsSessionProcessing()` for recently active sessions
3. Skip deletion if session is actively processing

### Session accumulation (too many sessions, slow API)

**Cause:** Sessions persist indefinitely in `~/.local/share/opencode/storage/`.

**Symptoms:** Slow dashboard API, high memory usage, 600+ sessions.

**Fix:** Use `orch clean --sessions` to delete sessions older than N days (default: 7):
```bash
orch clean --sessions                    # Delete sessions > 7 days old
orch clean --sessions --sessions-days 3  # Delete sessions > 3 days old
orch clean --sessions --dry-run          # Preview what would be deleted
```

**Notes:**
- Active sessions (IsSessionProcessing check) are skipped
- 461 sessions deleted in initial test (627 → 166)

---

## Key Decisions (from investigations)

These are settled. Don't re-investigate:

- **OpenCode handles OAuth auto-refresh** - The anthropic-auth plugin refreshes tokens transparently at fetch time. No orch implementation needed.
- **OpenCode is runtime infrastructure, not external tool** - 3,600+ LoC integration, writes to auth.json, manages session lifecycle. Different from beads.
- **Standalone + API Discovery is the recommended spawn approach** - Spawn with standalone mode, discover session via API, interact via HTTP.
- **Sessions created by standalone TUI ARE visible via API** - Python's discover_opencode_session() proves this works.
- **x-opencode-directory header controls disk vs memory session listing** - WITH header = disk (238+), WITHOUT = memory (2-3).
- **pkg/opencode/ provides the right abstraction level** - No additional abstraction needed on top of existing package.
- **Plugin system is the bridge for principle mechanization** - See `.kb/guides/opencode-plugins.md` for comprehensive reference on Gates, Context Injection, and Observation patterns.
- **session.idle is deprecated** - Prefer `session.status` event with `status.type === "idle"` check. Still functional but will be removed.
- **OpenCode sessions share central storage** - All servers query same `~/.local/share/opencode/storage/`; `x-opencode-directory` is for filtering, not isolation.
- **Question tool is `question`, not `AskUserQuestion`** - Skills corrected to use proper JSON interface with questions array containing question/header/options.

---

## What Lives Where

| Thing | Location | Purpose |
|-------|----------|---------|
| OpenCode client | `pkg/opencode/client.go` | HTTP API methods |
| SSE parsing | `pkg/opencode/sse.go` | Event stream handling |
| Completion monitor | `pkg/opencode/monitor.go` | Session completion detection |
| Completion service | `pkg/opencode/service.go` | High-level completion tracking |
| Auth management | `pkg/account/account.go` | OAuth token handling |
| OpenCode auth file | `~/.local/share/opencode/auth.json` | Token storage (orch writes this) |
| Session storage | `~/.local/share/opencode/storage/` | Disk session persistence |
| Global config | `~/.config/opencode/opencode.jsonc` | Global OpenCode settings |
| Project config | `{project}/opencode.json` | Per-project settings |
| Global plugins | `~/.config/opencode/plugin/` | User-wide plugins |
| Project plugins | `.opencode/plugin/` | Project-specific plugins |

---

## Configuration

### Native Instructions (Static)

OpenCode supports static file loading via `instructions` array:

```json
// opencode.json or ~/.config/opencode/opencode.jsonc
{
  "instructions": ["~/.claude/CLAUDE.md", "./CLAUDE.md"]
}
```

**Limitation:** Static paths only. Dynamic context requires plugins.

### Plugin System (Dynamic)

For dynamic context injection, use the plugin system:

- **session.created** event - Inject context at session start
- **experimental.chat.system.transform** hook - Modify system prompt transparently

```typescript
// ~/.config/opencode/plugin/session-context.ts
export default {
  name: "session-context",
  hooks: {
    "session.created": async ({ session }) => {
      // Inject dynamic context
    }
  }
}
```

### Theme System

28 built-in themes with JSON-based configuration:

```json
{
  "$schema": "https://opencode.ai/theme.json",
  "defs": { "nord8": "#88C0D0" },
  "theme": {
    "primary": "nord8",
    "background": { "dark": "#2E3440", "light": "#ECEFF4" }
  }
}
```

**Theme locations (later overrides earlier):**
1. Built-in themes (embedded)
2. `~/.config/opencode/themes/*.json`
3. `<project>/.opencode/themes/*.json`

---

## Debugging Checklist

Before spawning an investigation about OpenCode issues:

1. **Check kb:** `kb context "opencode"`
2. **Check this doc:** You're reading it
3. **Check API directly:**
   ```bash
   curl http://127.0.0.1:4096/session  # Should return 200
   ```
4. **Check auth token:**
   ```bash
   cat ~/.local/share/opencode/auth.json  # Check expires timestamp
   ```
5. **Check SSE stream:**
   ```bash
   curl -N http://127.0.0.1:4096/event  # Should stream events
   ```
6. **Check orch-go client code:**
   ```bash
   rg "ServerURL" pkg/opencode/ --type go  # Verify endpoint paths
   ```

If those don't answer your question, then investigate. But update this doc with what you learn.

---

## References

### Related Investigations (Synthesized)

| Date | Investigation | Key Finding |
|------|--------------|-------------|
| 2025-12-19 | OpenCode POC | Go POC validates spawn/monitor/ask pattern |
| 2025-12-19 | Client Package | pkg/opencode with 93.2% test coverage |
| 2025-12-20 | Integration Tradeoffs | Standalone + API Discovery recommended |
| 2025-12-20 | Refactor orch tail | API-based tail via `/session/{id}/message` |
| 2025-12-20 | Native Context Loading | Instructions static, plugins for dynamic |
| 2025-12-21 | Disk Session Cleanup | `--verify-opencode` flag for cleanup |
| 2025-12-23 | API Redirect Loop | Invalid endpoints proxied upstream |
| 2025-12-23 | oc Command Crash | Missing plugin dependency causes SIGTRAP |
| 2025-12-24 | Ecosystem Audit Addendum | OpenCode is runtime infrastructure |
| 2025-12-24 | Clean Verify-Opencode Fix | Active session detection added |
| 2025-12-25 | Crashes No User Message | OpenCode bug - missing await |
| 2025-12-26 | Auto-Refresh Tokens | OpenCode handles this automatically |
| 2025-12-26 | Session Accumulation | Monitor memory/goroutine leaks fixed |
| 2025-12-26 | Health Endpoint Redirect | Wrong endpoint tested (/sessions vs /session) |
| 2025-12-26 | Theme Selection System | 28 themes, JSON-based, hierarchical |
| 2025-12-26 | Port Theme System | Full theme system ported to dashboard |
| 2026-01-06 | Cross-Project Sessions | Sessions share storage; directory bug in spawn |
| 2026-01-06 | Session Cleanup Mechanism | `orch clean --sessions` implemented |
| 2026-01-07 | Question Tool Correction | Tool is `question` not `AskUserQuestion` |
| 2026-01-08 | Plugin Capabilities | Full hook analysis, 3 mechanization patterns |
| 2026-01-08 | Session Compaction | `experimental.session.compacting` preserves context |
| 2026-01-08 | Constraint Surfacing | Enhanced guarded-files plugin with kb context |
| 2026-01-08 | Event Reliability | file.edited reliable; session.idle deprecated |
| 2026-01-08 | Plugin Guide | Authoritative `.kb/guides/opencode-plugins.md` created |

### Source Code

- **pkg/opencode/** - Core OpenCode integration (client, SSE, monitor)
- **pkg/account/** - OAuth token management
- **cmd/orch/main.go** - CLI commands using OpenCode

### External

- **OpenCode API docs:** https://opencode.ai/docs/server
- **OpenCode source:** https://github.com/sst/opencode
- **OpenAPI spec:** http://127.0.0.1:4096/doc

---

## History

- **2026-01-06:** Created by synthesizing 16 investigations spanning 2025-12-19 to 2025-12-26
- **2026-01-08:** Updated with 8 new investigations (2026-01-06 to 2026-01-08), added plugin system decisions and new common problems
- **Evolution:** From POC (Dec 19) → full HTTP client → SSE monitoring → session cleanup → theme system → plugin system for principle mechanization (Jan 08)
