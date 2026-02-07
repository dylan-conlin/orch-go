# Investigation: Add force support to beads RPC close

**Status:** Complete

## Question

How should `force` be threaded through beads RPC close so `verify.CloseIssueForce` does not skip RPC when `force=true`?

## Findings

### Finding 1: verify layer bypassed RPC when force=true

**Evidence:** `pkg/verify/beads_api.go` used `client.CloseIssue(...)` only when `!force`, otherwise it skipped RPC and always used CLI fallback.
**Source:** `pkg/verify/beads_api.go`
**Significance:** Force-close behavior was inconsistent with other RPC-first operations and added unnecessary CLI shell-out.

### Finding 2: RPC close args had no force field

**Evidence:** `CloseArgs` only contained `id` and `reason`.
**Source:** `pkg/beads/types.go`
**Significance:** Even if call sites wanted force-close over RPC, there was no argument field to send.

### Finding 3: client API had no force-capable close method

**Evidence:** `pkg/beads/client.go` exposed `CloseIssue(id, reason)` only.
**Source:** `pkg/beads/client.go`
**Significance:** Callers needing force-close had to branch around RPC, causing inconsistent behavior.

## Outcome

Added `Force` to `CloseArgs`, added `Client.CloseIssueForce(id, reason, force)`, kept `CloseIssue` as wrapper for backward compatibility, and updated verify layer to call RPC for both normal and force closes before falling back to CLI.
