<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Multi-project filtering was incomplete - frontend didn't serialize includedProjects, backend only accepted single filter, and filtering used ProjectDir instead of Project field.

**Evidence:** Smoke tests confirm fix works: pw agent excluded from orch-go,orch-cli,beads filter (was showing before); pw agent included in pw filter (correct behavior).

**Knowledge:** Cross-project agents have ProjectDir=spawner-cwd, Project=target-project, so filtering must use Project field not ProjectDir for correct multi-project coordination.

**Next:** Close issue - fix implemented, tests passing, smoke tested successfully.

**Promote to Decision:** recommend-no (tactical fix, not architectural - implements existing partial design fully)

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Dashboard Follow Orchestrator Broken Implemented

**Question:** Why do price-watch agents show in orch-go dashboard when follow-orchestrator filtering should hide them?

**Started:** 2026-01-14 15:00
**Updated:** 2026-01-14 15:45
**Owner:** og-debug-dashboard-follow-orchestrator-14jan-2226
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Frontend stores included_projects but doesn't pass them to API

**Evidence:** The filter state in `context.ts` has both `project` and `includedProjects` fields (lines 23-24), and `setProjectFilter` accepts both parameters (line 125), but `buildFilterQueryString` only serializes `state.project` (lines 170-172), completely ignoring `includedProjects`.

**Source:** `web/src/lib/stores/context.ts:170-172`, `web/src/routes/+page.svelte:237`

**Significance:** The orchestrator context correctly identifies multi-project coordination (orch-go + orch-cli + beads + kb-cli + orch-knowledge + opencode), but this information never reaches the agents API, so filtering defaults to single-project mode.

---

### Finding 2: Backend filterByProject only accepts single filter string

**Evidence:** The `filterByProject` function signature is `func filterByProject(projectDir, filter string) bool` - it takes a single filter string and matches against a single projectDir. There's no array support or multi-project matching logic.

**Source:** `cmd/orch/serve_filter.go:63-86`

**Significance:** Even if the frontend passed all 6 included projects, the backend has no way to handle them - it would need to accept `[]string` and check if projectDir matches ANY of the filters.

---

### Finding 3: Price-watch agent returns project_dir pointing to orchestrator cwd

**Evidence:** curl test shows `pw-feat-implement-material-category` with `"project": "pw"` but `"project_dir": "/Users/dylanconlin/Documents/personal/orch-go"` - this is the orchestrator's working directory, not the price-watch repo path.

**Source:** `curl 'https://localhost:3348/api/agents?project=orch-go'` output

**Significance:** Cross-project agents spawned with `--workdir` have incorrect project_dir in the API response (shows spawner cwd instead of target dir), BUT this was already fixed in issue orch-go-j5h4w via kb projects integration. The real issue is that filtering isn't using the included_projects array.

---

## Synthesis

**Key Insights:**

1. **Beads filtering works, agents filtering incomplete** - The follow-orchestrator feature was partially implemented (commit b6567f78) for beads API with project_dir parameter and multi-project awareness, but agents API only got single-project support.

2. **Frontend correctly tracks multi-project context but doesn't use it** - The FilterState stores both `project` and `includedProjects`, orchestratorContext polls and updates them, but buildFilterQueryString only serializes the primary project, ignoring the 5 additional included projects.

3. **Backend needs array-based filtering** - The current filterByProject(projectDir, filter string) signature can't handle the "orch-go special case" where one orchestrator session coordinates work across 6 repos (orch-go, orch-cli, beads, kb-cli, orch-knowledge, opencode).

**Answer to Investigation Question:**

Price-watch agents show in the orch-go dashboard because the multi-project filtering implementation is incomplete. While the frontend correctly fetches and stores the included_projects array from orchestrator context (Finding 1), it only passes the primary project to the API (Finding 1), and the backend only supports single-project matching (Finding 2). The fix requires: (1) Frontend: update buildFilterQueryString to serialize includedProjects as URL params, and (2) Backend: update parseProjectFilter to return []string and filterByProject to check if projectDir matches ANY filter in the slice.

---

## Structured Uncertainty

**What's tested:**

- ✅ Frontend ignores includedProjects (verified: read context.ts:170-172, no mention of includedProjects in buildFilterQueryString)
- ✅ Backend accepts single filter string (verified: read serve_filter.go:48-52, parseProjectFilter returns string not []string)
- ✅ Orchestrator context contains included_projects (verified: curl /api/context shows 6-project array)
- ✅ Price-watch agent appears with orch-go project_dir (verified: curl /api/agents shows pw agent with orch-go dir)

**What's untested:**

- ⚠️ Multi-project filtering will solve the visibility issue (need to implement and verify price-watch agents disappear when not in included_projects)
- ⚠️ Performance impact of checking project_dir against 6-item slice vs single string (likely negligible, but not benchmarked)
- ⚠️ Edge case: what happens when orchestrator has NO included_projects (should fall back to primary project only)

**What would change this:**

- Finding would be wrong if price-watch agents have correct project_dir already (they don't - confirmed via curl)
- Finding would be wrong if buildFilterQueryString includes includedProjects (it doesn't - confirmed via code read)
- Finding would be wrong if backend already supports array filtering (it doesn't - confirmed via signature check)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Multi-project URL parameter serialization** - Serialize includedProjects as comma-separated values in the `project` URL parameter (e.g., `?project=orch-go,orch-cli,beads`)

**Why this approach:**
- RESTful standard pattern - many APIs use comma-separated values for multi-value params
- No schema change needed - reuses existing `project` param instead of introducing `projects` plural
- Backend change is minimal - split on comma, check if projectDir matches ANY value
- Matches how beads API already handles project_dir parameter (consistent design)

**Trade-offs accepted:**
- Can't have project names with commas in them (acceptable - our project names are kebab-case identifiers)
- URL params get longer with 6 projects (acceptable - still well under URL length limits)

**Implementation sequence:**
1. **Frontend first (buildFilterQueryString)** - Join includedProjects with comma, set as `project` param value. This is foundational because it's the data source.
2. **Backend parsing (parseProjectFilter)** - Split param value on comma, return []string instead of string. This enables multi-value handling.
3. **Backend filtering (filterByProject)** - Accept []string instead of string, iterate and return true if ANY match. This applies the actual filtering logic.

### Alternative Approaches Considered

**Option B: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Option C: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Rationale for recommendation:** [Brief synthesis of why Option A beats alternatives given investigation findings]

---

### Implementation Details

**What to implement first:**
- [Highest priority change based on findings]
- [Quick wins or foundational work]
- [Dependencies that need to be addressed early]

**Things to watch out for:**
- ⚠️ [Edge cases or gotchas discovered during investigation]
- ⚠️ [Areas of uncertainty that need validation during implementation]
- ⚠️ [Performance, security, or compatibility concerns to address]

**Areas needing further investigation:**
- [Questions that arose but weren't in scope]
- [Uncertainty areas that might affect implementation]
- [Optional deep-dives that could improve the solution]

**Success criteria:**
- ✅ [How to know the implementation solved the investigated problem]
- ✅ [What to test or validate]
- ✅ [Metrics or observability to add]

---

## References

**Files Examined:**
- [File path] - [What you looked at and why]
- [File path] - [What you looked at and why]

**Commands Run:**
```bash
# [Command description]
[command]

# [Command description]
[command]
```

**External Documentation:**
- [Link or reference] - [What it is and relevance]

**Related Artifacts:**
- **Decision:** [Path to related decision document] - [How it relates]
- **Investigation:** [Path to related investigation] - [How it relates]
- **Workspace:** [Path to related workspace] - [How it relates]

---

## Investigation History

**[YYYY-MM-DD HH:MM]:** Investigation started
- Initial question: [Original question as posed]
- Context: [Why this investigation was initiated]

**[YYYY-MM-DD HH:MM]:** [Milestone or significant finding]
- [Description of what happened or was discovered]

**[YYYY-MM-DD HH:MM]:** Investigation completed
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
