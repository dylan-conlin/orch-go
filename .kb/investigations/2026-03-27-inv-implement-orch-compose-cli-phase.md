## Summary (D.E.K.N.)

**Delta:** Implemented `orch compose` CLI command — Phase 1 of the brief composition layer. Scans .kb/briefs/, clusters by keyword overlap with mid-band document frequency filtering and seed-based clustering, matches clusters against threads, writes digest to .kb/digests/.

**Evidence:** 15 tests pass covering brief parsing, keyword extraction, clustering, thread matching, digest writing, and end-to-end composition. Smoke test against 71 real briefs produces 9 clusters with thread matches and 3 unclustered briefs.

**Knowledge:** Pure keyword clustering requires three techniques working together: (1) stopword removal for English + reasoning vocabulary, (2) mid-band document frequency filtering (remove words in >20% or <2 briefs), (3) seed-based clustering instead of single-linkage (prevents chaining). TF-IDF selects for uniqueness which is the opposite of what clustering needs — mid-band filtering is the right approach for brief-sized documents.

**Next:** Close. Phase 2 (orient integration) and Phase 3 (tension harvesting flow) are separate issues.

**Authority:** implementation - Implements the design from investigation 2026-03-26-design-brief-composition-layer.md within existing CLI patterns.

---

# Investigation: Implement Orch Compose CLI Phase 1

**Question:** Can keyword-based brief clustering produce meaningful clusters from 71 real briefs, and what filtering strategy makes it work?

**Started:** 2026-03-27
**Updated:** 2026-03-27
**Owner:** orch-go-hmdan
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| 2026-03-26-design-brief-composition-layer.md | implements | Yes — followed digest format, clustering approach, epistemic labeling | Found MinKeywordOverlap=3 chains everything with single-linkage; needed seed-based clustering |

---

## Findings

### Finding 1: Single-linkage clustering chains everything into one mega-cluster

**Evidence:** First attempt with simple keyword overlap (MinKeywordOverlap=3) and single-linkage produced 1 cluster containing all 71 briefs. Even after stopword removal and document-frequency filtering, briefs in the same project share enough vocabulary (~92 keywords/brief after filtering) that any pair can be connected through intermediaries.

**Source:** Debug analysis showed 969 out of 2485 pairs have 5+ overlap; 241 pairs have 3+ overlap. With single-linkage, these edges form one connected component.

**Significance:** The design's recommendation of "3+ keyword overlap" was correct for direct pairs but didn't account for transitive chaining. Seed-based clustering (where only the seed brief can recruit members) prevents this.

---

### Finding 2: TF-IDF selects for uniqueness, not clusterability

**Evidence:** After applying TF-IDF scoring to keep only top-15 keywords per brief, ALL keywords became unique to individual briefs (words like "exploration-judge", "backpropagatecompletion"). Zero pairs had 2+ overlap. TF-IDF maximizes distinctiveness, which is the opposite of what clustering needs.

**Source:** Debug analysis: avg 15.0 kw/brief after TF-IDF, max pairwise overlap = 1.

**Significance:** For clustering, the right approach is "mid-band" document frequency filtering — keep words shared by SOME briefs (2-20%) but not ALL. This selects the vocabulary where clustering signal lives.

---

### Finding 3: Mid-band filtering + seed clustering produces 9 meaningful clusters

**Evidence:** Final approach: (1) expanded stopwords for reasoning vocabulary, (2) mid-band DF filter (remove words in >20% or <2 briefs, keep top 25/brief), (3) seed-based clustering with MinKeywordOverlap=3. Result: 9 clusters from 71 briefs, ranging from 2-21 members, with 3 unclustered briefs and thread matches for all clusters.

**Source:** `go run /tmp/test_compose.go` against real `.kb/briefs/` directory.

**Significance:** The clustering quality is V1-appropriate. Cluster names from top shared keywords are rough but functional. Thread matching works via keyword overlap against thread content.

---

## Synthesis

**Key Insights:**

1. **Mid-band filtering is the key innovation** — not TF-IDF, not simple stopwords, but filtering to words that appear in 2-20% of briefs. This is the vocabulary band where clustering signal lives.

2. **Seed-based clustering prevents chaining** — single-linkage is wrong for same-domain corpora. When all documents share vocabulary, any transitive chain connects everything. Seed-based (star) clustering bounds each cluster to one hub.

3. **Brief text is short enough that keyword overlap works** — briefs average ~15 lines. At this scale, keyword overlap captures genuine content similarity without needing semantic/LLM approaches.

**Answer to Investigation Question:**

Yes, keyword-based clustering produces meaningful clusters from 71 real briefs, but only with the right filtering strategy. The three-technique combination (expanded stopwords + mid-band DF filtering + seed-based clustering) is necessary and sufficient for V1.

---

## Structured Uncertainty

**What's tested:**

- ✅ 15 unit tests pass covering all compose package functions
- ✅ Smoke test against 71 real briefs produces 9 clusters with thread matches
- ✅ Digest format matches design specification (frontmatter, epistemic label, clusters, tensions)

**What's untested:**

- ⚠️ Whether the 9 clusters match what Dylan would identify reading the same briefs
- ⚠️ Whether mid-band DF filtering generalizes as the brief corpus grows beyond 100
- ⚠️ Whether the thread matching produces meaningful connections (keyword overlap against full thread content may be too broad)

**What would change this:**

- If Dylan reports clusters don't match his mental model, may need semantic/LLM clustering (Phase 2 upgrade path per design)
- If corpus grows and clusters become too large, may need to lower MaxDocumentFrequency or increase MinKeywordOverlap

---

## References

**Files Created:**
- `pkg/compose/brief.go` — Brief parsing (Frame/Resolution/Tension extraction)
- `pkg/compose/keywords.go` — Keyword extraction, stopwords, mid-band DF filtering
- `pkg/compose/cluster.go` — Seed-based clustering algorithm
- `pkg/compose/threads.go` — Thread loading and cluster-to-thread matching
- `pkg/compose/digest.go` — Compose pipeline and digest rendering
- `pkg/compose/compose_test.go` — 15 tests
- `cmd/orch/compose_cmd.go` — CLI command

**Files Modified:**
- `cmd/orch/main.go` — Registered composeCmd

**Related Artifacts:**
- **Design:** `.kb/investigations/2026-03-26-design-brief-composition-layer.md`
