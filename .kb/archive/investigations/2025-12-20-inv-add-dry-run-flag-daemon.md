**TLDR:** Question: How to add --dry-run flag to daemon run command? Answer: Added --dry-run bool flag that triggers preview behavior when set, showing what would be processed without actually spawning agents. High confidence (95%) - straightforward flag addition following existing patterns.

---

# Investigation: Add --dry-run flag to daemon run command

**Question:** How to add --dry-run flag to daemon run command to preview what would be processed without spawning?

**Started:** 2025-12-20
**Updated:** 2025-12-20
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%)

---

## Findings

### Finding 1: Existing preview command provides the pattern

**Evidence:** The `daemon preview` subcommand already exists and shows what would be processed next without spawning. It uses `d.Preview()` which returns a `PreviewResult` with issue details and inferred skill.

**Source:** `cmd/orch/daemon.go:62-73` (daemonPreviewCmd) and `pkg/daemon/daemon.go:82-104` (Preview method)

**Significance:** The --dry-run flag can reuse the same Preview() method, providing consistent behavior with the existing preview subcommand.

---

### Finding 2: Flag infrastructure already in place

**Evidence:** The daemon run command already has a `--delay` flag using `daemonRunCmd.Flags().IntVar()`. Adding a bool flag follows the same pattern.

**Source:** `cmd/orch/daemon.go:85` - `daemonRunCmd.Flags().IntVar(&daemonDelay, "delay", 5, ...)`

**Significance:** No new infrastructure needed - just add another flag using the same pattern.

---

## Synthesis

**Key Insights:**

1. **Reuse existing Preview()** - The Preview() method in pkg/daemon already does exactly what dry-run needs - it shows what would be processed without actually spawning.

2. **Simple flag gating** - Just check if --dry-run is set at the start of runDaemonLoop() and call runDaemonDryRun() instead.

**Answer to Investigation Question:**

Added --dry-run flag by:

1. Adding `daemonDryRun bool` variable
2. Registering flag with `daemonRunCmd.Flags().BoolVar(&daemonDryRun, "dry-run", false, ...)`
3. Creating `runDaemonDryRun()` function that uses `d.Preview()` and displays results with "[DRY-RUN]" prefix
4. Gating runDaemonLoop() to call runDaemonDryRun() when flag is set

---

## Confidence Assessment

**Current Confidence:** Very High (95%)

**Why this level?**

Straightforward implementation following existing patterns. The Preview() method was already implemented and tested.

**What's certain:**

- ✅ Flag is registered correctly
- ✅ Dry-run mode uses Preview() which is tested
- ✅ Follows existing code patterns

**What's uncertain:**

- ⚠️ Haven't tested end-to-end with actual beads issues (requires beads setup)

---

## Implementation Details

**Changes made:**

1. Added `daemonDryRun bool` flag variable (line 79)
2. Registered flag in init() (line 88)
3. Updated command examples to include --dry-run (line 42)
4. Added check at start of runDaemonLoop() to call runDaemonDryRun() (lines 91-95)
5. Added runDaemonDryRun() function (lines 169-194)

**Files modified:**

- `cmd/orch/daemon.go`

---

## References

**Files Examined:**

- `cmd/orch/daemon.go` - CLI command implementation
- `pkg/daemon/daemon.go` - Daemon package with Preview() method
- `pkg/daemon/daemon_test.go` - Tests for daemon package

---

## Investigation History

**2025-12-20:** Investigation started

- Initial question: How to add --dry-run flag to daemon run command?
- Context: Currently only 'daemon preview' exists, need --dry-run on run command

**2025-12-20:** Investigation completed

- Final confidence: Very High (95%)
- Status: Complete
- Key outcome: Added --dry-run flag that triggers preview behavior when set
