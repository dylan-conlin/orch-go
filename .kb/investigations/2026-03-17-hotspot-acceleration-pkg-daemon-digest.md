---
Status: Complete
Question: Is pkg/daemon/digest.go a hotspot requiring extraction, and if so, what extraction plan minimizes risk?
Date: 2026-03-17
---

**TLDR:** digest.go is 775 lines, created 3 days ago in a single commit (burst creation, no growth trajectory). However, the file has 4 cleanly separable responsibilities and the daemon package already holds 1,037 lines of digest-related code (digest.go + digest_gate.go). Recommend extraction to `pkg/digest/` to prevent hotspot formation and fix a conceptual coupling where serve_digest.go imports `daemon` for data-layer operations.

## D.E.K.N. Summary

- **Delta:** digest.go growth is burst creation (single commit Mar 14), not sustained growth. BUT at 775 lines it contains 4 distinct responsibilities with zero-dependency seams. Combined with digest_gate.go (262 lines), the daemon package holds 1,037 lines of digest code — extraction is warranted to prevent hotspot formation.
- **Evidence:** `git log --numstat --since="30 days ago" -- pkg/daemon/digest.go` shows exactly 1 commit (714080661, Mar 14): +775 lines. All tests pass (17 tests in digest_test.go). Grep of consumers shows serve_digest.go imports `daemon.DigestStore`/`daemon.DigestProduct` for pure data-layer operations with no daemon dependency.
- **Knowledge:** DigestStore and DefaultDigestService have zero dependency on the Daemon struct. Only `RunPeriodicDigest` (~117 lines) accesses `d.Scheduler`, `d.Digest`, `d.DigestStatePath`, `d.DigestDir`. This is the natural extraction seam.
- **Next:** Recommend `architect` follow-up to create `pkg/digest/` package extracting types, store, service, and helpers (~655 lines), leaving only `RunPeriodicDigest` in daemon (~120 lines).

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
| --- | --- | --- | --- |
| 2026-03-17-hotspot-acceleration-pkg-daemon-trigger.md | Same pattern (burst creation) | yes | This file warrants extraction despite burst creation (trigger.go at 204 lines did not) |

## Findings

### Finding 1: Growth is burst creation, not sustained

Git history for `pkg/daemon/digest.go`:

| Commit | Date | Lines Added | Description |
| --- | --- | --- | --- |
| 714080661 | Mar 14 | +775 | feat: implement digest producer daemon task + API endpoints |
| **Total** | **1 commit** | **+775** | **File creation, no subsequent growth** |

This is the same pattern as trigger.go (false positive for growth trajectory). However, trigger.go was 204 lines — below any threshold. digest.go is 775 lines, just under the 800-line warning threshold.

### Finding 2: Four distinct responsibilities with clean seams

| Section | Lines | Daemon Dependency | Description |
| --- | --- | --- | --- |
| Types/Constants/Interfaces | 1-131 (~131) | None | Type definitions, constants, DigestService interface |
| DigestStore | 133-278 (~146) | None | Filesystem CRUD for products (Write, List, Get, UpdateState, Stats, ArchiveRead) |
| State persistence | 280-313 (~34) | None | LoadDigestState, SaveDigestState |
| RunPeriodicDigest | 315-431 (~117) | **Yes** (d.Scheduler, d.Digest, d.DigestStatePath, d.DigestDir) | Daemon periodic method |
| Internal helpers | 433-529 (~97) | None | classifySignificance, formatTitle, sanitizeSlug, fileHash, etc. |
| DefaultDigestService | 546-775 (~230) | None | Production scanning implementation (threads, models, investigations) |

Only `RunPeriodicDigest` (117 lines) touches the Daemon struct. Everything else (~655 lines) is independent.

### Finding 3: Conceptual coupling in consumers

`cmd/orch/serve_digest.go` imports `pkg/daemon` solely for DigestStore, DigestProduct, DigestListOpts, DigestStatsResponse, and DigestProductState/Type constants. These are data-layer operations with zero daemon logic. Importing the `daemon` package for CRUD operations is a misleading dependency.

Consumers that would change with extraction to `pkg/digest/`:

| File | Current Import | After Extraction |
| --- | --- | --- |
| `cmd/orch/serve_digest.go` | `daemon.DigestStore`, `daemon.DigestProduct`, etc. | `digest.Store`, `digest.Product`, etc. |
| `cmd/orch/serve_digest_test.go` | `daemon.DigestStatsResponse`, `daemon.DigestProduct` | `digest.StatsResponse`, `digest.Product` |
| `cmd/orch/daemon_loop.go` | `daemon.NewDefaultDigestService` | `digest.NewDefaultService` |
| `cmd/orch/daemon_periodic.go` | `daemon.DigestResult` | `digest.Result` |
| `pkg/daemon/daemon.go` | Internal (DigestService field type) | `digest.Service` |
| `pkg/daemon/digest.go` | Stays — RunPeriodicDigest uses `digest` package types | Import `pkg/digest` |
| `pkg/daemon/digest_gate.go` | Internal | Could move to `pkg/digest/` or stay (uses only its own types) |

### Finding 4: Total digest code in daemon package

| File | Lines |
| --- | --- |
| digest.go | 775 |
| digest_gate.go | 262 |
| digest_test.go | 563 |
| digest_gate_test.go | 424 |
| **Total** | **2,024** |

The digest subsystem accounts for 2,024 lines in `pkg/daemon/`. The non-test production code is 1,037 lines (digest.go + digest_gate.go). This is a substantial subsystem masquerading as a daemon feature.

### Finding 5: All tests pass

```
go test ./pkg/daemon/ -run "TestDigest" -count=1
```

17 digest tests pass. The test infrastructure uses `mockDigestService` (defined in digest_test.go) and temporary directories for store operations.

## Test Performed

1. **Git history verification:** `git log --numstat --since="30 days ago" -- pkg/daemon/digest.go` → 1 commit, +775 lines, Mar 14
2. **Dependency analysis:** Grep for all `daemon.Digest*` references across codebase → identified all consumers
3. **Seam analysis:** Read digest.go line-by-line, catalogued which functions reference `*Daemon` vs standalone
4. **Test verification:** `go test ./pkg/daemon/ -run "TestDigest"` → 17 tests pass

## Conclusion

**Verdict: Extraction warranted (preventive, not emergency).**

Unlike trigger.go (204 lines, well-decomposed across 10 files), digest.go at 775 lines contains a self-contained subsystem that belongs in its own package. The burst-creation pattern means there's no growth trajectory to panic about, but the file is already near the 800-line warning threshold and the conceptual coupling (serve_digest.go importing daemon for data operations) is a code smell.

**Recommended extraction plan:**

1. Create `pkg/digest/` with:
   - `types.go` — ProductType, ProductState, Product, Source, State, Stats, StatsResponse, ArtifactChange, ListOpts, Result, Service interface (~131 lines)
   - `store.go` — Store (renamed from DigestStore), all CRUD methods (~146 lines)
   - `state.go` — LoadState, SaveState (~34 lines)
   - `service.go` — DefaultService, scanning implementation (~230 lines)
   - `helpers.go` — classifySignificance, formatTitle, sanitizeSlug, fileHash, etc. (~97 lines)
   - `gate.go` — FeedbackState, adaptive thresholds (from digest_gate.go, ~262 lines)

2. Reduce `pkg/daemon/digest.go` to ~120 lines (RunPeriodicDigest + its helper struct)

3. Update consumers to import `pkg/digest` instead of `pkg/daemon` for data types

**Risk:** Low. Clean seams, all tests pass, no circular dependency risk (daemon imports digest, not vice versa).

**Priority:** P3 — preventive extraction. Not blocking anything today, but prevents the file from becoming a 1,500-line critical hotspot when the next feature lands.

## Recommendations

- Route to `architect` for extraction plan review before implementation
- Create `pkg/digest/` package following the extraction plan above
- Move digest_gate.go content into `pkg/digest/gate.go` (same zero-daemon-dependency pattern)
- Consider renaming types during extraction to drop "Digest" prefix (e.g., `digest.Store` instead of `digest.DigestStore`)
