## Summary (D.E.K.N.)

**Delta:** pkg/kbgate/publish.go hotspot acceleration is a false positive — 100% of additions (472/461 net) are birth churn from new feature implementation on 2026-03-10. All 4 commits occurred on the same day.

**Evidence:** `git log --numstat` shows: 286 lines (initial gate), +55 (claim-upgrade detector), -5 (scope fix), +125 (ledger validation). All on Mar 10. No subsequent modifications. File is 461 lines — well under 800-line advisory.

**Knowledge:** Unlike extraction false positives (status_infra.go, kb_ask_test.go), this is birth churn from new feature build-out. Same false positive conclusion: entire file existence counted as 30-day growth.

**Next:** Close as false positive. No action needed.

**Authority:** implementation - Tactical classification of false positive, no architectural impact.

---

# Investigation: Hotspot Acceleration — pkg/kbgate/publish.go

**Question:** Is the +472 lines/30d acceleration in pkg/kbgate/publish.go a genuine hotspot risk or a false positive?

**Started:** 2026-03-17
**Updated:** 2026-03-17
**Owner:** orch-go-zca40
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| 2026-03-17-hotspot-acceleration-cmd-orch-kb-ask-test.md | same FP pattern | yes | - |
| 2026-03-17-hotspot-acceleration-cmd-orch-status-infra.md | same FP pattern | yes | - |

## TLDR

publish.go was created on 2026-03-10 via 4 commits all on the same day, building the publish-time adversarial gate feature from scratch. The "+472 lines/30d" metric is the file's 7-day existence counted as growth. At 461 lines with clear cohesion (all publication gate logic), no extraction is needed.

## What I Tried

1. `git log --format="%h %ad %s" --date=short -- pkg/kbgate/publish.go` — all 4 commits on 2026-03-10
2. `git log --numstat -- pkg/kbgate/publish.go` — 286+55+6+125 additions, 11 deletions
3. `git show 6f5497dba --stat` — initial commit created file with 286 lines (new feature, not extraction)
4. `wc -l pkg/kbgate/*.go` — package total 3506 lines across 8 well-separated files

## What I Observed

- **All growth on one day:** 4 commits on 2026-03-10, zero modifications since
  - `6f5497dba`: +286 lines — initial publish gate (contract, challenge artifacts, lineage, banned language)
  - `372a28bd5`: +55 lines — claim-upgrade boundary detector
  - `fc39ba18c`: +6/-11 lines — scope fix (net -5)
  - `9d5b64210`: +125 lines — claim ledger validation
- **New feature, not extraction:** No corresponding file shrank. This was greenfield kbgate package build-out.
- **File size:** 461 lines, well under 800-line advisory and 1500-line critical thresholds
- **Package structure healthy:** kbgate/ has 8 files averaging ~438 lines each, with clear separation:
  - publish.go (461) — publication gate checks
  - claims.go (322) — claim scanning
  - challenge.go (314) — challenge protocol
  - model.go (391) — model gate checks
  - Plus corresponding test files

## Test Performed

Verified via git history that the file was created as a new feature on a single day. Confirmed no prior file was split to produce it (git show of initial commit shows only additions). Checked package-level structure to confirm healthy distribution.

```bash
git log --diff-filter=A --format="%h %ad %s" --date=short -- pkg/kbgate/publish.go
# 6f5497dba 2026-03-10 feat: implement publish-time adversarial gate (orch kb gate publish)

git log --numstat --format="%h %s" -- pkg/kbgate/publish.go
# 9d5b64210: 125+/0-   (ledger validation)
# fc39ba18c: 6+/11-    (scope fix)
# 372a28bd5: 55+/0-    (claim-upgrade detector)
# 6f5497dba: 286+/0-   (initial implementation)

wc -l pkg/kbgate/*.go
# 461 publish.go, 3506 total across 8 files
```

## Conclusion

**False positive.** The hotspot detector counted new-feature birth (4 commits, all on Mar 10) as organic acceleration. Unlike extraction false positives (status_infra.go, kb_ask_test.go), this file was built from scratch — but the false positive mechanism is the same: entire file existence within the 30-day window counted as growth. At 461 lines with clear single-responsibility (publication gate checks), the file is healthy and needs no extraction.
