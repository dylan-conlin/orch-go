<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Temporal density and repeated constraints are the highest value/noise signals for autonomous reflection; daemon is the recommended trigger mechanism.

**Evidence:** Found 4 duplicate constraint entries on "tmux fallback", 4 investigations on same topic (tmux fallback test), 37 kn entries in single day (12/21) vs 7 the day before - all measurable signals.

**Knowledge:** Reflection should be triggered by density thresholds, not time intervals; three mechanisms (hook, daemon, command) serve different use cases.

**Next:** Implement `orch reflect` command as MVP, daemon integration for autonomous overnight runs.

**Confidence:** High (80%) - Concrete data supports signal ranking; implementation details need validation.

---

# Investigation: Temporal Signals for Autonomous Reflection

**Question:** Which signals have highest value/noise ratio for triggering autonomous reflection? What's the trigger mechanism (hook vs daemon vs command)?

**Started:** 2025-12-21
**Updated:** 2025-12-21
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (80%)

---

## Findings

### Finding 1: Repeated Constraints Signal (HIGH VALUE)

**Evidence:** Found 5 kn entries about "tmux fallback" created within 2 hours:
- 3 constraints with nearly identical content
- 2 exact duplicates created 38 seconds apart
- Multiple agents encountering same edge case and documenting it independently

```bash
$ cat .kn/entries.jsonl | jq -s '[.[] | select(.content | test("tmux fallback"; "i"))] | length'
5
```

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/.kn/entries.jsonl`

**Significance:** Repeated constraints indicate either:
1. Knowledge not being discovered (agents recreating existing knowledge)
2. Confusion about the constraint (needs clarification/consolidation)
3. Important constraint worth promoting to principles

This is a high-value signal because:
- Low false positive rate (duplicate = definitely needs attention)
- High actionability (consolidate, clarify, or promote)
- Easy to detect programmatically

---

### Finding 2: Investigation Clustering Signal (HIGH VALUE)

**Evidence:** 4 investigations on "tmux fallback test" pattern in single day:
- `inv-test-tmux-fallback.md`
- `inv-test-tmux-fallback-10.md`  
- `inv-test-tmux-fallback-11.md`
- `inv-test-tmux-fallback-12.md`

46 investigations total on 2025-12-21, vs 95 on 2025-12-20.

**Source:** `ls .kb/investigations/*.md | grep "2025-12-21"`

**Significance:** Investigation clustering suggests:
1. Complex problem requiring multiple iterations
2. Potential for synthesis opportunity
3. Knowledge worth consolidating

High value because:
- Pattern is detectable (similar names, same date)
- Clear action (synthesize findings)
- Prevents knowledge fragmentation

---

### Finding 3: Temporal Density Signal (MEDIUM VALUE)

**Evidence:** kn entry creation patterns:
- 2025-12-21: 37 entries
- 2025-12-20: 7 entries  
- 2025-12-19: 1 entry

Within single sessions, entries created seconds apart:
- 23 pairs of entries created < 5 minutes apart
- Many clusters of 3+ entries in rapid succession

**Source:** `jq` analysis of `.kn/entries.jsonl` timestamps

**Significance:** High density indicates:
1. Intense learning period (worth consolidation)
2. Potential for duplicate/redundant entries
3. Cognitive overload (might miss patterns)

Medium value because:
- Higher false positive rate (intensity ≠ need for reflection)
- Context needed to evaluate (not all bursts are problematic)
- Good for triggering human review, less good for autonomous action

---

### Finding 4: Failed Attempts Signal (MEDIUM VALUE)

**Evidence:** 4 failed attempts recorded:
1. "debugging Insufficient Balance error" - wrong thing being checked
2. "orch tail on tmux agent" - missing fallback path
3. "Using empty string model resolution" - incorrect assumption
4. "orch clean to remove ghost sessions" - incomplete cleanup

**Source:** `jq '.[] | select(.type == "attempt" and .outcome == "failed")' .kn/entries.jsonl`

**Significance:** Failed attempts are valuable for:
1. Preventing future retry loops
2. Documenting dead ends
3. Informing design decisions

Medium value because:
- Requires semantic analysis to find related failures
- Some failures are one-off (not worth consolidating)
- Better as context for other signals than standalone trigger

---

### Finding 5: Citation Convergence Signal (LOW VALUE - Not Observable)

**Evidence:** No citation tracking mechanism exists in current system:
- kn entries have `ref_count` field but all are 0
- Investigations don't track which artifacts they reference
- No automatic linking between related artifacts

**Source:** `jq '.ref_count' .kn/entries.jsonl` - all zeros

**Significance:** This signal COULD be valuable but infrastructure doesn't exist to detect it. Would require:
1. Tracking when kn entries are used in spawn contexts
2. Tracking when investigations cite other investigations
3. Building a citation graph

Low value currently because:
- Not observable without significant infrastructure
- Would require changes to multiple tools
- Benefit uncertain until tested

---

### Finding 6: Staleness Signal (LOW VALUE - Hard to Detect)

**Evidence:** No reliable staleness indicators found:
- Timestamps exist but "old" doesn't mean "stale"
- Some 2-day-old entries are current truths
- Some same-day entries are already superseded

**Source:** Manual review of kn entries by date

**Significance:** Staleness is context-dependent:
1. A decision about Python orch-cli might be irrelevant now (Go rewrite)
2. A constraint about tmux is still valid
3. Can't determine staleness without semantic analysis

Low value because:
- High false positive rate (age ≠ staleness)
- Requires codebase context to evaluate
- Better handled by explicit supersession than detection

---

### Finding 7: Trigger Mechanism Analysis

**Evidence:** Three candidate mechanisms evaluated:

| Mechanism | Pros | Cons | Best For |
|-----------|------|------|----------|
| **Hook (SessionStart)** | Runs automatically, always-on | Context cost, adds latency | Surfacing existing suggestions |
| **Daemon** | Runs offline, no interaction cost | Delayed feedback, needs scheduling | Overnight batch analysis |
| **Command** | On-demand, explicit intent | Manual trigger required | Ad-hoc reflection requests |

Existing infrastructure:
- SessionStart hook already injects context (usage warning example)
- Daemon already runs autonomous work (`orch daemon run`)
- Commands are the primary CLI interface

**Source:** `/Users/dylanconlin/.claude/hooks/`, `pkg/daemon/daemon.go`

**Significance:** Each mechanism serves different use case:
1. **Hook**: "There are 3 potential duplicates to review" (surfacing)
2. **Daemon**: "Overnight analysis found 5 consolidation opportunities" (batch)
3. **Command**: `orch reflect` - "What needs attention?" (on-demand)

Recommended: **Daemon + Command hybrid**
- Daemon performs overnight analysis, stores results
- SessionStart hook surfaces "X items need review"
- Command allows on-demand triggering

---

## Synthesis

**Key Insights:**

1. **Signal Hierarchy Established** - Repeated constraints and investigation clustering are highest value (low noise, high actionability). Temporal density is medium (needs context). Citation convergence and staleness are low (not observable or high noise).

2. **Density Thresholds Over Time Intervals** - Reflection should trigger on "3+ similar constraints" or "4+ investigations on topic", not "weekly review". This matches actual patterns found in the data.

3. **Hybrid Trigger Architecture** - No single mechanism fits all use cases. Daemon for batch analysis + hook for surfacing + command for on-demand creates complete coverage.

**Answer to Investigation Question:**

**Which signals have highest value/noise ratio?**
1. **Repeated constraints** (HIGH) - 5 tmux fallback entries, 3 near-duplicates
2. **Investigation clustering** (HIGH) - 4 iterations of same test
3. **Temporal density** (MEDIUM) - 37 entries in one day, but needs context
4. **Failed attempts** (MEDIUM) - 4 documented failures, good context
5. **Citation convergence** (LOW) - Infrastructure doesn't exist
6. **Staleness** (LOW) - High false positive rate

**What's the trigger mechanism?**
- **Primary**: Daemon for overnight batch analysis
- **Secondary**: Command for on-demand reflection
- **Tertiary**: Hook for surfacing existing suggestions

The daemon can detect patterns (repeated constraints, investigation clusters) and create a `~/.orch/reflect-suggestions.json`. SessionStart hook can surface "X items need review" if suggestions exist. Command allows explicit `orch reflect` for immediate analysis.

---

## Confidence Assessment

**Current Confidence:** High (80%)

**Why this level?**
- Concrete data supports signal ranking (tested against real kn/kb state)
- Mechanism analysis based on existing infrastructure
- Implementation details still need validation

**What's certain:**

- ✅ Repeated constraints signal is measurable and actionable
- ✅ Investigation clustering is detectable and valuable
- ✅ Daemon infrastructure exists (pkg/daemon)
- ✅ Hook mechanism exists (session-start.sh)

**What's uncertain:**

- ⚠️ Exact thresholds for triggering (3+ duplicates? 4+ iterations?)
- ⚠️ Output format for reflection suggestions
- ⚠️ How to present actionable recommendations

**What would increase confidence to Very High:**

- Implementing MVP and testing with real usage
- Validating thresholds against historical data
- User feedback on suggestion quality

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Daemon-first with Command Escape Hatch** - Implement reflection analysis as daemon capability with `orch reflect` command.

**Why this approach:**
- Daemon already exists and runs overnight
- Command provides immediate access when needed
- Avoids context cost of hook injection

**Trade-offs accepted:**
- Not real-time (daemon runs periodically)
- Requires explicit check (unlike always-on hook)

**Implementation sequence:**
1. Add `orch reflect` command that analyzes kn/kb state
2. Add reflection analysis to daemon run cycle
3. Add SessionStart hook to surface suggestions (if exist)

### Alternative Approaches Considered

**Option B: Hook-Primary**
- **Pros:** Always-on, no manual trigger needed
- **Cons:** Context cost every session, even when nothing to report
- **When to use instead:** If reflection suggestions are extremely high value

**Option C: Command-Only**
- **Pros:** Minimal infrastructure, explicit intent
- **Cons:** Relies on user remembering to check
- **When to use instead:** MVP/prototype phase

**Rationale for recommendation:** Daemon-first matches the existing pattern (daemon runs overnight, surfaces issues next day). Hook surfaces results without doing heavy analysis. Command provides escape hatch for immediate needs.

---

### Implementation Details

**What to implement first:**
1. `orch reflect` command that checks for:
   - Duplicate kn entries (same content or nearly same)
   - Repeated constraints (same topic)
   - Investigation clusters (same topic, same day)
2. Output format: JSON or human-readable summary

**Detection algorithms:**

```go
// Repeated constraints detection
func FindRepeatedConstraints(entries []knEntry) []DuplicateGroup {
    // Group by normalized content (lowercase, strip punctuation)
    // Return groups with count > 1
}

// Investigation clustering detection  
func FindInvestigationClusters(files []string) []Cluster {
    // Extract topic from filename (remove date, inv- prefix)
    // Group by topic similarity
    // Filter to clusters with 3+ members
}
```

**Things to watch out for:**
- ⚠️ Don't count intentional similar entries as duplicates
- ⚠️ Some clusters are valid (iterative testing)
- ⚠️ Avoid false positives that train user to ignore suggestions

**Success criteria:**
- ✅ `orch reflect` runs in < 5 seconds
- ✅ Produces actionable suggestions (not just raw data)
- ✅ Low false positive rate (< 20% noise)

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-go/.kn/entries.jsonl` - kn entry analysis
- `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/` - investigation clustering
- `/Users/dylanconlin/.claude/hooks/session-start.sh` - existing hook pattern
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/daemon/daemon.go` - daemon infrastructure

**Commands Run:**
```bash
# Duplicate detection
cat .kn/entries.jsonl | jq -s '[.[] | select(.content | test("tmux fallback"; "i"))] | length'

# Investigation clustering
ls .kb/investigations/*.md | grep "2025-12-21" | wc -l

# Temporal density
cat .kn/entries.jsonl | jq -r '.created_at | split("T")[0]' | sort | uniq -c

# Near-duplicate analysis
cat .kn/entries.jsonl | jq -s 'reduce pairs...'
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-21-inv-knowledge-promotion-paths.md` - Knowledge flow mechanisms
- **Investigation:** `.kb/investigations/2025-12-21-synthesis-registry-evolution-and-orch-identity.md` - System architecture context

---

## Self-Review

- [x] Real test performed (not code review) - Ran jq queries against actual kn/kb data
- [x] Conclusion from evidence (not speculation) - Signal ranking based on measurable counts
- [x] Question answered - Both signal ranking and mechanism recommendation provided
- [x] File complete - All sections filled
- [x] D.E.K.N. filled - Summary section complete
- [x] NOT DONE claims verified - Checked citation infrastructure (ref_count all zeros)

**Self-Review Status:** PASSED

---

## Investigation History

**2025-12-21 14:30:** Investigation started
- Initial question: Which temporal signals trigger autonomous reflection?
- Context: Part of broader knowledge management architecture design

**2025-12-21 14:45:** Key discovery
- Found 5 duplicate kn entries on "tmux fallback"
- Found 4 investigation iterations on same topic
- Confirmed ref_count infrastructure unused

**2025-12-21 15:00:** Mechanism analysis complete
- Daemon-first approach recommended
- Hook for surfacing, command for on-demand

**2025-12-21 15:15:** Investigation completed
- Final confidence: High (80%)
- Status: Complete
- Key outcome: Temporal density + repeated constraints are highest value signals; daemon + command hybrid is recommended mechanism
