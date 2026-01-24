## Summary (D.E.K.N.)

**Delta:** Added --cross-project flag to daemon run, preview, and once commands enabling multi-project polling.

**Evidence:** Build successful, all 51 daemon tests pass, help text shows new flag correctly.

**Knowledge:** CrossProject field already existed in daemon.Config but wasn't wired to CLI flags.

**Next:** Close - implementation complete, ready for orchestrator review.

**Promote to Decision:** recommend-no (straightforward flag addition, no architectural changes)

---

# Investigation: Add Cross Project Flag Daemon

**Question:** How to add --cross-project flag to daemon commands?

**Started:** 2026-01-21
**Updated:** 2026-01-21
**Owner:** Worker
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: CrossProject config field already exists

**Evidence:** daemon.Config struct at pkg/daemon/daemon.go:37-40 already has CrossProject bool field

**Source:** `pkg/daemon/daemon.go:37-40`

**Significance:** No changes needed to daemon package, only CLI wiring required

---

### Finding 2: Flag wiring locations identified

**Evidence:** Flags are defined in var block (lines 107-122) and registered in init() (lines 124-152)

**Source:** `cmd/orch/daemon.go`

**Significance:** Consistent pattern to follow for adding new flag

---

## Implementation

Changes made to `cmd/orch/daemon.go`:

1. Added `daemonCrossProject bool` to var block (line 122)
2. Added flag to daemonRunCmd: `--cross-project` (line 151)
3. Added flag to daemonPreviewCmd: `--cross-project` (line 155)
4. Added flag to daemonOnceCmd: `--cross-project` (line 157)
5. Wired `daemonCrossProject` into daemon.Config in:
   - runDaemonLoop() (line 175)
   - runDaemonDryRun() (line 568)
   - runDaemonOnce() (line 614)
   - runDaemonPreview() (line 657)
6. Added startup message for cross-project mode (lines 224-226)
7. Updated help text with cross-project examples

Changes made to `.kb/guides/daemon.md`:
- Added preview and once command examples under "Enabling Cross-Project"
- Added note about backward compatibility

---

## Verification

```bash
# Build successful
/usr/local/go/bin/go build -o /tmp/orch-test ./cmd/orch/

# All 51 daemon tests pass
/usr/local/go/bin/go test ./pkg/daemon/... -v
# PASS ok github.com/dylan-conlin/orch-go/pkg/daemon 51.942s

# Help shows new flag
/tmp/orch-test daemon run --help | grep cross-project
#   --cross-project     Poll all kb-registered projects for issues
```

---

## Files Modified

- `cmd/orch/daemon.go` - Added flag, wiring, help text
- `.kb/guides/daemon.md` - Updated cross-project section
