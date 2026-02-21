# Probe: Beads Fork & Integration Architecture Audit

**Model:** beads-integration-architecture
**Date:** 2026-02-20
**Status:** Complete

---

## Question

The model claims: (1) beads uses RPC-first with CLI fallback via specific files (`pkg/beads/fallback.go`, `pkg/beads/lifecycle.go`, `pkg/beads/id.go`), (2) socket path is `~/.beads/daemon.sock`, (3) all beads integration goes through `pkg/beads` and never via direct `exec.Command("bd")`, (4) spawn integration is in `pkg/spawn/tracking.go` and `cmd/orch/spawn.go`. Additionally, the decision `2025-12-21-beads-oss-relationship-clean-slate.md` says "Drop all local features and use upstream beads as-is." How do these claims hold up against the actual codebase?

---

## What I Tested

### 1. File existence checks for model-referenced files

```bash
# Model references these files as primary evidence:
ls pkg/beads/fallback.go    # NOT FOUND
ls pkg/beads/lifecycle.go   # NOT FOUND
ls pkg/beads/id.go          # NOT FOUND
ls pkg/spawn/tracking.go    # NOT FOUND
ls cmd/orch/spawn.go        # NOT FOUND

# Actual files:
ls pkg/beads/
# client.go, cli_client.go, interface.go, types.go, mock_client.go
# + test files: client_test.go, cli_client_test.go, dedup_test.go, etc.

ls cmd/orch/spawn_cmd.go    # EXISTS (renamed from spawn.go)
```

### 2. Socket path verification

```bash
grep -n "daemon.sock\|bd.sock" pkg/beads/client.go
# Line 161: socketPath := filepath.Join(current, ".beads", "bd.sock")
# Socket is .beads/bd.sock (project-local), NOT ~/.beads/daemon.sock
```

### 3. Direct exec.Command("bd") calls outside pkg/beads

```bash
grep -rn 'exec\.Command("bd"' --include='*.go' | grep -v test | grep -v .kb/
# Found 11 direct calls across 7 files:
# pkg/daemon/issue_adapter.go:43,143,249
# pkg/daemon/extraction.go:257,272
# pkg/verify/beads_api.go:87,285
# pkg/focus/guidance.go:61
# cmd/orch/init.go:336
# cmd/orch/reconcile.go:510
# cmd/orch/status_cmd.go:1142
```

### 4. Beads fork analysis

```bash
cd ~/Documents/personal/beads
git remote -v
# origin = steveyegge/beads (upstream)
# fork = dylan-conlin/beads (Dylan's fork)

git log --oneline origin/main..HEAD | wc -l
# 43 commits ahead of upstream

git log --oneline --format="%h %ad %s" --date=short origin/main..HEAD
# All 43 commits dated 2025-12-30 to 2026-02-17 (ALL after clean-slate decision)
```

### 5. Fork feature usage in orch-go

Searched for usage of each fork-specific feature across orch-go codebase.

---

## What I Observed

### Finding 1: Model References 5 Files That Don't Exist

The model's "Primary Evidence" and "Source code" sections reference:
- `pkg/beads/fallback.go` → **deleted**, fallback functions consolidated into `client.go` (lines 710-1226)
- `pkg/beads/lifecycle.go` → **deleted**, lifecycle operations distributed across `client.go` methods
- `pkg/beads/id.go` → **deleted**, ID logic absorbed elsewhere
- `pkg/spawn/tracking.go` → **never existed** or renamed; tracking is in `spawn_cmd.go`
- `cmd/orch/spawn.go` → **renamed** to `cmd/orch/spawn_cmd.go`

The model's code examples (e.g., `NewClient()` constructor) are pseudo-code that doesn't match actual implementation. Real constructor is `NewClient(socketPath string, opts ...ClientOption)`.

### Finding 2: Socket Path is Project-Local, Not Global

Model claims: `~/.beads/daemon.sock`
Reality: `.beads/bd.sock` (project-local, found via directory tree walk)

The `FindSocketPath()` function walks up from the given directory looking for `.beads/bd.sock`. There is no global socket at `~/.beads/daemon.sock`.

### Finding 3: 11 Direct exec.Command("bd") Calls Violate Model Constraint

The model states: "All beads integration goes through `pkg/beads`, never direct `exec.Command('bd')`."

Reality: 11 direct `exec.Command("bd", ...)` calls exist across 7 Go source files outside `pkg/beads`:
- `pkg/daemon/issue_adapter.go` (3 calls) — **fallback functions** for when RPC client unavailable
- `pkg/daemon/extraction.go` (2 calls) — `bd create` and `bd dep add` for extraction issues
- `pkg/verify/beads_api.go` (2 calls) — `bd comments` and `bd label add` fallbacks
- `pkg/focus/guidance.go` (1 call) — `bd ready --json`
- `cmd/orch/init.go` (1 call) — `bd init`
- `cmd/orch/reconcile.go` (1 call) — various bd commands
- `cmd/orch/status_cmd.go` (1 call) — `bd config get issue_prefix`

Some are legitimate fallback paths, but `pkg/daemon/extraction.go`, `pkg/focus/guidance.go`, `cmd/orch/init.go`, `cmd/orch/reconcile.go`, and `cmd/orch/status_cmd.go` bypass `pkg/beads` entirely.

### Finding 4: Clean-Slate Decision Completely Reversed

Decision `2025-12-21-beads-oss-relationship-clean-slate.md` says: "Drop all local features and use upstream beads as-is."

Reality: **43 commits** have been added to the fork since that decision. All 43 are dated after Dec 21, 2025. The fork now has substantial local features that are actively used by orch-go:

**Fork features actively used by orch-go:**
| Feature | Fork Commit | orch-go Usage |
|---------|-------------|---------------|
| Question entity type | `2dc8f7dc` (Jan 18) | `serve_beads.go` dashboard API |
| Question gates/deps | `744af9cf` (Jan 18) | `unblocked_collector.go` dependency resolution |
| Title-based dedup | `e19ff3f8` (Feb 16) | `pkg/beads/types.go` CreateArgs.Force field |
| Phase: Complete gate on close | `be871d0c` (Dec 30) | Core verification pipeline |
| Investigation issue type | `d813a87c` (Feb 7) | Tier determination in verify |
| bd close non-zero exit | `a3f8729e` (Feb 5) | Error handling in reconcile.go |

**Fork features NOT used by orch-go (infrastructure improvements):**
| Feature | Purpose |
|---------|---------|
| JSONL-only default mode | Reliability under sandbox |
| Sandbox detection | Prevents SQLite WAL corruption in Claude Code |
| Pre-flight fingerprint validation | Daemon safety |
| Rapid restart loop prevention | Daemon stability |
| Cross-process file locking | Concurrent access safety |
| bd graph --all | Dependency visualization |
| Absorbed-by/supersedes | Relationship tracking |
| Epic readiness gate | Workflow enforcement |
| Pager support | UX improvement |

These "unused" features aren't called from orch-go code but improve beads reliability that orch-go depends on implicitly (e.g., sandbox detection prevents the SQLite corruption that plagued the system).

### Finding 5: RPC-First Pattern Is Confirmed But Implementation Differs

The model's core claim — RPC-first with CLI fallback — is accurate. But the implementation differs from what the model describes:

- Model shows a simple `Client` struct with `rpcClient *rpc.Client` and `fallback bool`
- Reality: `Client` struct uses raw Unix socket with line-oriented JSON protocol, auto-reconnect with configurable retries, `DefaultDir` for cross-project operations
- The `BeadsClient` interface (`interface.go`) is well-designed with `Client`, `CLIClient`, and `MockClient` implementations
- Real constructor: `NewClient(socketPath, ...ClientOption)` with functional options pattern

---

## Model Impact

- [x] **Contradicts** invariant: "Socket path is `~/.beads/daemon.sock`" — Actually `.beads/bd.sock` (project-local)
- [x] **Contradicts** invariant: "All beads integration goes through `pkg/beads`, never direct `exec.Command('bd')`" — 11 direct calls across 7 files
- [x] **Contradicts** file references: 5 of 6 "Primary Evidence" files don't exist (`fallback.go`, `lifecycle.go`, `id.go`, `tracking.go`, `spawn.go`)
- [x] **Contradicts** decision record: Clean-slate decision reversed with 43 fork commits, all post-decision
- [x] **Confirms** core pattern: RPC-first with CLI fallback is the correct architectural description
- [x] **Confirms** three integration points: spawn (create), work (comment), complete (close)
- [x] **Confirms** auto-tracking protocol: spawn creates issues unless `--no-track`
- [x] **Extends** model with: Fork is now a critical dependency with 5-6 actively-used features plus infrastructure improvements orch-go depends on implicitly

---

## Notes

### Model Update Needed

The model was last updated 2026-01-12. Since then:
1. File structure changed significantly (fallback.go, lifecycle.go consolidated)
2. Socket path was always project-local but model had it wrong
3. The exec.Command constraint is aspirational, not enforced — may need to either enforce it or update the constraint to reflect reality
4. The clean-slate decision should be formally superseded with a new decision acknowledging the fork

### Decision Record Staleness

The `2025-12-21-beads-oss-relationship-clean-slate.md` decision is the most stale artifact in the knowledge base:
- States: "Drop all local features and use upstream beads as-is"
- Reality: 43 commits of active fork development, 6+ features actively used
- The decision was effectively reversed within 9 days (first post-decision commit Dec 30, 2025)
- No superseding decision was ever recorded

### Improvement Opportunities

1. **Consolidate exec.Command("bd") calls**: 5 of the 11 direct calls could be migrated to pkg/beads methods
2. **Update model file references**: All "Primary Evidence" paths need updating
3. **Create superseding decision**: Document the actual fork relationship (active fork with upstream tracking)
4. **Fork contribution strategy**: Several fixes (3-char hash, sandbox detection) could be PRed upstream
