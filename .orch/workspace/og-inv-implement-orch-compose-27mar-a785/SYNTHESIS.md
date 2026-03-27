# Session Synthesis

**Agent:** og-inv-implement-orch-compose-27mar-a785
**Issue:** orch-go-hmdan
**Duration:** 2026-03-27 ~08:30 → ~08:45
**Outcome:** success

---

## TLDR

Built `orch compose` CLI command (Phase 1 of brief composition layer). The command scans .kb/briefs/, clusters them by keyword overlap using mid-band document frequency filtering and seed-based clustering, matches clusters against .kb/threads/, and writes a digest artifact to .kb/digests/. Tested against 71 real briefs — produces 9 clusters with thread matches.

---

## Plain-Language Summary

`orch compose` reads all the briefs your agents have written and groups them by what they're about. It does this by extracting keywords from each brief, filtering out words that are either too common (appear in everything) or too unique (appear in just one brief), and then clustering briefs that share enough of the remaining "middle-band" vocabulary. For each cluster, it finds threads that might be related, harvests the open questions from each brief's Tension section, and writes everything into a digest file. The digest explicitly labels itself as "unverified clustering" — it's a navigation aid for you, not a comprehension claim.

The interesting discovery: the initial approach (single-linkage clustering with keyword overlap) chains everything into one giant cluster because all your briefs are about the same project. The fix was two-fold: mid-band document frequency filtering (keep only words shared by 2-20% of briefs) and seed-based clustering (only the most-connected brief can recruit members, preventing long transitive chains).

---

## Delta (What Changed)

### Files Created
- `pkg/compose/brief.go` — Brief parser: extracts Frame, Resolution, Tension sections from .md files
- `pkg/compose/keywords.go` — Keyword extraction with stopwords, mid-band DF filtering
- `pkg/compose/cluster.go` — Seed-based clustering algorithm
- `pkg/compose/threads.go` — Thread loading and cluster-to-thread matching
- `pkg/compose/digest.go` — Compose pipeline orchestration and digest markdown rendering
- `pkg/compose/compose_test.go` — 15 tests covering all functions
- `cmd/orch/compose_cmd.go` — Cobra CLI command

### Files Modified
- `cmd/orch/main.go` — Added `rootCmd.AddCommand(composeCmd)`

---

## Evidence (What Was Observed)

- 15 unit tests pass: brief parsing, keyword extraction, clustering, thread matching, digest writing, DF filtering
- Smoke test against 71 real briefs: 9 clusters (2-21 members), 3 unclustered, all clusters have thread matches
- Single-linkage clustering fails catastrophically on same-domain corpora (produces 1 cluster of 71)
- TF-IDF scoring selects for uniqueness (opposite of what clustering needs) — zero overlap between briefs
- Mid-band DF filtering (words in 2-20% of briefs) is the right keyword selection strategy

### Tests Run
```bash
go test ./pkg/compose/ -v -count=1
# PASS: 15 tests, 0.379s
```

---

## Architectural Choices

### Seed-based clustering instead of single-linkage
- **What I chose:** Star/seed clustering where the most-connected brief is the seed and only it can recruit members
- **What I rejected:** Single-linkage (any member can recruit), complete-linkage (all members must overlap with each other), TF-IDF weighted clustering
- **Why:** Single-linkage chains everything in same-domain corpora. Complete-linkage is too restrictive. Seed-based prevents chains while allowing natural cluster formation.
- **Risk accepted:** Clusters are seed-dependent — different ordering could produce different clusters. Acceptable for V1.

### Mid-band document frequency filtering
- **What I chose:** Keep keywords appearing in 2-20% of briefs, remove ubiquitous and unique words
- **What I rejected:** TF-IDF (selects for uniqueness), simple threshold (doesn't remove unique words), no filtering (too much noise)
- **Why:** Clustering signal lives in the "middle band" — words shared by some but not all documents. TF-IDF maximizes distinctiveness which is the opposite goal.
- **Risk accepted:** The 20% threshold is empirically tuned for 71 briefs. May need adjustment as corpus grows.

---

## Knowledge (What Was Learned)

### Constraints Discovered
- Single-linkage clustering is wrong for same-domain document collections — transitive chaining connects everything
- TF-IDF is anti-correlated with clusterability — it selects unique words, not shared vocabulary
- Brief text (~15 lines) is short enough that keyword overlap captures genuine similarity without needing semantic approaches

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing (15/15)
- [x] Investigation file has Status: Complete
- [x] Ready for `orch complete orch-go-hmdan`

---

## Unexplored Questions

- Whether the 9 clusters match Dylan's mental model (requires human review of the digest)
- How cluster quality changes as the brief corpus grows beyond 100
- Whether LLM-powered naming would improve cluster labels (currently keyword-based)

---

## Friction

Friction: tooling: Full CLI build (`go build ./cmd/orch/`) fails due to pre-existing `VerificationTracker` error in `pkg/daemon`. Had to verify compose functionality via standalone test program instead. No impact on compose package itself.

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` — 15 tests pass, 4 design claims verified, smoke test produces 9 clusters from 71 real briefs.

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-inv-implement-orch-compose-27mar-a785/`
**Investigation:** `.kb/investigations/2026-03-27-inv-implement-orch-compose-cli-phase.md`
**Beads:** `bd show orch-go-hmdan`
