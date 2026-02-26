<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** `orch status` showed different phases depending on which project directory it was run from because beads comments were looked up using cwd instead of the beads issue's actual project.

**Evidence:** From orch-go, phases showed correctly. From snap, phases showed `-` because snap's `.beads/` had no matching issues.

**Knowledge:** The beads ID prefix (e.g., `orch-go-xxxx`) encodes the project name, which can be used to derive the correct project directory for cross-project visibility.

**Next:** Fix implemented and verified - close issue.

---

# Investigation: Fix Orch Status Showing Different

**Question:** Why does `orch status` show different results (phases, tasks) depending on which project directory you run it from?

**Started:** 2026-01-05
**Updated:** 2026-01-05
**Owner:** Agent (spawned for orch-go-u5a5)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Beads comments lookup used current working directory

**Evidence:** In `status_cmd.go` lines 235-256, `beadsProjectDirs` was populated by looking up workspaces only from the current project's `.orch/workspace/` directory. For cross-project agents, if no workspace existed in the current project, the fallback was to use cwd for beads lookup.

**Source:** `cmd/orch/status_cmd.go:235-256`

**Significance:** When running from snap, agents spawned in orch-go had no workspaces in snap's directory, so their beads comments were incorrectly looked up in snap's `.beads/` (which doesn't have orch-go issues).

---

### Finding 2: OpenCode session.Directory is always "/" for spawned agents

**Evidence:** API query `curl http://localhost:4096/session | jq '.[] | {directory}'` showed all sessions have `directory: "/"`. This means we cannot rely on session.Directory to determine the correct project.

**Source:** OpenCode API response for `/session`

**Significance:** Strategy 1 (using session.Directory) doesn't work because OpenCode doesn't track the actual project directory. Need an alternative approach.

---

### Finding 3: Beads ID prefix encodes project name

**Evidence:** 
- `orch-go-u5a5` → project name "orch-go"
- `kb-cli-xrm` → project name "kb-cli"
- `orch-knowledge-untracked-xxx` → project name "orch-knowledge-untracked"

The `extractProjectFromBeadsID` function already exists and correctly parses this.

**Source:** `cmd/orch/shared.go:95-109`, beads ID examples from `bd ready`

**Significance:** We can derive the project directory from the beads ID prefix by checking known project locations (e.g., `~/Documents/personal/{projectName}`).

---

## Synthesis

**Key Insights:**

1. **Session directory is unreliable** - OpenCode's session.Directory is "/" for all spawned agents, making it useless for cross-project resolution.

2. **Beads ID prefix is the source of truth** - The project name is encoded in the beads ID, and this can be mapped to project directories using a convention-based lookup.

3. **Three-strategy fallback is robust** - (1) Session directory if valid, (2) Workspace lookup from current project, (3) Derive from beads ID prefix. This covers all cases.

**Answer to Investigation Question:**

The bug occurred because `orch status` looked up beads comments using the current working directory when it couldn't find a workspace for the agent. The fix adds a third strategy: deriving the project directory from the beads ID prefix (e.g., `orch-go-xxxx` → `~/Documents/personal/orch-go`). This ensures correct cross-project visibility regardless of which directory you run `orch status` from.

---

## Structured Uncertainty

**What's tested:**

- ✅ From orch-go: phases show correctly (verified: `./orch status --json | jq '.agents[] | {beads_id, phase}'`)
- ✅ From snap: phases now show correctly for orch-go-* agents (verified: `/tmp/orch-debug status --json`)
- ✅ From beads: phases show correctly (verified: `/tmp/orch-debug status --json`)
- ✅ All existing tests pass (verified: `go test ./cmd/orch/...`)

**What's untested:**

- ⚠️ Performance impact of checking multiple directory paths (should be negligible - just stat() calls)
- ⚠️ Projects in non-standard locations won't be found by convention

**What would change this:**

- If project naming conventions change (e.g., beads ID prefixes no longer match directory names)
- If projects are stored in locations not covered by `findProjectDirByName` candidates

---

## Implementation Recommendations

### Recommended Approach ⭐

**Three-strategy project directory resolution** - Already implemented.

**Why this approach:**
- Handles all known cases (session dir, workspace, beads ID prefix)
- Falls back gracefully when strategies fail
- Uses existing `extractProjectFromBeadsID` function

**Trade-offs accepted:**
- Projects in non-standard locations won't be found (acceptable - they're rare)
- "untracked" beads IDs default to cwd (acceptable - they have no project by design)

**Implementation sequence:**
1. Check session.Directory (quick, usually "/" so skipped)
2. Look up workspace from current project (finds local cross-project spawns)
3. Derive from beads ID prefix (finds any project in known locations)

---

## References

**Files Examined:**
- `cmd/orch/status_cmd.go` - Main status command implementation
- `pkg/verify/beads_api.go` - GetCommentsBatchWithProjectDirs
- `cmd/orch/shared.go` - extractProjectFromBeadsID

**Commands Run:**
```bash
# Verify bug reproduction
cd ~/Documents/personal/snap && ./orch status --json | jq '.agents[] | {beads_id, phase}'

# Test fix
cd ~/Documents/personal/snap && /tmp/orch-debug status --json | jq '.agents[] | {beads_id, phase}'

# Run tests
go test ./cmd/orch/... -count=1
```

---

## Investigation History

**2026-01-05 21:00:** Investigation started
- Initial question: Why different results from different directories?
- Context: Spawned for orch-go-u5a5 issue

**2026-01-05 21:15:** Root cause identified
- `beadsProjectDirs` was only populated from current project's workspaces
- Cross-project agents defaulted to cwd for beads lookup

**2026-01-05 21:30:** Fix implemented and verified
- Added `findProjectDirByName` to derive project dir from beads ID prefix
- Three-strategy fallback now handles all cases
- Verified fix from multiple project directories
