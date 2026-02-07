# Model: Follow Orchestrator Mechanism

**Domain:** Dashboard / Tmux / Context Switching
**Last Updated:** 2026-01-15
**Synthesized From:** Investigations on dashboard context following, Ghostty window switching, tmux socket detection, and lsof fallback implementation

---

## Summary (30 seconds)

The "follow orchestrator" mechanism keeps the dashboard and workers Ghostty window synchronized with the orchestrator's current project context. Two independent systems work together: the **dashboard polls `/api/context`** to filter agents by project, and the **tmux `after-select-window` hook** switches the workers Ghostty to the matching `workers-{project}` session. Both rely on detecting the orchestrator pane's working directory, with an lsof fallback for when `#{pane_current_path}` is empty (e.g., running Claude Code).

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         ORCHESTRATOR TMUX SESSION                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐                         │
│  │ orch-go    │  │ price-watch │  │ specs-plat  │  ← Windows               │
│  │ window     │  │ window      │  │ window      │                          │
│  └──────┬──────┘  └─────────────┘  └─────────────┘                         │
│         │ active                                                            │
│         ▼                                                                   │
│  ┌──────────────────────────────────────────────────────────────┐          │
│  │ Pane running Claude Code (or shell)                          │          │
│  │ Process CWD: /Users/dylan/Documents/personal/orch-go         │          │
│  └──────────────────────────────────────────────────────────────┘          │
└─────────────────────────────────────────────────────────────────────────────┘
                              │
          ┌───────────────────┼───────────────────┐
          │                   │                   │
          ▼                   ▼                   ▼
┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│ DETECTION LAYER  │  │ DASHBOARD FOLLOW │  │ GHOSTTY FOLLOW   │
│                  │  │                  │  │                  │
│ 1. Try tmux      │  │ GET /api/context │  │ tmux hook:       │
│    #{pane_       │  │     ↓            │  │ after-select-    │
│    current_path} │  │ Returns:         │  │ window           │
│                  │  │ {                │  │     ↓            │
│ 2. If empty,     │  │   project:       │  │ sync-workers-    │
│    fallback to   │  │   "orch-go",     │  │ session.sh       │
│    lsof -p PID   │  │   included:      │  │     ↓            │
│                  │  │   ["orch-go",    │  │ tmux switch-     │
│ 3. Walk up to    │  │    "beads"...]   │  │ client to        │
│    find .orch/   │  │ }                │  │ workers-orch-go  │
└──────────────────┘  └──────────────────┘  └──────────────────┘
          │                   │                   │
          │                   ▼                   ▼
          │           ┌──────────────────┐  ┌──────────────────┐
          │           │ Dashboard UI     │  │ Right Ghostty    │
          │           │ filters agents   │  │ shows workers-   │
          │           │ to orch-go       │  │ orch-go session  │
          │           └──────────────────┘  └──────────────────┘
          │
          └──► Both mechanisms use same detection layer
```

---

## Core Mechanism

### 1. CWD Detection (Shared Layer)

Both follow mechanisms need to know the orchestrator's current working directory.

**Primary method:** `tmux display-message -t orchestrator -p '#{pane_current_path}'`

**Problem:** When the pane runs Claude Code (or other programs that don't update shell PWD tracking), `#{pane_current_path}` returns empty.

**Fallback method:** Query process CWD directly via `lsof`:
```bash
PANE_PID=$(tmux display-message -t orchestrator -p '#{pane_pid}')
lsof -p "$PANE_PID" | awk '/cwd/ {print $NF}'
```

**Implementation locations:**
- Shell script: `~/.local/bin/sync-workers-session.sh` (lines 20-30)
- Go code: `pkg/tmux/follower.go` → `GetTmuxCwd()` + `getPaneCwdViaLsof()`

### 2. Project Detection

Once CWD is known, walk up the directory tree to find project root:

```
/Users/dylan/Documents/personal/orch-go/cmd/orch/
                                        ↑ check for .orch/ - NO
/Users/dylan/Documents/personal/orch-go/cmd/
                                        ↑ check for .orch/ - NO
/Users/dylan/Documents/personal/orch-go/
                                        ↑ check for .orch/ - YES → project root
```

**Project indicators:** `.orch/` or `.beads/` directory

**Project name:** `basename` of project root directory

### 3. Multi-Project Configs

Some projects include related projects in their context (e.g., orch-go includes beads, kb-cli, etc.):

```go
// pkg/tmux/follower.go
func DefaultMultiProjectConfigs() map[string][]string {
    return map[string][]string{
        "orch-go": {"orch-go", "orch-cli", "beads", "kb-cli", "orch-knowledge", "opencode"},
        // ...
    }
}
```

The `/api/context` response includes `included_projects` for dashboard filtering.

---

## Dashboard Follow Mechanism

### How It Works

1. Dashboard polls `GET /api/context` every ~2 seconds
2. API calls `GetTmuxCwd("orchestrator")` (with lsof fallback)
3. API walks up to find project directory
4. API returns `{project, project_dir, cwd, included_projects}`
5. Dashboard filters agent list to show only matching projects

### API Endpoint

**Location:** `cmd/orch/serve_context.go`

**Response:**
```json
{
  "cwd": "/Users/dylan/Documents/personal/orch-go",
  "project_dir": "/Users/dylan/Documents/personal/orch-go",
  "project": "orch-go",
  "included_projects": ["orch-go", "orch-cli", "beads", "kb-cli", "orch-knowledge", "opencode"]
}
```

### Cache Behavior

- **TTL:** 1 second (short because context changes frequently)
- **Location:** `globalContextCache` in `serve_context.go`
- After switching orchestrator windows, dashboard updates within ~1-2 seconds

---

## Ghostty Follow Mechanism

### How It Works

1. User switches windows in orchestrator tmux session
2. Tmux fires `after-select-window` hook
3. Hook runs `~/.local/bin/sync-workers-session.sh`
4. Script detects new project from pane CWD
5. Script switches workers Ghostty client to `workers-{project}` session

### Tmux Hook Configuration

**Location:** `~/.tmux.conf.local`

```bash
set-hook -g after-select-window 'run-shell -b ~/.local/bin/sync-workers-session.sh'
```

**Verify enabled:**
```bash
tmux show-hooks -g | grep after-select-window
```

### Sync Script Logic

**Location:** `~/.local/bin/sync-workers-session.sh`

```bash
# 1. Only run if in orchestrator session
CURRENT_SESSION=$(tmux display-message -p '#{session_name}')
[[ "$CURRENT_SESSION" != "orchestrator" ]] && exit 0

# 2. Get pane CWD (with lsof fallback)
PANE_CWD=$(tmux display-message -p '#{pane_current_path}')
if [[ -z "$PANE_CWD" ]]; then
    PANE_PID=$(tmux display-message -p '#{pane_pid}')
    PANE_CWD=$(lsof -p "$PANE_PID" | awk '/cwd/ {print $NF}')
fi

# 3. Find project root (walk up to .orch/)
PROJECT_ROOT=$(find_project_root "$PANE_CWD")
PROJECT_NAME=$(basename "$PROJECT_ROOT")
TARGET_SESSION="workers-${PROJECT_NAME}"

# 4. Find workers client and switch it
WORKERS_TTY=$(find workers client TTY)
tmux switch-client -c "$WORKERS_TTY" -t "$TARGET_SESSION"
```

### Session Requirements

For Ghostty follow to work:
- `workers-{project}` session must exist (created by `orch spawn --tmux`)
- A Ghostty window must be attached to some `workers-*` session
- Tmux hook must be enabled

---

## Failure Modes

### Failure 1: Empty pane_current_path (Claude Code)

**Symptom:** Follow doesn't work when orchestrator window is running Claude Code

**Root cause:** Claude Code doesn't update tmux's shell PWD tracking

**Fix:** lsof fallback (implemented 2026-01-15)

**Verify:**
```bash
# Should return empty
tmux display-message -t orchestrator -p '#{pane_current_path}'

# Should return actual CWD
PANE_PID=$(tmux display-message -t orchestrator -p '#{pane_pid}')
lsof -p "$PANE_PID" | awk '/cwd/ {print $NF}'
```

### Failure 2: Wrong Tmux Socket (Overmind)

**Symptom:** Follow works outside overmind but fails when `orch serve` runs inside overmind

**Root cause:** Overmind creates its own tmux server; commands without `-S` flag target wrong server

**Fix:** Socket detection in `pkg/tmux/tmux.go` (implemented 2026-01-15)

**Verify:**
```bash
# If inside overmind, this fails:
tmux display-message -t orchestrator -p '#{window_index}'

# This works:
tmux -S /tmp/tmux-501/default display-message -t orchestrator -p '#{window_index}'
```

### Failure 3: Workers Session Doesn't Exist

**Symptom:** Orchestrator window switches, but Ghostty doesn't follow

**Root cause:** `workers-{project}` session not created yet

**Fix:** Spawn an agent with `--tmux` to create the session:
```bash
orch spawn --tmux investigation "create workers session" --workdir /path/to/project
```

### Failure 4: Cache Serving Stale Data

**Symptom:** Dashboard shows old project for ~1-2 seconds after switching

**Root cause:** Context cache TTL (1 second)

**This is expected behavior.** Dashboard will update after cache expires.

### Failure 5: Tmux Hook Disabled

**Symptom:** Ghostty never follows orchestrator

**Root cause:** Hook removed from tmux config

**Fix:** Re-add to `~/.tmux.conf.local`:
```bash
set-hook -g after-select-window 'run-shell -b ~/.local/bin/sync-workers-session.sh'
tmux source-file ~/.tmux.conf.local  # Reload
```

---

## Debugging Checklist

### Dashboard Not Following

1. **Check API response:**
   ```bash
   curl -sk https://localhost:3348/api/context | jq .
   ```

2. **If empty/wrong project:**
   - Check orchestrator window is active: `tmux list-windows -t orchestrator`
   - Check pane CWD detection: run commands from "Failure 1" section
   - Check orch serve is using new binary: `overmind restart api`

3. **If API correct but dashboard wrong:**
   - Hard refresh browser (Cmd+Shift+R)
   - Check dashboard is polling `/api/context`

### Ghostty Not Following

1. **Check hook is enabled:**
   ```bash
   tmux show-hooks -g | grep after-select-window
   ```

2. **Test script manually:**
   ```bash
   bash -x ~/.local/bin/sync-workers-session.sh
   ```

3. **Check workers session exists:**
   ```bash
   tmux list-sessions | grep workers
   ```

4. **Check workers client is attached:**
   ```bash
   tmux list-clients -F '#{client_tty} #{session_name}' | grep workers
   ```

---

## Configuration Files

| File | Purpose |
|------|---------|
| `~/.tmux.conf.local` | Tmux hook configuration |
| `~/.local/bin/sync-workers-session.sh` | Ghostty follow script |
| `cmd/orch/serve_context.go` | Dashboard context API |
| `pkg/tmux/follower.go` | `GetTmuxCwd()` + lsof fallback |
| `pkg/tmux/tmux.go` | Socket detection for overmind |

---

## Related Artifacts

**Investigations:**
- `2026-01-07-inv-dashboard-beads-follow-orchestrator-tmux.md` - Original implementation
- `2026-01-14-inv-dashboard-follow-orchestrator-broken-implemented.md` - Follow mode debugging
- `2026-01-15-inv-fix-tmux-socket-path-orch.md` - Overmind socket fix

**Models:**
- `escape-hatch-visibility-architecture.md` - Dual-window Ghostty setup (prerequisite for Ghostty follow)

**Guides:**
- `dashboard.md` - Dashboard overview
- `tmux-spawn-guide.md` - Workers session creation

---

## Evolution

### 2026-01-07: Initial Implementation
- Dashboard context API created
- Ghostty sync script created
- Basic `#{pane_current_path}` detection

### 2026-01-15: lsof Fallback + Socket Detection
- Added lsof fallback for Claude Code panes (empty pane_current_path)
- Added socket detection for overmind context (wrong tmux server)
- Both fixes applied to shell script and Go code
