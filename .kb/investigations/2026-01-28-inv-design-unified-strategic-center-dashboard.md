## Summary (D.E.K.N.)

**Delta:** Unified Strategic Center combines Decision Center (feat-052) with Knowledge State surfaces into a single meta-orchestrator dashboard section. Design extends the 4-category Decision Center with a 5th "Absorb Learning" category for knowledge hygiene signals (synthesis opportunities, pending promotions, stale decisions).

**Evidence:** Analyzed existing Decision Center design (.kb/investigations/2026-01-27-design-redesign-dashboard-ops-view-meta.md), kb reflect capabilities (8 reflection types implemented), kb quick entries (743 entries across 4 types), dashboard architecture model (.kb/models/dashboard-architecture.md), and principles (Session Amnesia, Surfacing Over Browsing).

**Knowledge:** Knowledge surfaces belong IN the Strategic Center (not separate) because knowledge hygiene IS a decision type ("should I consolidate these investigations?", "should I promote this to a decision?"). The key insight: Decision Center categories map to "what action do I take?" while Knowledge State surfaces map to "what needs my attention for system health?".

**Next:** Implement as extension of feat-052 with 5th category. Implementation: 1) /api/decisions endpoint gains knowledge_signals field, 2) decisions.ts store adds knowledgeSignals derived state, 3) decision-center.svelte adds "Absorb Learning" section below "Absorb Knowledge".

**Promote to Decision:** Issue created: orch-go-21085 (dashboard decision)

---

# Investigation: Design Unified Strategic Center Dashboard

**Question:** How should the dashboard combine Decision Center (agent completions as decisions) with Knowledge State surfaces (recent learnings, pending decisions, synthesis signals, stale models) into a unified Strategic Center?

**Started:** 2026-01-28
**Updated:** 2026-01-28
**Owner:** Architect session
**Phase:** Complete
**Next Step:** None (investigation complete)
**Status:** Complete

---

## Findings

### Finding 1: Decision Center design already establishes action-oriented categorization pattern

**Evidence:** The existing Decision Center investigation (2026-01-27) defines 4 categories:
1. **Absorb Knowledge** - Knowledge-producing skill completions (investigation, architect, research)
2. **Give Approvals** - Items requiring visual verification
3. **Answer Questions** - Strategic questions from questions store
4. **Handle Failures** - Failed verifications, escalated agents

These map escalation levels to user actions. The pattern: group by "what do I DO with this?" not "what IS this?"

**Source:** `.kb/investigations/2026-01-27-design-redesign-dashboard-ops-view-meta.md:114-132`

**Significance:** Knowledge State surfaces should follow the same pattern. The question isn't "what knowledge artifact type?" but "what action does the meta-orchestrator need to take?"

---

### Finding 2: kb reflect already provides 8 types of knowledge hygiene signals

**Evidence:** Running `kb reflect --help` shows implemented reflection types:
- **synthesis**: Investigations needing consolidation (3+ on same topic)
- **promote**: kn entries worth promoting to kb decisions  
- **stale**: Decisions with no citations (>7 days old)
- **drift**: Constraints that may be contradicted by code
- **open**: Investigations with unimplemented recommendations
- **refine**: kn entries that refine existing principles
- **skill-candidate**: kn entry clusters that may warrant skill updates
- **investigation-promotion**: Investigations marked recommend-yes awaiting decision creation

**Source:** `kb reflect --help` output, kb-cli implementation

**Significance:** The backend intelligence for knowledge hygiene ALREADY EXISTS. The dashboard just needs to surface it. This dramatically reduces implementation scope - no new algorithms needed, just API integration and UI.

---

### Finding 3: kb quick entries are substantial and categorized by type

**Evidence:** `.kb/quick/entries.jsonl` contains 743 entries:
- 478 decision entries
- 170 constraint entries  
- 67 attempt (tried/failed) entries
- 28 question entries

These are indexed by `kb context` and surfaced via `kb reflect --type promote`.

**Source:** `cat .kb/quick/entries.jsonl | jq -r '.type' | sort | uniq -c`

**Significance:** There's significant knowledge captured that could benefit from promotion visibility. The "pending promotions" signal is valuable because promoted knowledge has better discoverability.

---

### Finding 4: Dashboard has 666px minimum width constraint and existing section patterns

**Evidence:** From kb context: "Dashboard must be fully usable at 666px width (half MacBook Pro screen). No horizontal scrolling. All critical info visible without scrolling."

Existing sections (QuestionsSection, FrontierSection) follow pattern:
- Collapsible with count badge in header
- Preview text when collapsed
- Categorized content when expanded
- Color-coded severity (red=urgent, yellow=warning, green=resolved)

**Source:** `.kb/guides/dashboard.md`, `questions-section.svelte:73-110`, `frontier-section.svelte:43-78`

**Significance:** Knowledge surfaces must fit this pattern. They should NOT be full-width cards but compact list items like Questions or Frontier sections.

---

### Finding 5: Freshness thresholds should follow existing patterns

**Evidence:** Dashboard already uses time-based thresholds:
- Agent "dead": no activity for 3+ minutes
- Agent "stalled": same phase for 15+ minutes
- Agent "awaiting cleanup": completed but not closed
- kb reflect "stale": decisions with no citations >7 days

**Source:** `needs-attention.svelte:126-129`, `pkg/verify/synthesis_opportunities.go`

**Significance:** Knowledge freshness should use 7-day "recent" threshold (matching kb reflect stale) and 30-day "stale" threshold for decisions. These align with existing patterns.

---

## Synthesis

**Key Insights:**

1. **Knowledge hygiene IS a decision type** - "Should I consolidate these 12 dashboard investigations into a guide?" is a decision just like "Should I close this agent?" The Strategic Center should treat knowledge surfaces as a 5th category, not a separate section.

2. **Backend already exists via kb reflect** - The synthesis detection, promotion candidates, stale decision identification are all implemented in kb-cli. The dashboard integration is API exposure + UI, not algorithm development.

3. **Integration with Questions section is unnecessary** - Questions are about blocking work ("What should this agent do?"). Knowledge signals are about system health ("Should these investigations be consolidated?"). They serve different cognitive modes and should remain separate sections, with Knowledge in Strategic Center and Questions as its own section.

**Answer to Investigation Question:**

The unified Strategic Center should:

1. **Extend Decision Center with 5th category: "Tend Knowledge"** - Contains synthesis opportunities, pending promotions, stale decisions, investigation-promotions. Uses same compact list pattern as other categories.

2. **Add /api/kb-health endpoint** - Calls `kb reflect --format json --limit 5` (each type) and returns aggregated signals. Cached with 5-minute TTL (knowledge changes slowly).

3. **Keep Questions section separate** - Questions are about blocking work, not knowledge hygiene. They serve different cognitive modes.

4. **Use progressive disclosure** - Header shows count only. Expanded shows categorized signals with action buttons (e.g., "Create Guide" for synthesis, "Promote" for candidates).

5. **Freshness thresholds** - 7 days = "recent" (show as informational), 30 days = "stale" (show with warning).

---

## Structured Uncertainty

**What's tested:**

- ✅ Decision Center design follows action-oriented pattern (verified: read investigation)
- ✅ kb reflect provides 8 reflection types (verified: ran kb reflect --help)
- ✅ kb quick entries exist with 743 total (verified: analyzed entries.jsonl)
- ✅ Dashboard section patterns use collapsible + badges (verified: read components)

**What's untested:**

- ⚠️ Performance of kb reflect in API context (not benchmarked - may need background cron)
- ⚠️ User experience of 5-category Strategic Center (not user tested)
- ⚠️ Whether synthesis signals are actionable enough for inline display

**What would change this:**

- If kb reflect is too slow for API call (~500ms threshold), would need background refresh + cache
- If 5 categories creates visual overload, could collapse Knowledge into expandable sub-section of "Absorb Knowledge"
- If synthesis detection produces too many false positives, would need stricter filtering

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach: Extend Decision Center with "Tend Knowledge" Category

**Why this approach:**
- Treats knowledge hygiene as first-class operational concern (matches principle: "Knowledge surfaces need attention too")
- Leverages existing kb reflect backend (no new algorithms)
- Follows established Decision Center action-oriented pattern
- Minimal additional API surface (one endpoint)

**Trade-offs accepted:**
- Adds complexity to Decision Center (5 categories vs 4)
- Requires kb CLI available for API calls (graceful degradation if missing)
- Why that's acceptable: Knowledge management IS part of meta-orchestrator responsibilities; complexity is justified

**Implementation sequence:**

1. **Phase 1: Add /api/kb-health endpoint to serve.go**
   - Calls `kb reflect --format json --limit 5` per type
   - Aggregates into KBHealthResponse struct
   - 5-minute cache TTL
   - Graceful degradation if kb CLI unavailable
   - Why first: Backend foundation

2. **Phase 2: Extend decisions.ts store**
   - Add `knowledgeHealth` field to DecisionsResponse
   - Create `knowledgeSignals` derived state (counts per type)
   - Poll alongside decisions endpoint
   - Why second: Frontend data layer

3. **Phase 3: Add "Tend Knowledge" section to decision-center.svelte**
   - New category below existing 4
   - Purple border (matches Frontier/Questions pattern for strategic content)
   - Shows: synthesis opportunities, pending promotions, stale decisions, investigation-promotions
   - Each item: title + age + action button
   - Why third: User-facing integration

4. **Phase 4: Add action buttons for knowledge operations**
   - "Create Guide" for synthesis candidates (opens terminal with `kb create guide {topic}`)
   - "Promote" for pending decisions (opens terminal with `kb quick promote {id}`)
   - "Review" for investigation-promotions (links to investigation file)
   - Why fourth: Enables action from dashboard

### Alternative Approaches Considered

**Option B: Separate Knowledge State Section (Not Recommended)**
- **Pros:** Clean separation, easier to hide if not wanted
- **Cons:** Violates insight that knowledge hygiene IS a decision type, creates cognitive overhead switching between "decisions about work" and "decisions about knowledge"
- **When to use instead:** If users find 5 categories overwhelming in testing

**Option C: Integrate into Existing Questions Section (Not Recommended)**
- **Pros:** Fewer sections, simpler UI
- **Cons:** Questions are about blocking work; knowledge signals aren't blocking; different cognitive modes
- **When to use instead:** Never - these serve fundamentally different purposes

**Option D: Background Cron + Event-Driven Updates (Future Enhancement)**
- **Pros:** Zero latency on dashboard load
- **Cons:** Additional infrastructure (cron job), staleness window
- **When to use instead:** If kb reflect proves too slow for synchronous API call

**Rationale for recommendation:** Option A (extend Decision Center) best aligns with the insight that knowledge hygiene is a decision type and leverages the existing pattern.

---

### Implementation Details

**What to implement first:**
- /api/kb-health endpoint with basic kb reflect integration
- Graceful degradation (return empty if kb unavailable)
- Cache with 5-minute TTL

**Things to watch out for:**
- ⚠️ kb reflect may be slow with many investigations (667+ in orch-go) - consider --limit flag
- ⚠️ kb CLI path resolution in server context (use existing binutil pattern from feat-050)
- ⚠️ Cache invalidation when knowledge actually changes (acceptable: 5-min staleness)

**Areas needing further investigation:**
- Optimal --limit values per reflection type
- Whether to pre-compute in daemon vs on-demand in serve
- Integration with kb reflect --create-issue for auto-issue creation

**Success criteria:**
- ✅ Meta-orchestrator can see knowledge hygiene signals at a glance
- ✅ Signals grouped by action type (synthesize, promote, review)
- ✅ 666px width constraint maintained
- ✅ Actions enable quick triage (buttons lead to correct commands)

---

## API Design

### /api/kb-health Response

```json
{
  "synthesis": {
    "count": 3,
    "items": [
      {"topic": "dashboard", "investigation_count": 62, "oldest_days": 38}
    ]
  },
  "promote": {
    "count": 12,
    "items": [
      {"id": "kn-abc123", "type": "decision", "value": "Use beads close_reason as fallback"}
    ]
  },
  "stale": {
    "count": 5,
    "items": [
      {"path": ".kb/decisions/2025-12-15-foo.md", "title": "Foo Decision", "days_old": 44}
    ]
  },
  "investigation_promotion": {
    "count": 2,
    "items": [
      {"path": ".kb/investigations/2026-01-27-design-foo.md", "title": "Design Foo", "recommendation": "recommend-yes"}
    ]
  },
  "total": 22,
  "last_updated": "2026-01-28T10:30:00Z"
}
```

### decision-center.svelte Structure (Updated)

```
Decision Center
├── Absorb Knowledge (knowledge-producing completions)
├── Give Approvals (visual verification)
├── Answer Questions (strategic questions)
├── Handle Failures (failed/escalated)
└── Tend Knowledge (NEW - knowledge hygiene)
    ├── Synthesis Opportunities (N)
    ├── Pending Promotions (N)
    ├── Stale Decisions (N)
    └── Investigation Promotions (N)
```

---

## File Targets

**Backend (Go):**
- `cmd/orch/serve_kb_health.go` (new) - /api/kb-health endpoint
- `cmd/orch/serve.go` - Add endpoint registration

**Frontend (Svelte):**
- `web/src/lib/stores/kb-health.ts` (new) - Knowledge health store
- `web/src/lib/components/decision-center/decision-center.svelte` - Add "Tend Knowledge" section
- `web/src/routes/+page.svelte` - Wire kb-health store

---

## Component Structure

```
web/src/lib/components/decision-center/
├── decision-center.svelte (main container)
├── absorb-knowledge.svelte (knowledge completions)
├── give-approvals.svelte (visual verification)
├── answer-questions.svelte (strategic questions - may merge with existing)
├── handle-failures.svelte (failures/escalations)
└── tend-knowledge.svelte (NEW - knowledge hygiene)
```

---

## References

**Files Examined:**
- `.kb/investigations/2026-01-27-design-redesign-dashboard-ops-view-meta.md` - Decision Center design
- `.kb/models/dashboard-architecture.md` - Dashboard architecture patterns
- `.kb/guides/dashboard.md` - Dashboard usage and patterns
- `web/src/lib/components/questions-section/questions-section.svelte` - Section pattern
- `web/src/lib/components/frontier-section/frontier-section.svelte` - Section pattern
- `web/src/lib/components/needs-attention/needs-attention.svelte` - Current attention section
- `pkg/verify/escalation.go` - 5-tier escalation model
- `~/.kb/principles.md` - Session Amnesia, Surfacing Over Browsing principles

**Commands Run:**
```bash
# Check kb reflect capabilities
kb reflect --help

# Count kb quick entry types
cat .kb/quick/entries.jsonl | jq -r '.type' | sort | uniq -c

# List kb directory structure
ls -la .kb/
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-09-dashboard-reliability-architecture.md` - Dashboard as tier-0 infrastructure
- **Investigation:** `.kb/investigations/2026-01-27-design-redesign-dashboard-ops-view-meta.md` - Decision Center design
- **Feature:** `feat-052` in `.orch/features.json` - Decision Center implementation

---

## Investigation History

**2026-01-28 09:00:** Investigation started
- Initial question: How to combine Decision Center with Knowledge State surfaces?
- Context: Task spawned from orchestrator to extend Decision Center design

**2026-01-28 09:30:** Phase 1 (Problem Framing) completed
- Identified that knowledge hygiene IS a decision type
- Defined success criteria: unified view, action-oriented, 666px width

**2026-01-28 10:00:** Phase 2 (Exploration) completed
- Identified 5 decision forks
- Consulted substrate (principles, existing patterns, kb reflect capabilities)
- Key insight: backend for knowledge signals already exists via kb reflect

**2026-01-28 10:30:** Phase 3 (Synthesis) completed
- Navigated all forks with recommendations
- Designed "Tend Knowledge" category extending Decision Center
- Created API contract and component structure

**2026-01-28 11:00:** Investigation completed
- Status: Complete
- Key outcome: Extend Decision Center with 5th "Tend Knowledge" category using existing kb reflect backend
