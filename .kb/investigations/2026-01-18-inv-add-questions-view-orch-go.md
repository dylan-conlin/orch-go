<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Successfully implemented Questions view infrastructure in orch-go dashboard (API endpoint, store, component) that will display when question entity type becomes available.

**Evidence:** API returns `{"open":[],"investigating":[],"answered":[],"total_count":0}` correctly; component correctly renders only when questions exist.

**Knowledge:** Dashboard component implementation can be done before entity type exists; component conditionally renders based on data availability.

**Next:** Once task 1 (Add question entity type to beads) is complete, Questions will appear in dashboard.

**Promote to Decision:** recommend-no - Implementation task, not architectural decision.

---

# Investigation: Add Questions View to Orch-Go Dashboard

**Question:** How should the Questions view be implemented in the orch-go dashboard?

**Started:** 2026-01-18
**Updated:** 2026-01-18
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: API Endpoint Implementation Pattern

**Evidence:** Examined existing beads endpoints in `serve_beads.go`. Pattern uses CLI client to fetch data, transforms to API response types, and returns JSON.

**Source:**
- `cmd/orch/serve_beads.go:318-370` - handleBeadsReady pattern
- `pkg/beads/cli_client.go:126-152` - List function with filters

**Significance:** Followed same pattern for /api/questions endpoint. Used CLI client's List function with IssueType="question" filter.

---

### Finding 2: CLI Client Lacked Type Filter Support

**Evidence:** The CLIClient.List function only supported --status, --parent, and --limit flags. The --type flag was not passed through.

**Source:** `pkg/beads/cli_client.go:126-152` (before modification)

**Significance:** Added IssueType filter support to CLIClient.List to enable filtering by question type.

---

### Finding 3: Component Conditional Rendering Pattern

**Evidence:** ReadyQueueSection only renders when there are ready issues: `{#if $readyIssues && $readyIssues.count > 0}`. This prevents empty sections from cluttering the UI.

**Source:** `web/src/lib/components/ready-queue-section/ready-queue-section.svelte:36`

**Significance:** Applied same pattern to QuestionsSection - only renders when totalCount > 0.

---

### Finding 4: Question Entity Type Not Yet Available

**Evidence:** `bd create --type question` returns "invalid issue type: question" error. This is expected - task 1 of the epic adds the question entity type.

**Source:** `bd create --type question --title "Test" --priority 2` returns validation error

**Significance:** Implementation is complete and ready; will work automatically once question type exists in beads.

---

## Synthesis

**Key Insights:**

1. **Infrastructure First** - Dashboard infrastructure (API, store, component) can be implemented before entity type exists. Component conditionally renders based on data availability.

2. **Consistent Patterns** - Following existing patterns (beads API, store structure, component layout) ensures consistency and maintainability.

3. **Status Grouping** - Questions are grouped by status: open (needs answer), investigating (in_progress), answered (recently closed) with appropriate color coding (red/yellow/green).

**Answer to Investigation Question:**

The Questions view was implemented by:
1. Adding `/api/questions` endpoint that fetches questions via CLI client and groups by status
2. Creating `questions.ts` store following beads.ts pattern
3. Creating `QuestionsSection` component with status-based grouping and color coding
4. Integrating into dashboard operational mode after UpNextSection

---

## Structured Uncertainty

**What's tested:**

- ✅ API endpoint returns correct JSON structure (verified: curl https://localhost:3348/api/questions)
- ✅ Go code compiles (verified: go build ./cmd/orch/...)
- ✅ Svelte component has no type errors related to questions (verified: npm run check)
- ✅ Component is correctly imported and rendered in dashboard (verified: grep shows imports)

**What's untested:**

- ⚠️ Visual appearance with actual questions (question type not yet available)
- ⚠️ Blocking info display (requires questions with dependencies)
- ⚠️ Age formatting accuracy across different time ranges

**What would change this:**

- If question entity type structure differs from expected, API may need adjustment
- If blocking dependencies are stored differently than expected, blocking info extraction needs update

---

## Implementation Recommendations

**Purpose:** Document the implementation approach taken.

### Implemented Approach

**Status-Grouped Questions View** - Questions grouped by status with color-coded sections.

**Why this approach:**
- Matches design specification from investigation
- Provides clear visual hierarchy (open = urgent/red, investigating = warning/yellow, answered = success/green)
- Follows existing dashboard component patterns

**Trade-offs accepted:**
- Component only renders when questions exist (empty state handled by not showing)
- Only shows recently answered (last 7 days) to avoid clutter

**Implementation sequence:**
1. Added IssueType filter to CLIClient.List
2. Created /api/questions endpoint with status grouping
3. Created questions store
4. Created QuestionsSection component with status grouping
5. Integrated into dashboard

---

## References

**Files Modified:**
- `pkg/beads/cli_client.go:132-133` - Added IssueType filter
- `cmd/orch/serve_beads.go:461-592` - Added handleQuestions handler
- `cmd/orch/serve.go:66,160,282-283,395` - Registered /api/questions endpoint

**Files Created:**
- `web/src/lib/stores/questions.ts` - Questions store
- `web/src/lib/components/questions-section/questions-section.svelte` - Component
- `web/src/lib/components/questions-section/index.ts` - Export

**Files Updated:**
- `web/src/routes/+page.svelte` - Added imports, section state, fetch calls, component render

**Commands Run:**
```bash
# Verify API works
curl -sk https://localhost:3348/api/questions

# Verify build
go build ./cmd/orch/...

# Verify types
npm run check
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-18-inv-design-questions-first-class-entities.md` - Design reference with mockup
- **Epic:** `orch-go-5j2hx` - Parent epic for questions as first-class entities

---

## Investigation History

**2026-01-18 12:30:** Investigation started
- Initial question: How to implement Questions view in dashboard
- Context: Task 4 of epic orch-go-5j2hx

**2026-01-18 12:45:** Implementation complete
- Status: Complete
- Key outcome: Questions view infrastructure ready; will display once question entity type available
