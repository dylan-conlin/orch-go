# SYNTHESIS: Fix nil pointer dereference in daemon preview

## Plain-Language Summary

`orch daemon preview` crashed with a nil pointer dereference at `pkg/daemon/preview.go:65` because `Preview()` called `d.listIssuesFunc()` directly — a field that's only set in tests. The production code path requires calling `d.resolveListIssuesFunc()` instead, which falls back to `ListReadyIssues` when the mock field is nil. This is exactly how `NextIssueExcluding()` (the other consumer) works at `daemon.go:239`. One-line fix, plus a regression test.

## Delta

- **`pkg/daemon/preview.go:65`**: Changed `d.listIssuesFunc()` to `d.resolveListIssuesFunc()()` — matches the pattern used by `NextIssueExcluding` at `daemon.go:239`.
- **`pkg/daemon/preview_test.go`**: Added `TestDaemon_Preview_NilListIssuesFunc` regression test that creates a Daemon via `NewWithConfig` (production path) and calls `Preview()`, verifying no panic.

## Root Cause

The `listIssuesFunc` field on `Daemon` is a test-only mock. `NewWithConfig()` never sets it. The `resolveListIssuesFunc()` method exists specifically to handle this — it checks the mock first, then project registry, then falls back to `ListReadyIssues`. `Preview()` was the only caller that bypassed this resolver.

## Verification Contract

See `VERIFICATION_SPEC.yaml` — build, vet, regression test, full preview test suite, and smoke-test all pass.
