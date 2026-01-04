<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added `resolveShortBeadsID()` function to spawn_cmd.go that resolves short beads IDs to full IDs before generating SPAWN_CONTEXT.md.

**Evidence:** Build passes, all tests pass, manual testing confirms `oux7` → `orch-go-oux7` resolution works.

**Knowledge:** The beads RPC client already has `ResolveID()` method; the fix only required calling it from `determineBeadsID()`.

**Next:** Commit and close - fix is complete and verified.

---

# Investigation: Fix Short Beads Id Resolution

**Question:** How to fix short beads ID resolution in spawn so SPAWN_CONTEXT.md contains full IDs that `bd comment` can use?

**Started:** 2026-01-03
**Updated:** 2026-01-03
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: ResolveID RPC operation already exists

**Evidence:** The beads client already has `ResolveID(partialID string)` method at `pkg/beads/client.go:625-641` and the RPC operation `OpResolveID = "resolve_id"` at `pkg/beads/types.go:29`.

**Source:** `pkg/beads/client.go:625-641`, `pkg/beads/types.go:29`

**Significance:** No need to implement ID resolution logic - just need to call the existing method from spawn.

---

### Finding 2: determineBeadsID passed short IDs directly without resolution

**Evidence:** The function at `cmd/orch/spawn_cmd.go:1189-1210` returned `spawnIssue` directly without any resolution:
```go
if spawnIssue != "" {
    return spawnIssue, nil  // Was returning short ID directly
}
```

**Source:** `cmd/orch/spawn_cmd.go:1192-1195` (before fix)

**Significance:** This was the root cause - short IDs like "57dn" were passed to SPAWN_CONTEXT.md without being resolved to "orch-go-57dn".

---

### Finding 3: Fix is minimal - add resolveShortBeadsID helper

**Evidence:** Created `resolveShortBeadsID(id string)` function that:
1. Tries RPC client's `ResolveID()` first
2. Falls back to `FallbackShow()` which also resolves short IDs
3. Returns original ID with warning if resolution fails (graceful degradation)

**Source:** `cmd/orch/spawn_cmd.go:1213-1244` (after fix)

**Significance:** The fix is minimal and robust - handles both RPC and CLI fallback paths.

---

## Synthesis

**Key Insights:**

1. **Existing infrastructure was sufficient** - The beads package already had ID resolution capability, it just wasn't being used in spawn.

2. **Graceful degradation** - The fix returns the original ID with a warning if resolution fails, rather than blocking the spawn.

3. **Consistent with other orch-go patterns** - Uses the same RPC-first, CLI-fallback pattern used elsewhere in the codebase.

**Answer to Investigation Question:**

The fix required adding a `resolveShortBeadsID()` helper function and calling it from `determineBeadsID()` when `spawnIssue` is provided. The helper leverages the existing `beads.Client.ResolveID()` method for RPC resolution with `beads.FallbackShow()` as a fallback.

---

## Structured Uncertainty

**What's tested:**

- ✅ Build passes (`go build ./cmd/orch` succeeds)
- ✅ All tests pass (`go test ./...` shows all packages pass)
- ✅ Short ID resolution works (manual test: `oux7` → `orch-go-oux7`)
- ✅ Full ID resolution is no-op (manual test: `orch-go-oux7` → `orch-go-oux7`)

**What's untested:**

- ⚠️ End-to-end spawn with short ID (requires full agent spawn)
- ⚠️ Cross-project ID resolution (when using `--workdir`)
- ⚠️ Behavior when beads daemon is not running (should fall back to CLI)

**What would change this:**

- Finding would be wrong if beads CLI changed its short ID resolution behavior
- Finding would be incomplete if there are other code paths that produce short IDs

---

## Implementation Recommendations

**Purpose:** Document the implementation for future reference.

### Recommended Approach ⭐ (Implemented)

**Resolve short IDs in determineBeadsID** - Before generating SPAWN_CONTEXT, resolve any short beads ID to its full form.

**Why this approach:**
- Fixes the root cause at the source (spawn time, not agent time)
- Agents get correct IDs in SPAWN_CONTEXT, no guessing needed
- Minimal code change (single helper function)

**Trade-offs accepted:**
- Adds one beads lookup at spawn time (trivial overhead)
- Requires the issue to exist before spawning (already true for `--issue` flag)

**Implementation sequence:**
1. Add `resolveShortBeadsID()` helper function
2. Call it from `determineBeadsID()` when `spawnIssue != ""`
3. Handle errors gracefully (return original ID with warning)

---

## References

**Files Modified:**
- `cmd/orch/spawn_cmd.go` - Added `resolveShortBeadsID()` and modified `determineBeadsID()`

**Files Examined:**
- `pkg/beads/client.go:625-641` - Existing `ResolveID()` method
- `pkg/beads/types.go:29,254-257` - `OpResolveID` constant and `ResolveIDArgs` type
- `.kb/investigations/2026-01-03-inv-agents-report-phase-complete-via.md` - Prior investigation

**Commands Run:**
```bash
# Build verification
/opt/homebrew/bin/go build ./cmd/orch

# Test verification
/opt/homebrew/bin/go test ./... 

# Manual testing of short ID resolution
/opt/homebrew/bin/go run /tmp/test_resolve.go oux7
# Output: orch-go-oux7
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-03-inv-agents-report-phase-complete-via.md` - Root cause investigation that identified this bug

---

## Investigation History

**2026-01-03:** Investigation started
- Initial question: How to fix short beads ID resolution in spawn?
- Context: Prior investigation identified that short IDs in SPAWN_CONTEXT cause `bd comment` failures

**2026-01-03:** Found existing ResolveID method
- Discovered beads client already has ResolveID() method
- No new logic needed, just integration

**2026-01-03:** Implementation complete
- Added `resolveShortBeadsID()` helper function
- Modified `determineBeadsID()` to call it
- All tests pass, manual verification successful
- Status: Complete
