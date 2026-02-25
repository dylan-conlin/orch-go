# Probe: Preview() nil pointer crash on listIssuesFunc

**Status:** Complete
**Date:** 2026-02-24
**Model:** daemon-autonomous-operation
**Issue:** orch-go-1210

## Question

Does `Preview()` correctly resolve `listIssuesFunc` using the same fallback pattern as `NextIssueExcluding()`, or does it call the field directly and crash when nil?

## What I Tested

1. Read `preview.go:65` — prior code called `d.listIssuesFunc()` directly
2. Compared with `daemon.go:239` (`NextIssueExcluding`) which calls `d.resolveListIssuesFunc()()`
3. Verified `NewWithConfig()` and `NewWithPool()` do NOT set `listIssuesFunc` (it's nil by default)
4. Verified `resolveListIssuesFunc()` handles nil correctly: falls back to multi-project or single-project `ListReadyIssues`
5. Ran `TestDaemon_Preview_NilListIssuesFunc` — creates daemon via `NewWithConfig(Config{})` and calls `Preview()`, verifying no panic
6. Ran full daemon test suite — all tests pass

## What I Observed

- **Root cause:** `preview.go:65` called `d.listIssuesFunc()` directly instead of `d.resolveListIssuesFunc()()`. Since neither constructor sets `listIssuesFunc`, this was always nil in production, causing a nil pointer dereference.
- **Fix:** Changed line 65 from `d.listIssuesFunc()` to `d.resolveListIssuesFunc()()`, matching the pattern already used in `NextIssueExcluding()`.
- **Regression test:** `TestDaemon_Preview_NilListIssuesFunc` added to `preview_test.go` — creates a daemon via the production constructor path and calls `Preview()`, confirming no panic.
- **No other callsites affected:** Grep confirmed no other production code directly calls `d.listIssuesFunc()`.

## Model Impact

**Confirms:** The daemon's dependency injection pattern (mockable function fields with `resolve*` fallback methods) is the correct pattern. `listIssuesFunc` is for test mocking only; production callers must use `resolveListIssuesFunc()`.

**Extends:** The model's "Why This Fails" section should note a new failure mode:

### 4. Function Field Nil Dereference
**What happens:** New methods added to Daemon call function fields directly instead of through resolve methods.
**Root cause:** The `listIssuesFunc` field is nil in production (only set in tests). New code copies the field name without using the resolver.
**Prevention:** Always use `d.resolveListIssuesFunc()` (or equivalent resolve method) instead of `d.listIssuesFunc` directly. The resolve method handles nil fallback.
