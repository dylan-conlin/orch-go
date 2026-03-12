## Summary (D.E.K.N.)

**Delta:** The standalone harness repo already has ~70% of the core functionality (init, check, report, accretion, control, scaffold, events), but is missing 5 orch-go harness subcommands (lock, unlock, status, verify, snapshot), the harness report's full falsification/pipeline analytics, and the orch-specific features (beads close hook, full-mode init, serve_harness HTTP API).

**Evidence:** Line-by-line comparison of 3,486 lines across 8 orch-go harness files vs 2,169 lines across 13 standalone harness files. pkg/control/ is identical in both repos. Standalone repo has cleaner architecture (pkg/scaffold, pkg/accretion, pkg/report, pkg/events) vs orch-go's monolithic harness_init.go (1,342 lines).

**Knowledge:** The migration boundary is clear: standalone features (accretion, init-standalone, check, report, control plane lock/unlock/verify) move to harness repo; orch-specific features (beads hooks, full-mode init, serve_harness API, snapshot with orch events, falsification verdicts) stay in orch-go or are deferred.

**Next:** Architect review before implementation — 5 subcommands need to be added to standalone harness CLI, and orch-go needs to either depend on the harness module or thin-wrapper delegate to the `harness` binary.

**Authority:** architectural — Cross-boundary (two repos, shared code, import path changes)

---

# Investigation: Inventory Everything Orch Harness Migrate

**Question:** What harness functionality in orch-go needs to migrate to the standalone harness repo, what's already there, and what stays orch-specific?

**Started:** 2026-03-12
**Updated:** 2026-03-12
**Owner:** Investigation agent (orch-go-q69uj)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A - novel investigation | - | - | - |

---

## Findings

### Finding 1: Orch-go harness has 8 files totaling 3,486 lines across cmd/ and pkg/

**Evidence:**

| File | Lines | Purpose |
|------|-------|---------|
| `cmd/orch/harness_init.go` | 1,342 | Init (standalone + full modes), deny rules, hook scripts/registration, pre-commit gates, beads close hook |
| `cmd/orch/serve_harness.go` | 644 | HTTP API (`GET /api/harness`), HarnessResponse types, pipeline/verdicts/velocity computation |
| `cmd/orch/harness_snapshot.go` | 286 | `orch harness snapshot`, directory line-count snapshotting, daemon auto-snapshot |
| `cmd/orch/harness_report_cmd.go` | 230 | `orch harness report` CLI command, text/JSON output formatting |
| `cmd/orch/harness_cmd.go` | 197 | `lock`, `unlock`, `status`, `verify` subcommands |
| `pkg/verify/accretion.go` | 297 | Completion-time accretion verification (git diff-based) |
| `pkg/verify/accretion_precommit.go` | 224 | Staged file accretion checks (pre-commit gate) |
| `pkg/control/control.go` | 266 | chflags uchg immutability, file discovery, deny rules |

**Source:** `wc -l` on all files

**Significance:** The bulk of the code (1,342 lines) is in harness_init.go which contains both standalone and full-mode logic interleaved. This is the primary extraction target.

---

### Finding 2: Standalone harness repo has 13 files totaling 2,169 lines with clean package structure

**Evidence:**

| Package | Files | Lines | Purpose |
|---------|-------|-------|---------|
| `cmd/harness/main.go` | 1 | 306 | CLI entry point: `init`, `check`, `report` commands |
| `pkg/accretion/` | 4 | 508 | `check.go` (git diff accretion), `source.go` (file scanning/bloat), `precommit.go` (staged checks) |
| `pkg/scaffold/` | 5 | 780 | `init.go` (orchestrator), `hooks.go` (hook scripts), `denylist.go` (deny rules), `precommit.go` (pre-commit gate), `template.go` (CLAUDE.md governance) |
| `pkg/events/` | 1 | 94 | JSONL event logging (gate.fired, gate.bypassed, etc.) |
| `pkg/report/` | 2 | 215 | `velocity.go` (git log-based growth velocity), `history.go` (gate event history) |
| `pkg/control/` | 1 | 266 | Identical copy of orch-go's `pkg/control/control.go` |

**Source:** File listing and `wc -l` on standalone harness repo

**Significance:** The standalone repo has already extracted the core accretion/scaffold/events logic with proper package boundaries. The architecture is cleaner than orch-go's monolithic `harness_init.go`.

---

### Finding 3: Five orch-go subcommands are missing from standalone harness

**Evidence:**

| Subcommand | Orch-go Location | Standalone Status |
|------------|-----------------|-------------------|
| `harness lock` | `harness_cmd.go:34-44` | **MISSING** — `pkg/control/` has the implementation, but no CLI command |
| `harness unlock` | `harness_cmd.go:47-55` | **MISSING** |
| `harness status` | `harness_cmd.go:57-61` | **MISSING** |
| `harness verify` | `harness_cmd.go:63-73` | **MISSING** |
| `harness snapshot` | `harness_snapshot.go` | **MISSING** — no snapshot concept at all |

**Source:** `grep` for lock/unlock/verify/snapshot in standalone harness returned no matches

**Significance:** These 5 subcommands are straightforward to add since the underlying `pkg/control` package already exists in both repos. The `snapshot` command depends on orch-go's `pkg/events` (AccretionSnapshotData, DirectorySnapshot types) which is orch-specific.

---

### Finding 4: Standalone init is simpler (5 steps) vs orch-go's dual-mode (standalone + full, 6-7 steps)

**Evidence:**

Standalone harness `scaffold.Init()` does 6 steps:
1. Deny rules (user-level settings.json)
2. Hook script (gate-git-add-all.py)
3. Hook registration (PreToolUse in settings.json)
4. Pre-commit gate (self-contained bash script)
5. Control plane lock (chflags uchg)
6. Governance template (CLAUDE.md section) ← **NEW in standalone, not in orch-go**

Orch-go `harness init` has two code paths:
- **Standalone mode** (detected by absence of `~/.orch/hooks/`): project-level settings, inline hooks, self-contained pre-commit
- **Full mode**: user-level settings, references to `~/.orch/hooks/` scripts, beads close hook, `orch precommit accretion` delegation

**Source:** `cmd/orch/harness_init.go:58-377` (full mode detection + dual code paths), `pkg/scaffold/init.go:28-69` (standalone clean path)

**Significance:** The standalone harness repo has a cleaner init that adds a CLAUDE.md governance section (step 6). The orch-go init has extra orch-specific features: beads close hook creation, full-mode hook registration (gate-bd-close.py, gate-worker-git-add-all.py), and `orch precommit accretion` delegation.

---

### Finding 5: Harness report has two very different implementations

**Evidence:**

**Standalone** (`cmd/harness/main.go:175-306`):
- Velocity analysis (git log --numstat)
- Current hotspots (file bloat scan)
- Gate history (events.jsonl gate.fired/gate.bypassed)
- Trend classification (accelerating/stable/decelerating)
- ~130 lines

**Orch-go** (`serve_harness.go` + `harness_report_cmd.go`):
- Full harness pipeline model (13 components across 4 stages)
- Falsification verdicts (4 hypotheses: ceremony, irrelevant, inert, anecdotal)
- Exploration metrics
- Accretion velocity from snapshots
- Completion coverage
- Measurement coverage
- ~874 lines

**Source:** Direct file comparison

**Significance:** The orch-go harness report is deeply coupled to orch's event system (session.spawned, spawn.triage_bypassed, spawn.hotspot_bypassed, exploration.*, agent.completed). This is fundamentally orch-specific and should NOT migrate. The standalone report is appropriate for any project.

---

### Finding 6: pkg/control/control.go is identical in both repos

**Evidence:** Both files are 266 lines with identical function signatures: `DiscoverControlPlaneFiles`, `DefaultSettingsPath`, `FileStatus`, `Lock`, `EnsureLocked`, `Unlock`, `UnlockMarkerPath`, `WriteUnlockMarker`, `RemoveUnlockMarker`, `IsUnlockMarkerPresent`, `VerifyLocked`, `expandPath`, `DenyRules`. The only difference is the import path (`github.com/dylan-conlin/orch-go/pkg/control` vs `github.com/dylan-conlin/harness/pkg/control`).

**Source:** Side-by-side read of both files

**Significance:** This confirms the package was already copied wholesale. The harness repo should be the canonical owner. Orch-go could either import from harness or keep its copy for independence.

---

### Finding 7: Orch-go harness depends on orch-specific infrastructure

**Evidence:**

Dependencies that stay orch-specific:
1. `pkg/events/logger.go` — `AccretionSnapshotData`, `DirectorySnapshot`, `LogAccretionSnapshot`, `EventTypeAccretionSnapshot` (used by snapshot command)
2. `cmd/orch/serve_harness.go` — HTTP API serving the HarnessResponse (consumed by orch dashboard)
3. `cmd/orch/harness_init.go` — `ensureBeadsCloseHook()` (creates `.beads/hooks/on_close`)
4. `cmd/orch/harness_init.go` — Full-mode init paths (references `~/.orch/hooks/`, `gate-bd-close.py`, `gate-worker-git-add-all.py`)
5. `cmd/orch/harness_init.go` — `ensurePreCommitGate()` full mode delegates to `orch precommit accretion`
6. `cmd/orch/stats_cmd.go` — `StatsEvent` type used by `parseEvents` / `buildHarnessResponse`
7. `cmd/orch/control_cmd.go` — `settingsPath()` helper reads `ORCH_SETTINGS_PATH` env var

**Source:** Import analysis and grep across harness files

**Significance:** These orch-specific pieces should NOT migrate. They represent the integration layer between harness governance and orch orchestration.

---

## Synthesis

**Key Insights:**

1. **Clean boundary exists** — Standalone governance (accretion checks, init scaffolding, control plane immutability, basic reporting) is separable from orch-specific orchestration (pipeline analytics, beads integration, snapshot events, falsification framework). The standalone harness repo has already established the right package structure.

2. **5 missing CLI commands are low-hanging fruit** — `lock`, `unlock`, `status`, `verify` are thin CLI wrappers around `pkg/control/` which already exists in the standalone repo. Adding these is ~150 lines of straightforward CLI code.

3. **Snapshot is the one ambiguous piece** — The directory line-counting logic (`collectAllSnapshots`, `collectDirectorySnapshot`) is project-agnostic, but the event emission uses orch-go's `pkg/events` format. The standalone harness has its own simpler `pkg/events` system. Snapshot could migrate if it emits to the standalone event format instead.

4. **Orch-go should thin-wrapper or import** — After migration, orch-go's `orch harness *` commands should either (a) delegate to the `harness` binary, or (b) import the harness Go module. Option (b) is cleaner but creates a dependency. Option (a) requires the binary to be installed but keeps repos independent.

**Answer to Investigation Question:**

## Migration Inventory

### MOVES to standalone harness (missing today)

| Component | Source | Lines | Notes |
|-----------|--------|-------|-------|
| `harness lock` command | `harness_cmd.go:34-103` | ~70 | Thin CLI over `pkg/control.Lock()` |
| `harness unlock` command | `harness_cmd.go:106-128` | ~25 | Thin CLI over `pkg/control.Unlock()` |
| `harness status` command | `harness_cmd.go:131-169` | ~40 | Thin CLI over `pkg/control.FileStatus()` |
| `harness verify` command | `harness_cmd.go:172-197` | ~25 | Thin CLI over `pkg/control.VerifyLocked()` |
| `harness snapshot` (core logic) | `harness_snapshot.go:101-208` | ~110 | Directory snapshot collection, needs new event format |
| `shortPath()` helper | `control_cmd.go:203-210` | ~8 | Display helper for `~/` paths |

### ALREADY in standalone harness

| Component | Status | Fidelity |
|-----------|--------|----------|
| `pkg/control/` | ✅ Identical copy | 100% — same 266 lines |
| `pkg/accretion/check.go` | ✅ Extracted from `pkg/verify/accretion.go` | ~95% — exported function names |
| `pkg/accretion/precommit.go` | ✅ Extracted from `pkg/verify/accretion_precommit.go` | ~98% — near-identical |
| `pkg/accretion/source.go` | ✅ New — `IsSourceFile`, `CountLines`, `AnalyzeBloatFiles` | Enhanced vs orch-go |
| `pkg/scaffold/init.go` | ✅ Clean standalone-only init | 5 steps → 6 steps (adds CLAUDE.md) |
| `pkg/scaffold/hooks.go` | ✅ Hook script + registration | Same gate-git-add-all.py |
| `pkg/scaffold/denylist.go` | ✅ Deny rules management | Standalone rules (4 vs orch's 6) |
| `pkg/scaffold/precommit.go` | ✅ Pre-commit gate installation | Same bash script |
| `pkg/scaffold/template.go` | ✅ CLAUDE.md governance section | New — not in orch-go |
| `pkg/events/` | ✅ Simpler event system | Project-local `.harness/events.jsonl` |
| `pkg/report/velocity.go` | ✅ Git-log velocity analysis | Standalone git-based |
| `pkg/report/history.go` | ✅ Gate history from events | Standalone events |
| `harness init` | ✅ Standalone mode only | Clean 6-step init |
| `harness check` | ✅ File bloat scanning | New command not in orch-go CLI |
| `harness report` | ✅ Basic health report | Velocity + hotspots + gates + trend |

### STAYS in orch-go (orch-specific)

| Component | Source | Why it stays |
|-----------|--------|-------------|
| `serve_harness.go` (full HTTP API) | 644 lines | Serves orch dashboard, uses orch event types |
| `HarnessResponse` types + pipeline model | `serve_harness.go` | 13-component pipeline specific to orch orchestration |
| Falsification verdicts | `serve_harness.go` | Orch-specific hypotheses about orch's harness |
| Exploration metrics | `serve_harness.go` | Orch exploration feature |
| `ensureBeadsCloseHook()` | `harness_init.go:648-708` | Creates `.beads/hooks/on_close` — beads is orch |
| Full-mode init path | `harness_init.go` | References `~/.orch/hooks/`, `gate-bd-close.py`, `gate-worker-git-add-all.py` |
| Full-mode pre-commit delegation | `harness_init.go:1300-1342` | Delegates to `orch precommit accretion` |
| `harness report` (orch version) | `harness_report_cmd.go` | Uses orch event types (session.spawned, etc.) |
| Snapshot event emission | `harness_snapshot.go:62-69` | Uses `pkg/events.LogAccretionSnapshot()` |
| Daemon auto-snapshot | `harness_snapshot.go:230-255` | Called from orch daemon periodic tasks |
| `StatsEvent` parsing | Referenced by `serve_harness.go` | Orch event format |

### NEEDS NEW CODE

| Component | Why | Effort |
|-----------|-----|--------|
| `harness lock/unlock/status/verify` CLI | Wire existing `pkg/control/` to cobra commands | Small (~160 lines) |
| `harness snapshot` command | Rewrite event emission to use harness's `pkg/events` format | Small (~100 lines) |
| Event type for snapshots | `pkg/events/events.go` needs `snapshot.recorded` event type | Trivial |
| Orch-go thin wrapper decision | Either import harness module or delegate to binary | Architectural decision |

---

## Structured Uncertainty

**What's tested:**

- ✅ pkg/control/control.go is identical in both repos (verified by reading both files)
- ✅ Standalone harness has init/check/report commands (verified by reading cmd/harness/main.go)
- ✅ 5 subcommands missing from standalone (verified by grep for lock/unlock/verify/snapshot)
- ✅ Standalone init has 6 steps including CLAUDE.md governance (verified by reading scaffold/init.go)

**What's untested:**

- ⚠️ Whether orch-go tests pass if harness code is removed and delegated (not tested)
- ⚠️ Whether standalone harness binary actually builds and runs (binary exists at harness/harness but not tested)
- ⚠️ Whether orch-go's harness_init full-mode features are actually used (no frequency data checked)

**What would change this:**

- If orch-go's harness tests depend on internals of the init helpers, refactoring would be more complex
- If Dylan wants orch-go to be fully self-contained (no harness dependency), the architecture changes

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add 5 missing CLI commands to standalone harness | implementation | Straightforward wiring of existing pkg/control |
| Decide orch-go → harness dependency model | architectural | Cross-repo dependency, affects both build systems |
| Decide what happens to orch-go harness code post-migration | strategic | Irreversible — removing code from orch-go changes maintenance surface |

### Recommended Approach ⭐

**Phase 1: Complete standalone harness, then thin-wrap orch-go** — Add the 5 missing commands to standalone harness first, validate it works independently, then decide the orch-go integration model.

**Why this approach:**
- Standalone harness should be usable in any project without orch — it's the product
- The 5 missing commands are trivial to add since pkg/control already exists
- Orch-go integration is a separate decision that doesn't block standalone completeness

**Trade-offs accepted:**
- Temporary code duplication between repos (acceptable during migration)
- Orch-go harness commands continue to work unchanged during transition

**Implementation sequence:**
1. Add `lock`, `unlock`, `status`, `verify` commands to standalone harness (wiring pkg/control to CLI)
2. Add `snapshot` command to standalone harness (with local event format)
3. Architect decision: orch-go integration model (import module vs delegate to binary vs keep duplication)
4. Execute orch-go integration based on architect decision

### Alternative Approaches Considered

**Option B: Import harness as Go module in orch-go**
- **Pros:** Single source of truth, no code duplication, type-safe
- **Cons:** Creates cross-repo dependency, complicates orch-go build, version coupling
- **When to use instead:** If both repos are released together or harness API is very stable

**Option C: Delete harness code from orch-go entirely, require harness binary**
- **Pros:** Clean separation, no duplication
- **Cons:** Breaks `orch harness *` commands for existing users, adds install dependency
- **When to use instead:** If harness is widely adopted and always installed alongside orch

**Rationale for recommendation:** Phase 1 (complete standalone first) is risk-free and delivers value independently. The orch-go integration can be deferred without blocking either repo.

---

### Implementation Details

**What to implement first:**
- `harness lock/unlock/status/verify` — these are essential for control plane management
- Test coverage for the new commands

**Things to watch out for:**
- ⚠️ `UnlockMarkerPath()` uses `~/.orch/harness-unlocked` — standalone should use `~/.harness/harness-unlocked` or similar
- ⚠️ `DenyRules()` in orch-go includes `~/.orch/hooks/**` paths that are orch-specific
- ⚠️ `settingsPath()` in orch-go reads `ORCH_SETTINGS_PATH` env var — standalone should have its own or use default

**Areas needing further investigation:**
- Whether standalone harness should have its own marker path vs using orch's
- Whether deny rules should be configurable per-project vs hardcoded

**Success criteria:**
- ✅ `harness lock`, `harness unlock`, `harness status`, `harness verify` all work from standalone binary
- ✅ `harness snapshot` emits events to `.harness/events.jsonl`
- ✅ All existing standalone tests continue to pass
- ✅ Orch-go harness commands continue to work unchanged

---

## References

**Files Examined:**
- `cmd/orch/harness_cmd.go` (197 lines) — lock/unlock/status/verify subcommands
- `cmd/orch/harness_init.go` (1342 lines) — dual-mode init with all helper functions
- `cmd/orch/harness_snapshot.go` (286 lines) — snapshot command and daemon integration
- `cmd/orch/harness_report_cmd.go` (230 lines) — CLI report command
- `cmd/orch/serve_harness.go` (644 lines) — HTTP API and full pipeline model
- `pkg/control/control.go` (266 lines) — control plane immutability
- `pkg/verify/accretion.go` (297 lines) — completion-time accretion checks
- `pkg/verify/accretion_precommit.go` (224 lines) — staged file accretion checks
- `~/Documents/personal/harness/cmd/harness/main.go` (306 lines) — standalone CLI
- `~/Documents/personal/harness/pkg/scaffold/init.go` (129 lines) — standalone init orchestrator
- `~/Documents/personal/harness/pkg/accretion/` (508 lines total) — standalone accretion package
- `~/Documents/personal/harness/pkg/control/control.go` (266 lines) — identical copy
- `~/Documents/personal/harness/pkg/events/events.go` (94 lines) — standalone events
- `~/Documents/personal/harness/pkg/report/` (215 lines total) — standalone reporting

---

## Investigation History

**2026-03-12:** Investigation started
- Initial question: What harness functionality in orch-go needs to migrate to standalone harness repo?
- Context: Standalone harness repo exists with partial functionality; need complete inventory

**2026-03-12:** Investigation completed
- Status: Complete
- Key outcome: 70% already migrated, 5 CLI commands missing, clear boundary between standalone and orch-specific code
