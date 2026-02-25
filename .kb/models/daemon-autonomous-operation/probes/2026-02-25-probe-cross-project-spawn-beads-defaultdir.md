# Probe: Cross-Project Spawn Fails Due to beads.DefaultDir Not Set

**Model:** daemon-autonomous-operation
**Date:** 2026-02-25
**Status:** Complete

---

## Question

The daemon model documents failure modes for capacity starvation, duplicate spawns, and skill inference mismatch. Does the daemon handle cross-project spawning correctly? Specifically: when `SpawnWork` calls `orch work --workdir`, do all beads lookups in `runWork()` use the target project's database?

---

## What I Tested

1. Confirmed bug reproduction: `bd show toolshed-164` from orch-go directory → "no issue found"
2. Traced code path: `SpawnWork` → `orch work --workdir X beadsID` → `runWork()` → `verify.GetIssue(beadsID)` → `beads.FindSocketPath("")` → uses CWD (orch-go), not workdir
3. Verified fix: `beads.DefaultDir` controls `FindSocketPath("")` default directory (client.go:147)
4. Verified daemon preview without ProjectRegistry shows only local issues; with registry shows all projects

```bash
# Reproduction
bd show toolshed-164 --json  # From orch-go: "no issue found"
cd /path/to/toolshed && bd show toolshed-164 --json  # Works

# Verification after fix
go run ./cmd/orch/ daemon preview  # Shows toolshed-*, specs-platform-*, bd-* issues
```

---

## What I Observed

- `runWork()` calls `verify.GetIssue()` at line 396 BEFORE consulting `spawnWorkdir`
- `verify.GetIssue()` → `beads.FindSocketPath("")` → defaults to `os.Getwd()` → orch-go's `.beads/bd.sock`
- Cross-project issue lookup fails → `runWork` returns error → daemon records spawn failure
- This repeats every poll cycle (user reported 103 consecutive failures)
- Secondary: `daemon preview` and `daemon dry-run` didn't initialize ProjectRegistry, so cross-project issues were invisible in diagnostic output
- The daemon's internal flow (status checks, status updates, workdir passing) was correct — the bug was in `orch work`, the command it shells out to

---

## Model Impact

- [x] **Extends** model with: New failure mode — "Cross-project spawn CWD mismatch"

### New Failure Mode: Cross-Project Spawn CWD Mismatch

**What happens:** Daemon polls cross-project issues correctly (via ProjectRegistry + ListReadyIssuesMultiProject), marks them in_progress, and passes --workdir to `orch work`. But `orch work` uses the daemon's CWD for beads lookups before consulting --workdir.

**Root cause:** `runWork()` calls `verify.GetIssue(beadsID)` before any --workdir-aware code. All beads calls default to `FindSocketPath("")` which uses `os.Getwd()` or `beads.DefaultDir`.

**Impact:** 100% failure rate for cross-project spawns. All cross-project issues silently dropped on every poll cycle.

**Fix:** Set `beads.DefaultDir` from `--workdir` at the start of `runWork()`, before any beads calls.

**Prevention:** Any beads-dependent function called from a cross-project context must either accept a projectDir parameter or ensure `beads.DefaultDir` is set.

---

## Notes

- `CheckBlockingDependencies` in `NextIssueExcluding` also doesn't pass projectDir for cross-project issues, but this is fail-open (continues without dependency check) rather than fail-closed
- Content-aware dedup (`FindInProgressByTitle`) only checks local beads database — not a bug currently but a blind spot for cross-project dedup
