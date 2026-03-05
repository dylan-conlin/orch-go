## Summary (D.E.K.N.)

**Delta:** The "Ready to Complete" section needs escalation-aware partitioning and TLDR-first layout to answer "Can I safely close this, and what did it do?"

**Evidence:** Current UI shows truncated titles, "runtime unknown", "tokens unknown" — all noise. The API already serves Synthesis.TLDR, Outcome, Recommendation, Skill, DeltaSummary, and NextActions. Escalation level (none/info/review/block/failed) exists in pkg/verify/escalation.go but is NOT exposed in the API — currently only computed during `orch complete`.

**Knowledge:** The escalation level is the key safety signal for bulk-close decisions. It can be computed client-side from outcome + recommendation + skill with ~90% accuracy, but the authoritative computation needs server context (workspace files, git diff). A two-phase approach gives immediate value with a clean upgrade path.

**Next:** Implement the redesign as a feature-impl task using the phased approach below. Phase 1 (client-side escalation + new layout) is self-contained and testable.

**Authority:** architectural - Cross-component design (API contract changes, frontend layout, escalation bridging between server and client)

---

# Investigation: Redesign Ready to Complete Section

**Question:** How should the "Ready to Complete" section be redesigned so the user can answer "Can I safely close this, and what did it do?" at a glance?

**Started:** 2026-03-04
**Updated:** 2026-03-04
**Owner:** architect agent
**Phase:** Complete
**Next Step:** Route to feature-impl for implementation
**Status:** Complete

---

## Findings

### Finding 1: Current UI is information-poor and noisy

**Evidence:** The current `ReadyToCompleteItem` interface captures only: id, title, type, priority, runtime, tokenTotal, completionAt, tldr, deltaSummary. The rendered row shows:
- `item.id` (beads ID, 120px fixed width)
- `item.title` (truncated via CSS `truncate`)
- `item.runtime || 'runtime unknown'` — always shows something, even noise
- `formatTokenTotal(item.tokenTotal)` — "tokens unknown" when null
- Completion timestamp
- Close button

The second line (tldr + deltaSummary) only renders if either exists, but tldr is `truncate max-w-[400px]` — often cutting off the most useful information.

**Source:** `web/src/routes/work-graph/+page.svelte:956-989` (the rendering block), lines 104-114 (ReadyToCompleteItem interface)

**Significance:** Two of the five visible data points ("runtime unknown", "tokens unknown") are noise. The most useful data (TLDR — what the agent actually did) is truncated and demoted to a secondary line. The user sees beads IDs and unknown metadata instead of actionable completion context.

---

### Finding 2: Rich synthesis data exists but isn't surfaced

**Evidence:** The `Agent` interface in `agents.ts` (lines 66-68) includes:
- `synthesis.tldr` — one-line summary of what was done
- `synthesis.outcome` — success/partial/blocked/failed
- `synthesis.recommendation` — close/continue/escalate
- `synthesis.delta_summary` — "3 files created, 2 modified, 5 commits"
- `synthesis.next_actions` — follow-up items
- `skill` — what kind of work (investigation, feature-impl, etc.)
- `close_reason` — fallback when synthesis is null

The reactive block building `readyToCompleteItems` (lines 359-404) already iterates over `$agents` and has access to all these fields, but only extracts `tldr` and `deltaSummary`.

**Source:** `web/src/lib/stores/agents.ts:11-17` (Synthesis interface), `web/src/routes/work-graph/+page.svelte:359-404` (reactive block)

**Significance:** The data needed to answer "what did it do?" is already flowing from the server to the frontend. The redesign is primarily a frontend layout + data extraction change, not an API redesign.

---

### Finding 3: Escalation level is server-only — not in API response

**Evidence:** `AgentAPIResponse` in `serve_agents_types.go` has no escalation field. `DetermineEscalation()` in `pkg/verify/escalation.go` is only called from `cmd/orch/complete_cmd.go` and `cmd/orch/daemon.go` during the actual completion flow. The computation requires `EscalationInput` which includes `WorkspacePath`, `ProjectDir`, `HasWebChanges`, `FileCount` — server-side data not available to the frontend.

However, a useful approximation is possible client-side from fields already in the API:
- `outcome != "success"` → review or higher
- `recommendation == "escalate"` → block
- `next_actions.length > 0` → info or review
- `skill` in knowledge-producing set → review
- All clean → none

This covers ~90% of cases. The main gap: visual verification detection (needs filesystem check) and file count thresholds (needs git).

**Source:** `pkg/verify/escalation.go:110-164` (DetermineEscalation), `cmd/orch/serve_agents_types.go:1-65` (no escalation field)

**Significance:** Client-side escalation approximation gives immediate value. Server-side escalation can be added as a follow-up without breaking the UI — the client simply prefers the server value when present.

---

### Finding 4: "Close All" is currently unsafe

**Evidence:** The `acknowledgeAll()` function (lines 656-679) closes every item in `readyToCompleteItems` regardless of outcome, recommendation, or escalation level. It calls `/api/issues/close-batch` with all IDs. There's no filtering — a "failed" completion with recommendation "escalate" gets batch-closed alongside clean successes.

The daemon has a verification pause mechanism (`$daemon?.verification?.is_paused`) that prevents spawning new work until completions are reviewed, but the "Close All" button short-circuits this safety by closing everything without discrimination.

**Source:** `web/src/routes/work-graph/+page.svelte:656-679` (acknowledgeAll), lines 916-921 (Close All button in daemon paused banner)

**Significance:** Bulk-close needs escalation awareness. Items with escalation >= review should not be batch-closeable. The UI should make it impossible to accidentally close something that needs human attention.

---

## Synthesis

**Key Insights:**

1. **TLDR is the primary content** — The TLDR should be the most prominent text, not the beads ID or truncated title. It directly answers "what did it do?" The beads ID is metadata, not the headline.

2. **Escalation level is the key safety signal** — The user needs to know at a glance whether an item is safe to close (none/info) or needs individual review (review/block/failed). This determines both visual treatment and batch-close eligibility.

3. **Hide noise, surface signal** — "runtime unknown" and "tokens unknown" provide negative value. Omit unknown data rather than showing it. Show runtime/tokens only when available, and even then as secondary metadata.

4. **Two-group partition enables safe batch-close** — Splitting items into "safe to close" (escalation none/info, outcome=success) and "needs review" (everything else) makes the Close All button safe by restricting it to the safe group only.

**Answer to Investigation Question:**

The redesign should:
1. Add client-side escalation approximation to partition items into safe/needs-review groups
2. Make TLDR the primary text (with outcome badge for visual scanning)
3. Move from flat list to two-group layout: safe items (batch-closeable) and review items (individual close only)
4. Hide unknown fields instead of showing "unknown" placeholders
5. Add expand/collapse for per-item details (recommendation, next_actions, skill)

---

## Structured Uncertainty

**What's tested:**

- ✅ API already returns synthesis.outcome, synthesis.recommendation, synthesis.next_actions, skill (verified: read serve_agents_handlers.go lines 244-249, 334-339)
- ✅ Frontend Agent interface already includes all synthesis fields (verified: read agents.ts lines 11-17, 66-68)
- ✅ Escalation logic maps cleanly to available client-side data for most cases (verified: read escalation.go DetermineEscalation decision tree)

**What's untested:**

- ⚠️ Client-side escalation approximation accuracy vs server-side (not benchmarked against real completions)
- ⚠️ Visual verification detection gap — client can't know if web/ files were changed (needs server)
- ⚠️ Performance of expanded details rendering with many items (not profiled)

**What would change this:**

- If visual verification is commonly the deciding factor for escalation, client-side approximation would be insufficient and server-side should be prioritized
- If the dashboard typically has 20+ items in ready-to-complete, the two-group layout might need pagination

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Client-side escalation approximation | implementation | Uses existing data, no API changes, reversible |
| Add `escalation_level` to AgentAPIResponse | architectural | API contract change, cross-component (Go + TypeScript) |
| Two-group layout with safe batch-close | implementation | Frontend-only, existing data |
| Hide unknown fields | implementation | CSS/conditional rendering only |

### Recommended Approach ⭐

**Two-Phase Escalation-Aware Redesign** — Phase 1 is client-side only (immediate value); Phase 2 adds server-side escalation (authoritative).

**Why this approach:**
- Phase 1 requires zero API changes — pure frontend, deployable immediately
- Client-side escalation is correct for ~90% of cases (outcome + recommendation + skill covers the common paths)
- Phase 2 adds the server field as an upgrade, not a redesign — client prefers server value when present
- Each phase is independently testable and shippable

**Trade-offs accepted:**
- Phase 1 may misclassify some items (visual verification detection gap)
- Client-side logic is technically a duplicate of server logic (acceptable for Phase 1, resolved in Phase 2)

**Implementation sequence:**

#### Phase 1: Frontend Redesign (feature-impl)

**1a. Extend `ReadyToCompleteItem` interface:**
```typescript
interface ReadyToCompleteItem {
  id: string;
  title: string;
  type: string;
  priority: number;
  skill?: string;              // NEW
  outcome?: string;            // NEW: success/partial/blocked/failed
  recommendation?: string;     // NEW: close/continue/escalate
  nextActions?: string[];      // NEW: follow-up items
  runtime?: string;
  tokenTotal: number | null;
  completionAt: string;
  tldr?: string;
  deltaSummary?: string;
  escalation: 'safe' | 'review' | 'blocked'; // NEW: computed client-side
}
```

**1b. Client-side escalation computation:**
```typescript
function computeEscalation(item: {
  outcome?: string;
  recommendation?: string;
  nextActions?: string[];
  skill?: string;
}): 'safe' | 'review' | 'blocked' {
  // Failed/blocked outcomes need review
  if (item.outcome === 'failed' || item.outcome === 'blocked') return 'blocked';
  if (item.outcome === 'partial') return 'review';

  // Escalate recommendations need review
  if (item.recommendation === 'escalate') return 'blocked';
  if (item.recommendation === 'continue' || item.recommendation === 'resume') return 'review';

  // Knowledge-producing skills need review
  const knowledgeSkills = new Set(['investigation', 'architect', 'research', 'design-session', 'codebase-audit', 'issue-creation']);
  if (item.skill && knowledgeSkills.has(item.skill)) return 'review';

  // Has follow-up actions → informational (still safe to close, but notable)
  // These are escalation=info in server terms — safe to batch-close

  return 'safe';
}
```

**1c. New layout structure:**

```
┌─────────────────────────────────────────────────────────────────┐
│ Ready to Complete                          3 awaiting review    │
├─────────────────────────────────────────────────────────────────┤
│ ┌─ NEEDS REVIEW (amber border, if any) ──────────────────────┐ │
│ │ ⚠ partial  TLDR text here full width no truncation...      │ │
│ │   architect · orch-go-abc1 · 3 files, 2 modified · 12m ago │ │
│ │   ▸ 2 follow-up actions                          [Close]   │ │
│ ├────────────────────────────────────────────────────────────-┤ │
│ │ ✗ failed   TLDR text describing what went wrong...         │ │
│ │   feature-impl · orch-go-abc2 · 5m ago           [Close]   │ │
│ └────────────────────────────────────────────────────────────-┘ │
│ ┌─ SAFE TO CLOSE (green border) ─────────── [Close All (4)] ─┐ │
│ │ ✓ success  Implemented auth middleware for API endpoints    │ │
│ │   feature-impl · orch-go-def3 · 5 files, 3 mod · 20m ago  │ │
│ ├─────────────────────────────────────────────────────────────┤ │
│ │ ✓ success  Fixed race condition in token refresh            │ │
│ │   systematic-debugging · orch-go-ghi4 · 8m ago    [Close]  │ │
│ └─────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

**Key layout decisions:**
- **Line 1 (primary):** Outcome badge + TLDR (full width, no truncation — wrap if needed)
- **Line 2 (metadata):** Skill · Beads ID · DeltaSummary · Relative time
- **Line 3 (expandable, only if next_actions exist):** Expand arrow showing follow-up actions
- **Close button:** Right-aligned on metadata line
- **"Close All" button:** Only on the "Safe to Close" group header — never on "Needs Review"
- **Outcome badge colors:** success=green, partial=amber, blocked=amber, failed=red, missing=gray
- **Hide when null:** runtime, tokens, deltaSummary — just omit, don't show "unknown"

**1d. Expand/collapse per item:**
Clicking the expand arrow on an item with next_actions shows:
- Next actions list (bulleted)
- Recommendation text (if not "close")
- Runtime + token count (if available)

#### Phase 2: Server-side escalation (follow-up task)

**2a. Add `EscalationLevel` to `AgentAPIResponse`:**
```go
type AgentAPIResponse struct {
    // ... existing fields ...
    EscalationLevel string `json:"escalation_level,omitempty"` // none, info, review, block, failed
}
```

**2b. Compute during agent list assembly:**
In `serve_agents_handlers.go`, after synthesis is parsed, compute escalation:
```go
if agents[i].Synthesis != nil && phaseComplete {
    input := verify.EscalationInput{
        VerificationPassed: true, // Phase: Complete implies verification ran
        SkillName:          agents[i].Skill,
        Outcome:            agents[i].Synthesis.Outcome,
        Recommendation:     agents[i].Synthesis.Recommendation,
        NextActions:        agents[i].Synthesis.NextActions,
        WorkspacePath:      workspacePath,
        ProjectDir:         projectDir,
    }
    agents[i].EscalationLevel = verify.DetermineEscalation(input).String()
}
```

**2c. Frontend prefers server value:**
```typescript
function computeEscalation(item): 'safe' | 'review' | 'blocked' {
    // Prefer server-computed escalation when available
    if (item.serverEscalation) {
        if (item.serverEscalation === 'block' || item.serverEscalation === 'failed') return 'blocked';
        if (item.serverEscalation === 'review') return 'review';
        return 'safe';
    }
    // Fallback to client-side approximation
    // ... existing logic ...
}
```

### Alternative Approaches Considered

**Option B: Server-side only (compute escalation only on server)**
- **Pros:** Single source of truth, no logic duplication
- **Cons:** Requires API change before any UI improvement can ship; blocks frontend work on backend deployment
- **When to use instead:** If Phase 2 is prioritized and shipped first

**Option C: Flat list with color-coded left border (no two-group split)**
- **Pros:** Simpler layout, less visual complexity
- **Cons:** "Close All" remains unsafe unless we add a confirmation dialog. Doesn't solve the batch-close safety problem structurally.
- **When to use instead:** If the typical ready-to-complete list is 1-3 items (rarely needs batch close)

**Rationale for recommendation:** The two-group layout makes batch-close safety a structural property of the UI rather than relying on user discipline. The phased approach (client-side first, server upgrade second) delivers value immediately while maintaining an upgrade path.

---

### Implementation Details

**What to implement first:**
- Extract more fields from `$agents` into `ReadyToCompleteItem` (skill, outcome, recommendation, nextActions)
- Add `computeEscalation()` function
- Split `readyToCompleteItems` into two derived arrays: `safeItems` and `reviewItems`
- Redesign the rendering block with the two-group layout

**Things to watch out for:**
- ⚠️ Tab indentation in Svelte files — use `cat -vet` before Edit tool operations (per CLAUDE.md)
- ⚠️ The `max-h-48` constraint on the list container may need adjustment for the two-group layout
- ⚠️ Agents without synthesis (synthesis=null) should be treated as "review" (unknown state = needs attention)
- ⚠️ Defect Class 1 (Filter Amnesia): ensure the `readyToCompleteIds` exclusion set still works correctly after splitting into two groups

**Areas needing further investigation:**
- Whether `NeedsVisualApproval` is commonly the deciding factor for escalation (would prioritize Phase 2)
- Optimal expand/collapse UX for next_actions (accordion vs inline)

**Success criteria:**
- ✅ User can answer "what did it do?" at a glance (TLDR visible without truncation)
- ✅ User can distinguish safe-to-close from needs-review items visually
- ✅ "Close All" only affects safe items (structural safety)
- ✅ Unknown fields (runtime, tokens) are hidden, not displayed as "unknown"
- ✅ Outcome badge provides instant visual triage (green/amber/red)

---

## File Targets

| File | Action | Description |
|------|--------|-------------|
| `web/src/routes/work-graph/+page.svelte` | Modify | Extend ReadyToCompleteItem, add computeEscalation, split into groups, redesign rendering |
| `cmd/orch/serve_agents_types.go` | Modify (Phase 2) | Add EscalationLevel field to AgentAPIResponse |
| `cmd/orch/serve_agents_handlers.go` | Modify (Phase 2) | Compute escalation during agent list assembly |
| `web/src/lib/stores/agents.ts` | No change needed | Synthesis interface already has all needed fields |

---

## References

**Files Examined:**
- `web/src/routes/work-graph/+page.svelte` — Current "Ready to Complete" rendering and data flow
- `cmd/orch/serve_agents_types.go` — AgentAPIResponse and SynthesisResponse structs
- `pkg/verify/escalation.go` — Escalation level definitions and computation logic
- `cmd/orch/serve_agents_handlers.go:220-350` — How agent responses are assembled (synthesis parsing)
- `web/src/lib/stores/agents.ts` — Agent and Synthesis TypeScript interfaces

**Related Artifacts:**
- **Decision:** Escalation system design is implicit in `pkg/verify/escalation.go` — no formal decision record found
- **Investigation:** This is the first investigation specifically about the Ready to Complete UI
