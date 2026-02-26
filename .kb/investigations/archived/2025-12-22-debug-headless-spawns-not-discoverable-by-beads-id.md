<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** `runTail()` and `runQuestion()` used a naive lookup `strings.Contains(entry.Name(), beadsID)` that fails for headless spawns because workspace names don't contain beads IDs.

**Evidence:** Workspace names are generated as `og-{prefix}-{task-slug}-{date}` (e.g., `og-debug-headless-spawns-not-22dec`), never containing the beads ID. The beads ID is only in SPAWN_CONTEXT.md.

**Knowledge:** `findWorkspaceByBeadsID()` correctly scans SPAWN_CONTEXT.md for the authoritative beads issue declaration, but the duplicate lookup logic in `runTail()` and `runQuestion()` didn't use it.

**Next:** Fix applied - replaced naive lookups with `findWorkspaceByBeadsID()` calls. Merged.

**Confidence:** Very High (95%) - Root cause identified, fix verified, all tests pass.

---

# Investigation: Headless Spawns Not Discoverable by Beads ID

**Question:** Why can't headless agents be found by their beads ID after spawning?

**Started:** 2025-12-22
**Updated:** 2025-12-22
**Owner:** Agent og-debug-headless-spawns-not-22dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%)

---

## Findings

### Finding 1: Workspace names never contain beads ID

**Evidence:** `GenerateWorkspaceName()` in `pkg/spawn/config.go:55-81` creates names in format `og-{prefix}-{task-slug}-{date}`:
```go
return fmt.Sprintf("og-%s-%s-%s", prefix, slug, date)
```
Example: `og-debug-headless-spawns-not-22dec` - no beads ID present.

**Source:** `pkg/spawn/config.go:55-81`

**Significance:** Any lookup using `strings.Contains(entry.Name(), beadsID)` will NEVER match because the beads ID isn't in the directory name.

---

### Finding 2: runTail() used naive lookup that can't find headless agents

**Evidence:** `runTail()` at lines 415-428 iterated workspace directories checking:
```go
if entry.IsDir() && strings.Contains(entry.Name(), beadsID) {
```
This check fails for standard workspace names.

**Source:** `cmd/orch/main.go:415-428` (before fix)

**Significance:** `orch tail orch-go-xxxx` failed to find headless agents because their workspace directories don't contain the beads ID.

---

### Finding 3: findWorkspaceByBeadsID() correctly scans SPAWN_CONTEXT.md

**Evidence:** `findWorkspaceByBeadsID()` at lines 2019-2063 correctly searches for the authoritative beads issue declaration:
```go
// Look for "spawned from beads issue:" pattern in SPAWN_CONTEXT.md
if strings.Contains(lineLower, "spawned from beads issue:") {
    if strings.Contains(line, beadsID) {
        return dirPath, dirName
    }
```

**Source:** `cmd/orch/main.go:2015-2063`

**Significance:** The correct lookup function existed but wasn't being used by `runTail()` and `runQuestion()`.

---

## Synthesis

**Key Insights:**

1. **Beads ID is only in SPAWN_CONTEXT.md** - Workspace names are designed for human readability (task-based), not machine lookup. The beads ID lives in SPAWN_CONTEXT.md.

2. **Duplicate lookup logic diverged** - `resolveSessionID()` correctly uses `findWorkspaceByBeadsID()`, but `runTail()` and `runQuestion()` had copied a naive lookup that only works when beads ID is in the directory name.

3. **Tmux spawns work accidentally** - Tmux window names include the beads ID (format: `emoji workspace-name [beads-id]`), so the tmux fallback path worked. Headless spawns have no tmux window, exposing the bug.

**Answer to Investigation Question:**

Headless agents couldn't be found by beads ID because `runTail()` and `runQuestion()` used `strings.Contains(entry.Name(), beadsID)` to search workspace directories. Since workspace names are task-based (like `og-debug-task-22dec`), not beads-based, this never matched. The fix is to use `findWorkspaceByBeadsID()` which correctly scans SPAWN_CONTEXT.md for the authoritative beads issue declaration.

---

## Confidence Assessment

**Current Confidence:** Very High (95%)

**Why this level?**

Root cause identified through code tracing, fix implemented and verified, all tests pass.

**What's certain:**

- ✅ Workspace names don't contain beads IDs (verified in `GenerateWorkspaceName()`)
- ✅ `findWorkspaceByBeadsID()` correctly scans SPAWN_CONTEXT.md (verified by existing tests)
- ✅ Fix makes `orch tail orch-go-ttbc` work (smoke tested)

**What's uncertain:**

- ⚠️ Other commands might have similar duplicate lookup logic (but `resolveSessionID()` is the main path)

---

## Implementation (Completed)

**Changes made:**

1. `cmd/orch/main.go:runTail()` - Replaced naive workspace lookup with `findWorkspaceByBeadsID()` call
2. `cmd/orch/main.go:runQuestion()` - Replaced naive workspace lookup with `findWorkspaceByBeadsID()` call
3. `cmd/orch/main_test.go` - Added documentation comment explaining headless spawn scenario

**Verification:**
- All tests pass: `go test ./...`
- Smoke test: `orch tail orch-go-ttbc` returns output via API

---

## References

**Files Examined:**
- `cmd/orch/main.go` - Contains runTail(), runQuestion(), findWorkspaceByBeadsID()
- `pkg/spawn/config.go` - Contains GenerateWorkspaceName()
- `pkg/spawn/context.go` - Contains SPAWN_CONTEXT template
- `cmd/orch/main_test.go` - Contains TestFindWorkspaceByBeadsID

**Commands Run:**
```bash
# Verify lookup tests pass
go test ./cmd/orch/ -run "TestFindWorkspaceByBeadsID" -v

# Build and smoke test
go build -o /tmp/orch-test ./cmd/orch
/tmp/orch-test tail orch-go-ttbc
```

---

## Investigation History

**2025-12-22 12:15:** Investigation started
- Initial question: Why can't headless agents be found by beads ID?
- Context: Discovered during registry removal validation

**2025-12-22 12:25:** Root cause identified
- `runTail()` and `runQuestion()` use naive lookup that can't find standard workspaces
- `findWorkspaceByBeadsID()` exists and works correctly

**2025-12-22 12:30:** Fix implemented and verified
- Replaced naive lookups with `findWorkspaceByBeadsID()` calls
- All tests pass, smoke test confirms fix works
