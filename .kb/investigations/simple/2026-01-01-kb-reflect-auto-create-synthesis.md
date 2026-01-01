# Investigation: kb reflect auto-create synthesis issues

## Summary (D.E.K.N.)

**Delta:** The kb reflect --create-issue feature already existed but failed due to PATH issues when spawned agents couldn't find the bd binary.

**Evidence:** Running `kb reflect --type synthesis --create-issue` now shows `[ISSUE CREATED]` for 51-investigation dashboard topic; beads shows 5+ new synthesis issues with triage:review label.

**Knowledge:** OpenCode spawned agents inherit a minimal PATH that excludes ~/go/bin, ~/.bun/bin, and ~/.local/bin. Tool CLIs must implement fallback PATH resolution when invoking other tools.

**Next:** Close - implementation complete in kb-cli. Spawned agents can now auto-create synthesis issues.

---

**Question:** How do we implement kb reflect auto-creation of beads issues for synthesis candidates?

**Started:** 2026-01-01
**Updated:** 2026-01-01
**Owner:** Agent og-feat-kb-reflect-auto-01jan
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Feature Already Exists

**Evidence:** `kb reflect --help` shows `--create-issue` flag. When run, it attempted to create issues but failed with `exec: "bd": executable file not found in $PATH`.

**Source:** `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/reflect.go:534-539`

**Significance:** No new feature needed - just PATH resolution fix.

---

### Finding 2: PATH Issue Root Cause

**Evidence:** OpenCode server inherits minimal PATH. Symlinks at `~/.bun/bin/bd` exist but `~/.bun/bin` is not in the spawned agent's PATH.

**Source:** CLAUDE.md documents this as known issue with workaround.

**Significance:** This is a recurring pattern - any CLI that invokes other CLIs needs fallback PATH resolution.

---

### Finding 3: Fix Implementation

**Evidence:** Created `beads.go` helper with `findBdPath()` that checks:
1. Standard PATH via `exec.LookPath("bd")`
2. Fallback locations: `~/.bun/bin/bd`, `~/go/bin/bd`, `~/.local/bin/bd`, `~/Documents/personal/beads/build/bd`

Updated `reflect.go`, `link.go`, `context.go` to use `runBdCommand()` helper instead of direct `exec.Command("bd", ...)`.

**Source:** `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/beads.go`

**Significance:** All kb commands that invoke bd now work in spawned agent contexts.

---

## Structured Uncertainty

**What's tested:**

- ✅ kb reflect --type synthesis --create-issue creates issues (verified: ran command, saw [ISSUE CREATED])
- ✅ beads issues have correct title format and triage:review label (verified: bd list --status open | grep synthesize)
- ✅ Tests pass (verified: go test ./cmd/kb/... -run TestFindBdPath)

**What's untested:**

- ⚠️ Other kb commands using bd (kb link, kb context --stale) - not exercised but use same helper

**What would change this:**

- bd binary moves to unexpected location
- OpenCode changes PATH behavior

---

## References

**Files Modified:**
- `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/beads.go` - New helper for bd PATH resolution
- `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/beads_test.go` - Tests for helper
- `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/reflect.go` - Use runBdCommand helper
- `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/link.go` - Use runBdCommand helper
- `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/context.go` - Use runBdCommand helper
