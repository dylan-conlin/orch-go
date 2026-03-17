---
title: "Hotspot acceleration: pkg/account/capacity.go"
status: Complete
date: 2026-03-17
beads: orch-go-s8j3g
---

## TLDR

`capacity.go` hotspot alert (+654 lines/30d, now 654 lines) is a **false positive** caused by birth churn. The file was created on Mar 10 by extracting capacity-tracking code from `account.go` (which dropped from 1162→513 lines). Zero commits post-birth. No action needed.

## D.E.K.N. Summary

- **Delta:** Investigated hotspot acceleration flag for `pkg/account/capacity.go`
- **Evidence:** Git history shows exactly 1 commit touching this file: the birth extraction commit (5c1982748, Mar 10). `git log --numstat --follow` shows the file was renamed from account.go with 82 lines added and 590 lines carried over. Zero additional commits since extraction.
- **Knowledge:** File was created specifically to reduce account.go below the hotspot threshold. The extraction was successful — account.go dropped from 1162→513 lines. 100% of the "growth" detected is birth churn from the extraction refactor.
- **Next:** Close as false positive. The file is 654 lines, well below the 1500-line threshold. No post-birth growth trend exists.

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| 2026-03-17-hotspot-acceleration-pkg-spawn-kbmodel.md | parallel (same false-positive pattern) | yes | - |

## Question

Is `pkg/account/capacity.go` (+654 lines/30d, now 654 lines) at risk of becoming a critical hotspot requiring extraction?

## Findings

### Finding 1: File was created via extraction, not organic growth

**Tested:** `git log --numstat --follow -- pkg/account/capacity.go`

**Observed:**
- The file has a single commit in its direct history: `5c1982748` (Mar 10, 2026)
- That commit message: "refactor: extract capacity tracking from account.go into capacity.go (orch-go-yqxng)"
- Numstat shows: `82 590 pkg/account/{account.go => capacity.go}` — Git tracked it as a rename with 82 added lines
- Pre-extraction, account.go was 1162 lines; post-extraction: account.go=513, capacity.go=654

### Finding 2: Zero post-birth commits

**Tested:** `git log --since="2026-03-10" -- pkg/account/capacity.go`

**Observed:** Empty output. No commits have touched capacity.go since its creation. The +654 line delta is 100% birth churn.

### Finding 3: File content is cohesive and well-bounded

**Tested:** Read the full 654-line file

**Observed:** The file contains a coherent domain:
- `CapacityInfo` type + methods (IsHealthy, IsLow, IsCritical) — lines 25-68
- API response types — lines 70-88
- `GetCurrentCapacity` / `GetAccountCapacity` — lines 99-182 (API fetching)
- `fetchCapacityWithToken` / `fetchProfileEmail` — lines 184-281 (HTTP internals)
- `ListAccountsWithCapacity` / `FindBestAccount` — lines 283-365 (account selection)
- `RecommendAccount` — lines 376-447 (recommendation algorithm)
- Auto-switch types + logic — lines 449-654 (threshold-based switching)

The file has clear responsibility boundaries. The auto-switch section (lines 449-654, ~200 lines) is the largest sub-domain and could be extracted if the file grows further, but at 654 lines it's well within healthy bounds.

### Finding 4: Package distribution is healthy

**Tested:** `wc -l pkg/account/*.go`

**Observed:**
| File | Lines |
|---|---|
| account.go | 513 |
| account_test.go | 1452 |
| cache.go | 69 |
| cache_test.go | 157 |
| capacity.go | 654 |
| oauth.go | 350 |
| oauth_test.go | 245 |
| **Total** | **3440** |

No file exceeds the 1500-line threshold. The test file (1452 lines) is the largest but test files are exempt from accretion enforcement.

## Test Performed

`git log --since="2026-03-10" -- pkg/account/capacity.go` — confirmed zero post-birth commits.

## Conclusion

**False positive.** 100% birth churn from extraction. The file was intentionally created on Mar 10, 2026 to split the oversized `account.go` (1162 lines). No organic growth has occurred since. At 654 lines, the file is healthy and well below the 1500-line threshold. No action required.
