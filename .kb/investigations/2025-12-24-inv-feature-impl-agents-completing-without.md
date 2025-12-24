<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** feature-impl agents don't create SYNTHESIS.md because they use "light tier" by default, which explicitly skips the synthesis requirement.

**Evidence:** 199/378 workspaces have SYNTHESIS.md (~53%); config.go:21-34 explicitly sets `feature-impl: TierLight`; SPAWN_CONTEXT.md for light tier says "SYNTHESIS.md is NOT required".

**Knowledge:** The tiering system is working as designed - it's a deliberate optimization, not a bug. Agents creating synthesis anyway (53%) are either full-tier skills or agents ignoring the "skip" instruction.

**Next:** Close - this is working as designed per .kb/decisions/2025-12-22-template-ownership-model.md. If synthesis is desired for feature-impl, use `--full` flag.

**Confidence:** Very High (95%) - code is explicit and documented; observed behavior matches design.

---

# Investigation: Feature-Impl Agents Completing Without SYNTHESIS.md

**Question:** Why are feature-impl agents completing without creating SYNTHESIS.md, causing the dashboard to show no TLDR for completed agents?

**Started:** 2025-12-24
**Updated:** 2025-12-24
**Owner:** og-debug-feature-impl-agents-24dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** Very High (95%)

---

## Findings

### Finding 1: The Tiering System is Explicit and Working as Designed

**Evidence:** `pkg/spawn/config.go:21-34` explicitly defines tier defaults:

```go
var SkillTierDefaults = map[string]string{
    // Full tier: Investigation-type skills that produce knowledge artifacts
    "investigation":        TierFull,
    "architect":            TierFull,
    "research":             TierFull,
    "codebase-audit":       TierFull,
    "design-session":       TierFull,
    "systematic-debugging": TierFull,

    // Light tier: Implementation-focused skills
    "feature-impl":        TierLight,  // <-- This is the key setting
    "reliability-testing": TierLight,
    "issue-creation":      TierLight,
}
```

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/config.go:21-34`

**Significance:** The light tier for feature-impl is a deliberate design decision, not an oversight. The code explicitly categorizes "implementation-focused skills" as light tier.

---

### Finding 2: SPAWN_CONTEXT.md Explicitly Tells Light-Tier Agents to Skip Synthesis

**Evidence:** `pkg/spawn/context.go:22-27` and `context.go:43-47` show the template:

```go
{{if eq .Tier "light"}}
⚡ LIGHT TIER: This is a lightweight spawn. SYNTHESIS.md is NOT required.
   Focus on completing the task efficiently. Skip session synthesis documentation.
{{else}}
📚 FULL TIER: This spawn requires SYNTHESIS.md for knowledge externalization.
   Document your findings, decisions, and learnings in SYNTHESIS.md before completing.
{{end}}
```

And in the completion protocol:

```go
{{if eq .Tier "light"}}
1. Run: `bd comment {{.BeadsID}} "Phase: Complete - [1-2 sentence summary of deliverables]"`
2. Run: `/exit` to close the agent session

⚡ LIGHT TIER: SYNTHESIS.md is NOT required for this spawn.
{{else}}
1. Ensure SYNTHESIS.md is created and committed in your workspace.
2. Run: `bd comment {{.BeadsID}} "Phase: Complete - [1-2 sentence summary of deliverables]"`
3. Run: `/exit` to close the agent session
{{end}}
```

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/context.go:18-196`

**Significance:** Agents are following explicit instructions. The SPAWN_CONTEXT.md they receive tells them SYNTHESIS.md is NOT required.

---

### Finding 3: Dashboard TLDR Display Works Correctly for Full-Tier Agents

**Evidence:** 
- 199 out of 378 workspaces have SYNTHESIS.md (~53%)
- The dashboard correctly parses and displays TLDR from SYNTHESIS.md via `verify.ParseSynthesis()`
- Completed workspaces with SYNTHESIS.md appear correctly in the dashboard with TLDR displayed

Sample verified workspaces with SYNTHESIS.md:
- og-feat-fix-pre-spawn-22dec (has SYNTHESIS.md, 2,584 bytes)
- og-debug-fix-beads-database-22dec
- og-inv-orchestrator-skill-says-24dec

**Source:** 
- `find .orch/workspace -name "SYNTHESIS.md" | wc -l` → 199
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/serve.go:291-333` (synthesis parsing)
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/verify/check.go:142-191` (ParseSynthesis implementation)

**Significance:** The system is working correctly. Agents that create SYNTHESIS.md have their TLDR displayed. Agents that skip it (per light-tier instructions) don't.

---

### Finding 4: Prior Decision Documents This Design Choice

**Evidence:** The spawn context I received references:
- `.kb/decisions/2025-12-22-template-ownership-model.md` - Template ownership decision
- Prior Decision: "Progressive disclosure for skill bloat" - mentions 89% of feature-impl spawns use only 2-3 phases, reducing 1757→~500 lines

**Source:** SPAWN_CONTEXT.md lines 32-37, referencing prior decisions

**Significance:** This is a documented, intentional optimization for the common case where feature-impl agents don't need heavy documentation overhead.

---

## Synthesis

**Key Insights:**

1. **This is working as designed, not a bug** - The tiering system was deliberately implemented to reduce overhead for implementation-focused skills.

2. **Light tier is an optimization** - For agents that primarily produce code (feature-impl, reliability-testing), the synthesis step adds overhead without proportional value.

3. **Dashboard shows TLDR when available** - The 53% of workspaces with SYNTHESIS.md display correctly. The "missing TLDR" is intentional for light-tier agents.

**Answer to Investigation Question:**

Feature-impl agents complete without SYNTHESIS.md because they default to "light tier" (`pkg/spawn/config.go:31`), which explicitly skips the synthesis requirement. The SPAWN_CONTEXT.md template tells them "SYNTHESIS.md is NOT required" for light tier spawns. This is working as designed per the progressive disclosure optimization documented in prior decisions.

If synthesis is desired for a specific feature-impl spawn, use `orch spawn --full feature-impl "task"` to override the default tier.

---

## Confidence Assessment

**Current Confidence:** Very High (95%)

**Why this level?**

The code is explicit, well-documented, and observed behavior matches design intent. The tiering system has clear comments explaining the rationale.

**What's certain:**

- ✅ `feature-impl` defaults to `TierLight` (line 31 of config.go)
- ✅ Light tier SPAWN_CONTEXT.md says "SYNTHESIS.md is NOT required"
- ✅ Dashboard correctly parses and displays SYNTHESIS.md when present
- ✅ 199/378 workspaces have SYNTHESIS.md (agents following full-tier or ignoring light-tier skip)

**What's uncertain:**

- ⚠️ Whether 53% synthesis rate is "good enough" for orchestrator visibility
- ⚠️ Whether some feature-impl work should require synthesis (complex features, architectural changes)

**What would increase confidence to 100%:**

- Review with project owner to confirm this matches intent
- Check if any feature-impl agents with `--full` flag create expected synthesis

---

## Implementation Recommendations

**Purpose:** Clarify options if the behavior change is desired.

### Option A: Accept Current Behavior (Recommended) ⭐

**No code changes needed** - This is working as designed.

**Why this approach:**
- Reduces overhead for implementation-focused work
- Agents that need synthesis can use `--full` flag
- 53% synthesis rate means important work is likely being captured

**Trade-offs accepted:**
- Dashboard won't show TLDR for light-tier completed agents
- Orchestrator needs other signals (beads comments, git commits) for visibility

### Option B: Make feature-impl Full Tier by Default

Change `config.go:31` from `TierLight` to `TierFull`:

```go
"feature-impl": TierFull,  // Changed from TierLight
```

- **Pros:** All feature-impl agents would create SYNTHESIS.md, better dashboard visibility
- **Cons:** More overhead for simple tasks, goes against progressive disclosure optimization
- **When to use instead:** If orchestrator visibility is more important than agent efficiency

### Option C: Add TLDR Without Full Synthesis

Create a "micro synthesis" for light-tier agents - just TLDR from Phase: Complete comment.

- **Pros:** Gets TLDR without full synthesis overhead
- **Cons:** Requires code changes to parse beads comments for TLDR
- **When to use instead:** If dashboard visibility is important but full synthesis is overkill

**Rationale for recommendation:** Current behavior is documented and intentional. If change is needed, it should be a deliberate decision with orchestrator approval.

---

## References

**Files Examined:**
- `pkg/spawn/config.go:21-34` - Tier defaults for skills
- `pkg/spawn/context.go:18-196` - SPAWN_CONTEXT.md template with tier-conditional synthesis instructions
- `cmd/orch/serve.go:291-333` - Dashboard synthesis parsing
- `pkg/verify/check.go:142-191` - ParseSynthesis implementation
- `.orch/workspace/og-feat-add-focus-drift-24dec/.tier` - Contains "light"
- `.orch/workspace/og-feat-fix-pre-spawn-22dec/SYNTHESIS.md` - Example of agent that created synthesis

**Commands Run:**
```bash
# Count workspaces with SYNTHESIS.md
find .orch/workspace -name "SYNTHESIS.md" -type f | wc -l
# Result: 199

# Count total workspaces
ls .orch/workspace | wc -l
# Result: 378

# Check tier file for a feature-impl workspace
cat .orch/workspace/og-feat-add-focus-drift-24dec/.tier
# Result: light
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2025-12-22-template-ownership-model.md` - Template ownership
- **Prior Knowledge:** "Progressive disclosure for skill bloat" decision

---

## Investigation History

**2025-12-24 14:35:** Investigation started
- Initial question: Why are feature-impl agents completing without SYNTHESIS.md?
- Context: Dashboard shows no TLDR for completed feature-impl agents

**2025-12-24 14:45:** Root cause identified
- Found explicit tier defaults in config.go
- Confirmed light tier skips synthesis requirement

**2025-12-24 14:55:** Investigation completed
- Final confidence: Very High (95%)
- Status: Complete
- Key outcome: This is working as designed - feature-impl uses light tier which skips SYNTHESIS.md
