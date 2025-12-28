<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added `kb reflect --type skill-candidate` that clusters kn entries by topic and surfaces candidates with 3+ entries for skill consolidation.

**Evidence:** Tests pass for clustering logic, topic extraction, recent entry counting, and limit handling.

**Knowledge:** Topic keywords (daemon, spawn, session, etc.) provide better clustering than pure NLP; fallback to first meaningful word handles uncategorized entries.

**Next:** Implementation complete. Deploy kb-cli binary for testing in real environments.

---

# Investigation: Kb Reflect Type Skill Candidate

**Question:** How should kb reflect --type skill-candidate cluster kn entries by topic and surface candidates for skill updates?

**Started:** 2025-12-27
**Updated:** 2025-12-27
**Owner:** Agent (og-feat-kb-reflect-type-27dec)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Topic clustering uses keyword-first, fallback-second approach

**Evidence:** The `clusterEntriesByTopic` function first checks for domain-specific keywords (daemon, spawn, skill, agent, session, etc.) and only falls back to `extractPrimaryTopic` when no keyword matches.

**Source:** kb-cli/cmd/kb/reflect.go:1249-1289

**Significance:** This approach ensures that operational domains are correctly identified even when entries use varied phrasing. Example: both "Daemon must run via launchd" and "Use daemon for overnight work" cluster under "daemon".

---

### Finding 2: Skill candidate threshold is 3+ entries

**Evidence:** `findSkillCandidates` only returns clusters where `len(clusterEntries) >= 3`, consistent with the synthesis threshold pattern.

**Source:** kb-cli/cmd/kb/reflect.go:1223-1228

**Significance:** Matches the existing reflection pattern (synthesis uses 3+ investigations). Prevents noise from single mentions while surfacing genuine topic density.

---

### Finding 3: Recent entry counting provides temporal signal

**Evidence:** `SkillCandidate` struct includes `RecentCount` field counting entries from the last 7 days. This enables orchestrators to prioritize actively-evolving topics.

**Source:** kb-cli/cmd/kb/reflect.go:79-84

**Significance:** Aligns with the Dec 25 example from spawn context - multiple daemon-related kn entries in a short period triggered skill update consideration.

---

## Synthesis

**Key Insights:**

1. **Keyword-based clustering is more reliable than NLP** - Domain-specific keywords (daemon, spawn, session) provide consistent clustering because they represent actual operational boundaries.

2. **Two-repo change required** - kb-cli defines the clustering logic; orch-go/pkg/daemon/reflect.go needs matching types for the daemon to process skill-candidate suggestions.

3. **Temporal density is the real signal** - Per prior decision "Temporal density and repeated constraints are highest value reflection signals", the RecentCount field enables prioritization.

**Answer to Investigation Question:**

The skill-candidate reflection type clusters kn entries by topic keyword matching (fallback to first meaningful word), surfaces clusters with 3+ entries, and includes recent entry counts for temporal prioritization. The implementation spans kb-cli (clustering logic) and orch-go (type definitions for daemon integration).

---

## Structured Uncertainty

**What's tested:**

- ✅ Topic clustering correctly groups entries by keyword (TestClusterEntriesByTopic)
- ✅ Skill candidates found for 3+ entry clusters (TestReflectSkillCandidateFindsClusters)
- ✅ Recent entry counting works (TestReflectSkillCandidateCountsRecentEntries)
- ✅ Limit parameter caps results (TestReflectSkillCandidateWithLimit)
- ✅ Missing .kn handled gracefully (TestReflectSkillCandidateHandlesMissingKnDir)

**What's untested:**

- ⚠️ Real-world clustering accuracy with actual .kn entries (tested only with synthetic data)
- ⚠️ Performance with large .kn files (no benchmark)
- ⚠️ Integration with orch daemon reflect command (only type definitions added)

**What would change this:**

- If real kn entries don't cluster well with current keywords, the topicKeywords list may need expansion
- If performance is an issue with large files, buffered reading may be needed

---

## Implementation Recommendations

**Purpose:** Implementation is complete.

### Implemented Approach ⭐

**Keyword-based topic clustering with fallback** - Entries are clustered by matching domain-specific keywords first, with fallback to extracting the primary topic from content.

**What was implemented:**
- `SkillCandidate` struct with Topic, Count, Entries, EntryTypes, RecentCount, Suggestion
- `findSkillCandidates` function that clusters entries and filters to 3+
- `clusterEntriesByTopic` for keyword-based clustering
- `extractPrimaryTopic` for fallback topic extraction
- Updated Reflect command to handle `--type skill-candidate`
- Updated printReflectText to display skill-candidate results
- orch-go types for daemon integration

---

## References

**Files Examined:**
- kb-cli/cmd/kb/reflect.go - Main reflect implementation
- kb-cli/cmd/kb/reflect_test.go - Test file
- orch-go/pkg/daemon/reflect.go - Daemon reflect types

**Commands Run:**
```bash
# Build kb-cli
go build ./...

# Run skill-candidate tests
go test ./cmd/kb/... -run "SkillCandidate|ExtractPrimaryTopic|ClusterEntries" -v

# Build orch-go
go build ./...

# Run orch-go reflect tests
go test ./pkg/daemon/... -v -run Reflect
```

**Related Artifacts:**
- **Decision:** kb reflect uses single command with --type flag for reflection modes
- **Decision:** Temporal density and repeated constraints are highest value reflection signals

---

## Investigation History

**2025-12-27 19:xx:** Investigation started
- Initial question: How to implement skill-candidate reflection type
- Context: Pattern observed - multiple kn entries on same topic should trigger skill update consideration

**2025-12-27 20:xx:** Implementation completed
- Added SkillCandidate type and findSkillCandidates function to kb-cli
- Added skill-candidate handling to reflect command and output
- Added corresponding types to orch-go/pkg/daemon/reflect.go
- All tests passing

**2025-12-27 20:xx:** Investigation completed
- Status: Complete
- Key outcome: kb reflect --type skill-candidate implemented with topic clustering and 3+ threshold
