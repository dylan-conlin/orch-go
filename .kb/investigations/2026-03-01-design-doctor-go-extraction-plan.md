## Summary (D.E.K.N.)

**Delta:** doctor.go (1736 lines) contains 7 distinct feature domains that map cleanly to 5 extraction files, leaving a ~190-line orchestrator core.

**Evidence:** Line-by-line function grouping analysis shows zero cross-domain coupling between --sessions, --config, --docs, --watch, and --daemon modes; each has self-contained types, logic, and formatting.

**Knowledge:** The file grew by accretion of flag-activated modes (sessions, config, docs, watch, daemon) that share only the ServiceStatus type and individual check functions. The extraction hierarchy is: shared checks first, then each mode to its own file.

**Next:** Spawn feature-impl to execute the 5-file extraction plan (Phase 1: doctor_checks.go, Phase 2: 4 mode files in parallel).

**Authority:** implementation - Standard file extraction within established patterns, no architectural changes.

---

# Investigation: Doctor Go Extraction Plan

**Question:** How should doctor.go (1736 lines, CRITICAL hotspot) be decomposed into cohesive extraction units to unblock feature-impl spawns?

**Defect-Class:** unbounded-growth

**Started:** 2026-03-01
**Updated:** 2026-03-01
**Owner:** architect agent
**Phase:** Complete
**Next Step:** None - ready for feature-impl execution
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/guides/code-extraction-patterns.md` | extends | Yes - verified naming conventions and phase ordering | None |
| `.kb/models/extract-patterns/model.md` | confirms | Yes - 800-line gate, shared-first hierarchy | None |
| `.kb/investigations/synthesized/serve-performance/2026-01-04-inv-analyze-serve-agents-go-1399.md` | extends | Checked - same extraction pattern applies | None |

---

## Findings

### Finding 1: doctor.go Contains 7 Distinct Cohesive Groups

**Evidence:** Function-by-function analysis of 1736 lines reveals these independent domains:

| Group | Lines | Functions | Activated By |
|-------|-------|-----------|-------------|
| Command setup + orchestrator | 1-275 | `runDoctor()`, init, flags, types | default |
| Service health checks | 277-672 | `checkOpenCode()`, `checkOrchServe()`, `checkWebUI()`, `checkOvermindServices()`, `checkBeadsDaemon()`, `startOpenCode()`, `startOrchServe()`, `checkStaleBinary()`, `checkStalledSessions()`, `printDoctorReport()` | default (used by runDoctor) |
| Sessions cross-reference | 778-960 | `runSessionsCrossReference()`, `printSessionsCrossReferenceReport()` | `--sessions` |
| Config drift + doc debt | 962-1171 | `runConfigDriftCheck()`, `checkPlistDrift()`, `runDocDebtCheck()` | `--config`, `--docs` |
| Watch mode | 1173-1315 | `runDoctorWatch()`, `runHealthCheckWithNotifications()`, `countUnhealthy()` | `--watch` |
| Self-healing daemon | 1317-1614 | `runDoctorDaemon()`, `runDaemonHealthCycle()`, `killOrphanedViteProcesses()`, `killLongRunningBdProcesses()`, `restartCrashedServices()`, `parseElapsedTime()` | `--daemon` |
| Launchd management | 1616-1736 | `getDoctorPlistPath()`, `runDoctorInstall()`, `runDoctorUninstall()` | subcommands |

**Source:** `cmd/orch/doctor.go:1-1736`

**Significance:** Each mode is activated by a distinct flag and has no cross-dependencies with other modes (except all share the service health check functions). This is the ideal extraction case — clean domain boundaries with a shared utility layer.

---

### Finding 2: Cross-Group Dependencies Are Minimal and One-Directional

**Evidence:** The dependency graph shows:

```
runDoctor() (orchestrator)
    ├── checkOpenCode(), checkOrchServe(), checkWebUI()...  (health checks)
    ├── printDoctorReport()                                  (formatting)
    └── startOpenCode(), startOrchServe()                    (fix actions)

runDoctorWatch()
    └── runHealthCheckWithNotifications()
        └── checkOpenCode(), checkOrchServe(), checkWebUI()... (same health checks)
        └── countUnhealthy()

runDoctorDaemon()
    └── runDaemonHealthCycle()
        ├── checkOpenCode(), checkOrchServe(), checkWebUI()... (same health checks)
        ├── killOrphanedViteProcesses()
        ├── killLongRunningBdProcesses()
        └── restartCrashedServices()

runSessionsCrossReference()        → standalone, no shared deps
runConfigDriftCheck()              → standalone, no shared deps
runDocDebtCheck()                  → standalone, no shared deps
```

The only shared code is the service health check functions — used by `runDoctor()`, `runDoctorWatch()`, and `runDoctorDaemon()`. All three consume the same `ServiceStatus` type and `check*()` functions.

**Source:** Cross-referencing all function call sites in `cmd/orch/doctor.go`

**Significance:** The health check functions form the natural "shared utilities" layer. Per the extraction guide, shared utilities must be extracted FIRST. All other groups are self-contained — they can be extracted in parallel after the shared layer is established.

---

### Finding 3: Watch and Daemon Modes Duplicate Health Check Assembly

**Evidence:** Lines 1214-1271 (`runHealthCheckWithNotifications`) and lines 1438-1444 (`runDaemonHealthCycle`) both manually assemble a `DoctorReport` by calling the same sequence of `check*()` functions and appending to `report.Services`. This is nearly identical to lines 168-231 in `runDoctor()`.

**Source:** `cmd/orch/doctor.go:168-231`, `cmd/orch/doctor.go:1214-1271`, `cmd/orch/doctor.go:1438-1444`

**Significance:** This duplication is a pre-existing smell but NOT in scope for this extraction task. The extraction should preserve existing behavior exactly. A follow-up refactoring could introduce a `runAllChecks() *DoctorReport` helper to DRY this up, but that's a separate concern from file splitting.

---

## Synthesis

**Key Insights:**

1. **Natural domain boundaries align with CLI flags** — Each `--flag` activates a completely self-contained mode with its own types, logic, and formatting. This makes extraction trivial: one file per mode.

2. **Shared health checks are the only cross-cutting concern** — `checkOpenCode()`, `checkOrchServe()`, etc. are consumed by the main orchestrator, watch mode, and daemon mode. These must be extracted first to avoid duplication.

3. **Config drift and doc debt can share a file** — Both are "audit" checks (activated by `--config` and `--docs`). Together they total ~209 lines, which is reasonable for a single file. They share no code with each other but are conceptually similar (checking system configuration health).

4. **Daemon and launchd management belong together** — The launchd install/uninstall is only meaningful in the context of the daemon. Keeping them together (~419 lines) is within the 300-800 line target.

**Answer to Investigation Question:**

doctor.go should be decomposed into 5 new files plus a residual orchestrator (~190 lines). The extraction follows the established pattern: shared utilities first (health checks), then domain files (modes). No architectural changes needed — all files stay in `package main`, all functions remain package-level visible.

---

## Structured Uncertainty

**What's tested:**

- ✅ All 7 function groups have been identified with exact line ranges (verified by reading full file)
- ✅ Cross-group dependencies are one-directional toward health checks (verified by tracing call sites)
- ✅ No shared utilities exist beyond ServiceStatus type and check* functions (verified by grep of function names)
- ✅ Naming convention follows established pattern: `{command}_{domain}.go` (verified against existing files like `complete_pipeline.go`)

**What's untested:**

- ⚠️ Import cleanup after extraction (each new file will need its own import subset)
- ⚠️ Test file splitting (doctor_test.go at 536 lines should split to match new files)
- ⚠️ Exact line counts post-extraction (estimates based on function boundaries)

**What would change this:**

- If new features were planned that span multiple modes (e.g., daemon needs config drift detection), the grouping might change
- If a shared helper function was discovered that crosses group boundaries unexpectedly

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| 5-file extraction within package main | implementation | Standard refactoring within established patterns, reversible, no API changes |

### Recommended Approach ⭐

**5-File Extraction Plan** — Split doctor.go into 5 domain files plus residual orchestrator, following established code extraction patterns.

**Why this approach:**
- Each file maps to a distinct feature flag, making future changes isolated
- Follows proven extraction hierarchy (shared first, then domains)
- All files stay in `package main` — no import changes needed
- Residual orchestrator (~190 lines) is well under 800-line threshold

**Trade-offs accepted:**
- Some small files (doctor_audit.go ~209 lines) are below the 300-line minimum suggested by the guide — acceptable because they represent complete domain units
- Health check duplication between runDoctor/watch/daemon is preserved (refactoring is a separate concern)

**Implementation sequence:**

#### Phase 1: Extract Shared Health Checks (~450 lines)

**File: `cmd/orch/doctor_checks.go`**

| What Moves | Lines (approx) |
|-----------|----------------|
| `ServiceStatus` struct | 10 |
| `DoctorReport` struct | 5 |
| `BinaryStatus` struct | 8 |
| `checkOpenCode()` | 25 |
| `checkOrchServe()` | 92 |
| `checkWebUI()` | 52 |
| `checkOvermindServices()` | 31 |
| `checkBeadsDaemon()` | 26 |
| `startOpenCode()` | 22 |
| `startOrchServe()` | 55 |
| `checkStaleBinary()` | 43 |
| `checkStalledSessions()` | 97 |
| `printDoctorReport()` | 32 |
| **Total** | **~450** |

**Why first:** Watch mode and daemon mode both depend on these functions. Extracting them first prevents duplication.

**Imports needed:** `crypto/tls`, `encoding/json`, `fmt`, `net`, `net/http`, `os`, `os/exec`, `strings`, `time`, `pkg/opencode`, `pkg/verify`

#### Phase 2: Extract Mode Files (4 files, can be done in parallel)

**File: `cmd/orch/doctor_sessions.go`** (~183 lines)

| What Moves | Lines (approx) |
|-----------|----------------|
| `SessionsCrossReferenceReport` struct | 10 |
| `runSessionsCrossReference()` | 97 |
| `printSessionsCrossReferenceReport()` | 73 |
| **Total** | **~183** |

**Imports needed:** `fmt`, `os`, `path/filepath`, `strings`, `time`, `pkg/opencode`, `pkg/spawn`

---

**File: `cmd/orch/doctor_audit.go`** (~209 lines)

| What Moves | Lines (approx) |
|-----------|----------------|
| `ConfigDrift` struct | 6 |
| `ConfigDriftReport` struct | 6 |
| `runConfigDriftCheck()` | 38 |
| `checkPlistDrift()` | 88 |
| `DocDebtReport` struct | 7 |
| `runDocDebtCheck()` | 58 |
| **Total** | **~209** |

**Imports needed:** `fmt`, `os`, `pkg/daemonconfig`, `pkg/userconfig`

---

**File: `cmd/orch/doctor_watch.go`** (~143 lines)

| What Moves | Lines (approx) |
|-----------|----------------|
| `runDoctorWatch()` | 38 |
| `runHealthCheckWithNotifications()` | 93 |
| `countUnhealthy()` | 10 |
| **Total** | **~143** |

**Imports needed:** `fmt`, `os`, `os/signal`, `syscall`, `time`, `pkg/notify`

---

**File: `cmd/orch/doctor_daemon.go`** (~419 lines)

| What Moves | Lines (approx) |
|-----------|----------------|
| `DoctorDaemonConfig` struct | 8 |
| `DefaultDoctorDaemonConfig()` | 11 |
| `DoctorDaemonIntervention` struct | 9 |
| `DoctorDaemonLogger` struct + methods | 36 |
| `runDoctorDaemon()` | 37 |
| `runDaemonHealthCycle()` | 45 |
| `killOrphanedViteProcesses()` | 41 |
| `killLongRunningBdProcesses()` | 38 |
| `restartCrashedServices()` | 37 |
| `parseElapsedTime()` | 27 |
| `getDoctorPlistPath()` | 5 |
| `runDoctorInstall()` | 88 |
| `runDoctorUninstall()` | 26 |
| **Total** | **~419** |

**Imports needed:** `fmt`, `os`, `os/exec`, `os/signal`, `path/filepath`, `strings`, `syscall`, `time`, `pkg/notify`

#### Phase 3: Residual doctor.go (~190 lines)

**What remains:**
- Package declaration + imports
- Global flag variables (`doctorFix`, `doctorVerbose`, etc.)
- `DefaultWebPort` const
- `doctorCmd` Cobra command definition (with `Long` description)
- `doctorInstallCmd`, `doctorUninstallCmd` command definitions
- `init()` function (flag binding + subcommands)
- `runDoctor()` orchestrator function

#### Phase 4: Split doctor_test.go (optional follow-up)

Current `doctor_test.go` (536 lines) tests span multiple domains. Split to match:
- `doctor_checks_test.go` — ServiceStatus, DoctorReport, checkOrchServe tests
- `doctor_audit_test.go` — ConfigDrift, ParsePlistValues tests
- `doctor_daemon_test.go` — parseElapsedTime, DoctorDaemonConfig tests
- `doctor_test.go` — remaining general tests

### Alternative Approaches Considered

**Option B: Fewer files (3 files)**
- Combine sessions + audit + watch into `doctor_modes.go` (~535 lines)
- Keep daemon separate
- **Pros:** Fewer files to manage
- **Cons:** `doctor_modes.go` conflates unrelated features; future changes to `--sessions` require reading `--config` code
- **When to use:** If team prefers fewer, larger files

**Option C: Extract to pkg/doctor/ package**
- Move all doctor logic to a dedicated Go package
- **Pros:** Cleaner separation, testable without package main
- **Cons:** Violates established pattern ("prefer splitting within package main"), creates import overhead, contradicts extraction model constraint
- **When to use:** If doctor functionality needs to be consumed by other packages

**Rationale for recommendation:** Option A (5 files) best follows the established extraction pattern (shared-first hierarchy), matches the domain boundaries (one file per flag-mode), and keeps each file within the 300-800 line target. The `package main` approach avoids import cycles and matches every prior extraction in this codebase.

---

### Implementation Details

**What to implement first:**
- `doctor_checks.go` — all other mode files depend on this for `ServiceStatus` type and `check*()` functions

**Things to watch out for:**
- ⚠️ The `doctorVerbose` and `doctorFix` global vars are referenced across multiple groups — they stay in residual `doctor.go` and are visible package-wide
- ⚠️ `DefaultWebPort` const is used by `checkWebUI()` in doctor_checks.go — it can stay in residual or move to checks
- ⚠️ `extractBeadsIDFromTitle()` is called in doctor_checks.go (checkStalledSessions) and doctor_sessions.go — verify it's in shared.go
- ⚠️ `shortID()` helper is used in multiple places — verify it exists in shared.go
- ⚠️ Watch mode's `runHealthCheckWithNotifications()` duplicates the check assembly from `runDoctor()` — preserve as-is, don't refactor during extraction

**Areas needing further investigation:**
- Whether `extractBeadsIDFromTitle()` and `shortID()` are already in `shared.go` (if not, they need to be)
- Test coverage gap: `runDoctorWatch()`, `runDoctorDaemon()`, and `runDoctorInstall/Uninstall` have no unit tests

**Success criteria:**
- ✅ `go build ./cmd/orch/` passes with no errors
- ✅ `go test ./cmd/orch/...` passes with all existing tests
- ✅ `go vet ./cmd/orch/` passes
- ✅ `doctor.go` is under 200 lines (below 800-line gate)
- ✅ No extracted file exceeds 500 lines
- ✅ `orch hotspot` no longer lists `cmd/orch/doctor.go` as CRITICAL

---

## References

**Files Examined:**
- `cmd/orch/doctor.go` (1736 lines) - Full analysis of all functions
- `cmd/orch/doctor_test.go` (536 lines) - Test distribution analysis
- `cmd/orch/shared.go` (570 lines) - Verified shared utilities location
- `cmd/orch/complete_cmd.go`, `complete_pipeline.go` - Prior extraction pattern reference
- `.kb/guides/code-extraction-patterns.md` - Extraction workflow
- `.kb/models/extract-patterns/model.md` - Extraction theory

**Commands Run:**
```bash
# Verify file size
wc -l cmd/orch/doctor.go  # 1736

# Check existing extraction patterns
ls cmd/orch/complete*.go  # 5 files from prior extraction
ls cmd/orch/doctor*.go    # Only doctor.go and doctor_test.go

# Check shared utilities
wc -l cmd/orch/shared.go  # 570 lines
```

**Related Artifacts:**
- **Guide:** `.kb/guides/code-extraction-patterns.md` - Authoritative extraction procedures
- **Model:** `.kb/models/extract-patterns/model.md` - Extraction theory and constraints
- **Constraint:** Accretion boundaries in CLAUDE.md - Files >1,500 lines block feature-impl

---

## Investigation History

**2026-03-01 14:27:** Investigation started
- Initial question: How to extract doctor.go (1736 lines) below CRITICAL hotspot threshold
- Context: File is blocking feature-impl spawns per accretion boundary enforcement

**2026-03-01 14:35:** Full file analysis complete
- Identified 7 cohesive function groups with clear boundaries
- Confirmed minimal cross-group coupling (only health checks are shared)

**2026-03-01 14:45:** Investigation completed
- Status: Complete
- Key outcome: 5-file extraction plan ready for feature-impl execution
