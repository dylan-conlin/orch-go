# Investigation: Hotspot Acceleration — cmd/orch/hotspot_spawn.go

**TLDR:** `hotspot_spawn.go` (+340 lines/30d) is a false positive — the file was born on March 10 as an extraction from `hotspot.go` (1050→387 lines). No further extraction needed at 339 lines.

## D.E.K.N. Summary

- **Delta:** The +340 lines/30d growth metric for `hotspot_spawn.go` is a false positive. The file was created on 2026-03-10 as an intentional extraction from `hotspot.go` (which dropped from 1050 to 387 lines). The file is well-structured at 339 lines, far below the 800-line warning threshold.
- **Evidence:** `git log --follow --diff-filter=A -- cmd/orch/hotspot_spawn.go` shows creation at commit `68ce5ff42` (2026-03-10). `git show 68ce5ff42^:cmd/orch/hotspot.go` was 1050 lines before extraction, 387 after.
- **Knowledge:** The hotspot growth metric counts file births as growth. Extractions that create new files will always trigger false positives for the new file. The hotspot system should ideally distinguish "born via extraction" from "organic growth."
- **Next:** No code changes needed. Recommend the orchestrator file this as a known false positive pattern: "new files born from extraction trigger +N lines/30d alerts."

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| N/A — novel investigation | - | - | - |

---

**Status:** Complete
**Question:** Does `cmd/orch/hotspot_spawn.go` (+340 lines/30d, now 339 lines) need extraction to prevent becoming a critical hotspot?
**Answer:** No. The +340 line growth is a false positive — the entire file was born as an extraction from `hotspot.go` on 2026-03-10. No further action needed.

## Finding 1: File birth history

**What I tested:** Checked when `hotspot_spawn.go` was first created and from what.

```bash
git log --oneline --follow --diff-filter=A -- cmd/orch/hotspot_spawn.go
# → 68ce5ff42 session: relabel probes as confirmatory... (2026-03-10)

git show 68ce5ff42^:cmd/orch/hotspot.go | wc -l
# → 1050 (before extraction)

git show 68ce5ff42:cmd/orch/hotspot.go | wc -l
# → 387 (after extraction)
```

**What I observed:** The file was created on 2026-03-10 by extracting spawn-integration code from `hotspot.go`. Before extraction, `hotspot.go` was 1050 lines. After, it dropped to 387 lines and `hotspot_spawn.go` was born at 339 lines. This was the accretion enforcement system working correctly.

## Finding 2: Current hotspot file landscape

**What I tested:** Line counts across all hotspot files.

```bash
wc -l cmd/orch/hotspot*.go
#   364 hotspot_analysis.go
#   315 hotspot_coupling_test.go
#   389 hotspot_coupling.go
#   339 hotspot_spawn.go
#  1692 hotspot_test.go
#   389 hotspot.go
#  3488 total
```

**What I observed:** All non-test source files are between 339-389 lines — well-balanced distribution. The extraction from the original 1050-line `hotspot.go` into 4 files was effective. No file is near the 800-line warning threshold.

## Finding 3: Code duplication between RunHotspotCheckForSpawn and runHotspot

**What I tested:** Compared `RunHotspotCheckForSpawn` (hotspot_spawn.go:298-339) with `runHotspot` (hotspot.go:130-193).

**What I observed:** Both functions construct a `HotspotReport`, call all four `analyze*` functions, and collect hotspots. `RunHotspotCheckForSpawn` hardcodes defaults (28 days, threshold 5, bloat 800) while `runHotspot` uses CLI flags. This is mild duplication (~40 lines) but justified — the spawn path needs stable defaults independent of CLI flags, and the functions diverge after collection (one checks task text, the other outputs reports). Not worth extracting a shared helper.

## Conclusion

`hotspot_spawn.go` is healthy at 339 lines. The +340 lines/30d growth metric is a false positive triggered by file birth within the measurement window. The hotspot file family (4 source files, 2 test files) is well-balanced with all source files between 339-389 lines. No extraction needed.

## Sources

None — investigation used only git history and local code inspection.
