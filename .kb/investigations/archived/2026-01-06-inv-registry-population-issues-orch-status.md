## Summary (D.E.K.N.)

**Delta:** The "registry.json" file doesn't exist - the actual file is `~/.orch/sessions.json` which IS correctly populated with orchestrator sessions.

**Evidence:** Verified `~/.orch/sessions.json` contains 11 sessions; `orch status --json` shows all orchestrator_sessions correctly; `~/.orch/registry.json` doesn't exist (was misconception from prior investigation).

**Knowledge:** The system has TWO registry mechanisms: (1) `sessions.json` for orchestrator sessions (correctly working), (2) `agent-registry.json` for legacy/archived agent tracking. The prior investigation's Gap #4 was based on a misunderstanding of the filename.

**Next:** Close as not-a-bug. Update prior investigation to correct the filename misconception. Document the registry architecture for future reference.

---

# Investigation: Registry Population Issues - orch status vs registry.json

**Question:** Why does `~/.orch/registry.json` appear empty while `orch status` shows sessions?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** Agent og-arch-registry-population-issues-06jan-c0d1
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: The "registry.json" File Doesn't Exist - Wrong Filename

**Evidence:** 
- `ls -la ~/.orch/` shows no file named `registry.json`
- The actual file for orchestrator sessions is `~/.orch/sessions.json`
- The file `~/.orch/agent-registry.json` exists but is a DIFFERENT system (legacy agent tracking)

**Source:** 
- `ls -la ~/.orch/` output - no `registry.json` present
- `pkg/session/registry.go:20-22` - `RegistryPath()` returns `~/.orch/sessions.json`

**Significance:** The problem statement was based on a filename misconception. The prior investigation (Gap #4) mentioned "registry.json" but the actual file is `sessions.json`.

---

### Finding 2: sessions.json IS Correctly Populated

**Evidence:** 
`~/.orch/sessions.json` contains 11 orchestrator sessions with full data:
```json
{
  "sessions": [
    {
      "workspace_name": "meta-orch-continue-previous-session-06jan-cacd",
      "session_id": "",
      "project_dir": "/Users/dylanconlin/Documents/personal/orch-go",
      "spawn_time": "2026-01-06T07:17:21.196228-08:00",
      "goal": "Continue from previous session...",
      "status": "completed"
    },
    // ... 10 more sessions
  ]
}
```

**Source:** `cat ~/.orch/sessions.json` - contains all expected data

**Significance:** The orchestrator session registry IS working correctly. The "population issue" doesn't exist.

---

### Finding 3: orch status Correctly Uses sessions.json

**Evidence:** 
`cmd/orch/status_cmd.go:626-627`:
```go
func getOrchestratorSessions(project string) []OrchestratorSessionInfo {
    registry := session.NewRegistry("")
    sessions, err := registry.ListActive()
```

The `NewRegistry("")` call uses `RegistryPath()` which returns `~/.orch/sessions.json`.

`orch status --json` output confirms:
```json
{
  "orchestrator_sessions": [
    {
      "workspace_name": "meta-orch-continue-meta-orch-06jan-2c9a",
      "goal": "...",
      "duration": "2h 55m",
      "project": "orch-go",
      "status": "active"
    },
    // ... more sessions
  ]
}
```

**Source:** 
- `cmd/orch/status_cmd.go:626-656`
- `pkg/session/registry.go:19-21`
- `orch status --json` output

**Significance:** The data flow is correct: sessions.json → Registry.ListActive() → orch status output.

---

### Finding 4: Two Separate Registry Systems (Source of Confusion)

**Evidence:** The codebase has TWO separate registry-like systems:

1. **Orchestrator Session Registry** (`pkg/session/registry.go`):
   - File: `~/.orch/sessions.json`
   - Purpose: Tracks orchestrator sessions (workspace name, goal, status)
   - Used by: `orch status` for "ORCHESTRATOR SESSIONS" section
   - Status: Working correctly

2. **Legacy Agent Registry** (`~/.orch/agent-registry.json`):
   - File: `~/.orch/agent-registry.json`
   - Purpose: Legacy tracking of ALL spawns (archived)
   - Used by: Nothing currently (legacy/archived)
   - Status: Contains old data from December 23

The comment in `session.go:4` confirms:
```go
// Unlike agent-registry which tracks ALL spawns, session
// only tracks spawns made during the current session.
```

**Source:**
- `pkg/session/session.go:2-5` - comment referencing agent-registry
- `~/.orch/agent-registry.json` - contains December 23 data
- `~/.orch/sessions.json` - contains current orchestrator sessions

**Significance:** The confusion in the prior investigation likely came from conflating these two systems. The "agent-registry" is legacy; the "session registry" (sessions.json) is the active system.

---

## Synthesis

**Key Insights:**

1. **No bug exists** - The "registry.json" file mentioned in the issue description doesn't exist because that's not the filename. The actual file `sessions.json` is correctly populated and `orch status` correctly reads it.

2. **Naming confusion between registries** - The codebase has evolved and now has two "registry" concepts: the legacy `agent-registry.json` (unused) and the current `sessions.json` (orchestrator sessions). The prior investigation's Gap #4 likely confused these.

3. **Empty session_id field is expected** - Many sessions have `session_id: ""` because orchestrators spawn via tmux (not headless), so the OpenCode session ID isn't captured at spawn time. This is a separate concern from "registry not populated."

**Answer to Investigation Question:**

The premise of the question was incorrect. There is no `~/.orch/registry.json` file. The actual orchestrator session registry is `~/.orch/sessions.json`, which IS correctly populated (11 sessions visible) and IS being correctly used by `orch status` (6 active sessions displayed). 

The confusion arose from Gap #4 in the prior investigation which mentioned "registry.json" - this was likely a misremembering or conflation with the legacy `agent-registry.json` file.

---

## Structured Uncertainty

**What's tested:**

- ✅ `sessions.json` exists and contains expected session data (verified: `cat ~/.orch/sessions.json`)
- ✅ `orch status` reads from sessions.json correctly (verified: `orch status --json` shows orchestrator_sessions)
- ✅ `registry.json` doesn't exist (verified: `ls -la ~/.orch/` shows no such file)
- ✅ Sessions are being registered on spawn (verified: code path in `spawn_cmd.go:1099`)

**What's untested:**

- ⚠️ Empty `session_id` fields might indicate a separate issue with OpenCode session capture
- ⚠️ Whether the legacy `agent-registry.json` should be cleaned up or removed

**What would change this:**

- If a `registry.json` file was recently deleted, the issue might have been valid at the time of reporting
- If there's code elsewhere expecting `registry.json`, it would indicate an incomplete migration

---

## Implementation Recommendations

**Purpose:** No implementation needed - this was not a bug.

### Recommended Approach ⭐

**Close as not-a-bug** - The reported issue was based on an incorrect filename.

**Why this approach:**
- The actual file `sessions.json` is working correctly
- `orch status` shows correct orchestrator session data
- No code path references a `registry.json` file

**Follow-up actions:**
1. Update prior investigation `.kb/investigations/2026-01-06-inv-workspace-session-architecture.md` Gap #4 to note this was a filename misconception
2. Consider whether the legacy `agent-registry.json` should be deprecated/removed

### Alternative Approaches Considered

**Option B: Investigate empty session_id fields**
- **Pros:** Would improve session tracking completeness
- **Cons:** Different issue from what was reported
- **When to use instead:** If session resumption for orchestrators is needed, this becomes relevant

---

## References

**Files Examined:**
- `cmd/orch/status_cmd.go:112-497` - runStatus implementation
- `cmd/orch/status_cmd.go:624-657` - getOrchestratorSessions function
- `pkg/session/registry.go` - Full registry implementation
- `pkg/session/session.go` - Session store with agent-registry reference

**Commands Run:**
```bash
# Check registry files
ls -la ~/.orch/ | grep registry
# Shows: agent-registry.json (legacy)

# Check sessions file
cat ~/.orch/sessions.json
# Shows: 11 sessions, correctly populated

# Check orch status output
orch status --json
# Shows: orchestrator_sessions array with 6 active sessions
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-06-inv-workspace-session-architecture.md` - Gap #4 was the source of this issue

---

## Investigation History

**2026-01-06 17:53:** Investigation started
- Initial question: Why does registry.json appear empty while orch status shows sessions?
- Context: Referenced from Gap #4 in workspace-session-architecture investigation

**2026-01-06 17:55:** Discovered filename mismatch
- Found that `registry.json` doesn't exist
- Actual file is `sessions.json`

**2026-01-06 17:58:** Investigation completed
- Status: Complete
- Key outcome: Not a bug - reported issue was based on incorrect filename
