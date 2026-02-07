<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** resolveBackend() only accepted "claude" and "opencode" as valid config values, silently ignoring "docker".

**Evidence:** backend.go:69 checked `globalCfg.Backend == "claude" || globalCfg.Backend == "opencode"` without including "docker". Tests now pass with docker config.

**Knowledge:** When adding new backend types, must update ALL validation points (--backend flag, project config, global config).

**Next:** Commit fix - root cause addressed, tests passing.

**Promote to Decision:** recommend-no (tactical bug fix, not architectural)

---

# Investigation: Fix Global Config Backend Docker

**Question:** Why is `~/.orch/config.yaml` with `backend: docker` not being respected by `orch spawn`?

**Started:** 2026-01-20
**Updated:** 2026-01-20
**Owner:** og-debug-fix-global-config-20jan-2a4c
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Docker backend missing from config validation

**Evidence:** In `cmd/orch/backend.go`, the resolveBackend() function checks for valid backend values in config files:

```go
// Line 69 (global config)
if globalCfg.Backend == "claude" || globalCfg.Backend == "opencode" {

// Line 56 (project config)
if projCfg.SpawnMode == "claude" || projCfg.SpawnMode == "opencode" {
```

Neither check includes "docker" as a valid value.

**Source:** `cmd/orch/backend.go:56,69`

**Significance:** When config has `backend: docker`, the validation fails and falls through to the warning ("Invalid global backend...") and then defaults to opencode.

---

### Finding 2: --backend flag already supported docker

**Evidence:** The explicit flag check on line 36 already includes docker:

```go
if backendFlag != "claude" && backendFlag != "opencode" && backendFlag != "docker" {
```

**Source:** `cmd/orch/backend.go:36`

**Significance:** The docker backend was partially implemented - the flag worked, but config file support was incomplete.

---

### Finding 3: Fix verified with unit tests

**Evidence:** Added two new test cases:
- `global config backend: docker`
- `project config spawn_mode: docker`

All 19 tests now pass including the new docker config tests.

**Source:** `cmd/orch/backend_test.go` - test run output shows all PASS

**Significance:** Root cause is fixed and regression tests are in place.

---

## Synthesis

**Key Insights:**

1. **Incomplete validation pattern** - When docker backend was added, the flag validation was updated but config file validation was missed.

2. **Silent fallthrough masks bugs** - The code logs a warning about invalid config but then silently falls through to opencode default, making it hard to notice the bug.

**Answer to Investigation Question:**

The config was being read correctly, but `resolveBackend()` rejected "docker" as an invalid value because the validation conditionals only checked for "claude" or "opencode". Fix: add "|| projCfg.SpawnMode == "docker"` and `|| globalCfg.Backend == "docker"` to the validation checks.

---

## Structured Uncertainty

**What's tested:**

- ✅ Global config `backend: docker` now returns docker backend (unit test passes)
- ✅ Project config `spawn_mode: docker` now returns docker backend (unit test passes)
- ✅ All 19 backend resolution tests pass

**What's untested:**

- ⚠️ Full end-to-end spawn with docker backend via config (would need Docker running)

**What would change this:**

- None - this is a straightforward validation bug with clear fix

---

## References

**Files Examined:**
- `cmd/orch/backend.go` - resolveBackend() function with validation logic
- `cmd/orch/backend_test.go` - existing tests, added docker config tests
- `pkg/userconfig/userconfig.go` - confirmed Backend field correctly typed as string
- `~/.orch/config.yaml` - confirmed contains `backend: docker`

**Commands Run:**
```bash
# Build and test
go build -o orch ./cmd/orch
go test ./cmd/orch/... -v -run TestResolveBackend
```

---

## Investigation History

**2026-01-20:** Investigation started
- Initial question: Global config `backend: docker` not respected
- Context: User set docker backend in config but spawns use opencode

**2026-01-20:** Root cause identified
- Missing "docker" in validation conditionals at lines 56 and 69

**2026-01-20:** Investigation completed
- Status: Complete
- Key outcome: Two-line fix to add "docker" to validation checks, plus test coverage
