# Investigation: Beads Fork & Integration Architecture Audit

**Date:** 2026-02-20
**Status:** Complete
**Investigator:** Claude (codebase-audit skill)
**Trigger:** Orientation frame — documentation says fork was dropped, but fork exists with 43 commits
**Beads:** orch-go-1165
**Confidence:** High (primary evidence from code and git history)
**Resolution-Status:** Resolved

---

## TLDR

The beads fork at `~/Documents/personal/beads` has **43 commits beyond upstream**, all made after the Dec 21, 2025 "clean slate" decision that said to drop the fork. The decision was effectively reversed within 9 days and never formally superseded. Of these 43 commits, **6 features are actively used by orch-go** and ~10 more provide infrastructure reliability that orch-go depends on implicitly. The integration model (`beads-integration-architecture.md`) references 5 files that no longer exist, has the wrong socket path, and states a constraint ("never use exec.Command directly") that is violated 11 times across 7 files. The model's core architectural claim (RPC-first with CLI fallback, three integration points) remains accurate.

---

## Scope

**Focus:** Beads fork inventory, orch-go integration patterns, model/decision staleness, improvement opportunities
**Boundaries:** orch-go and beads codebases; no changes implemented
**Probe file:** `.kb/models/beads-integration-architecture/probes/2026-02-20-beads-fork-integration-audit.md`

---

## Findings

### 1. Fork Inventory (43 Commits Beyond Upstream)

**Remotes:**
- `origin` → steveyegge/beads (upstream)
- `fork` → dylan-conlin/beads (Dylan's GitHub fork)

**Timeline:** All 43 commits are post-Dec 21 (clean-slate decision date). First fork commit: Dec 30, 2025. Most recent: Feb 17, 2026.

**Categories:**

| Category | Count | Examples |
|----------|-------|---------|
| Stability/reliability | 10 | Sandbox detection, JSONL-only mode, WAL corruption prevention, restart loops |
| Feature additions | 9 | Question type, dedup, investigation type, absorbed-by, epic readiness |
| Bug fixes | 8 | File locking, close exit codes, migration idempotency, stale locks |
| Workflow enforcement | 5 | Phase: Complete gate, push confirmation, bd prime skip |
| Documentation/chore | 6 | SYNTHESIS files, gitignore, investigations |
| Infrastructure | 5 | Auto-rebuild, symlink install, decidability fields |

### 2. Fork Features Used by orch-go

**Actively used (called from orch-go code):**

| Feature | Commit | orch-go Location | Usage |
|---------|--------|-------------------|-------|
| Question entity type | `2dc8f7dc` | `serve_beads.go:804+` | Dashboard questions API |
| Question gate deps | `744af9cf` | `attention/unblocked_collector.go` | Dependency blocking |
| Title-based dedup | `e19ff3f8` | `pkg/beads/types.go` (Force field) | Prevent duplicate issues |
| Phase: Complete gate | `be871d0c` | Core verify pipeline | `bd close` checks for completion |
| Investigation type | `d813a87c` | `verify/unverified.go` | Tier determination |
| Close exit codes | `a3f8729e` | `cmd/orch/reconcile.go` | Error detection |

**Implicitly depended on (infrastructure improvements):**

| Feature | Why orch-go depends on it |
|---------|--------------------------|
| Sandbox detection | Prevents SQLite corruption when agents run in Claude Code sandbox |
| JSONL-only mode | Reliable storage when SQLite unavailable |
| Rapid restart prevention | Daemon stability (orch daemon polls beads daemon) |
| Cross-process file locking | Concurrent agent access to beads data |
| Pre-flight fingerprint validation | Daemon doesn't start in bad state |

**Not used by orch-go:**

| Feature | Status |
|---------|--------|
| bd graph --all | No references |
| Absorbed-by/supersedes | Documentation only |
| Epic readiness gate | Not called |
| Pager for bd list | Not called |
| --repro flag | Not called |
| could-not-reproduce close outcome | Not called |

### 3. Model Staleness Assessment

**File: `.kb/models/beads-integration-architecture/model.md`**
Last updated: 2026-01-12

| Model Claim | Status | Reality |
|-------------|--------|---------|
| Socket at `~/.beads/daemon.sock` | **WRONG** | Socket is `.beads/bd.sock` (project-local, found via tree walk) |
| Source: `pkg/beads/fallback.go` | **DELETED** | Fallback functions consolidated into `client.go:710-1226` |
| Source: `pkg/beads/lifecycle.go` | **DELETED** | Lifecycle operations distributed across `client.go` methods |
| Source: `pkg/beads/id.go` | **DELETED** | ID logic absorbed elsewhere |
| Source: `pkg/spawn/tracking.go` | **NEVER EXISTED** | Tracking logic is in `spawn_cmd.go` |
| Source: `cmd/orch/spawn.go` | **RENAMED** | Now `cmd/orch/spawn_cmd.go` |
| "Never use exec.Command('bd') directly" | **VIOLATED** | 11 direct calls across 7 files |
| RPC-first with CLI fallback | **CORRECT** | Core pattern confirmed |
| Three integration points (spawn/work/complete) | **CORRECT** | Confirmed |
| Auto-tracking unless --no-track | **CORRECT** | Confirmed |
| NewClient() constructor style | **OUTDATED** | Now `NewClient(socketPath, ...ClientOption)` with functional options |
| Performance: RPC ~2-5ms, CLI ~50-100ms | **PLAUSIBLE** | Not benchmarked but architecture supports this |

**5 of 6 "Primary Evidence" file paths are wrong.** The model's code examples are pseudo-code that doesn't match actual implementation.

### 4. Decision Staleness Assessment

**File: `.kb/decisions/2025-12-21-beads-oss-relationship-clean-slate.md`**

| Decision Claim | Status | Reality |
|----------------|--------|---------|
| "Drop all local features" | **REVERSED** | 43 new local features added |
| "Use upstream beads as-is" | **REVERSED** | Active fork with substantial divergence |
| "Zero rebase maintenance going forward" | **REVERSED** | Fork needs periodic upstream reconciliation |
| "No fork to manage" | **REVERSED** | Fork is actively managed at `dylan-conlin/beads` |
| "Features not used by orch-go" | **WAS TRUE** | True for the old features (ai-help, health, tree) — new features are actively used |

The decision addressed a different set of local features (bd ai-help, bd health, bd tree) that genuinely weren't used. The decision was correct for those features. However, in the 2 months since, **entirely new features** were developed and committed to the fork, effectively creating a new fork relationship that was never formally documented.

**No superseding decision exists.** The knowledge base treats beads as upstream-only while the reality is active fork development.

### 5. exec.Command Constraint Violation Analysis

The model's constraint says: "All beads integration goes through `pkg/beads`, never direct `exec.Command('bd')`."

**11 violations across 7 files:**

| File | Calls | Justification | Should migrate? |
|------|-------|---------------|-----------------|
| `pkg/daemon/issue_adapter.go` | 3 | Fallback functions for RPC client | **YES** — should use `CLIClient` |
| `pkg/verify/beads_api.go` | 2 | CLI fallback for comments/labels | **PARTIAL** — already have RPC primary |
| `pkg/daemon/extraction.go` | 2 | `bd create` and `bd dep add` | **YES** — should use `BeadsClient` |
| `pkg/focus/guidance.go` | 1 | `bd ready --json` | **YES** — should use `BeadsClient.Ready()` |
| `cmd/orch/init.go` | 1 | `bd init` | **OK** — one-time setup, no RPC equivalent |
| `cmd/orch/reconcile.go` | 1 | Various bd commands | **PARTIAL** — some could migrate |
| `cmd/orch/status_cmd.go` | 1 | `bd config get` | **OK** — config query, no RPC equivalent |

**5-7 calls could be migrated** to use `BeadsClient` interface. 2-3 are legitimate (one-time operations with no RPC equivalent).

### 6. Integration Architecture (Actual State)

**pkg/beads/ (actual files):**
- `interface.go` — `BeadsClient` interface with 11 methods
- `client.go` (33KB) — RPC client with auto-reconnect, fallback functions consolidated here
- `cli_client.go` (8KB) — CLI-only client implementation
- `types.go` (14KB) — RPC protocol types and data structures
- `mock_client.go` (10KB) — Mock for testing with dedup support

**Integration points confirmed:**
1. **Spawn** → `spawn_cmd.go` creates issue via `BeadsClient.Create()`
2. **Work** → Agents report via `bd comments add` (CLI from agent process)
3. **Complete** → `complete_cmd.go` closes via verification pipeline → `BeadsClient.CloseIssue()`
4. **Dashboard** → `serve_beads.go` queries via RPC for real-time display
5. **Daemon** → `daemon.go` polls via `BeadsClient.Ready()` for auto-spawn

---

## Recommendations

### High Priority

1. **Supersede the clean-slate decision** — Create `.kb/decisions/2026-02-20-beads-fork-active-development.md` documenting the actual relationship: active fork with upstream tracking, 6+ features used by orch-go, infrastructure improvements implicitly depended on.

2. **Update model file references** — The model's "Primary Evidence" and "Source code" sections reference 5 non-existent files. Update to reflect actual file structure (`client.go`, `cli_client.go`, `interface.go`, `types.go`, `spawn_cmd.go`).

3. **Fix socket path in model** — Change `~/.beads/daemon.sock` to `.beads/bd.sock` throughout.

### Medium Priority

4. **Migrate exec.Command calls** — Move 5-7 direct `bd` calls to use `BeadsClient` interface, particularly in `pkg/daemon/extraction.go` and `pkg/focus/guidance.go` where the RPC path exists but isn't used.

5. **Establish upstream contribution strategy** — Several fork fixes are generally useful (sandbox detection, file locking, restart loop prevention). Consider PRing these upstream to reduce fork maintenance burden.

6. **Update model code examples** — Replace pseudo-code with actual constructor signature and usage patterns.

### Low Priority

7. **Evaluate unused fork features** — bd graph, absorbed-by/supersedes, epic readiness gate, and pager support are not used. Determine if these should be removed from the fork or adopted by orch-go.

8. **Add BeadsClient methods for missing operations** — `bd dep add`, `bd config get`, and `bd init` have no interface methods, forcing direct exec.Command. Adding these would allow full migration.

---

## Reproducibility

**Commands used:**
```bash
# Fork analysis
cd ~/Documents/personal/beads && git log --oneline origin/main..HEAD | wc -l
cd ~/Documents/personal/beads && git log --oneline --format="%h %ad %s" --date=short origin/main..HEAD

# File existence
ls pkg/beads/fallback.go pkg/beads/lifecycle.go pkg/beads/id.go pkg/spawn/tracking.go cmd/orch/spawn.go

# Socket path
grep -n "daemon.sock\|bd.sock" pkg/beads/client.go

# exec.Command violations
grep -rn 'exec\.Command("bd"' --include='*.go' | grep -v test | grep -v .kb/

# Fork feature usage
grep -rn "question\|dedup\|Force.*bool\|investigation.*type\|graph.*--all\|absorbed\|supersedes" pkg/ cmd/
```

**Metrics baseline:**
- Fork commits ahead of upstream: 43
- Direct exec.Command("bd") calls outside pkg/beads: 11 across 7 files
- Model file references that don't exist: 5 of 6
- Fork features actively used by orch-go: 6
- Fork features implicitly depended on: 5
- Fork features not used: 6

---

## Related

- **Model:** `.kb/models/beads-integration-architecture/model.md` (needs update)
- **Decision:** `.kb/decisions/2025-12-21-beads-oss-relationship-clean-slate.md` (needs superseding)
- **Guide:** `.kb/guides/beads-integration.md` (mostly accurate, needs minor updates)
- **Probe:** `.kb/models/beads-integration-architecture/probes/2026-02-20-beads-fork-integration-audit.md`
- **Beads fork:** `~/Documents/personal/beads`
