## Summary (D.E.K.N.)

**Delta:** Implemented tiered KB context filtering that queries local project first, expands to orch ecosystem if sparse, and applies per-category limits to prevent context flood.

**Evidence:** All tests pass (6 new tests for filtering logic). Implementation reduces noise by filtering out ~33% irrelevant repos (price-watch, dotfiles, scs-slack) while preserving cross-repo orch ecosystem knowledge.

**Knowledge:** Tiered filtering with explicit allowlist is the right approach - local-first reduces noise for project-specific work, ecosystem expansion preserves valuable cross-repo decisions/constraints.

**Next:** Test in production spawns to validate noise reduction. Consider making allowlist configurable in future.

**Confidence:** High (85%) - Implementation complete with tests, but real-world validation pending.

---

# Investigation: Implement Tiered KB Context Filtering

**Question:** How should runPreSpawnKBCheck implement tiered filtering to reduce noise while preserving cross-repo orch ecosystem knowledge?

**Started:** 2025-12-22
**Updated:** 2025-12-22
**Owner:** Feature implementation agent
**Phase:** Complete
**Next Step:** None - implementation complete
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: RunKBContextCheck needed tiered search strategy

**Evidence:** Original implementation used `--global` flag unconditionally, returning 1,200+ results including noise from irrelevant repos (price-watch, dotfiles, scs-slack). The prior investigation showed 33% of results were from non-orch repos.

**Source:** `pkg/spawn/kbcontext.go:63-94` (original implementation)

**Significance:** The global search was returning too much noise, overwhelming agent context. A tiered approach (local first, then global with filtering) addresses this.

---

### Finding 2: Orch ecosystem repos form a stable, identifiable set

**Evidence:** The orch ecosystem consists of 6 repos that consistently need cross-repo knowledge:
- orch-go, orch-cli, kb-cli, orch-knowledge, beads, kn

**Source:** Prior investigation analysis in `.kb/investigations/2025-12-22-inv-pre-spawn-kb-context-noise-filtering.md`

**Significance:** This set is stable enough to hardcode as an allowlist. The filtering logic extracts project names from `[project]` prefixes in match titles.

---

### Finding 3: Per-category limits prevent investigation flood

**Evidence:** Prior testing showed local searches could return 100+ investigations even with targeted queries. Limiting to 20 per category (constraint, decision, investigation, guide) keeps context manageable.

**Source:** MaxMatchesPerCategory constant set to 20; prior investigation findings

**Significance:** Limits prevent any single category from dominating the context while still surfacing the most relevant entries.

---

## Synthesis

**Key Insights:**

1. **Tiered search reduces noise** - By trying local first and only expanding to global when sparse (<3 matches), most spawns get targeted results without cross-repo noise.

2. **Post-filtering is effective** - Even when global search is used, filtering to orch ecosystem repos eliminates irrelevant content (price-watch, dotfiles, scs-slack).

3. **Merge with deduplication** - When combining local and global results, deduplication by type+title prevents duplicate entries in spawn context.

**Answer to Investigation Question:**

Implemented tiered filtering in `RunKBContextCheck()` with three stages:
1. Query current project first (no --global)
2. If sparse (<3 matches), expand to global with orch ecosystem post-filter
3. Apply per-category limits (20) to prevent flood

This preserves valuable cross-repo knowledge while eliminating noise.

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

Implementation is complete with comprehensive tests. The approach follows the prior investigation's recommendations and addresses all identified issues.

**What's certain:**

- ✅ Tiered search logic works correctly (tested)
- ✅ Orch ecosystem filtering removes irrelevant repos (tested)
- ✅ Per-category limits cap results appropriately (tested)
- ✅ Merge logic deduplicates correctly (tested)

**What's uncertain:**

- ⚠️ Real-world noise reduction hasn't been measured in production spawns
- ⚠️ MinMatchesForLocalSearch threshold (3) might need tuning
- ⚠️ Allowlist is hardcoded - may need updates when repos are added

**What would increase confidence to Very High:**

- Production testing with actual spawns
- Measurement of context size before/after filtering
- User feedback on context relevance

---

## Implementation Recommendations

### Recommended Approach ⭐

**Tiered filtering with orch ecosystem allowlist** - Implemented as described in prior investigation recommendations.

**Why this approach:**
- Immediate noise reduction (filters 33% of irrelevant repos)
- Preserves cross-repo knowledge within orch ecosystem
- Minimal latency impact (only runs second query when local is sparse)

**Trade-offs accepted:**
- Hardcoded allowlist requires maintenance
- Post-filtering is less efficient than server-side filtering
- Why acceptable: orch ecosystem is stable, performance is acceptable

**Implementation sequence:**
1. ✅ Add OrchEcosystemRepos allowlist constant
2. ✅ Implement tiered search in RunKBContextCheck
3. ✅ Add post-filtering and per-category limits
4. ✅ Add tests for all new functions

### Alternative Approaches Considered

**Option B: kb-cli --project flag**
- **Pros:** Server-side filtering would be more efficient
- **Cons:** Requires kb-cli changes, not available now
- **When to use instead:** Follow-up enhancement when kb-cli adds --project to context command

**Option C: Better keyword extraction**
- **Pros:** More specific queries reduce noise at source
- **Cons:** Risk of missing relevant content
- **When to use instead:** As additional improvement, not replacement

---

## References

**Files Modified:**
- `pkg/spawn/kbcontext.go` - Added tiered search, filtering, and limit logic
- `pkg/spawn/kbcontext_test.go` - Added 6 new tests for filtering functions

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-22-inv-pre-spawn-kb-context-noise-filtering.md` - Prior investigation with root cause analysis

---

## Self-Review

- [x] Real test performed (all tests pass)
- [x] Conclusion from evidence (not speculation)
- [x] Question answered (implementation complete)
- [x] File complete

**Self-Review Status:** PASSED

---

## Investigation History

**2025-12-22 17:00:** Implementation started
- Initial question: How to implement tiered KB context filtering
- Context: Follow-up to noise filtering investigation

**2025-12-22 17:30:** Implementation complete
- Added OrchEcosystemRepos allowlist
- Implemented tiered search in RunKBContextCheck
- Added post-filtering and per-category limits
- Added 6 tests covering all new functions
- All tests passing

**2025-12-22 17:35:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Tiered filtering implemented with tests, ready for production validation
