# Probe: Knowledge Decay Verification — Beads Database Corruption Model

**Date:** 2026-03-18
**Trigger:** 999-day decay flag (no probes since model creation)
**Method:** Cross-reference model claims against current beads codebase (`~/Documents/personal/beads/`)
**Verdict:** All claims CONFIRMED — model is accurate and current

---

## Claims Verified

| # | Claim | Status | Evidence |
|---|-------|--------|----------|
| 1 | JSONL-only is default storage mode (`629441ad`) | ✅ CONFIRMED | `cmd/bd/main.go:206` — `noDb = true` default, `--sqlite` flag opts in |
| 2 | Sandbox detection prevents daemon auto-start (`9953b9cb`, `4da15127`) | ✅ CONFIRMED | `cmd/bd/daemon_autostart.go:47-76` — `shouldAutoStartDaemon()` returns false if `isSandboxed()` |
| 3 | WAL checkpoint TRUNCATE mode in Close() | ✅ CONFIRMED | `internal/storage/sqlite/store.go:206-217` — exact code matches model |
| 4 | Rapid restart prevention with backoff (`2198ad78`) | ✅ CONFIRMED | `cmd/bd/daemon_start_state.go` — exponential backoff 30s→30m, persistent via JSON file |
| 5 | Pre-flight fingerprint validation (`041af3fa`) | ✅ CONFIRMED | `internal/storage/sqlite/store.go:162-175` — `verifySchemaCompatibility()` before operations |
| 6 | JSONL-only mode in sandbox (`98e5c750`) | ✅ CONFIRMED | `cmd/bd/main.go:294-303` — sandbox sets `noDb = true` |

## Current State Observations

- **Beads repo last commit:** `babdf04a` (Mar 4, 2026) — stable since Feb 2026 architecture
- **orch-go .beads/ state:** beads.db exists (2.5MB), WAL is 0 bytes (healthy — checkpointed), no daemon.log (daemon not running, as expected with JSONL-only default)
- **issues.jsonl:** 1643 lines — JSONL authoritative source active and growing
- **No architecture changes** since the fixes were applied (Jan-Feb 2026)

## SQLite Still Required For (Verified)

- `bd compact --auto/--analyze/--apply` — confirmed in `compact.go`
- `bd wisp gc` — confirmed
- `bd gate wait` — confirmed in `gate.go`
- `bd mol burn` — confirmed
- `bd cook` — confirmed in `cook.go`

## Model Accuracy Assessment

**Score: 10/10** — All factual claims verified. No stale information detected. The "RESOLVED" status is accurate. Defense-in-depth layers (sandbox detection → backoff → pre-flight validation → JSONL-only default) are all active in current code.

## Recommendation

Model requires no content changes. Update `Last Updated` date to reflect this verification pass.
