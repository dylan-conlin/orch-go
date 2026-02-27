# SYNTHESIS: Fix daemon preview nil pointer crash

## Plain-Language Summary

`orch daemon preview` was crashing with a nil pointer dereference because `Preview()` called `d.listIssuesFunc()` directly, but that field is nil in production — it's only set in tests for mocking. The fix changes line 65 of `preview.go` to use `d.resolveListIssuesFunc()()`, which provides the correct fallback (multi-project or single-project `ListReadyIssues`). This matches the pattern already used by `NextIssueExcluding()`. A regression test confirms no panic when the daemon is created via the production constructor.

## Delta

- **`pkg/daemon/preview.go:65`**: Changed `d.listIssuesFunc()` to `d.resolveListIssuesFunc()()`
- **`pkg/daemon/preview_test.go`**: Added `TestDaemon_Preview_NilListIssuesFunc` regression test

## Verification Contract

See `VERIFICATION_SPEC.yaml` in this workspace. Key outcome: `orch daemon preview` no longer crashes when `listIssuesFunc` is nil (the production default).

## Root Cause

The `Daemon` struct uses dependency injection: `listIssuesFunc` is a function field for test mocking, with `resolveListIssuesFunc()` providing a nil-safe fallback. `Preview()` was added without using the resolver, creating a nil dereference on every production call path.

## Probe

Path: `.kb/models/daemon-autonomous-operation/probes/2026-02-24-probe-preview-nil-pointer-listissuesfunc.md`

Extends the daemon model with a new failure mode: "Function Field Nil Dereference" — new methods that call function fields directly instead of through resolve methods will crash in production.
